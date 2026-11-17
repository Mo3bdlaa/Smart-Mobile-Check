package iphone

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	ios "github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/afc"
	"github.com/danielpaulus/go-ios/ios/installationproxy"
	"github.com/danielpaulus/go-ios/ios/mcinstall"
)

// DeviceBrief is a connected device as shown in the device picker.
type DeviceBrief struct {
	UDID           string `json:"udid"`
	ConnectionType string `json:"connectionType"`
	ProductType    string `json:"productType,omitempty"`
	ProductVersion string `json:"productVersion,omitempty"`
	DeviceClass    string `json:"deviceClass,omitempty"`
	DeviceName     string `json:"deviceName,omitempty"`
	ModelName      string `json:"modelName,omitempty"`
	Paired         bool   `json:"paired"`
	PairState      string `json:"pairState"`
	PairMessage    string `json:"pairMessage,omitempty"`
}

// ProgressFunc receives scan progress (step index, total, label).
type ProgressFunc func(step, total int, label string)

// ServiceAvailable reports whether the Apple USB multiplexer service
// (Apple Mobile Device Service on Windows, usbmuxd elsewhere) is reachable.
func ServiceAvailable() error {
	_, err := withTimeout(4*time.Second, func() (ios.DeviceList, error) {
		return ios.ListDevices()
	})
	return err
}

// ListDevices returns the connected devices with basic, pre-pairing info.
func ListDevices() ([]DeviceBrief, error) {
	list, err := withTimeout(6*time.Second, func() (ios.DeviceList, error) { return ios.ListDevices() })
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []DeviceBrief
	for _, d := range list.DeviceList {
		udid := d.Properties.SerialNumber
		if seen[udid] {
			// Same device over USB and Wi-Fi: keep the first (USB is listed first).
			continue
		}
		seen[udid] = true
		b := DeviceBrief{UDID: udid, ConnectionType: d.Properties.ConnectionType}
		vals, _ := withTimeout(5*time.Second, func() (map[string]any, error) { return unpairedValues(d) })
		b.ProductType = str(vals["ProductType"])
		b.ProductVersion = str(vals["ProductVersion"])
		b.DeviceClass = str(vals["DeviceClass"])
		b.DeviceName = str(vals["DeviceName"])
		b.PairState, b.PairMessage = pairState(d)
		b.Paired = b.PairState == "paired"
		out = append(out, b)
	}
	return out, nil
}

// FindDevice returns the usbmux entry for a UDID (USB preferred).
func FindDevice(udid string) (ios.DeviceEntry, error) {
	list, err := withTimeout(6*time.Second, func() (ios.DeviceList, error) { return ios.ListDevices() })
	if err != nil {
		return ios.DeviceEntry{}, err
	}
	var fallback *ios.DeviceEntry
	for i, d := range list.DeviceList {
		if d.Properties.SerialNumber != udid {
			continue
		}
		if strings.EqualFold(d.Properties.ConnectionType, "USB") {
			return d, nil
		}
		if fallback == nil {
			fallback = &list.DeviceList[i]
		}
	}
	if fallback != nil {
		return *fallback, nil
	}
	return ios.DeviceEntry{}, fmt.Errorf("device %s is no longer connected", udid)
}

// unpairedValues reads the handful of values lockdown exposes without a
// trusted session (ProductType, ProductVersion, ...).
func unpairedValues(d ios.DeviceEntry) (map[string]any, error) {
	mux, err := ios.NewUsbMuxConnectionSimple()
	if err != nil {
		return nil, err
	}
	defer mux.Close()
	ld, err := mux.ConnectLockdown(d.DeviceID)
	if err != nil {
		return nil, err
	}
	defer ld.Close()
	out := map[string]any{}
	for _, k := range []string{"ProductType", "ProductVersion", "DeviceClass", "DeviceName"} {
		if v, err := ld.GetValue(k); err == nil && v != nil {
			out[k] = v
		}
	}
	return out, nil
}

