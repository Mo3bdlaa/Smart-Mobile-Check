// Package android inspects Android phones over ADB: it reads system
// properties, accounts, device policy, battery, sensors, storage and crash
// history (all read-only shell commands that work without root), then runs
// authenticity, lock, root and health checks against the spec database.
package android

import (
	"bufio"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mo3bdlaa/smart-mobile-check/internal/adb"
)

// AppEntry is an installed package.
type AppEntry struct {
	BundleID  string `json:"bundleId"` // package name (field name shared with the iPhone UI)
	Name      string `json:"name"`
	Type      string `json:"type"` // User | System
	Installer string `json:"installer,omitempty"`
}

// Account is a signed-in account on the phone.
type Account struct {
	Type string `json:"type"`
	Name string `json:"name"` // masked
}

// Snapshot is everything read from the phone.
type Snapshot struct {
	CollectedAt   time.Time         `json:"collectedAt"`
	Serial        string            `json:"serial"`
	Props         map[string]string `json:"props"`
	MemTotalKB    int64             `json:"memTotalKB,omitempty"`
	DataTotalKB   int64             `json:"dataTotalKB,omitempty"`
	DataUsedKB    int64             `json:"dataUsedKB,omitempty"`
	ScreenSize    string            `json:"screenSize,omitempty"`
	Density       string            `json:"density,omitempty"`
	Battery       map[string]string `json:"battery,omitempty"`    // dumpsys battery
	BatterySys    map[string]string `json:"batterySys,omitempty"` // /sys/class/power_supply/battery
	Accounts      []Account         `json:"accounts,omitempty"`
	AccountsRead  bool              `json:"accountsRead"`
	DeviceOwner   string            `json:"deviceOwner,omitempty"`
	ProfileOwners []string          `json:"profileOwners,omitempty"`
	PolicyRead    bool              `json:"policyRead"`
	Apps          []AppEntry        `json:"apps,omitempty"`
	AppsScanned   bool              `json:"appsScanned"`
	SuPaths       []string          `json:"suPaths,omitempty"`
	SELinux       string            `json:"selinux,omitempty"`
	Sensors       []string          `json:"sensors,omitempty"`
	SensorsRead   bool              `json:"sensorsRead"`
	CameraCount   int               `json:"cameraCount"`
	CamerasRead   bool              `json:"camerasRead"`
	Dropbox       map[string]int    `json:"dropbox,omitempty"`
	DropboxRead   bool              `json:"dropboxRead"`
	IMEIs         []string          `json:"imeis,omitempty"`
	Settings      map[string]string `json:"settings,omitempty"`
	UptimeSec     int64             `json:"uptimeSec,omitempty"`
	Errors        map[string]string `json:"errors,omitempty"`
	Demo          bool              `json:"demo,omitempty"`
}

// DeviceBrief is a connected phone as shown in the device picker.
type DeviceBrief struct {
	Serial    string `json:"serial"`
	State     string `json:"state"`
	Model     string `json:"model,omitempty"`
	Product   string `json:"product,omitempty"`
	Device    string `json:"device,omitempty"`
	ModelName string `json:"modelName,omitempty"`
}

// ProgressFunc receives scan progress.
type ProgressFunc func(step, total int, label string)

// ListDevices lists phones known to the adb server.
func ListDevices() ([]DeviceBrief, error) {
	ds, err := adb.New().Devices()
	if err != nil {
		return nil, err
	}
	var out []DeviceBrief
	for _, d := range ds {
		out = append(out, DeviceBrief{Serial: d.Serial, State: d.State, Model: d.Model, Product: d.Product, Device: d.Device})
	}
	return out, nil
}

type step struct {
	key, label string
	fn         func(c *adb.Client, s *Snapshot) error
}

var steps = []step{
	{"props", "Reading device identity", collectProps},
	{"battery", "Reading battery", collectBattery},
	{"hardware", "Checking memory, storage & screen", collectHardware},
	{"accounts", "Checking accounts & locks", collectAccounts},
	{"apps", "Auditing installed apps", collectApps},
	{"root", "Checking for root & bootloader unlock", collectRoot},
	{"sensors", "Testing sensors & cameras", collectSensors},
	{"crashes", "Reading crash history", collectCrashes},
}

// StepCount is the number of progress steps (collection + evaluation).
var StepCount = len(steps) + 1

// Collect reads everything from the phone with the given serial.
func Collect(serial string, progress ProgressFunc) (*Snapshot, error) {
	c := adb.New()
	s := &Snapshot{CollectedAt: time.Now(), Serial: serial, Props: map[string]string{}, Errors: map[string]string{}}
	for i, st := range steps {
		if progress != nil {
			progress(i+1, StepCount, st.label)
		}
		if err := st.fn(c, s); err != nil {
			s.Errors[st.key] = err.Error()
			if st.key == "props" {
				return s, fmt.Errorf("could not read the phone: %w", err)
			}
		}
	}
	if progress != nil {
		progress(StepCount, StepCount, "Evaluating results")
	}
	return s, nil
}

