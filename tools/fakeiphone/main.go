// Command fakeiphone runs a simulated Apple Mobile Device Service with one
// connected iPhone, for testing Smart Mobile Check without hardware.
//
//	go run ./tools/fakeiphone -profile genuine            (listens on 127.0.0.1:27015)
//	USBMUXD_SOCKET_ADDRESS=tcp://127.0.0.1:27015 go run ./cmd/smartcheck --browser
//
// On Windows, stop "Apple Mobile Device Service" first: the simulator uses the
// same port, so the app connects to it without any configuration.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/mo3bdlaa/smart-mobile-check/internal/fakedevice"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:27015", "listen address")
	profile := flag.String("profile", "genuine", "genuine | tampered")
	verbose := flag.Bool("v", false, "log protocol traffic")
	flag.Parse()
	dev := fakedevice.IPhone15Pro()
	if *profile == "tampered" {
		dev = fakedevice.Tampered()
	}
	srv, err := fakedevice.Start(*addr, dev)
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		srv.Log = log.New(os.Stderr, "fake ", log.Ltime)
	}
	log.Printf("simulated iPhone (%s, UDID %s) on %s", *profile, dev.UDID, srv.Addr())
	select {}
}