// pairState tells whether we hold a working, trusted pairing for the device.
func pairState(d ios.DeviceEntry) (string, string) {
	_, err := ios.ReadPairRecord(d.Properties.SerialNumber)
	if err != nil {
		return "unpaired", "This computer is not trusted by the iPhone yet."
	}
	_, err = withTimeout(6*time.Second, func() (bool, error) {
		ld, err := ios.ConnectLockdownWithSession(d)
		if err != nil {
			return false, err
		}
		ld.Close()
		return true, nil
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "InvalidHostID") || strings.Contains(msg, "PairingDialog") {
			return "unpaired", "The saved trust record is no longer valid. Trust this computer again."
		}
		if strings.Contains(msg, "PasswordProtected") {
			return "locked", "Unlock the iPhone to continue."
		}
		return "unpaired", "Trust check failed: " + msg
	}
	return "paired", ""
}

// Pair asks the device to trust this computer. The first call shows the
// "Trust This Computer?" prompt on the iPhone; the UI keeps calling until the
// user accepts.
func Pair(udid string) (string, string) {
	d, err := FindDevice(udid)
	if err != nil {
		return "error", err.Error()
	}
	if st, _ := pairState(d); st == "paired" {
		return "paired", "Already trusted."
	}
	_, err = withTimeout(25*time.Second, func() (bool, error) { return true, ios.Pair(d) })
	if err == nil {
		return "paired", "Trusted."
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "PairingDialog"):
		return "waiting", "Tap \"Trust\" on the iPhone and enter its passcode."
	case strings.Contains(msg, "PasswordProtected"):
		return "locked", "Unlock the iPhone (enter its passcode), then keep it on the Home Screen."
	case strings.Contains(msg, "UserDeniedPairing"):
		return "denied", "\"Don't Trust\" was selected. Unplug and reconnect the cable, then tap \"Trust\"."
	default:
		return "error", msg
	}
}

// Collect reads everything we can from the device. Every step is isolated:
// a failure is recorded in Snapshot.Errors and the scan carries on.
func Collect(udid string, progress ProgressFunc) (*Snapshot, error) {
	d, err := FindDevice(udid)
	if err != nil {
		return nil, err
	}
	snap := &Snapshot{
		CollectedAt:    time.Now(),
		UDID:           udid,
		ConnectionType: d.Properties.ConnectionType,
		Domains:        map[string]map[string]any{},
		Errors:         map[string]string{},
	}
	steps := []struct {
		label string
		fn    func() error
		limit time.Duration
		key   string
	}{
		{"Reading device identity", func() error { return collectLockdown(d, snap) }, 25 * time.Second, "lockdown"},
		{"Reading battery gauge", func() error { return collectBattery(d, snap) }, 20 * time.Second, "battery"},
		{"Running hardware diagnostics", func() error { return collectDiagnostics(d, snap) }, 20 * time.Second, "diagnostics"},
		{"Measuring storage", func() error { return collectFS(d, snap) }, 15 * time.Second, "storage"},
		{"Auditing installed apps", func() error { return collectApps(d, snap) }, 60 * time.Second, "apps"},
		{"Checking management & profiles", func() error { return collectProfiles(d, snap) }, 20 * time.Second, "profiles"},
		{"Probing for jailbreak services", func() error { return probeJailbreak(d, snap) }, 15 * time.Second, "jailbreak"},
		{"Analysing crash & panic logs", func() error { return collectCrashes(d, snap) }, 90 * time.Second, "crashes"},
	}
	total := len(steps) + 1
	for i, s := range steps {
		if progress != nil {
			progress(i+1, total, s.label)
		}
		_, err := withTimeout(s.limit, func() (bool, error) { return true, s.fn() })
		if err != nil {
			snap.Errors[s.key] = err.Error()
			if s.key == "lockdown" {
				return snap, fmt.Errorf("could not read the device: %w", err)
			}
		}
	}
	if progress != nil {
		progress(total, total, "Evaluating results")
	}
	return snap, nil
}

var lockdownDomains = []string{
	"com.apple.disk_usage",
	"com.apple.disk_usage.factory",
	"com.apple.mobile.battery",
	"com.apple.fmip",
	"com.apple.international",
	"com.apple.PurpleBuddy",
	"com.apple.mobile.data_sync",
	"com.apple.security.mac.amfi",
	"com.apple.mobile.wireless_lockdown",
	"com.apple.mobile.restriction",
	"com.apple.mobile.iTunes",
	"com.apple.mobile.lockdown_cache",
}