var propRe = regexp.MustCompile(`^\[([^\]]+)\]: \[(.*)\]$`)

func parseProps(out string) map[string]string {
	m := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if mm := propRe.FindStringSubmatch(strings.TrimSpace(sc.Text())); mm != nil {
			m[mm[1]] = mm[2]
		}
	}
	return m
}

func collectProps(c *adb.Client, s *Snapshot) error {
	out, err := c.Shell(s.Serial, "getprop")
	if err != nil {
		return err
	}
	s.Props = parseProps(out)
	if len(s.Props) == 0 {
		return fmt.Errorf("no system properties returned")
	}
	// IMEI: readable by the shell user on many devices (best effort).
	for _, cmd := range []string{
		"service call iphonesubinfo 1 s16 com.android.shell",
		"service call iphonesubinfo 4 i32 1 s16 com.android.shell",
		"service call iphonesubinfo 4 i32 2 s16 com.android.shell",
	} {
		if o, err := c.Shell(s.Serial, cmd); err == nil {
			if im := parseParcelString(o); len(im) == 15 && isDigits(im) && !contains(s.IMEIs, im) {
				s.IMEIs = append(s.IMEIs, im)
			}
		}
	}
	s.Settings = map[string]string{}
	for _, k := range []string{"global device_provisioned", "secure user_setup_complete", "global development_settings_enabled", "global adb_enabled", "secure bluetooth_address"} {
		if o, err := c.Shell(s.Serial, "settings get "+k); err == nil {
			s.Settings[strings.Fields(k)[1]] = strings.TrimSpace(o)
		}
	}
	if o, err := c.Shell(s.Serial, "cat /proc/uptime"); err == nil {
		if f := strings.Fields(o); len(f) > 0 {
			v, _ := strconv.ParseFloat(f[0], 64)
			s.UptimeSec = int64(v)
		}
	}
	return nil
}

var parcelRe = regexp.MustCompile(`'([^']*)'`)

// parseParcelString decodes the UTF-16 string printed by "service call".
func parseParcelString(out string) string {
	var b strings.Builder
	for _, m := range parcelRe.FindAllStringSubmatch(out, -1) {
		b.WriteString(m[1])
	}
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, b.String())
}

func kvLines(out, sep string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, sep)
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if k != "" {
			m[k] = strings.TrimSpace(v)
		}
	}
	return m
}

func collectBattery(c *adb.Client, s *Snapshot) error {
	out, err := c.Shell(s.Serial, "dumpsys battery")
	if err != nil {
		return err
	}
	s.Battery = kvLines(out, ":")
	files := []string{"cycle_count", "charge_full", "charge_full_design", "battery_cycle", "fg_asoc", "health", "capacity", "charge_counter", "batt_capacity_max"}
	cmd := "for f in " + strings.Join(files, " ") + "; do echo \"$f=$(cat /sys/class/power_supply/battery/$f 2>/dev/null)\"; done"
	if o, err := c.Shell(s.Serial, cmd); err == nil {
		s.BatterySys = map[string]string{}
		for k, v := range kvLines(o, "=") {
			if v != "" {
				s.BatterySys[k] = v
			}
		}
	}
	return nil
}

func collectHardware(c *adb.Client, s *Snapshot) error {
	if o, err := c.Shell(s.Serial, "cat /proc/meminfo"); err == nil {
		for _, line := range strings.Split(o, "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				f := strings.Fields(line)
				if len(f) >= 2 {
					s.MemTotalKB, _ = strconv.ParseInt(f[1], 10, 64)
				}
			}
		}
	}
	if o, err := c.Shell(s.Serial, "df -k /data"); err == nil {
		lines := strings.Split(strings.TrimSpace(o), "\n")
		if len(lines) >= 2 {
			f := strings.Fields(lines[len(lines)-1])
			if len(f) >= 4 {
				s.DataTotalKB, _ = strconv.ParseInt(f[1], 10, 64)
				s.DataUsedKB, _ = strconv.ParseInt(f[2], 10, 64)
			}
		}
	}
	if o, err := c.Shell(s.Serial, "wm size"); err == nil {
		for _, line := range strings.Split(o, "\n") {
			if strings.HasPrefix(line, "Physical size:") {
				s.ScreenSize = strings.TrimSpace(strings.TrimPrefix(line, "Physical size:"))
			}
		}
	}
	if o, err := c.Shell(s.Serial, "wm density"); err == nil {
		for _, line := range strings.Split(o, "\n") {
			if strings.HasPrefix(line, "Physical density:") {
				s.Density = strings.TrimSpace(strings.TrimPrefix(line, "Physical density:"))
			}
		}
	}
	return nil
}

var accountRe = regexp.MustCompile(`Account \{name=([^,]*), type=([^}]*)\}`)

