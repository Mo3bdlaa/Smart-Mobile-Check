package iphone

import (
	"regexp"
	"strings"
	"testing"
)

var quoted = regexp.MustCompile(`"[^"]*"|\([^)]*\)`)
var englishRun = regexp.MustCompile(`\b[a-z]{2,} [a-z]{2,} [a-z]{2,}\b`)

// variants builds snapshots that drive every branch of the check engine.
func variants() map[string]*Snapshot {
	out := map[string]*Snapshot{}
	add := func(name, base string, mut func(s *Snapshot)) {
		s := DemoSnapshot(base)
		mut(s)
		out[name] = s
	}
	add("healthy", "healthy", func(s *Snapshot) {})
	add("tampered", "tampered", func(s *Snapshot) {})
	add("new", "new", func(s *Snapshot) {})
	add("worn", "healthy", func(s *Snapshot) {
		s.Battery["NominalChargeCapacity"] = uint64(4150) // ~82%
		s.Battery["CycleCount"] = uint64(800)
		s.Battery["Temperature"] = uint64(4100)
	})
	add("dead", "healthy", func(s *Snapshot) {
		s.Battery["NominalChargeCapacity"] = uint64(3500)
		s.Battery["CycleCount"] = uint64(1300)
		s.Battery["Temperature"] = uint64(4800)
		s.Battery["DesignCapacity"] = uint64(4000)
	})
	add("nobattery", "healthy", func(s *Snapshot) { s.Battery = nil; s.Diagnostics = nil })
	add("unknown", "healthy", func(s *Snapshot) { s.Lockdown["ProductType"] = "iPhone99,9"; s.FS.Model = "iPhone99,9" })
	add("misc", "healthy", func(s *Snapshot) {
		s.Lockdown["BuildVersion"] = "23B5050f"
		s.Lockdown["ProductionSOC"] = false
		s.Lockdown["ModelNumber"] = "3MYP4"
		s.Lockdown["RegionInfo"] = "QQ/A"
		s.Lockdown["SIMStatus"] = "kCTSIMSupportSIMStatusReady"
		s.Lockdown["SerialNumber"] = "bad!"
		s.Lockdown["InternationalMobileEquipmentIdentity2"] = "356789123456791"
		s.Lockdown["BluetoothAddress"] = ""
		s.Lockdown["HardwareModel"] = "X99AP"
		s.Lockdown["RegulatoryModelNumber"] = "A0000"
		s.Lockdown["ChipID"] = uint64(0x8120)
		s.Lockdown["UniqueDeviceID"] = "00008120-001A2C3E0E10001C"
		s.Lockdown["ProductVersion"] = "17.0"
		s.Lockdown["ActivationState"] = "Unactivated"
		s.Lockdown["BrickState"] = true
		s.Lockdown["PasswordProtected"] = false
		s.Domains["com.apple.security.mac.amfi"] = map[string]any{"DeveloperModeStatus": true}
		s.Domains["com.apple.disk_usage"] = map[string]any{"TotalDiskCapacity": uint64(300_000_000_000)}
		s.Domains["com.apple.fmip"] = nil
		s.CloudConfig = map[string]any{"IsSupervised": true}
		s.Profiles = []ProfileEntry{{Identifier: "x", DisplayName: "VPN"}}
		s.Apps = append(s.Apps, AppEntry{BundleID: "com.rileytestut.AltStore", Name: "AltStore", Type: "User"},
			AppEntry{BundleID: "com.x.scarlet", Name: "Scarlet", Type: "User"})
		s.Diagnostics = map[string]any{"NAND": map[string]any{"Status": "Failure"}}
		s.Panics = append(s.Panics, PanicLog{Summary: "x"}, PanicLog{Summary: "y"})
	})
	add("misc2", "healthy", func(s *Snapshot) {
		s.Lockdown["ModelNumber"] = "FYP4"
		s.Lockdown["SIMStatus"] = "kCTSIMSupportSIMStatusPINLocked"
		s.Lockdown["SerialNumber"] = ""
		s.Lockdown["InternationalMobileEquipmentIdentity"] = ""
		s.Lockdown["BasebandVersion"] = ""
		s.Lockdown["ProductVersion"] = "99.0"
		s.Gestalt = map[string]any{"BuildVersion": "22A1"}
		s.CloudConfig = nil
		s.AppsScanned = false
		s.CrashesRead = false
		s.Battery["CycleCount"] = uint64(0)
		s.Battery["NominalChargeCapacity"] = uint64(4500)
	})
	add("misc3", "healthy", func(s *Snapshot) {
		s.Lockdown["ModelNumber"] = "NYP4"
		s.Lockdown["SIMStatus"] = "kCTSIMSupportSIMStatusOther"
		s.Lockdown["ActivationState"] = "Activated"
		s.Apps = append(s.Apps, AppEntry{BundleID: "com.evil.thing", Name: "Thing", Type: "System"})
		s.Profiles = []ProfileEntry{{Identifier: "x", DisplayName: "VPN"}}
		s.Battery["CycleCount"] = uint64(12)
		s.Domains["com.apple.disk_usage"] = map[string]any{"TotalDiskCapacity": uint64(256_000_000_000), "TotalDataCapacity": uint64(243_000_000_000), "TotalDataAvailable": uint64(100_000_000_000)}
	})
	add("misc4", "healthy", func(s *Snapshot) {
		s.Lockdown["ModelNumber"] = "XYP4"
		s.Lockdown["UniqueDeviceID"] = "aabbccddeeff00112233445566778899aabbccdd"
		s.Apps = append(s.Apps, AppEntry{BundleID: "com.x.trollstoreish", Name: "TrollStore Helper X", Type: "User"})
	})
	return out
}

