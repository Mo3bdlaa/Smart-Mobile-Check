package android

import (
	"time"

	"github.com/mo3bdlaa/smart-mobile-check/internal/inspect"
)

// DemoProfiles are the built-in sample Android phones.
var DemoProfiles = []struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Mode  string `json:"mode"`
	Ar    string `json:"titleAr"`
}{
	{"a-healthy", "Used Galaxy S24 Ultra — clean", "A genuine, healthy used Samsung.", inspect.ModeUsed, "جالاكسي S24 ألترا مستعمل — سليم"},
	{"a-clone", "\"Galaxy S24 Ultra\" — fake clone", "A clone with a cheap chip, locks and root.", inspect.ModeUsed, "\"جالاكسي S24 ألترا\" — تقليد"},
	{"a-new", "New Pixel 9 Pro — sealed", "A brand-new Google Pixel in new-device mode.", inspect.ModeNew, "بيكسل 9 برو جديد — متبرشم"},
}

func baseProps() map[string]string {
	return map[string]string{
		"ro.build.type": "user", "ro.build.tags": "release-keys", "ro.debuggable": "0", "ro.secure": "1",
		"ro.boot.verifiedbootstate": "green", "ro.boot.flash.locked": "1", "ro.boot.vbmeta.device_state": "locked",
		"gsm.sim.state": "ABSENT,ABSENT", "persist.sys.usb.config": "mtp,adb",
	}
}

// DemoSnapshot returns a realistic snapshot for a demo profile.
func DemoSnapshot(id string) *Snapshot {
	switch id {
	case "a-clone":
		return demoClone()
	case "a-new":
		return demoNewPixel()
	}
	return demoS24()
}

func demoS24() *Snapshot {
	p := baseProps()
	for k, v := range map[string]string{
		"ro.product.manufacturer": "samsung", "ro.product.brand": "samsung", "ro.product.model": "SM-S928B",
		"ro.product.device": "e3q", "ro.product.name": "e3qxxx", "ro.product.vendor.model": "SM-S928B",
		"ro.product.odm.model": "SM-S928B", "ro.product.system.model": "SM-S928B",
		"ro.build.fingerprint":        "samsung/e3qxxx/e3q:16/BP2A.250605.031.A3/S928BXXU5CYJ1:user/release-keys",
		"ro.vendor.build.fingerprint": "samsung/e3qxxx/e3q:16/BP2A.250605.031.A3/S928BXXU5CYJ1:user/release-keys",
		"ro.build.version.release":    "16", "ro.build.version.sdk": "36", "ro.build.version.security_patch": "2026-10-01",
		"ro.build.display.id": "BP2A.250605.031.A3.S928BXXU5CYJ1", "ro.board.platform": "pineapple", "ro.soc.model": "SM8650",
		"ro.serialno": "R5CX21ABCDE", "gsm.version.baseband": "S928BXXU5CYJ1", "ro.csc.sales_code": "EGY",
		"ro.boot.warranty_bit": "0", "knox.kg.state": "Completed", "gsm.sim.state": "READY,ABSENT",
	} {
		p[k] = v
	}
	return &Snapshot{
		Demo: true, CollectedAt: time.Now(), Serial: "R5CX21ABCDE", Props: p,
		MemTotalKB: 11_480_212, DataTotalKB: 230_118_400, DataUsedKB: 96_442_880,
		ScreenSize: "1440x3120", Density: "600",
		Battery:    map[string]string{"AC powered": "false", "USB powered": "true", "status": "2", "health": "2", "level": "73", "voltage": "4012", "temperature": "301", "technology": "Li-ion"},
		BatterySys: map[string]string{"battery_cycle": "286", "fg_asoc": "92", "charge_full_design": "4855000"},
		Accounts:   nil, AccountsRead: true, PolicyRead: true,
		Apps: []AppEntry{
			{BundleID: "com.whatsapp", Name: "com.whatsapp", Type: "User", Installer: "com.android.vending"},
			{BundleID: "com.instagram.android", Name: "com.instagram.android", Type: "User", Installer: "com.android.vending"},
			{BundleID: "com.samsung.android.app.notes", Name: "com.samsung.android.app.notes", Type: "System"},
			{BundleID: "com.google.android.gms", Name: "com.google.android.gms", Type: "System"},
		},
		AppsScanned: true, SELinux: "Enforcing",
		Sensors:     []string{"accelerometer", "gyroscope", "light", "magnetic_field", "pressure", "proximity", "step_counter"},
		SensorsRead: true, CameraCount: 5, CamerasRead: true,
		Dropbox:     map[string]int{"SYSTEM_BOOT": 3, "data_app_crash": 4, "SYSTEM_LAST_KMSG": 1},
		DropboxRead: true, IMEIs: []string{"356789123456780", "356789123456798"},
		Settings: map[string]string{"device_provisioned": "1", "user_setup_complete": "1"},
		Errors:   map[string]string{},
	}
}

