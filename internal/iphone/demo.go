package iphone

import (
	"fmt"
	"time"
)

// DemoProfiles lists the built-in sample devices used by demo mode, so the
// interface can be explored without an iPhone connected.
var DemoProfiles = []struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Mode  string `json:"mode"`
}{
	{"healthy", "Used iPhone 17 Pro Max — clean", "A genuine, healthy used phone with a few minor notes.", ModeUsed},
	{"tampered", "Used iPhone 13 Pro — red flags", "Activation Lock, jailbreak, replaced battery and spoofed readings.", ModeUsed},
	{"new", "New iPhone 17 Pro Max — sealed", "A brand-new phone checked in new-device mode.", ModeNew},
}

// DemoSnapshot builds a realistic snapshot for a demo profile.
func DemoSnapshot(profile string) *Snapshot {
	switch profile {
	case "tampered":
		return demoTampered()
	case "new":
		return demoNew()
	}
	return demoHealthy()
}

func demoHealthy() *Snapshot {
	chip := uint64(0x8150)
	ecid := uint64(0x001A2C3E0E10001C)
	return &Snapshot{
		Demo:           true,
		CollectedAt:    time.Now(),
		UDID:           fmt.Sprintf("%08X-%016X", chip, ecid),
		ConnectionType: "USB",
		Lockdown: map[string]any{
			"DeviceName":                            "Sample iPhone",
			"ProductType":                           "iPhone18,2",
			"HardwareModel":                         "V54AP",
			"RegulatoryModelNumber":                 "A3257",
			"ModelNumber":                           "MFYP4",
			"RegionInfo":                            "LL/A",
			"SerialNumber":                          "K4F7QW9M2X",
			"ProductVersion":                        "26.1",
			"BuildVersion":                          "23B85",
			"ChipID":                                chip,
			"UniqueChipID":                          ecid,
			"UniqueDeviceID":                        fmt.Sprintf("%08X-%016X", chip, ecid),
			"ProductionSOC":                         true,
			"ActivationState":                       "Activated",
			"PasswordProtected":                     true,
			"InternationalMobileEquipmentIdentity":  "356789123456780",
			"InternationalMobileEquipmentIdentity2": "356789123456798",
			"BasebandVersion":                       "1.10.02",
			"WiFiAddress":                           "a4:c3:61:2b:9e:10",
			"BluetoothAddress":                      "a4:c3:61:2b:9e:11",
			"DeviceEnclosureColor":                  "3",
			"SIMStatus":                             "kCTSIMSupportSIMStatusNotInserted",
			"CPUArchitecture":                       "arm64e",
		},
		Domains: map[string]map[string]any{
			"com.apple.disk_usage":     {"TotalDiskCapacity": uint64(256_000_000_000), "TotalDataCapacity": uint64(243_000_000_000), "TotalDataAvailable": uint64(181_400_000_000)},
			"com.apple.mobile.battery": {"BatteryCurrentCapacity": uint64(78), "BatteryIsCharging": true, "ExternalConnected": true},
			"com.apple.fmip":           {"IsAssociated": false},
		},
		Battery: map[string]any{
			"CycleCount": uint64(212), "DesignCapacity": uint64(5088), "NominalChargeCapacity": uint64(4681),
			"Temperature": uint64(3120), "Voltage": uint64(4021), "Serial": "F8Y4512Z9KQ1N7A",
		},
		BatterySource: "AppleSmartBattery",
		Diagnostics:   map[string]any{"NAND": map[string]any{"Status": "Success"}},
		FS:            &FSInfo{Model: "iPhone18,2", TotalBytes: 243_000_000_000, FreeBytes: 181_400_000_000},
		Apps: []AppEntry{
			{BundleID: "com.apple.mobilesafari", Name: "Safari", Type: "System"},
			{BundleID: "com.apple.camera", Name: "Camera", Type: "System"},
			{BundleID: "net.whatsapp.WhatsApp", Name: "WhatsApp", Type: "User"},
			{BundleID: "com.spotify.client", Name: "Spotify", Type: "User"},
		},
		AppsScanned:  true,
		ProfilesRead: true,
		CloudConfig:  map[string]any{},
		CrashReports: []string{"JetsamEvent-2026-10-02.ips", "Analytics-2026-11-10.ips.ca.synced", "panic-full-2026-09-28-101522.000.ips"},
		CrashesRead:  true,
		Panics: []PanicLog{
			{File: "panic-full-2026-09-28-101522.000.ips", Timestamp: "2026-09-28 10:15:22", OSVersion: "iPhone OS 26.0", Summary: "panic(cpu 4 caller 0xfffffff0283c1a4c): userspace watchdog timeout: no successful checkins from thermalmonitord"},
		},
		AFC2Probed: true,
		Errors:     map[string]string{},
	}
}