func TestArabicCoversEveryCheck(t *testing.T) {
	db := loadDB(t)
	seen := map[string]bool{}
	for name, snap := range variants() {
		for _, mode := range []string{ModeUsed, ModeNew} {
			r := EvaluateMode(snap, db, "test", mode)
			manual := map[string]ManualResult{}
			for _, mt := range ManualTests {
				manual[mt.ID] = ManualResult{ID: mt.ID, Status: []Status{Pass, Fail, Skip}[len(manual)%3]}
			}
			r = ApplyManual(r, manual)
			ar := Localize(r, "ar")
			if !hasArabic(ar.VerdictText) {
				t.Errorf("%s/%s verdict untranslated: %q", name, mode, ar.VerdictText)
			}
			for i, c := range ar.Checks {
				seen[c.ID] = true
				for field, v := range map[string]string{"title": c.Title, "summary": c.Summary, "advice": c.Advice} {
					if v != "" && !hasArabic(v) {
						t.Errorf("%s/%s %s %s untranslated: %q", name, mode, r.Checks[i].ID, field, v)
					}
					if m := englishRun.FindString(quoted.ReplaceAllString(v, "")); m != "" {
						t.Errorf("%s/%s %s %s half-translated (%q): %q", name, mode, r.Checks[i].ID, field, m, v)
					}
				}
				if strings.Contains(c.Expected, "rated for") || strings.Contains(c.Expected, "or later") {
					t.Errorf("%s expected untranslated: %q", c.ID, c.Expected)
				}
			}
			for _, kv := range ar.Identity {
				if kv.Label == r.Identity[0].Label && kv.Label != "الموديل" {
					t.Errorf("identity label untranslated: %q", kv.Label)
				}
			}
		}
	}
	if len(seen) < 45 {
		t.Errorf("only %d distinct checks exercised", len(seen))
	}
	for _, mt := range ManualTests {
		if mt.TitleAr == "" || mt.GroupAr == "" || len(mt.StepsAr) != len(mt.Steps) {
			t.Errorf("manual test %s missing Arabic (%d/%d steps)", mt.ID, len(mt.StepsAr), len(mt.Steps))
		}
	}
	if Localize(EvaluateMode(DemoSnapshot("healthy"), db, "x", ModeUsed), "en").Checks[0].Title != "Model recognised" {
		t.Error("English must stay unchanged")
	}
}