func demoClone() *Snapshot {
	s := demoS24()
	s.Serial = "0123456789ABCDEF"
	for k, v := range map[string]string{
		"ro.board.platform": "mt6765", "ro.soc.model": "MT6765", "ro.hardware": "mt6765",
		"ro.product.vendor.model": "S24 Ultra", "ro.serialno": "0123456789ABCDEF",
		"ro.build.fingerprint":        "samsung/e3qxxx/e3q:16/BP2A.250605.031.A3/S928BXXU5CYJ1:user/release-keys",
		"ro.vendor.build.fingerprint": "alps/full_k62v1_64_bsp/k62v1_64_bsp:11/RP1A.200720.011/1:user/release-keys",
		"ro.build.version.release":    "11", "ro.build.version.security_patch": "2021-07-05",
		"ro.boot.verifiedbootstate": "orange", "ro.boot.flash.locked": "0", "ro.boot.warranty_bit": "1",
		"knox.kg.state": "Active",
	} {
		s.Props[k] = v
	}
	s.MemTotalKB = 3_846_000
	s.DataTotalKB = 51_000_000
	s.ScreenSize = "720x1600"
	s.BatterySys = map[string]string{"charge_full_design": "4000000", "charge_full": "2980000", "cycle_count": "640"}
	s.Battery["health"] = "2"
	s.Accounts = []Account{{Type: "com.google", Name: "ah••••@gmail.com"}, {Type: "com.osp.app.signin", Name: "ah••••@gmail.com"}}
	s.DeviceOwner = "com.example.mdm"
	s.SuPaths = []string{"/system/xbin/su"}
	s.SELinux = "Permissive"
	s.Apps = append(s.Apps, AppEntry{BundleID: "com.topjohnwu.magisk", Name: "com.topjohnwu.magisk", Type: "User"},
		AppEntry{BundleID: "com.chelpus.lackypatch", Name: "com.chelpus.lackypatch", Type: "User"})
	s.Sensors = []string{"accelerometer", "light"}
	s.CameraCount = 2
	s.Dropbox = map[string]int{"SYSTEM_LAST_KMSG": 4, "system_server_crash": 6}
	s.IMEIs = []string{"356789123456781"}
	return s
}

func demoNewPixel() *Snapshot {
	p := baseProps()
	for k, v := range map[string]string{
		"ro.product.manufacturer": "Google", "ro.product.brand": "google", "ro.product.model": "Pixel 9 Pro",
		"ro.product.device": "caiman", "ro.product.name": "caiman", "ro.product.vendor.model": "Pixel 9 Pro",
		"ro.build.fingerprint":        "google/caiman/caiman:16/BP3A.251005.004/13950000:user/release-keys",
		"ro.vendor.build.fingerprint": "google/caiman/caiman:16/BP3A.251005.004/13950000:user/release-keys",
		"ro.build.version.release":    "16", "ro.build.version.sdk": "36", "ro.build.version.security_patch": "2026-11-05",
		"ro.build.display.id": "BP3A.251005.004", "ro.board.platform": "zumapro", "ro.soc.model": "Tensor G4",
		"ro.serialno": "4B061FDJH00123", "gsm.version.baseband": "g5400c-251001-251005-B-14009999",
	} {
		p[k] = v
	}
	return &Snapshot{
		Demo: true, CollectedAt: time.Now(), Serial: "4B061FDJH00123", Props: p,
		MemTotalKB: 15_620_000, DataTotalKB: 226_000_000, DataUsedKB: 9_800_000,
		ScreenSize: "1280x2856", Density: "480",
		Battery:      map[string]string{"USB powered": "true", "status": "2", "health": "2", "level": "64", "voltage": "3985", "temperature": "268"},
		BatterySys:   map[string]string{"cycle_count": "1", "charge_full": "4690000", "charge_full_design": "4700000"},
		AccountsRead: true, PolicyRead: true,
		Apps: []AppEntry{
			{BundleID: "com.google.android.apps.photos", Name: "com.google.android.apps.photos", Type: "System"},
			{BundleID: "com.google.android.gms", Name: "com.google.android.gms", Type: "System"},
		},
		AppsScanned: true, SELinux: "Enforcing",
		Sensors:     []string{"accelerometer", "gyroscope", "light", "magnetic_field", "pressure", "proximity"},
		SensorsRead: true, CameraCount: 4, CamerasRead: true,
		Dropbox: map[string]int{"SYSTEM_BOOT": 1}, DropboxRead: true,
		Settings: map[string]string{"device_provisioned": "1", "user_setup_complete": "1"},
		Errors:   map[string]string{},
	}
}
