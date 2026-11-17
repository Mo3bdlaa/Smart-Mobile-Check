package iphone

import (
	"testing"

	smartcheck "github.com/mo3bdlaa/smart-mobile-check"
	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

func loadDB(t *testing.T) *specs.DB {
	t.Helper()
	db := specs.Load(smartcheck.Specs, "")
	if len(db.Warnings) > 0 {
		t.Fatalf("spec warnings: %v", db.Warnings)
	}
	if len(db.All()) < 40 {
		t.Fatalf("expected the full iPhone database, got %d", len(db.All()))
	}
	return db
}

func find(r *Result, id string) *Check {
	for i := range r.Checks {
		if r.Checks[i].ID == id {
			return &r.Checks[i]
		}
	}
	return nil
}

func TestHealthyDemo(t *testing.T) {
	db := loadDB(t)
	r := Evaluate(DemoSnapshot("healthy"), db, "test")
	if r.ModelName != "iPhone 17 Pro Max" {
		t.Fatalf("model = %q", r.ModelName)
	}
	for _, c := range r.Checks {
		if c.Status == Fail {
			t.Errorf("unexpected fail: %s — %s (%s)", c.ID, c.Summary, c.Actual)
		}
	}
	for _, id := range []string{"udid", "imei", "storage", "battery-original", "findmy", "ios-plausible", "a-number", "board"} {
		if c := find(r, id); c == nil || c.Status != Pass {
			t.Errorf("%s should pass, got %+v", id, c)
		}
	}
	if r.Verdict != "good" {
		t.Errorf("verdict = %s (%d)", r.Verdict, r.Score)
	}
}

func TestTamperedDemo(t *testing.T) {
	db := loadDB(t)
	r := Evaluate(DemoSnapshot("tampered"), db, "test")
	want := map[string]Status{
		"model-consistency": Fail, "udid": Fail, "serial": Fail, "findmy": Fail, "mdm": Fail,
		"jb-afc2": Fail, "jb-apps": Fail, "imei": Fail, "panics": Fail, "battery-original": Warn,
		"battery-cycles": Warn, "part-number": Warn,
	}
	for id, st := range want {
		c := find(r, id)
		if c == nil || c.Status != st {
			t.Errorf("%s: want %s, got %+v", id, st, c)
		}
	}
	if r.Verdict != "risk" {
		t.Errorf("verdict = %s", r.Verdict)
	}
}

func TestManualMerge(t *testing.T) {
	db := loadDB(t)
	base := Evaluate(DemoSnapshot("healthy"), db, "test")
	r := ApplyManual(base, map[string]ManualResult{"account": {ID: "account", Status: Fail}})
	if r.Verdict != "risk" {
		t.Errorf("failed critical hands-on test should make verdict risk, got %s", r.Verdict)
	}
	if len(base.Checks) == len(r.Checks) {
		t.Errorf("manual check not merged")
	}
}

func TestHelpers(t *testing.T) {
	if !luhnValid("356789123456780") || luhnValid("356789123456781") {
		t.Error("luhn")
	}
	if compareVersions("18.6.2", "18") != 0 || compareVersions("26.1", "18") != 1 || compareVersions("17.0", "18.1") != -1 {
		t.Error("compareVersions")
	}
}
