package iphone

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

// Check categories, in display order.
const (
	CatIdentity = "Identity & authenticity"
	CatOwner    = "Ownership & locks"
	CatSoftware = "Software integrity"
	CatBattery  = "Battery"
	CatHardware = "Hardware & diagnostics"
	CatHands    = "Hands-on tests"
)

// Categories lists the categories in display order.
var Categories = []string{CatIdentity, CatOwner, CatSoftware, CatBattery, CatHardware, CatHands}

type evaluator struct {
	s      *Snapshot
	spec   *specs.Device
	db     *specs.DB
	checks []Check
}

func (e *evaluator) add(c Check) {
	if c.Severity == "" {
		c.Severity = Note
	}
	e.checks = append(e.checks, c)
}

func (e *evaluator) ld(key string) string { return str(e.s.Lockdown[key]) }

func (e *evaluator) dom(domain, key string) any {
	if m := e.s.Domains[domain]; m != nil {
		return m[key]
	}
	return nil
}

func (e *evaluator) gestalt(key string) string {
	if e.s.Gestalt == nil {
		return ""
	}
	return str(e.s.Gestalt[key])
}

// Evaluate runs every check against a snapshot.
func Evaluate(s *Snapshot, db *specs.DB, appVersion string) *Result {
	e := &evaluator{s: s, db: db}
	if s.Lockdown == nil {
		s.Lockdown = map[string]any{}
	}
	pt := e.ld("ProductType")
	e.spec = db.ByProductType(pt)

	e.identityChecks()
	e.ownershipChecks()
	e.softwareChecks()
	batt := e.batteryChecks()
	e.hardwareChecks()

	res := &Result{
		Snapshot:    s,
		Checks:      e.checks,
		Battery:     batt,
		GeneratedAt: time.Now(),
		AppVersion:  appVersion,
		Storage:     e.storageInfo(),
	}
	if e.spec != nil {
		res.Spec = e.spec
		res.ModelName = e.spec.Name
	} else {
		res.ModelName = firstNonEmpty(pt, "Unknown iPhone")
	}
	res.Identity = e.identity(res.ModelName)
	Score(res)
	return res
}

// Score computes counts, score and verdict from the checks.
func Score(r *Result) {
	r.Counts = map[Status]int{}
	score := 100.0
	critical := false
	majorFail := false
	for _, c := range r.Checks {
		r.Counts[c.Status]++
		switch c.Status {
		case Fail:
			switch c.Severity {
			case Critical:
				score -= 45
				critical = true
			case Major:
				score -= 20
				majorFail = true
			case Minor:
				score -= 8
			}
		case Warn:
			switch c.Severity {
			case Critical:
				score -= 15
			case Major:
				score -= 8
			case Minor:
				score -= 3
			}
		}
	}
	if score < 0 {
		score = 0
	}
	r.Score = int(math.Round(score))
	switch {
	case critical:
		r.Verdict = "risk"
		r.VerdictText = "Do not buy — a critical problem was found"
	case r.Score >= 85 && !majorFail:
		r.Verdict = "good"
		r.VerdictText = "Looks genuine and healthy"
	case r.Score >= 60:
		r.Verdict = "caution"
		r.VerdictText = "Buy with caution — review the warnings"
	default:
		r.Verdict = "risk"
		r.VerdictText = "High risk — several problems were found"
	}
}

// ---------------------------------------------------------------- identity

