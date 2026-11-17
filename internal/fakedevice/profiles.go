package fakedevice

import "fmt"

// IPhone15Pro returns a realistic, genuine-looking used iPhone 15 Pro that is
// not yet trusted by the host (the user has to tap "Trust" once).
func IPhone15Pro() *Device {
	chip := uint64(0x8130)
	ecid := uint64(0x000E1D2C3B4A5968)
	udid := fmt.Sprintf("%08X-%016X", chip, ecid)
	panic1 := `{"bug_type":"210","timestamp":"2026-10-21 09:14:03.00 +0300","os_version":"iPhone OS 26.0.1 (23A355)","incident_id":"8C1E"}
{"product":"iPhone16,1","panicString":"panic(cpu 2 caller 0xfffffff01b2c4e10): watchdog timeout: no checkins from watchdogd in 182 seconds\nDebugger message: panic\nMemory ID: 0x6\n"}`
	return &Device{
		UDID:       udid,
		TrustAfter: 2,
		Values: map[string]any{
			"ActivationState": "Activated", "BasebandVersion": "2.60.01", "BasebandStatus": "BBInfoAvailable",
			"BluetoothAddress": "f4:d4:88:5a:21:c3", "BoardId": uint64(4), "BuildVersion": "23B85",
			"CPUArchitecture": "arm64e", "ChipID": chip, "DeviceClass": "iPhone", "DeviceColor": "1",
			"DeviceEnclosureColor": "9", "DeviceName": "Ahmed's iPhone", "FirmwareVersion": "iBoot-11881.42.2",
			"HardwareModel": "D83AP", "HardwarePlatform": "t8130", "InternationalMobileEquipmentIdentity": "353028470212832",
			"InternationalMobileEquipmentIdentity2": "353028470364211", "ModelNumber": "MTUV3", "PasswordProtected": true,
			"ProductName": "iPhone OS", "ProductType": "iPhone16,1", "ProductVersion": "26.1", "ProductionSOC": true,
			"RegionInfo": "AB/A", "RegulatoryModelNumber": "A3102", "SerialNumber": "G6WQ9X2LJ7",
			"SIMStatus": "kCTSIMSupportSIMStatusReady", "TimeZone": "Asia/Riyadh", "UniqueChipID": ecid,
			"UniqueDeviceID": udid, "WiFiAddress": "f4:d4:88:5a:21:c2", "TrustedHostAttached": true,
		},
		Domains: map[string]map[string]any{
			"com.apple.disk_usage": {"TotalDiskCapacity": uint64(256_060_514_304), "TotalDataCapacity": uint64(243_571_458_048),
				"TotalDataAvailable": uint64(97_314_201_600), "TotalSystemCapacity": uint64(12_489_056_256)},
			"com.apple.mobile.battery":    {"BatteryCurrentCapacity": uint64(64), "BatteryIsCharging": false, "ExternalConnected": true},
			"com.apple.fmip":              {"IsAssociated": false},
			"com.apple.international":     {"Language": "ar", "Locale": "ar_SA"},
			"com.apple.PurpleBuddy":       {"SetupDone": true},
			"com.apple.security.mac.amfi": {"DeveloperModeStatus": false},
		},
		Battery: map[string]any{
			"CycleCount": uint64(431), "DesignCapacity": uint64(3274), "NominalChargeCapacity": uint64(2913),
			"AppleRawMaxCapacity": uint64(2861), "Temperature": uint64(2985), "Voltage": uint64(3917),
			"InstantAmperage": int64(-412), "CurrentCapacity": uint64(64), "IsCharging": false,
			"BatteryData": map[string]any{"Serial": "F5D3341A2Q1PXWH1M", "CycleCount": uint64(431), "DesignCapacity": uint64(3274)},
		},
		Diagnostics: map[string]any{"NAND": map[string]any{"Status": "Success"}, "WiFi": map[string]any{"Active": "true", "Status": "Success"}},
		Apps: []map[string]any{
			app("com.apple.mobilesafari", "Safari", "System"), app("com.apple.camera", "Camera", "System"),
			app("com.apple.Preferences", "Settings", "System"), app("net.whatsapp.WhatsApp", "WhatsApp", "User"),
			app("com.burbn.instagram", "Instagram", "User"), app("com.google.Maps", "Google Maps", "User"),
			app("com.careem.acma", "Careem", "User"), app("com.apple.Keynote", "Keynote", "User"),
		},
		CloudConfig: map[string]any{"CloudConfigurationUIComplete": true, "IsSupervised": false},
		CrashFiles: map[string]string{
			"panic-full-2026-10-21-091403.000.ips":        panic1,
			"JetsamEvent-2026-11-02-221004.ips":           `{"bug_type":"298"}` + "\n{}",
			"Retired/stacks-2026-08-14-101010.ips":        `{"bug_type":"288"}` + "\n{}",
			"Analytics-2026-11-15-030000.ips.ca.synced":   "{}",
			"Assistant/SiriSearchFeedback-2026-11-01.ips": "{}",
		},
		FSModel: "iPhone16,1",
		FSTotal: 243_571_458_048,
		FSFree:  97_314_201_600,
	}
}

// Tampered returns a phone with spoofed readings and locks, to exercise the
// failure paths end to end.
func Tampered() *Device {
	d := IPhone15Pro()
	d.TrustAfter = 0
	d.Values["SerialNumber"] = "C39XK2ABHG7F"
	d.Values["UniqueDeviceID"] = "00008130-000E1D2C3B4A0000"
	d.UDID = "00008130-000E1D2C3B4A0000"
	d.Values["InternationalMobileEquipmentIdentity"] = "353028470212833"
	d.FSModel = "iPhone14,2"
	d.Domains["com.apple.fmip"] = map[string]any{"IsAssociated": true}
	d.Battery["DesignCapacity"] = uint64(2900)
	d.Battery["CycleCount"] = uint64(0)
	d.AFC2 = true
	d.Apps = append(d.Apps, app("com.opa334.TrollStore", "TrollStore", "System"), app("com.tigisoftware.Filza", "Filza", "User"))
	d.CloudConfig = map[string]any{"OrganizationName": "Example Logistics LLC", "IsMDMUnremovable": true, "IsSupervised": true}
	d.Profiles = []Profile{{ID: "com.example.mdm", Name: "Example Device Management", Description: "MDM", RemovalDisallowed: true}}
	d.CrashFiles["panic-full-2026-11-12-120000.000.ips"] = `{"bug_type":"210","timestamp":"2026-11-12 12:00:00.00 +0300","os_version":"iPhone OS 26.1 (23B85)"}
{"panicString":"panic(cpu 0 caller 0xfffffff0): SMC PANIC - ASSERTION FAILED: Missing sensor(s): mic1, prs0\n"}`
	return d
}

func app(id, name, typ string) map[string]any {
	return map[string]any{"CFBundleIdentifier": id, "CFBundleDisplayName": name, "CFBundleName": name,
		"CFBundleShortVersionString": "1.0", "ApplicationType": typ}
}
