package android_test

import (
	"strings"
	"testing"

	smartcheck "github.com/mo3bdlaa/smart-mobile-check"
	"github.com/mo3bdlaa/smart-mobile-check/internal/android"
	"github.com/mo3bdlaa/smart-mobile-check/internal/fakedevice"
	"github.com/mo3bdlaa/smart-mobile-check/internal/inspect"
	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

func find(r *android.Result, id string) *inspect.Check {
	for i := range r.Checks {
		if r.Checks[i].ID == id {
			return &r.Checks[i]
		}
	}
	return nil
}

func db(t *testing.T) *specs.DB {
	d := specs.Load(smartcheck.Specs, "")
	for _, w := range d.Warnings {
		t.Errorf("spec warning: %s", w)
	}
	if len(d.Android()) < 80 {
		t.Errorf("expected the Android spec database, got %d models", len(d.Android()))
	}
	return d
}

// End to end over the real ADB wire protocol against the emulator.
func TestEndToEndGenuine(t *testing.T) {
	want := android.DemoSnapshot("a-healthy")
	srv, err := fakedevice.StartADB("127.0.0.1:0", want, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	t.Setenv("SMC_ADB_ADDR", srv.Addr())

	list, err := android.ListDevices()
	if err != nil || len(list) != 1 || list[0].State != "unauthorized" {
		t.Fatalf("first poll should be unauthorized: %+v %v", list, err)
	}
	if _, err := android.Collect(want.Serial, nil); err == nil {
		t.Fatal("collect must fail while unauthorized")
	}
	list, _ = android.ListDevices()
	list, _ = android.ListDevices()
	if list[0].State != "device" || list[0].Model != "SM-S928B" {
		t.Fatalf("after allow: %+v", list[0])
	}
	var labels []string
	got, err := android.Collect(want.Serial, func(i, n int, l string) { labels = append(labels, l) })
	if err != nil {
		t.Fatalf("collect: %v %v", err, got.Errors)
	}
	if len(got.Errors) > 0 {
		t.Errorf("step errors: %v", got.Errors)
	}
	if len(labels) != android.StepCount {
		t.Errorf("progress labels %d", len(labels))
	}
	checks := map[string]bool{
		"props":    got.Props["ro.product.model"] == "SM-S928B" && got.Props["ro.board.platform"] == "pineapple",
		"imei":     strings.Join(got.IMEIs, ",") == strings.Join(want.IMEIs, ","),
		"mem":      got.MemTotalKB == want.MemTotalKB,
		"data":     got.DataTotalKB == want.DataTotalKB && got.DataUsedKB == want.DataUsedKB,
		"screen":   got.ScreenSize == want.ScreenSize,
		"battery":  got.Battery["level"] == "73" && got.BatterySys["fg_asoc"] == "92",
		"accounts": got.AccountsRead && len(got.Accounts) == 0,
		"policy":   got.PolicyRead && got.DeviceOwner == "",
		"apps":     got.AppsScanned && len(got.Apps) == len(want.Apps),
		"selinux":  got.SELinux == "Enforcing",
		"sensors":  len(got.Sensors) == len(want.Sensors),
		"cameras":  got.CameraCount == 5,
		"dropbox":  got.Dropbox["SYSTEM_LAST_KMSG"] == 1,
	}
	for k, ok := range checks {
		if !ok {
			t.Errorf("%s not read correctly", k)
		}
	}

	r := android.Evaluate(got, db(t), "test", "used", "en", nil)
	for _, c := range r.Checks {
		if c.Status == inspect.Fail {
			t.Errorf("unexpected fail on genuine phone: %s — %s (%s)", c.ID, c.Summary, c.Actual)
		}
	}
	if r.Verdict == "risk" {
		t.Errorf("verdict %s score %d", r.Verdict, r.Score)
	}
	t.Logf("%s: score %d %s %v", r.ModelName, r.Score, r.Verdict, r.Counts)
}

func TestCloneAndLocksCaught(t *testing.T) {
	srv, err := fakedevice.StartADB("127.0.0.1:0", android.DemoSnapshot("a-clone"), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	t.Setenv("SMC_ADB_ADDR", srv.Addr())
	snap, err := android.Collect("0123456789ABCDEF", nil)
	if err != nil {
		t.Fatal(err)
	}
	r := android.Evaluate(snap, db(t), "test", "used", "ar", nil)
	ids := []string{"consistency", "accounts", "mdm", "knox-guard", "bootloader", "root", "selinux", "knox-warranty", "sensors", "serial", "imei"}
	if r.Spec != nil {
		// Spec-based checks need the model in the Android spec database.
		ids = append(ids, "android-version", "chip", "ram", "storage")
		if c := find(r, "screen"); c == nil || c.Status != inspect.Warn {
			t.Errorf("screen mismatch should warn: %+v", c)
		}
	} else {
		t.Log("Galaxy S24 Ultra spec not in database yet; spec-based checks not asserted")
	}
	for _, id := range ids {
		if c := find(r, id); c == nil || c.Status != inspect.Fail {
			t.Errorf("%s should fail: %+v", id, c)
		}
	}
	if r.Verdict != "risk" || !strings.Contains(r.VerdictText, "لا تشتري") {
		t.Errorf("verdict %s %q", r.Verdict, r.VerdictText)
	}
}

func TestNewPixelAndArabic(t *testing.T) {
	d := db(t)
	r := android.Evaluate(android.DemoSnapshot("a-new"), d, "test", "new", "ar", map[string]android.ManualResult{"box-seal": {ID: "box-seal", Status: inspect.Pass}, "body": {ID: "body", Status: inspect.Fail}})
	if find(r, "manual-box-seal") == nil || find(r, "manual-body") != nil {
		t.Error("manual tests not filtered by mode")
	}
	for _, c := range r.Checks {
		for _, v := range []string{c.Title, c.Summary, c.Advice} {
			if v != "" && !hasArabic(v) && v != "IMEI" {
				t.Errorf("%s untranslated: %q", c.ID, v)
			}
		}
	}
	// A used phone presented as new must be caught.
	u := android.Evaluate(android.DemoSnapshot("a-healthy"), d, "test", "new", "en", nil)
	if c := find(u, "new-cycles"); c == nil || c.Status != inspect.Fail {
		t.Errorf("used phone in new mode: %+v", c)
	}
	for _, m := range android.ManualTests {
		if m.TitleAr == "" || len(m.StepsAr) != len(m.Steps) {
			t.Errorf("manual %s missing Arabic", m.ID)
		}
	}
}

func hasArabic(s string) bool {
	for _, r := range s {
		if r >= 0x0600 && r <= 0x06FF {
			return true
		}
	}
	return false
}

func TestBatteryFromDumpsysOnly(t *testing.T) {
	s := android.DemoSnapshot("a-healthy")
	s.BatterySys = nil
	s.Battery["Charging cycles"] = "412"
	s.Battery["State of health"] = "88%"
	r := android.Evaluate(s, db(t), "test", "used", "en", nil)
	if !r.Battery.CyclesKnown || r.Battery.CycleCount != 412 || r.Battery.HealthPercent != 88 {
		t.Errorf("battery from dumpsys: %+v", r.Battery)
	}
}
