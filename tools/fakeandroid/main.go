// Command fakeandroid runs a simulated adb server with one Android phone, for
// testing Smart Mobile Check without hardware.
//
//	go run ./tools/fakeandroid -profile a-healthy     (listens on 127.0.0.1:5037)
//
// The app connects to an adb server already listening on 5037, so no real adb
// is needed while the simulator runs (stop any real "adb" server first).
package main

import (
	"flag"
	"log"
	"os"

	"github.com/mo3bdlaa/smart-mobile-check/internal/android"
	"github.com/mo3bdlaa/smart-mobile-check/internal/fakedevice"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:5037", "listen address")
	profile := flag.String("profile", "a-healthy", "a-healthy | a-clone | a-new")
	unauth := flag.Int("unauthorized-polls", 3, "device-list polls that report the phone as unauthorized (simulates the Allow USB debugging prompt)")
	verbose := flag.Bool("v", false, "log protocol traffic")
	flag.Parse()
	snap := android.DemoSnapshot(*profile)
	snap.Demo = false
	srv, err := fakedevice.StartADB(*addr, snap, *unauth)
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		srv.Log = log.New(os.Stderr, "fake ", log.Ltime)
	}
	log.Printf("simulated Android phone (%s, serial %s) on %s", *profile, snap.Serial, srv.Addr())
	select {}
}
