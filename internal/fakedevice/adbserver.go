package fakedevice

import (
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mo3bdlaa/smart-mobile-check/internal/android"
)

// ADBServer emulates an adb server (port 5037) with one Android phone whose
// shell answers are rendered from an android.Snapshot.
type ADBServer struct {
	Snap *android.Snapshot
	Log  *log.Logger

	ln         net.Listener
	mu         sync.Mutex
	unauthLeft int // device-list polls that still report "unauthorized"
}

// StartADB starts the emulator. unauthorizedPolls is how many device-list
// requests report the phone as "unauthorized" before the user "allows" it.
func StartADB(addr string, snap *android.Snapshot, unauthorizedPolls int) (*ADBServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &ADBServer{Snap: snap, ln: ln, unauthLeft: unauthorizedPolls, Log: log.New(io.Discard, "", 0)}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(c)
		}
	}()
	return s, nil
}

// Addr returns the listening address.
func (s *ADBServer) Addr() string { return s.ln.Addr().String() }

// Close stops the emulator.
func (s *ADBServer) Close() error { return s.ln.Close() }

func readReq(c net.Conn) (string, error) {
	h := make([]byte, 4)
	if _, err := io.ReadFull(c, h); err != nil {
		return "", err
	}
	n, err := strconv.ParseUint(string(h), 16, 32)
	if err != nil {
		return "", err
	}
	b := make([]byte, n)
	_, err = io.ReadFull(c, b)
	return string(b), err
}

func okay(c net.Conn, payload string) {
	fmt.Fprintf(c, "OKAY%04x%s", len(payload), payload)
}

func failMsg(c net.Conn, msg string) {
	fmt.Fprintf(c, "FAIL%04x%s", len(msg), msg)
}

func (s *ADBServer) authorized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unauthLeft <= 0
}

func (s *ADBServer) handle(c net.Conn) {
	defer c.Close()
	req, err := readReq(c)
	if err != nil {
		return
	}
	s.Log.Printf("adb %s", req)
	switch {
	case req == "host:version":
		okay(c, "0029")
	case req == "host:devices-l" || req == "host:devices":
		s.mu.Lock()
		state := "device"
		if s.unauthLeft > 0 {
			state = "unauthorized"
			s.unauthLeft--
		}
		s.mu.Unlock()
		p := s.Snap.Props
		line := fmt.Sprintf("%s\t%s usb:1-1 product:%s model:%s device:%s transport_id:1\n", s.Snap.Serial, state,
			p["ro.product.name"], strings.ReplaceAll(p["ro.product.model"], " ", "_"), p["ro.product.device"])
		if state == "unauthorized" {
			line = s.Snap.Serial + "\tunauthorized usb:1-1 transport_id:1\n"
		}
		okay(c, line)
	case strings.HasPrefix(req, "host:transport:"):
		if strings.TrimPrefix(req, "host:transport:") != s.Snap.Serial {
			failMsg(c, "device '"+strings.TrimPrefix(req, "host:transport:")+"' not found")
			return
		}
		if !s.authorized() {
			failMsg(c, "device unauthorized.")
			return
		}
		c.Write([]byte("OKAY"))
		cmd, err := readReq(c)
		if err != nil || !strings.HasPrefix(cmd, "shell:") {
			failMsg(c, "unsupported")
			return
		}
		c.Write([]byte("OKAY"))
		c.Write([]byte(s.shell(strings.TrimPrefix(cmd, "shell:"))))
	default:
		failMsg(c, "unknown host service")
	}
}