func demoTampered() *Snapshot {
	chip := uint64(0x8110)
	ecid := uint64(0x000A11223344556F)
	return &Snapshot{
		Demo:           true,
		CollectedAt:    time.Now(),
		UDID:           "00008110-000A1122334455AA",
		ConnectionType: "USB",
		Lockdown: map[string]any{
			"DeviceName":                           "iPhone",
			"ProductType":                          "iPhone14,2",
			"HardwareModel":                        "D63AP",
			"RegulatoryModelNumber":                "A2638",
			"ModelNumber":                          "FLTP3",
			"RegionInfo":                           "AB/A",
			"SerialNumber":                         "H7XQ2M9LKP",
			"ProductVersion":                       "26.1",
			"BuildVersion":                         "23B85",
			"ChipID":                               chip,
			"UniqueChipID":                         ecid,
			"UniqueDeviceID":                       "00008110-000A1122334455AA",
			"ProductionSOC":                        true,
			"ActivationState":                      "Activated",
			"PasswordProtected":                    false,
			"InternationalMobileEquipmentIdentity": "353912110837465",
			"BasebandVersion":                      "4.02.01",
			"WiFiAddress":                          "8c:86:1e:44:01:7a",
			"BluetoothAddress":                     "8c:86:1e:44:01:7b",
			"SIMStatus":                            "kCTSIMSupportSIMStatusReady",
		},
		Domains: map[string]map[string]any{
			"com.apple.disk_usage":     {"TotalDiskCapacity": uint64(512_000_000_000), "TotalDataAvailable": uint64(402_000_000_000)},
			"com.apple.mobile.battery": {"BatteryCurrentCapacity": uint64(54)},
			"com.apple.fmip":           {"IsAssociated": true},
		},
		Battery: map[string]any{
			"CycleCount": uint64(0), "DesignCapacity": uint64(3500), "NominalChargeCapacity": uint64(2690),
			"Temperature": uint64(3390), "Voltage": uint64(3874),
		},
		BatterySource: "AppleSmartBattery",
		Gestalt:       map[string]any{"ProductType": "iPhone14,2", "SerialNumber": "C39XK2ABHG7F"},
		FS:            &FSInfo{Model: "iPhone12,3", TotalBytes: 512_000_000_000, FreeBytes: 402_000_000_000},
		Apps: []AppEntry{
			{BundleID: "com.apple.mobilesafari", Name: "Safari", Type: "System"},
			{BundleID: "com.opa334.TrollStore", Name: "TrollStore", Type: "System"},
			{BundleID: "com.tigisoftware.Filza", Name: "Filza", Type: "System"},
			{BundleID: "org.coolstar.SileoStore", Name: "Sileo", Type: "User"},
		},
		AppsScanned:  true,
		ProfilesRead: true,
		Profiles:     []ProfileEntry{{Identifier: "com.example.mdm", DisplayName: "Example Corp Device Management", RemovalDisallowed: true}},
		CloudConfig:  map[string]any{"OrganizationName": "Example Corp", "IsMDMUnremovable": true, "IsSupervised": true},
		CrashesRead:  true,
		CrashReports: []string{"panic-full-2026-11-01-080011.ips", "panic-full-2026-11-03-191201.ips", "panic-full-2026-11-09-120000.ips"},
		Panics: []PanicLog{
			{File: "panic-full-2026-11-09-120000.ips", Timestamp: "2026-11-09 12:00:00", Summary: "panic(cpu 0 caller 0xfffffff02...): SMC PANIC - ASSERTION FAILED: Missing sensor(s): mic1, prs0", MissingSensors: []string{"mic1", "prs0"}},
			{File: "panic-full-2026-11-03-191201.ips", Timestamp: "2026-11-03 19:12:01", Summary: "Missing sensor(s): mic1", MissingSensors: []string{"mic1"}},
			{File: "panic-full-2026-11-01-080011.ips", Timestamp: "2026-11-01 08:00:11", Summary: "userspace watchdog timeout"},
		},
		AFC2Probed: true,
		AFC2Open:   true,
		Errors:     map[string]string{},
	}
}

func demoNew() *Snapshot {
	s := demoHealthy()
	s.Lockdown["DeviceName"] = "iPhone"
	s.Lockdown["ActivationState"] = "Unactivated"
	s.Lockdown["PasswordProtected"] = false
	s.Lockdown["SerialNumber"] = "M2QX7KV9TD"
	s.Domains["com.apple.disk_usage"] = map[string]any{"TotalDiskCapacity": uint64(256_000_000_000), "TotalDataCapacity": uint64(243_000_000_000), "TotalDataAvailable": uint64(234_100_000_000)}
	s.Domains["com.apple.mobile.battery"] = map[string]any{"BatteryCurrentCapacity": uint64(81), "ExternalConnected": true}
	s.Battery = map[string]any{"CycleCount": uint64(1), "DesignCapacity": uint64(5088), "NominalChargeCapacity": uint64(5141),
		"Temperature": uint64(2710), "Voltage": uint64(4105), "Serial": "F8Y5108K2QW1N3B"}
	s.Apps = []AppEntry{
		{BundleID: "com.apple.mobilesafari", Name: "Safari", Type: "System"},
		{BundleID: "com.apple.camera", Name: "Camera", Type: "System"},
		{BundleID: "com.apple.Keynote", Name: "Keynote", Type: "User"},
	}
	s.CrashReports = []string{"Analytics-2026-11-17-090000.ips.ca.synced"}
	s.Panics = nil
	return s
}