func collectLockdown(d ios.DeviceEntry, snap *Snapshot) error {
	ld, err := ios.ConnectLockdownWithSession(d)
	if err != nil {
		return err
	}
	defer ld.Close()
	all, err := lockdownGet(ld, "", "")
	if err != nil {
		return err
	}
	m, ok := all.(map[string]any)
	if !ok {
		return errors.New("unexpected lockdown response")
	}
	snap.Lockdown = m
	for _, dom := range lockdownDomains {
		v, err := lockdownGet(ld, dom, "")
		if err != nil {
			continue
		}
		if mm, ok := v.(map[string]any); ok && len(mm) > 0 {
			snap.Domains[dom] = mm
		}
	}
	// A few keys are not part of the "all values" dictionary on every iOS.
	for _, k := range []string{"RegulatoryModelNumber", "ProductionSOC", "ActivationState", "BrickState", "IntegratedCircuitCardIdentity", "kCTPostponementStatus"} {
		if _, ok := snap.Lockdown[k]; ok {
			continue
		}
		if v, err := lockdownGet(ld, "", k); err == nil && v != nil {
			snap.Lockdown[k] = v
		}
	}
	return nil
}

func lockdownGet(ld *ios.LockDownConnection, domain, key string) (any, error) {
	req := map[string]any{"Label": "smartcheck", "Request": "GetValue"}
	if domain != "" {
		req["Domain"] = domain
	}
	if key != "" {
		req["Key"] = key
	}
	if err := ld.Send(req); err != nil {
		return nil, err
	}
	resp, err := ld.ReadMessage()
	if err != nil {
		return nil, err
	}
	p, err := ios.ParsePlist(resp)
	if err != nil {
		return nil, err
	}
	if e, ok := p["Error"]; ok {
		return nil, fmt.Errorf("%v", e)
	}
	return p["Value"], nil
}

// diagRequest sends one request to diagnostics_relay over a fresh connection.
func diagRequest(d ios.DeviceEntry, req map[string]any) (map[string]any, error) {
	conn, err := ios.ConnectToService(d, "com.apple.mobile.diagnostics_relay")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	codec := ios.NewPlistCodec()
	b, err := codec.Encode(req)
	if err != nil {
		return nil, err
	}
	if err := conn.Send(b); err != nil {
		return nil, err
	}
	resp, err := codec.Decode(conn.Reader())
	if err != nil {
		return nil, err
	}
	p, err := ios.ParsePlist(resp)
	if err != nil {
		return nil, err
	}
	// Say goodbye politely; ignore failures.
	if gb, err := codec.Encode(map[string]any{"Request": "Goodbye"}); err == nil {
		_ = conn.Send(gb)
		_, _ = codec.Decode(conn.Reader())
	}
	if st, _ := p["Status"].(string); st != "" && st != "Success" {
		return p, fmt.Errorf("diagnostics status: %s", st)
	}
	return p, nil
}

func ioreg(d ios.DeviceEntry, field, value string) (map[string]any, error) {
	p, err := diagRequest(d, map[string]any{"Request": "IORegistry", field: value})
	if err != nil {
		return nil, err
	}
	diag, _ := p["Diagnostics"].(map[string]any)
	reg, _ := diag["IORegistry"].(map[string]any)
	if len(reg) == 0 {
		return nil, errors.New("empty IORegistry entry")
	}
	return reg, nil
}

func collectBattery(d ios.DeviceEntry, snap *Snapshot) error {
	tries := []struct{ field, value string }{
		{"EntryName", "AppleSmartBattery"},
		{"EntryClass", "IOPMPowerSource"},
		{"EntryName", "AppleARMPMUCharger"},
	}
	var lastErr error
	for _, t := range tries {
		reg, err := ioreg(d, t.field, t.value)
		if err != nil {
			lastErr = err
			continue
		}
		if _, ok := reg["DesignCapacity"]; !ok {
			if _, ok2 := reg["CycleCount"]; !ok2 {
				lastErr = fmt.Errorf("%s has no battery data", t.value)
				continue
			}
		}
		snap.Battery = reg
		snap.BatterySource = t.value
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("battery registry not available")
	}
	return lastErr
}

