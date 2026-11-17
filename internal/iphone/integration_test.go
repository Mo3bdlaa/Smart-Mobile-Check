package iphone

import (
	"testing"

	"github.com/mo3bdlaa/smart-mobile-check/internal/fakedevice"
)

// TestEndToEndAgainstEmulatedDevice runs the real collector over the real
// wire protocols (usbmux, lockdown + TLS, pairing, diagnostics_relay,
// installation_proxy, MCInstall, AFC, crash reports) against the emulator.
func TestEndToEndAgainstEmulatedDevice(t *testing.T) {
	dev := fakedevice.IPhone15Pro()
	srv, err := fakedevice.Start("127.0.0.1:0", dev)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	t.Setenv("USBMUXD_SOCKET_ADDRESS", "tcp://"+srv.Addr())

	if err := ServiceAvailable(); err != nil {
		t.Fatalf("service: %v", err)
	}
	list, err := ListDevices()
	if err != nil || len(list) != 1 {
		t.Fatalf("devices: %v %v", list, err)
	}
	d := list[0]
	if d.ProductType != "iPhone16,1" || d.ProductVersion != "26.1" || d.Paired || d.PairState != "unpaired" {
		t.Fatalf("unexpected brief before trust: %+v", d)
	}

	// Trust flow: first attempts show the dialog, then the user accepts.
	var st string
	for i := 0; i < 4 && st != "paired"; i++ {
		st, _ = Pair(d.UDID)
		if i == 0 && st != "waiting" {
			t.Fatalf("first pair attempt should wait for Trust, got %s", st)
		}
	}
	if st != "paired" {
		t.Fatalf("pairing did not complete: %s", st)
	}
	list, _ = ListDevices()
	if !list[0].Paired {
		t.Fatalf("device should be trusted now: %+v", list[0])
	}

	var steps []string
	snap, err := Collect(d.UDID, func(step, total int, label string) { steps = append(steps, label) })
	if err != nil {
		t.Fatalf("collect: %v (errors %v)", err, snap.Errors)
	}
	if len(steps) != 9 {
		t.Errorf("progress steps = %d", len(steps))
	}
	// MobileGestalt is deprecated on modern iOS; everything else must work.
	for k, v := range snap.Errors {
		if k != "diagnostics" {
			t.Errorf("step %s failed: %s", k, v)
		}
	}
	if snap.Lockdown["SerialNumber"] != "G6WQ9X2LJ7" || snap.FS == nil || snap.FS.Model != "iPhone16,1" {
		t.Errorf("identity/fs not read: %v %+v", snap.Lockdown["SerialNumber"], snap.FS)
	}
	if len(snap.Apps) != 8 || !snap.AppsScanned {
		t.Errorf("apps = %d", len(snap.Apps))
	}
	if snap.BatterySource == "" || snap.Battery["CycleCount"] == nil {
		t.Errorf("battery not read: %v", snap.Battery)
	}
	if !snap.CrashesRead || len(snap.CrashReports) != 5 || len(snap.Panics) != 1 || snap.Panics[0].Timestamp == "" {
		t.Errorf("crash reports: %d files, panics %+v", len(snap.CrashReports), snap.Panics)
	}
	if !snap.ProfilesRead || snap.CloudConfig == nil || !snap.AFC2Probed || snap.AFC2Open {
		t.Errorf("profiles/jailbreak probe: %+v", snap)
	}

	res := Evaluate(snap, loadDB(t), "test")
	if res.ModelName != "iPhone 15 Pro" {
		t.Errorf("model = %s", res.ModelName)
	}
	for _, c := range res.Checks {
		if c.Status == Fail {
			t.Errorf("unexpected fail on genuine device: %s — %s (%s)", c.ID, c.Summary, c.Actual)
		}
	}
	for _, id := range []string{"model-consistency", "udid", "imei", "storage", "board", "a-number", "chip", "battery-original", "findmy", "mdm", "profiles", "jb-afc2", "jb-apps", "ios-plausible", "radios", "baseband"} {
		if c := find(res, id); c == nil || c.Status != Pass {
			t.Errorf("%s should pass: %+v", id, c)
		}
	}
	if res.Battery.CycleCount != 431 || res.Battery.HealthPercent < 88 || res.Battery.HealthPercent > 90 {
		t.Errorf("battery report: %+v", res.Battery)
	}
	t.Logf("score %d verdict %s; %v", res.Score, res.Verdict, res.Counts)
}

func TestEndToEndTamperedDevice(t *testing.T) {
	srv, err := fakedevice.Start("127.0.0.1:0", fakedevice.Tampered())
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	t.Setenv("USBMUXD_SOCKET_ADDRESS", "tcp://"+srv.Addr())
	list, err := ListDevices()
	if err != nil || len(list) != 1 || !list[0].Paired {
		t.Fatalf("devices: %+v %v", list, err)
	}
	snap, err := Collect(list[0].UDID, nil)
	if err != nil {
		t.Fatal(err)
	}
	res := Evaluate(snap, loadDB(t), "test")
	for _, id := range []string{"model-consistency", "udid", "imei", "findmy", "mdm", "profiles", "jb-afc2", "jb-apps", "panics"} {
		if c := find(res, id); c == nil || c.Status != Fail {
			t.Errorf("%s should fail: %+v", id, c)
		}
	}
	if res.Verdict != "risk" {
		t.Errorf("verdict %s", res.Verdict)
	}
}