var (
	serialRe = regexp.MustCompile(`^[A-Z0-9]{10,12}$`)
	newUDID  = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{16}$`)
	oldUDID  = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	chipNum  = regexp.MustCompile(`A\d+`)
)

func (e *evaluator) identityChecks() {
	pt := e.ld("ProductType")

	// 1. Model known
	if e.spec != nil {
		e.add(Check{ID: "model", Category: CatIdentity, Title: "Model recognised", Status: Pass,
			Summary: fmt.Sprintf("%s (%s) found in the spec database.", e.spec.Name, pt), Actual: pt})
	} else {
		e.add(Check{ID: "model", Category: CatIdentity, Title: "Model recognised", Status: Warn, Severity: Minor,
			Summary: fmt.Sprintf("Identifier %q is not in the spec database, so spec cross-checks were skipped.", pt),
			Advice:  "Add a spec file for this model to the specs folder next to the app."})
	}

	// 2. Model reported consistently by independent services
	sources := map[string]string{"lockdown": pt}
	if e.s.FS != nil && e.s.FS.Model != "" {
		sources["file service"] = e.s.FS.Model
	}
	if g := e.gestalt("ProductType"); g != "" {
		sources["MobileGestalt"] = g
	}
	if len(sources) > 1 {
		mismatch := false
		var parts []string
		for k, v := range sources {
			parts = append(parts, k+"="+v)
			if !strings.EqualFold(v, pt) {
				mismatch = true
			}
		}
		sort.Strings(parts)
		if mismatch {
			e.add(Check{ID: "model-consistency", Category: CatIdentity, Title: "Model identity consistency", Status: Fail, Severity: Critical,
				Summary: "Different system services report different models — the device readings have been tampered with.",
				Actual:  strings.Join(parts, ", ")})
		} else {
			e.add(Check{ID: "model-consistency", Category: CatIdentity, Title: "Model identity consistency", Status: Pass,
				Summary: fmt.Sprintf("%d independent services agree on the model.", len(sources)), Actual: strings.Join(parts, ", ")})
		}
	}

	// 3. Board / hardware model
	hw := e.ld("HardwareModel")
	if e.spec != nil && len(e.spec.HardwareModels) > 0 && hw != "" {
		if containsFold(e.spec.HardwareModels, hw) {
			e.add(Check{ID: "board", Category: CatIdentity, Title: "Logic board model", Status: Pass,
				Summary: "Board identifier matches the model.", Expected: strings.Join(e.spec.HardwareModels, " / "), Actual: hw})
		} else {
			e.add(Check{ID: "board", Category: CatIdentity, Title: "Logic board model", Status: Warn, Severity: Major,
				Summary: "Board identifier does not match the expected value for this model.", Expected: strings.Join(e.spec.HardwareModels, " / "), Actual: hw,
				Advice: "Possible logic-board swap or spoofed readings. Inspect carefully."})
		}
	}

	// 4. Regulatory model (A-number)
	reg := firstNonEmpty(e.ld("RegulatoryModelNumber"))
	if e.spec != nil && len(e.spec.RegulatoryModels) > 0 && reg != "" {
		if containsFold(e.spec.RegulatoryModels, reg) {
			e.add(Check{ID: "a-number", Category: CatIdentity, Title: "Regulatory model (A-number)", Status: Pass,
				Summary: "A-number belongs to this model.", Expected: strings.Join(e.spec.RegulatoryModels, ", "), Actual: reg})
		} else {
			e.add(Check{ID: "a-number", Category: CatIdentity, Title: "Regulatory model (A-number)", Status: Warn, Severity: Major,
				Summary: "A-number is not one of the known numbers for this model.", Expected: strings.Join(e.spec.RegulatoryModels, ", "), Actual: reg,
				Advice: "Compare with the A-number printed on the SIM tray/box and Settings › General › About."})
		}
	}

	// 5. Chip ID vs SoC
	chipID, hasChip := num(e.s.Lockdown["ChipID"])
	if e.spec != nil && hasChip && e.spec.Chip != "" {
		key := chipNum.FindString(e.spec.Chip)
		expected := e.db.ChipIDs(key)
		actual := fmt.Sprintf("0x%04X", int64(chipID))
		if len(expected) > 0 {
			ok := false
			for _, x := range expected {
				if v, err := strconv.ParseInt(strings.TrimPrefix(strings.ToLower(x), "0x"), 16, 64); err == nil && v == int64(chipID) {
					ok = true
				}
			}
			if ok {
				e.add(Check{ID: "chip", Category: CatIdentity, Title: "Processor (SoC)", Status: Pass,
					Summary: fmt.Sprintf("Chip ID matches %s.", e.spec.Chip), Expected: strings.Join(expected, " / "), Actual: actual})
			} else {
				e.add(Check{ID: "chip", Category: CatIdentity, Title: "Processor (SoC)", Status: Warn, Severity: Major,
					Summary: fmt.Sprintf("Chip ID does not match the %s expected for this model.", e.spec.Chip), Expected: strings.Join(expected, " / "), Actual: actual})
			}
		}
	}

	// 6. UDID integrity (derived from ChipID + ECID, or SHA1 on older models)
	udid := firstNonEmpty(e.ld("UniqueDeviceID"), e.s.UDID)
	ecid, hasECID := num(e.s.Lockdown["UniqueChipID"])
	switch {
	case newUDID.MatchString(udid) && hasChip && hasECID:
		want := fmt.Sprintf("%08X-%016X", uint64(chipID), uint64(ecid))
		if strings.EqualFold(want, udid) {
			e.add(Check{ID: "udid", Category: CatIdentity, Title: "Hardware ID integrity", Status: Pass,
				Summary: "UDID is consistent with the chip ID and ECID burned into the processor.", Actual: udid})
		} else {
			e.add(Check{ID: "udid", Category: CatIdentity, Title: "Hardware ID integrity", Status: Fail, Severity: Critical,
				Summary: "UDID does not match the processor's chip ID / ECID — identity has been altered.", Expected: want, Actual: udid})
		}
	case oldUDID.MatchString(udid):
		if legacyUDIDMatches(udid, e.ld("SerialNumber"), uint64(ecid), e.ld("WiFiAddress"), e.ld("BluetoothAddress")) {
			e.add(Check{ID: "udid", Category: CatIdentity, Title: "Hardware ID integrity", Status: Pass,
				Summary: "Legacy UDID is consistent with serial number, ECID and MAC addresses.", Actual: udid})
		} else {
			e.add(Check{ID: "udid", Category: CatIdentity, Title: "Hardware ID integrity", Status: Info,
				Summary: "Legacy-format UDID; cannot be fully re-derived on this model.", Actual: udid})
		}
	}

	// 7. Serial number
	serial := e.ld("SerialNumber")
	switch {
	case serial == "":
		e.add(Check{ID: "serial", Category: CatIdentity, Title: "Serial number", Status: Fail, Severity: Major,
			Summary: "The device did not report a serial number."})
	case !serialRe.MatchString(serial):
		e.add(Check{ID: "serial", Category: CatIdentity, Title: "Serial number", Status: Fail, Severity: Major,
			Summary: "Serial number has an invalid format.", Actual: serial})
	default:
		g := e.gestalt("SerialNumber")
		if g != "" && !strings.EqualFold(g, serial) {
			e.add(Check{ID: "serial", Category: CatIdentity, Title: "Serial number", Status: Fail, Severity: Critical,
				Summary: "Serial number differs between system services — readings were modified.", Expected: g, Actual: serial})
		} else {
			e.add(Check{ID: "serial", Category: CatIdentity, Title: "Serial number", Status: Pass,
				Summary: "Valid serial number format.", Actual: serial,
				Advice: "Verify coverage at checkcoverage.apple.com and that it matches the box / Settings."})
		}
	}

	// 8. Production silicon
	if v, ok := boolv(e.s.Lockdown["ProductionSOC"]); ok {
		if v {
			e.add(Check{ID: "production", Category: CatIdentity, Title: "Production hardware", Status: Pass,
				Summary: "Retail (production-fused) processor."})
		} else {
			e.add(Check{ID: "production", Category: CatIdentity, Title: "Production hardware", Status: Fail, Severity: Major,
				Summary: "Development / prototype processor — not a retail unit."})
		}
	}

	// 9. Sales model (part number prefix)
	mn := e.ld("ModelNumber")
	region := e.ld("RegionInfo")
	if mn != "" {
		prefix := strings.ToUpper(mn[:1])
		meaning := e.db.Reference.PartPrefixes[prefix]
		full := mn + region
		c := Check{ID: "part-number", Category: CatIdentity, Title: "Sales model (part number)", Actual: full, Summary: meaning}
		switch prefix {
		case "M":
			c.Status = Pass
		case "F":
			c.Status, c.Severity = Warn, Minor
			c.Advice = "Refurbished units are fine if priced accordingly — make sure the seller disclosed it."
		case "N":
			c.Status, c.Severity = Warn, Minor
			c.Advice = "A replacement unit issued by Apple service. Usually fine, but not the original retail device."
		case "3":
			c.Status, c.Severity = Warn, Major
			c.Advice = "Ex-demo units have been on display 24/7; expect heavier battery wear."
		case "P":
			c.Status = Pass
		default:
			c.Status, c.Severity = Warn, Minor
			c.Summary = "Unusual part-number prefix."
		}
		if c.Summary == "" {
			c.Summary = "Part number " + full
		}
		e.add(c)
	}

	// 10. Region
	if region != "" {
		r := e.db.Region(region)
		c := Check{ID: "region", Category: CatIdentity, Title: "Sales region", Status: Info, Actual: region}
		if r != nil {
			c.Summary = "Sold for: " + strings.Join(r.Countries, ", ")
		} else {
			c.Summary = "Region code " + region
		}
		if e.spec != nil && len(e.spec.Cellular.ESIMOnlyRegions) > 0 && r != nil {
			for _, esim := range e.spec.Cellular.ESIMOnlyRegions {
				for _, ctry := range r.Countries {
					if strings.Contains(strings.ToLower(ctry), strings.ToLower(esim)) {
						c.Advice = "This region's model is eSIM-only (no physical SIM tray). Make sure your carrier supports eSIM."
					}
				}
			}
		}
		e.add(c)
	}

	// 11. IMEI
	imei := strings.ReplaceAll(e.ld("InternationalMobileEquipmentIdentity"), " ", "")
	imei2 := strings.ReplaceAll(e.ld("InternationalMobileEquipmentIdentity2"), " ", "")
	switch {
	case imei == "":
		e.add(Check{ID: "imei", Category: CatIdentity, Title: "IMEI", Status: Fail, Severity: Major,
			Summary: "No IMEI reported — the cellular modem may be damaged or the baseband is not responding.",
			Advice:  "Check Settings › General › About. A missing IMEI usually means no cellular service."})
	case len(imei) != 15 || !luhnValid(imei):
		e.add(Check{ID: "imei", Category: CatIdentity, Title: "IMEI", Status: Fail, Severity: Critical,
			Summary: "IMEI fails the checksum — it is invalid or has been altered.", Actual: imei})
	default:
		c := Check{ID: "imei", Category: CatIdentity, Title: "IMEI", Status: Pass, Actual: imei,
			Summary: "Valid IMEI (checksum OK).",
			Advice:  "Check it against the box / SIM tray and a blacklist (lost/stolen) database before paying."}
		if imei2 != "" {
			c.Actual += " · " + imei2
			if len(imei2) != 15 || !luhnValid(imei2) {
				c.Status, c.Severity = Fail, Critical
				c.Summary = "Second IMEI fails the checksum — altered or invalid."
			}
		}
		e.add(c)
	}

	// 12. Storage capacity vs marketed options
	total, _ := num(e.dom("com.apple.disk_usage", "TotalDiskCapacity"))
	if total == 0 && e.s.FS != nil {
		total = float64(e.s.FS.TotalBytes)
	}
	if total > 0 && e.spec != nil && len(e.spec.StorageOptionsGB) > 0 {
		gb := total / 1e9
		best, bestDiff := 0, math.MaxFloat64
		for _, o := range e.spec.StorageOptionsGB {
			marketed := float64(o)
			if o >= 1024 {
				marketed = float64(o) / 1024 * 1000
			}
			if gb <= marketed*1.03 && gb >= marketed*0.85 {
				if d := math.Abs(marketed - gb); d < bestDiff {
					best, bestDiff = o, d
				}
			}
		}
		opts := []string{}
		for _, o := range e.spec.StorageOptionsGB {
			opts = append(opts, gbLabel(o))
		}
		if best > 0 {
			e.add(Check{ID: "storage", Category: CatIdentity, Title: "Storage capacity", Status: Pass,
				Summary:  fmt.Sprintf("%s model — capacity matches an official configuration.", gbLabel(best)),
				Expected: strings.Join(opts, " / "), Actual: humanBytes(total)})
		} else {
			e.add(Check{ID: "storage", Category: CatIdentity, Title: "Storage capacity", Status: Fail, Severity: Major,
				Summary:  "Reported capacity does not match any official configuration — storage may have been modified.",
				Expected: strings.Join(opts, " / "), Actual: humanBytes(total)})
		}
	}
}

func legacyUDIDMatches(udid, serial string, ecid uint64, wifi, bt string) bool {
	if serial == "" || ecid == 0 {
		return false
	}
	ecids := []string{strconv.FormatUint(ecid, 10), strconv.FormatUint(ecid, 16), strings.ToUpper(strconv.FormatUint(ecid, 16)), fmt.Sprintf("%016X", ecid)}
	for _, ec := range ecids {
		for _, w := range []string{strings.ToLower(wifi), strings.ToUpper(wifi)} {
			for _, b := range []string{strings.ToLower(bt), strings.ToUpper(bt)} {
				sum := sha1.Sum([]byte(serial + ec + w + b))
				if strings.EqualFold(hex.EncodeToString(sum[:]), udid) {
					return true
				}
			}
		}
	}
	return false
}

// --------------------------------------------------------------- ownership

func (e *evaluator) ownershipChecks() {
	// Activation
	switch st := e.ld("ActivationState"); st {
	case "":
	case "Activated", "FactoryActivated":
		e.add(Check{ID: "activation", Category: CatOwner, Title: "Activation", Status: Pass, Actual: st,
			Summary: "Device is activated."})
	default:
		e.add(Check{ID: "activation", Category: CatOwner, Title: "Activation", Status: Warn, Severity: Major, Actual: st,
			Summary: "Device is not activated.",
			Advice:  "Activate it in front of you (insert a SIM / connect to Wi-Fi). If it asks for the previous owner's Apple Account, it is Activation Locked — do not buy."})
	}

	// Find My / Activation Lock
	if v, ok := boolv(e.dom("com.apple.fmip", "IsAssociated")); ok {
		if v {
			e.add(Check{ID: "findmy", Category: CatOwner, Title: "Find My / Activation Lock", Status: Fail, Severity: Critical,
				Summary: "Find My is ON — the phone is locked to someone's Apple Account (Activation Lock).",
				Advice:  "Ask the seller to sign out of their Apple Account (Settings › [name] › Sign Out), turn off Find My and erase the phone in front of you. Re-scan afterwards."})
		} else {
			e.add(Check{ID: "findmy", Category: CatOwner, Title: "Find My / Activation Lock", Status: Pass,
				Summary: "Find My is off — no Activation Lock."})
		}
	} else {
		e.add(Check{ID: "findmy", Category: CatOwner, Title: "Find My / Activation Lock", Status: Warn, Severity: Major,
			Summary: "Find My status could not be read.",
			Advice:  "Check Settings › [name] › Find My manually. It must be OFF before you pay."})
	}

	// Remote management (DEP / MDM)
	if cc := e.s.CloudConfig; cc != nil {
		org := str(cc["OrganizationName"])
		mdmUnremovable, _ := boolv(cc["IsMDMUnremovable"])
		supervised, _ := boolv(cc["IsSupervised"])
		cfgURL := str(cc["ConfigurationURL"])
		mandatory, _ := boolv(cc["IsMandatory"])
		switch {
		case org != "" || cfgURL != "" || mdmUnremovable || mandatory:
			e.add(Check{ID: "mdm", Category: CatOwner, Title: "Remote Management (MDM)", Status: Fail, Severity: Critical,
				Summary: "This iPhone is enrolled in an organisation's device management — it belongs to a company or school.",
				Actual:  firstNonEmpty(org, cfgURL, "enrolled"),
				Advice:  "Company-owned devices can be remotely locked or wiped. Only buy with written release from the organisation."})
		case supervised:
			e.add(Check{ID: "mdm", Category: CatOwner, Title: "Remote Management (MDM)", Status: Warn, Severity: Major,
				Summary: "Device is supervised (managed configuration)."})
		default:
			e.add(Check{ID: "mdm", Category: CatOwner, Title: "Remote Management (MDM)", Status: Pass,
				Summary: "Not enrolled in device management and not supervised."})
		}
	} else {
		e.add(Check{ID: "mdm", Category: CatOwner, Title: "Remote Management (MDM)", Status: Skip,
			Summary: "Management status could not be read.", Advice: "Check Settings › General › VPN & Device Management."})
	}

	// Configuration profiles
	if e.s.ProfilesRead {
		if len(e.s.Profiles) == 0 {
			e.add(Check{ID: "profiles", Category: CatOwner, Title: "Configuration profiles", Status: Pass,
				Summary: "No configuration profiles installed."})
		} else {
			var names []string
			locked := false
			for _, p := range e.s.Profiles {
				names = append(names, firstNonEmpty(p.DisplayName, p.Identifier))
				if p.RemovalDisallowed {
					locked = true
				}
			}
			c := Check{ID: "profiles", Category: CatOwner, Title: "Configuration profiles", Status: Warn, Severity: Major,
				Summary: fmt.Sprintf("%d profile(s) installed.", len(e.s.Profiles)), Actual: strings.Join(names, ", "),
				Advice: "Profiles can install certificates, VPNs or restrictions. Ask the seller to remove them (Settings › General › VPN & Device Management)."}
			if locked {
				c.Status, c.Severity = Fail, Major
				c.Summary += " At least one cannot be removed by the user."
			}
			e.add(c)
		}
	}

	// Brick state
	if v, ok := boolv(e.s.Lockdown["BrickState"]); ok && v {
		e.add(Check{ID: "brick", Category: CatOwner, Title: "Device state", Status: Fail, Severity: Critical,
			Summary: "Device reports a bricked (unusable) state."})
	}

	// Passcode (informational)
	if v, ok := boolv(e.s.Lockdown["PasswordProtected"]); ok {
		s := "No passcode is set."
		if v {
			s = "A passcode is set. Ask the seller to erase the phone (Settings › General › Transfer or Reset) before handover."
		}
		e.add(Check{ID: "passcode", Category: CatOwner, Title: "Passcode", Status: Info, Summary: s})
	}
}

// --------------------------------------------------------------- software

func (e *evaluator) softwareChecks() {
	ver := e.ld("ProductVersion")
	build := e.ld("BuildVersion")

	// iOS version plausibility for this hardware
	if e.spec != nil && ver != "" && e.spec.LaunchIOS != "" {
		tooOld := compareVersions(ver, e.spec.LaunchIOS) < 0
		tooNew := e.spec.MaxIOS != nil && *e.spec.MaxIOS != "" && compareVersions(ver, *e.spec.MaxIOS) > 0
		rng := "iOS " + e.spec.LaunchIOS + " or later"
		if e.spec.MaxIOS != nil && *e.spec.MaxIOS != "" {
			rng = "iOS " + e.spec.LaunchIOS + " – " + *e.spec.MaxIOS + ".x"
		}
		if tooOld || tooNew {
			e.add(Check{ID: "ios-plausible", Category: CatSoftware, Title: "iOS version plausibility", Status: Fail, Severity: Critical,
				Summary:  "This iOS version cannot officially run on this model — the version/model readings were spoofed.",
				Expected: rng, Actual: "iOS " + ver + " (" + build + ")"})
		} else {
			e.add(Check{ID: "ios-plausible", Category: CatSoftware, Title: "iOS version plausibility", Status: Pass,
				Summary: "iOS version is valid for this model.", Expected: rng, Actual: "iOS " + ver + " (" + build + ")"})
		}
	}

	// Build consistency across services
	if g := e.gestalt("BuildVersion"); g != "" && build != "" && !strings.EqualFold(g, build) {
		e.add(Check{ID: "build-consistency", Category: CatSoftware, Title: "Firmware build consistency", Status: Fail, Severity: Critical,
			Summary: "System services disagree about the installed iOS build — firmware readings were modified.", Expected: g, Actual: build})
	}

	// Beta firmware
	if build != "" {
		last := build[len(build)-1]
		if last >= 'a' && last <= 'z' {
			e.add(Check{ID: "beta", Category: CatSoftware, Title: "Release firmware", Status: Warn, Severity: Minor,
				Summary: "A beta (pre-release) iOS build is installed.", Actual: build,
				Advice: "Beta firmware can be unstable. Ask the seller to leave the beta programme and update to a public release."})
		} else {
			e.add(Check{ID: "beta", Category: CatSoftware, Title: "Release firmware", Status: Pass,
				Summary: "Public release build of iOS.", Actual: build})
		}
	}

	// Jailbreak: afc2
	if e.s.AFC2Probed {
		if e.s.AFC2Open {
			e.add(Check{ID: "jb-afc2", Category: CatSoftware, Title: "Root filesystem service", Status: Fail, Severity: Critical,
				Summary: "A root-filesystem access service (afc2) is running — the device is jailbroken."})
		} else {
			e.add(Check{ID: "jb-afc2", Category: CatSoftware, Title: "Root filesystem service", Status: Pass,
				Summary: "No jailbreak filesystem services exposed."})
		}
	}

	// Jailbreak / TrollStore / sideloading apps
	if e.s.AppsScanned {
		known := map[string]string{}
		for _, a := range e.db.Reference.JailbreakApps {
			known[strings.ToLower(a.BundleID)] = a.Name
		}
		side := map[string]string{}
		for _, a := range e.db.Reference.SideloadApps {
			side[strings.ToLower(a.BundleID)] = a.Name
		}
		var jb, sys, sus, sl []string
		for _, a := range e.s.Apps {
			id := strings.ToLower(a.BundleID)
			label := firstNonEmpty(a.Name, a.BundleID) + " (" + a.BundleID + ")"
			if n, ok := known[id]; ok {
				jb = append(jb, n+" ("+a.BundleID+")")
				continue
			}
			if n, ok := side[id]; ok {
				sl = append(sl, n)
				continue
			}
			if strings.EqualFold(a.Type, "System") && !strings.HasPrefix(id, "com.apple.") {
				sys = append(sys, label)
				continue
			}
			for _, frag := range e.db.Reference.SuspiciousNames {
				if strings.Contains(strings.ToLower(a.Name), strings.ToLower(frag)) || strings.Contains(id, strings.ToLower(frag)) {
					sus = append(sus, label)
					break
				}
			}
		}
		switch {
		case len(jb) > 0:
			e.add(Check{ID: "jb-apps", Category: CatSoftware, Title: "Jailbreak tools", Status: Fail, Severity: Critical,
				Summary: "Jailbreak / exploit tools are installed — the system has been modified.", Actual: strings.Join(jb, ", "),
				Advice: "A jailbroken phone can hide modifications and fake readings. Ask for a full erase + restore, then re-scan."})
		case len(sys) > 0:
			e.add(Check{ID: "jb-apps", Category: CatSoftware, Title: "Jailbreak tools", Status: Fail, Severity: Major,
				Summary: "Non-Apple apps are registered as system apps — typical of TrollStore or a jailbreak.", Actual: strings.Join(sys, ", ")})
		case len(sus) > 0:
			e.add(Check{ID: "jb-apps", Category: CatSoftware, Title: "Jailbreak tools", Status: Warn, Severity: Major,
				Summary: "Apps with names associated with jailbreak/sideloading tools were found.", Actual: strings.Join(sus, ", ")})
		default:
			e.add(Check{ID: "jb-apps", Category: CatSoftware, Title: "Jailbreak tools", Status: Pass,
				Summary: fmt.Sprintf("Scanned %d apps — no jailbreak tools found.", len(e.s.Apps))})
		}
		if len(sl) > 0 {
			e.add(Check{ID: "sideload", Category: CatSoftware, Title: "Sideloading", Status: Warn, Severity: Minor,
				Summary: "Sideloading tools are installed (apps from outside the App Store).", Actual: strings.Join(sl, ", ")})
		}
	} else {
		e.add(Check{ID: "jb-apps", Category: CatSoftware, Title: "Jailbreak tools", Status: Skip,
			Summary: "Installed apps could not be listed."})
	}

	// Developer mode
	if v, ok := boolv(e.dom("com.apple.security.mac.amfi", "DeveloperModeStatus")); ok && v {
		e.add(Check{ID: "devmode", Category: CatSoftware, Title: "Developer Mode", Status: Warn, Severity: Minor,
			Summary: "Developer Mode is enabled.", Advice: "Not harmful by itself, but unusual for a personal phone."})
	}
}

// ---------------------------------------------------------------- battery

func (e *evaluator) batteryChecks() BatteryReport {
	b := e.s.Battery
	r := BatteryReport{}
	if pct, ok := num(e.dom("com.apple.mobile.battery", "BatteryCurrentCapacity")); ok {
		r.ChargePercent = int(pct)
	}
	if v, ok := boolv(e.dom("com.apple.mobile.battery", "BatteryIsCharging")); ok {
		r.IsCharging = v
	}
	if v, ok := boolv(e.dom("com.apple.mobile.battery", "ExternalConnected")); ok {
		r.ExternalConnected = v
	}
	if e.spec != nil {
		r.ExpectedDesign = e.spec.Battery.DesignCapacityMah
		r.RatedCycles = e.spec.Battery.RatedCycles
	}
	if r.RatedCycles == 0 {
		r.RatedCycles = 500
	}
	if len(b) == 0 {
		if gg := sub(e.s.Diagnostics, "GasGauge"); gg != nil {
			b = gg
		}
	}
	if len(b) == 0 {
		e.add(Check{ID: "battery", Category: CatBattery, Title: "Battery health", Status: Skip,
			Summary: "Battery gauge could not be read on this iOS version.", Advice: "Check Settings › Battery › Battery Health manually."})
		return r
	}
	r.Available = true
	bd := sub(b, "BatteryData")

	getN := func(keys ...string) (float64, bool) {
		for _, k := range keys {
			if v, ok := num(b[k]); ok {
				return v, true
			}
			if bd != nil {
				if v, ok := num(bd[k]); ok {
					return v, true
				}
			}
		}
		return 0, false
	}
	if v, ok := getN("CycleCount"); ok {
		r.CycleCount = int(v)
	}
	if v, ok := getN("DesignCapacity"); ok {
		r.DesignCapacity = int(v)
	}
	if v, ok := getN("NominalChargeCapacity", "AppleRawMaxCapacity", "FullChargeCapacity"); ok {
		r.FullChargeCap = int(v)
	}
	if v, ok := getN("Temperature"); ok {
		r.TemperatureC = v / 100
	}
	if v, ok := getN("Voltage"); ok {
		r.VoltageV = v / 1000
	}
	if r.ChargePercent == 0 {
		if v, ok := getN("CurrentCapacity"); ok && v <= 100 {
			r.ChargePercent = int(v)
		}
	}
	r.Serial = firstNonEmpty(str(b["Serial"]), str(b["BatterySerialNumber"]), str(bd["Serial"]))

	if v, ok := getN("MaximumCapacityPercent"); ok && v > 0 && v <= 110 {
		r.HealthPercent, r.HealthSource = v, "gauge (maximum capacity)"
	} else if r.DesignCapacity > 0 && r.FullChargeCap > 0 {
		r.HealthPercent = math.Min(100, float64(r.FullChargeCap)/float64(r.DesignCapacity)*100)
		r.HealthSource = "full-charge ÷ design capacity"
	}
	r.HealthPercent = math.Round(r.HealthPercent*10) / 10

	// Health
	switch h := r.HealthPercent; {
	case h == 0:
		e.add(Check{ID: "battery-health", Category: CatBattery, Title: "Battery health", Status: Skip,
			Summary: "Maximum capacity not reported.", Advice: "Check Settings › Battery › Battery Health."})
	case h >= 85:
		e.add(Check{ID: "battery-health", Category: CatBattery, Title: "Battery health", Status: Pass,
			Summary: fmt.Sprintf("Maximum capacity %.0f%% — good condition.", h), Actual: fmt.Sprintf("%.1f%%", h)})
	case h >= 80:
		e.add(Check{ID: "battery-health", Category: CatBattery, Title: "Battery health", Status: Warn, Severity: Major,
			Summary: fmt.Sprintf("Maximum capacity %.0f%% — worn, replacement will be needed soon.", h), Actual: fmt.Sprintf("%.1f%%", h),
			Advice: "Negotiate the price to cover a battery replacement."})
	default:
		e.add(Check{ID: "battery-health", Category: CatBattery, Title: "Battery health", Status: Fail, Severity: Major,
			Summary: fmt.Sprintf("Maximum capacity %.0f%% — below 80%%, battery is degraded and needs service.", h), Actual: fmt.Sprintf("%.1f%%", h),
			Advice: "Expect poor battery life and possible performance throttling. Factor a battery replacement into the price."})
	}

	// Cycles
	ratio := float64(r.CycleCount) / float64(r.RatedCycles)
	cyc := Check{ID: "battery-cycles", Category: CatBattery, Title: "Charge cycles", Actual: fmt.Sprintf("%d cycles", r.CycleCount),
		Expected: fmt.Sprintf("rated for %d cycles at 80%% capacity", r.RatedCycles)}
	switch {
	case ratio < 0.6:
		cyc.Status, cyc.Summary = Pass, fmt.Sprintf("%d cycles — %.0f%% of rated life used.", r.CycleCount, ratio*100)
	case ratio < 1:
		cyc.Status, cyc.Severity, cyc.Summary = Warn, Minor, fmt.Sprintf("%d cycles — %.0f%% of rated life used.", r.CycleCount, ratio*100)
	default:
		cyc.Status, cyc.Severity, cyc.Summary = Warn, Major, fmt.Sprintf("%d cycles — beyond the rated cycle life.", r.CycleCount)
	}
	if r.CycleCount == 0 && r.HealthPercent > 0 && r.HealthPercent < 95 {
		cyc.Status, cyc.Severity = Warn, Major
		cyc.Summary = "0 cycles reported but capacity is worn — cycle counter may have been reset (battery swapped or gauge reprogrammed)."
	}
	e.add(cyc)

	// Design capacity vs spec (original battery?)
	if r.DesignCapacity > 0 && len(r.ExpectedDesign) > 0 {
		match := false
		for _, x := range r.ExpectedDesign {
			if math.Abs(float64(r.DesignCapacity-x))/float64(x) <= 0.04 {
				match = true
			}
		}
		if match {
			e.add(Check{ID: "battery-original", Category: CatBattery, Title: "Battery matches factory spec", Status: Pass,
				Summary: "Design capacity matches the original battery for this model.", Expected: joinInts(r.ExpectedDesign, " mAh"), Actual: fmt.Sprintf("%d mAh", r.DesignCapacity)})
		} else {
			e.add(Check{ID: "battery-original", Category: CatBattery, Title: "Battery matches factory spec", Status: Warn, Severity: Major,
				Summary: "Design capacity differs from the factory battery — likely a replacement or third-party battery.", Expected: joinInts(r.ExpectedDesign, " mAh"), Actual: fmt.Sprintf("%d mAh", r.DesignCapacity),
				Advice: "Look for \"Unknown Part\" / \"Service\" messages in Settings › Battery › Battery Health and Settings › General › About."})
		}
	}

	// Temperature
	if r.TemperatureC > 0 {
		t := Check{ID: "battery-temp", Category: CatBattery, Title: "Battery temperature", Actual: fmt.Sprintf("%.1f °C", r.TemperatureC)}
		switch {
		case r.TemperatureC > 45:
			t.Status, t.Severity, t.Summary = Warn, Major, "Battery is running hot."
		case r.TemperatureC > 38:
			t.Status, t.Severity, t.Summary = Warn, Minor, "Battery is warm (charging or heavy use?)."
		default:
			t.Status, t.Summary = Pass, "Normal temperature."
		}
		e.add(t)
	}

	// Gauge serial
	if r.Serial != "" {
		e.add(Check{ID: "battery-serial", Category: CatBattery, Title: "Battery gauge identity", Status: Pass,
			Summary: "Battery management chip reports a serial number.", Actual: r.Serial})
	}
	return r
}

// --------------------------------------------------------------- hardware

func (e *evaluator) hardwareChecks() {
	// Panic logs
	if e.s.CrashesRead {
		var missing []string
		seen := map[string]bool{}
		for _, p := range e.s.Panics {
			for _, m := range p.MissingSensors {
				if !seen[m] {
					seen[m] = true
					missing = append(missing, m)
				}
			}
		}
		n := len(e.s.Panics)
		switch {
		case len(missing) > 0:
			e.add(Check{ID: "panics", Category: CatHardware, Title: "Kernel panics", Status: Fail, Severity: Major,
				Summary: fmt.Sprintf("%d panic report(s); the system reports missing sensors — a component is disconnected, faulty or non-original.", n),
				Actual:  "Missing: " + strings.Join(missing, ", "),
				Advice:  "Typical after a bad repair (e.g. charging port flex, microphone, proximity sensor). Sensor names: mic=microphone, prs=pressure/barometer, tg=thermal/gauge."})
		case n >= 3:
			e.add(Check{ID: "panics", Category: CatHardware, Title: "Kernel panics", Status: Warn, Severity: Major,
				Summary: fmt.Sprintf("%d panic reports found — the phone has been restarting unexpectedly.", n), Actual: firstPanic(e.s.Panics)})
		case n > 0:
			e.add(Check{ID: "panics", Category: CatHardware, Title: "Kernel panics", Status: Warn, Severity: Minor,
				Summary: fmt.Sprintf("%d panic report(s) found.", n), Actual: firstPanic(e.s.Panics)})
		default:
			e.add(Check{ID: "panics", Category: CatHardware, Title: "Kernel panics", Status: Pass,
				Summary: fmt.Sprintf("No kernel panics in %d stored diagnostic reports.", len(e.s.CrashReports))})
		}
	} else {
		e.add(Check{ID: "panics", Category: CatHardware, Title: "Kernel panics", Status: Skip, Summary: "Crash reports could not be read."})
	}

	// Baseband / modem
	bb := e.ld("BasebandVersion")
	if bb != "" {
		e.add(Check{ID: "baseband", Category: CatHardware, Title: "Cellular modem", Status: Pass,
			Summary: "Modem firmware is loaded and responding.", Actual: bb})
	} else if _, ok := e.s.Lockdown["BasebandVersion"]; ok || e.ld("InternationalMobileEquipmentIdentity") == "" {
		e.add(Check{ID: "baseband", Category: CatHardware, Title: "Cellular modem", Status: Fail, Severity: Major,
			Summary: "No modem firmware reported — cellular hardware may be faulty (no service)."})
	}

	// Radios
	wifi := e.ld("WiFiAddress")
	bt := e.ld("BluetoothAddress")
	if wifi != "" && bt != "" {
		e.add(Check{ID: "radios", Category: CatHardware, Title: "Wi-Fi & Bluetooth", Status: Pass,
			Summary: "Wi-Fi and Bluetooth chips report valid hardware addresses.", Actual: wifi + " · " + bt})
	} else {
		e.add(Check{ID: "radios", Category: CatHardware, Title: "Wi-Fi & Bluetooth", Status: Warn, Severity: Major,
			Summary: "Wi-Fi or Bluetooth hardware address missing — the wireless module may be faulty.",
			Actual:  firstNonEmpty(wifi, "—") + " · " + firstNonEmpty(bt, "—")})
	}

	// NAND / storage controller
	if nand := sub(e.s.Diagnostics, "NAND"); nand != nil {
		if st := str(nand["Status"]); st != "" {
			c := Check{ID: "nand", Category: CatHardware, Title: "Storage controller", Actual: st}
			if strings.EqualFold(st, "Success") || strings.EqualFold(st, "OK") {
				c.Status, c.Summary = Pass, "Storage self-test passed."
			} else {
				c.Status, c.Severity, c.Summary = Warn, Major, "Storage self-test did not report success."
			}
			e.add(c)
		}
	}

	// SIM
	if sim := e.ld("SIMStatus"); sim != "" {
		c := Check{ID: "sim", Category: CatHardware, Title: "SIM", Status: Info, Actual: sim}
		switch sim {
		case "kCTSIMSupportSIMStatusReady":
			c.Summary = "A SIM is inserted and ready."
		case "kCTSIMSupportSIMStatusNotInserted":
			c.Summary = "No SIM inserted — insert yours to test calls and data."
		case "kCTSIMSupportSIMStatusPINLocked":
			c.Summary = "SIM is PIN-locked."
		default:
			c.Summary = "SIM state: " + strings.TrimPrefix(sim, "kCTSIMSupportSIMStatus")
		}
		c.Advice = "Check Settings › General › About › Carrier Lock shows \"No SIM restrictions\"."
		e.add(c)
	}
}

func firstPanic(p []PanicLog) string {
	for _, x := range p {
		if x.Summary != "" {
			return x.Summary
		}
	}
	return ""
}

// --------------------------------------------------------------- identity table

func (e *evaluator) identity(model string) []KV {
	l := e.ld
	var kv []KV
	addKV := func(label, value string, mono bool) {
		if strings.TrimSpace(value) != "" {
			kv = append(kv, KV{Label: label, Value: value, Mono: mono})
		}
	}
	addKV("Model", model, false)
	addKV("Device name", l("DeviceName"), false)
	addKV("Identifier", l("ProductType"), true)
	addKV("Model number", l("RegulatoryModelNumber"), true)
	addKV("Part number", l("ModelNumber")+l("RegionInfo"), true)
	if r := e.db.Region(l("RegionInfo")); r != nil {
		addKV("Region", strings.Join(r.Countries, ", "), false)
	}
	addKV("Serial number", l("SerialNumber"), true)
	addKV("IMEI", l("InternationalMobileEquipmentIdentity"), true)
	addKV("IMEI 2", l("InternationalMobileEquipmentIdentity2"), true)
	addKV("MEID", l("MobileEquipmentIdentifier"), true)
	if v := l("ProductVersion"); v != "" {
		addKV("iOS", v+" ("+l("BuildVersion")+")", false)
	}
	if e.spec != nil {
		addKV("Chip", e.spec.Chip, false)
	}
	addKV("Board", l("HardwareModel"), true)
	if s := e.storageInfo(); s != nil {
		if t, ok := s["totalLabel"].(string); ok {
			addKV("Capacity", t, false)
		}
	}
	addKV("Colour code", firstNonEmpty(l("DeviceEnclosureColor"), l("DeviceColor")), true)
	addKV("Activation", l("ActivationState"), false)
	addKV("Baseband", l("BasebandVersion"), true)
	addKV("Wi-Fi address", l("WiFiAddress"), true)
	addKV("Bluetooth", l("BluetoothAddress"), true)
	if ecid, ok := num(e.s.Lockdown["UniqueChipID"]); ok {
		addKV("ECID", fmt.Sprintf("%X", uint64(ecid)), true)
	}
	addKV("UDID", firstNonEmpty(l("UniqueDeviceID"), e.s.UDID), true)
	addKV("ICCID", l("IntegratedCircuitCardIdentity"), true)
	addKV("Phone number", l("PhoneNumber"), true)
	return kv
}

func (e *evaluator) storageInfo() map[string]any {
	total, _ := num(e.dom("com.apple.disk_usage", "TotalDiskCapacity"))
	data, _ := num(e.dom("com.apple.disk_usage", "TotalDataCapacity"))
	avail, _ := num(e.dom("com.apple.disk_usage", "TotalDataAvailable"))
	if avail == 0 {
		avail, _ = num(e.dom("com.apple.disk_usage", "AmountDataAvailable"))
	}
	if total == 0 && e.s.FS != nil {
		total = float64(e.s.FS.TotalBytes)
		avail = float64(e.s.FS.FreeBytes)
	}
	if total == 0 {
		return nil
	}
	out := map[string]any{"totalBytes": total, "totalLabel": humanBytes(total)}
	if avail > 0 {
		used := total - avail
		out["freeBytes"] = avail
		out["usedBytes"] = used
		out["freeLabel"] = humanBytes(avail)
		out["usedLabel"] = humanBytes(used)
	}
	if data > 0 {
		out["dataBytes"] = data
	}
	return out
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}