func collectAccounts(c *adb.Client, s *Snapshot) error {
	var errs []string
	if o, err := c.Shell(s.Serial, "dumpsys account"); err == nil {
		s.AccountsRead = true
		seen := map[string]bool{}
		for _, m := range accountRe.FindAllStringSubmatch(o, -1) {
			key := m[2] + "|" + m[1]
			if seen[key] {
				continue
			}
			seen[key] = true
			s.Accounts = append(s.Accounts, Account{Type: strings.TrimSpace(m[2]), Name: maskAccount(m[1])})
		}
	} else {
		errs = append(errs, err.Error())
	}
	if o, err := c.Shell(s.Serial, "dumpsys device_policy"); err == nil {
		s.PolicyRead = true
		parsePolicy(o, s)
	} else {
		errs = append(errs, err.Error())
	}
	if len(errs) == 2 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

var adminRe = regexp.MustCompile(`admin=ComponentInfo\{([^/}]+)`)

func parsePolicy(out string, s *Snapshot) {
	section := ""
	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "Device Owner:"):
			section = "device"
		case strings.HasPrefix(t, "Profile Owner"):
			section = "profile"
		case t == "" || strings.HasSuffix(t, ":") && !strings.Contains(t, "admin"):
			if !strings.HasPrefix(t, "Device Owner") && !strings.HasPrefix(t, "Profile Owner") {
				section = ""
			}
		}
		if m := adminRe.FindStringSubmatch(t); m != nil {
			switch section {
			case "device":
				s.DeviceOwner = m[1]
			case "profile":
				if !contains(s.ProfileOwners, m[1]) {
					s.ProfileOwners = append(s.ProfileOwners, m[1])
				}
			}
		}
	}
}

func maskAccount(name string) string {
	name = strings.TrimSpace(name)
	at := strings.Index(name, "@")
	if at > 2 {
		return name[:2] + strings.Repeat("•", 4) + name[at:]
	}
	if len(name) > 3 {
		return name[:2] + strings.Repeat("•", 4)
	}
	return "••••"
}

func collectApps(c *adb.Client, s *Snapshot) error {
	third, err := c.Shell(s.Serial, "pm list packages -3 -i")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(third, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "package:") {
			continue
		}
		f := strings.Fields(strings.TrimPrefix(line, "package:"))
		if len(f) == 0 {
			continue
		}
		a := AppEntry{BundleID: f[0], Name: f[0], Type: "User"}
		for _, x := range f[1:] {
			if v, ok := strings.CutPrefix(x, "installer="); ok && v != "null" {
				a.Installer = v
			}
		}
		s.Apps = append(s.Apps, a)
	}
	if sys, err := c.Shell(s.Serial, "pm list packages -s"); err == nil {
		for _, line := range strings.Split(sys, "\n") {
			if p, ok := strings.CutPrefix(strings.TrimSpace(line), "package:"); ok && p != "" {
				s.Apps = append(s.Apps, AppEntry{BundleID: p, Name: p, Type: "System"})
			}
		}
	}
	sort.Slice(s.Apps, func(i, j int) bool {
		if s.Apps[i].Type != s.Apps[j].Type {
			return s.Apps[i].Type == "User"
		}
		return s.Apps[i].BundleID < s.Apps[j].BundleID
	})
	s.AppsScanned = true
	return nil
}

func collectRoot(c *adb.Client, s *Snapshot) error {
	o, err := c.Shell(s.Serial, "for p in /system/bin/su /system/xbin/su /sbin/su /su/bin/su /system/sbin/su /vendor/bin/su /data/local/su /data/local/bin/su /data/local/xbin/su; do [ -e $p ] && echo $p; done; which su 2>/dev/null")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(o, "\n") {
		if p := strings.TrimSpace(line); strings.HasPrefix(p, "/") && !contains(s.SuPaths, p) {
			s.SuPaths = append(s.SuPaths, p)
		}
	}
	if o, err := c.Shell(s.Serial, "getenforce"); err == nil {
		s.SELinux = strings.TrimSpace(o)
	}
	return nil
}

var sensorTypeRe = regexp.MustCompile(`android\.sensor\.([a-z_]+)`)

func collectSensors(c *adb.Client, s *Snapshot) error {
	if o, err := c.Shell(s.Serial, "dumpsys sensorservice"); err == nil {
		s.SensorsRead = true
		seen := map[string]bool{}
		for _, m := range sensorTypeRe.FindAllStringSubmatch(o, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				s.Sensors = append(s.Sensors, m[1])
			}
		}
		sort.Strings(s.Sensors)
	}
	if o, err := c.Shell(s.Serial, "dumpsys media.camera"); err == nil {
		for _, line := range strings.Split(o, "\n") {
			if strings.Contains(line, "Number of camera devices:") {
				f := strings.Fields(line)
				s.CameraCount, _ = strconv.Atoi(f[len(f)-1])
				s.CamerasRead = true
			}
		}
	}
	return nil
}

var dropboxRe = regexp.MustCompile(`^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d ([A-Za-z0-9_]+)`)

func collectCrashes(c *adb.Client, s *Snapshot) error {
	o, err := c.Shell(s.Serial, "dumpsys dropbox")
	if err != nil {
		return err
	}
	s.DropboxRead = true
	s.Dropbox = map[string]int{}
	for _, line := range strings.Split(o, "\n") {
		if m := dropboxRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			s.Dropbox[m[1]]++
		}
	}
	return nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