func collectDiagnostics(d ios.DeviceEntry, snap *Snapshot) error {
	var errs []string
	if p, err := diagRequest(d, map[string]any{"Request": "All"}); err == nil {
		if diag, ok := p["Diagnostics"].(map[string]any); ok {
			snap.Diagnostics = diag
		}
	} else {
		errs = append(errs, "all: "+err.Error())
	}
	keys := []string{
		"ProductType", "SerialNumber", "UniqueChipID", "ChipID", "HardwarePlatform", "BoardId",
		"RegionCode", "RegionInfo", "ModelNumber", "MainScreenWidth", "MainScreenHeight",
		"main-screen-width", "main-screen-height", "DeviceColor", "DeviceEnclosureColor",
		"HasBaseband", "BasebandSerialNumber", "ArtworkTraits", "PhysicalHardwareNameString",
		"BatteryIsFullyCharged", "BatteryCurrentCapacity", "DiskUsage", "InternalBuild",
		"IsUIBuild", "ReleaseType", "BuildVersion", "ProductVersion",
	}
	if p, err := diagRequest(d, map[string]any{"Request": "MobileGestalt", "MobileGestaltKeys": keys}); err == nil {
		if diag, ok := p["Diagnostics"].(map[string]any); ok {
			if mg, ok := diag["MobileGestalt"].(map[string]any); ok {
				snap.Gestalt = mg
			}
		}
	} else {
		errs = append(errs, "gestalt: "+err.Error())
	}
	if len(errs) == 2 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func collectFS(d ios.DeviceEntry, snap *Snapshot) error {
	c, err := afc.New(d)
	if err != nil {
		return err
	}
	defer c.Close()
	info, err := c.DeviceInfo()
	if err != nil {
		return err
	}
	snap.FS = &FSInfo{Model: info.Model, TotalBytes: info.TotalBytes, FreeBytes: info.FreeBytes}
	return nil
}

func collectApps(d ios.DeviceEntry, snap *Snapshot) error {
	conn, err := installationproxy.New(d)
	if err != nil {
		return err
	}
	defer conn.Close()
	apps, err := conn.BrowseAllApps()
	if err != nil {
		return err
	}
	for _, a := range apps {
		e := AppEntry{
			BundleID: a.CFBundleIdentifier(),
			Name:     firstNonEmpty(str(a["CFBundleDisplayName"]), a.CFBundleName()),
			Version:  a.CFBundleShortVersionString(),
			Type:     str(a["ApplicationType"]),
			Signer:   str(a["SignerIdentity"]),
		}
		snap.Apps = append(snap.Apps, e)
	}
	sort.Slice(snap.Apps, func(i, j int) bool { return strings.ToLower(snap.Apps[i].Name) < strings.ToLower(snap.Apps[j].Name) })
	snap.AppsScanned = true
	return nil
}

func collectProfiles(d ios.DeviceEntry, snap *Snapshot) error {
	conn, err := mcinstall.New(d)
	if err != nil {
		return err
	}
	defer conn.Close()
	var errs []string
	if cfg, err := conn.GetCloudConfiguration(); err == nil {
		if inner, ok := cfg["CloudConfiguration"].(map[string]any); ok {
			snap.CloudConfig = inner
		} else {
			snap.CloudConfig = cfg
		}
	} else {
		errs = append(errs, err.Error())
	}
	if list, err := conn.HandleList(); err == nil {
		snap.ProfilesRead = true
		for _, p := range list {
			snap.Profiles = append(snap.Profiles, ProfileEntry{
				Identifier:        p.Identifier,
				DisplayName:       p.Metadata.PayloadDisplayName,
				Description:       firstNonEmpty(p.Metadata.PayloadDescription, p.Manifest.Description),
				RemovalDisallowed: p.Metadata.PayloadRemovalDisallowed,
			})
		}
	} else {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 && !snap.ProfilesRead && snap.CloudConfig == nil {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func probeJailbreak(d ios.DeviceEntry, snap *Snapshot) error {
	// "com.apple.afc2" gives full filesystem access and only exists on
	// jailbroken devices (installed by jailbreak tools / "afc2add").
	snap.AFC2Probed = true
	ld, err := ios.ConnectLockdownWithSession(d)
	if err != nil {
		snap.AFC2Probed = false
		return err
	}
	defer ld.Close()
	resp, err := ld.StartService("com.apple.afc2")
	if err == nil && resp.Port != 0 {
		snap.AFC2Open = true
	}
	return nil
}

var (
	panicFileRe      = regexp.MustCompile(`(?i)^panic-(full|base)`)
	missingSensorsRe = regexp.MustCompile(`(?i)missing sensor\(s\):\s*([A-Za-z0-9_, ]+)`)
)

func collectCrashes(d ios.DeviceEntry, snap *Snapshot) error {
	// Ask crash_mover to flush pending reports into the readable area.
	if mover, err := ios.ConnectToService(d, "com.apple.crashreportmover"); err == nil {
		buf := make([]byte, 4)
		_ = mover.Conn().SetReadDeadline(time.Now().Add(8 * time.Second))
		_, _ = mover.Reader().Read(buf)
		mover.Close()
	}
	conn, err := ios.ConnectToService(d, "com.apple.crashreportcopymobile")
	if err != nil {
		return err
	}
	defer conn.Close()
	client := afc.NewFromConn(conn)
	var panics []string
	err = client.WalkDir(".", func(p string, info afc.FileInfo, err error) error {
		if err != nil || info.Type == afc.S_IFDIR {
			return nil
		}
		snap.CrashReports = append(snap.CrashReports, p)
		if panicFileRe.MatchString(filepath.Base(p)) {
			panics = append(panics, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	snap.CrashesRead = true
	sort.Sort(sort.Reverse(sort.StringSlice(panics)))
	if len(panics) > 15 {
		panics = panics[:15]
	}
	tmp, err := os.MkdirTemp("", "smartcheck-panics-")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(tmp)
	for i, p := range panics {
		dst := filepath.Join(tmp, fmt.Sprintf("%02d.ips", i))
		pl := PanicLog{File: filepath.Base(p)}
		if err := client.PullSingleFile(p, dst); err == nil {
			if raw, err := os.ReadFile(dst); err == nil {
				parsePanic(raw, &pl)
			}
		}
		snap.Panics = append(snap.Panics, pl)
	}
	return nil
}

// parsePanic extracts the useful parts of an .ips panic report (a JSON header
// line followed by a JSON body).
func parsePanic(raw []byte, pl *PanicLog) {
	text := string(raw)
	if nl := strings.IndexByte(text, '\n'); nl > 0 {
		var hdr map[string]any
		if json.Unmarshal([]byte(text[:nl]), &hdr) == nil {
			pl.Timestamp = str(hdr["timestamp"])
			pl.OSVersion = str(hdr["os_version"])
		}
		var body map[string]any
		if json.Unmarshal([]byte(text[nl+1:]), &body) == nil {
			if ps := str(body["panicString"]); ps != "" {
				text = ps
			}
		}
	}
	first := strings.TrimSpace(text)
	if i := strings.Index(first, "\n"); i > 0 {
		first = first[:i]
	}
	if len(first) > 220 {
		first = first[:220] + "…"
	}
	pl.Summary = first
	if m := missingSensorsRe.FindStringSubmatch(text); m != nil {
		for _, s := range strings.Split(m[1], ",") {
			if s = strings.TrimSpace(s); s != "" {
				pl.MissingSensors = append(pl.MissingSensors, s)
			}
		}
	}
}

// withTimeout runs fn and gives up after d. go-ios calls can block forever
// on a misbehaving connection, so every device call goes through here.
func withTimeout[T any](d time.Duration, fn func() (T, error)) (T, error) {
	type res struct {
		v   T
		err error
	}
	ch := make(chan res, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				var zero T
				ch <- res{zero, fmt.Errorf("internal error: %v", r)}
			}
		}()
		v, err := fn()
		ch <- res{v, err}
	}()
	select {
	case r := <-ch:
		return r.v, r.err
	case <-time.After(d):
		var zero T
		return zero, fmt.Errorf("timed out after %s", d)
	}
}