// shell renders the output of a command from the snapshot.
func (s *ADBServer) shell(cmd string) string {
	sn := s.Snap
	var b strings.Builder
	switch {
	case cmd == "getprop":
		keys := make([]string, 0, len(sn.Props))
		for k := range sn.Props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "[%s]: [%s]\n", k, sn.Props[k])
		}
	case strings.HasPrefix(cmd, "service call iphonesubinfo"):
		idx := 0
		if strings.Contains(cmd, "i32 2") {
			idx = 1
		}
		if idx < len(sn.IMEIs) {
			b.WriteString(parcel(sn.IMEIs[idx]))
		} else {
			b.WriteString("Result: Parcel(ffffffec '....')\n")
		}
	case strings.HasPrefix(cmd, "settings get "):
		k := strings.Fields(cmd)[3]
		v, ok := sn.Settings[k]
		if !ok {
			v = "null"
		}
		b.WriteString(v + "\n")
	case cmd == "cat /proc/uptime":
		b.WriteString("183422.51 1290233.10\n")
	case cmd == "dumpsys battery":
		b.WriteString("Current Battery Service state:\n")
		keys := make([]string, 0, len(sn.Battery))
		for k := range sn.Battery {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s: %s\n", k, sn.Battery[k])
		}
	case strings.HasPrefix(cmd, "for f in cycle_count"):
		for _, f := range []string{"cycle_count", "charge_full", "charge_full_design", "battery_cycle", "fg_asoc", "health", "capacity", "charge_counter", "batt_capacity_max"} {
			fmt.Fprintf(&b, "%s=%s\n", f, sn.BatterySys[f])
		}
	case cmd == "cat /proc/meminfo":
		fmt.Fprintf(&b, "MemTotal:       %d kB\nMemFree:          812344 kB\nMemAvailable:    5233112 kB\n", sn.MemTotalKB)
	case cmd == "df -k /data":
		fmt.Fprintf(&b, "Filesystem      1K-blocks     Used Available Use%% Mounted on\n/dev/block/dm-50 %d %d %d %d%% /data\n",
			sn.DataTotalKB, sn.DataUsedKB, sn.DataTotalKB-sn.DataUsedKB, sn.DataUsedKB*100/max64(sn.DataTotalKB, 1))
	case cmd == "wm size":
		b.WriteString("Physical size: " + sn.ScreenSize + "\n")
	case cmd == "wm density":
		b.WriteString("Physical density: " + sn.Density + "\n")
	case cmd == "dumpsys account":
		fmt.Fprintf(&b, "User UserInfo{0:Owner:c13}:\n  Accounts: %d\n", len(sn.Accounts))
		for _, a := range sn.Accounts {
			fmt.Fprintf(&b, "    Account {name=%s, type=%s}\n", a.Name, a.Type)
		}
	case cmd == "dumpsys device_policy":
		b.WriteString("Current Device Policy Manager state:\n")
		if sn.DeviceOwner != "" {
			fmt.Fprintf(&b, "  Device Owner: \n    admin=ComponentInfo{%s/%s.AdminReceiver}\n    name=Device management\n", sn.DeviceOwner, sn.DeviceOwner)
		}
		for _, p := range sn.ProfileOwners {
			fmt.Fprintf(&b, "  Profile Owner (User 10): \n    admin=ComponentInfo{%s/.Admin}\n", p)
		}
		b.WriteString("\n  Enabled Device Admins (User 0, provisioningState: 3):\n")
	case cmd == "pm list packages -3 -i":
		for _, a := range sn.Apps {
			if a.Type == "User" {
				inst := a.Installer
				if inst == "" {
					inst = "null"
				}
				fmt.Fprintf(&b, "package:%s  installer=%s\n", a.BundleID, inst)
			}
		}
	case cmd == "pm list packages -s":
		for _, a := range sn.Apps {
			if a.Type == "System" {
				fmt.Fprintf(&b, "package:%s\n", a.BundleID)
			}
		}
	case strings.HasPrefix(cmd, "for p in /system/bin/su"):
		for _, p := range sn.SuPaths {
			b.WriteString(p + "\n")
		}
	case cmd == "getenforce":
		b.WriteString(sn.SELinux + "\n")
	case cmd == "dumpsys sensorservice":
		b.WriteString("Sensor List:\n")
		for i, t := range sn.Sensors {
			fmt.Fprintf(&b, "0x%08x) Sensor %d | Vendor | ver: 1 | type: android.sensor.%s(%d) | perm: n/a\n", i+1, i, t, i+1)
		}
	case cmd == "dumpsys media.camera":
		fmt.Fprintf(&b, "== Service global info: ==\n\nNumber of camera devices: %d\nNumber of normal camera devices: %d\n", sn.CameraCount, sn.CameraCount)
	case cmd == "dumpsys dropbox":
		b.WriteString("Drop box contents: 12 entries\nMax entries: 1000\n\n")
		tags := make([]string, 0, len(sn.Dropbox))
		for k := range sn.Dropbox {
			tags = append(tags, k)
		}
		sort.Strings(tags)
		for _, t := range tags {
			for i := 0; i < sn.Dropbox[t]; i++ {
				fmt.Fprintf(&b, "2026-11-%02d 10:%02d:00 %s (text, 1234 bytes)\n", 1+i, i, t)
			}
		}
	default:
		b.WriteString("/system/bin/sh: " + cmd + ": inaccessible or not found\n")
	}
	return b.String()
}

// parcel renders a string the way "service call" prints a Parcel.
func parcel(s string) string {
	var b strings.Builder
	b.WriteString("Result: Parcel(\n")
	// Simple form: put the digits in quoted chunks the parser concatenates.
	for i := 0; i < len(s); i += 4 {
		end := i + 4
		if end > len(s) {
			end = len(s)
		}
		chunk := strings.Join(strings.Split(s[i:end], ""), ".")
		fmt.Fprintf(&b, "  0x%08x: 00000000 00000000 00000000 00000000 '%s.'\n", 16+i*4, chunk)
	}
	b.WriteString("  0x00000040: 00000000                            '....        ')\n")
	return b.String()
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
