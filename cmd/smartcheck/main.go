// Command smartcheck is the Smart Mobile Check desktop app: a portable
// inspector for used phones. It serves its UI from an embedded web bundle on
// a loopback port and shows it in a native window (WebView2 on Windows).
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	smartcheck "github.com/mo3bdlaa/smart-mobile-check"
	"github.com/mo3bdlaa/smart-mobile-check/internal/server"
	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

// Version is set at build time with -ldflags "-X main.Version=...".
var Version = "0.1.0-dev"

func main() {
	browser := flag.Bool("browser", false, "open the UI in the default browser instead of the app window")
	flag.Parse()

	logDir := appDataDir()
	_ = os.MkdirAll(logDir, 0o755)
	if f, err := os.OpenFile(filepath.Join(logDir, "smartcheck.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		defer f.Close()
	}
	log.Printf("Smart Mobile Check %s starting", Version)

	exe, _ := os.Executable()
	external := filepath.Join(filepath.Dir(exe), "specs")
	db := specs.Load(smartcheck.Specs, external)
	log.Printf("loaded %d device specs (%d warnings)", len(db.All()), len(db.Warnings))
	for _, w := range db.Warnings {
		log.Printf("spec warning: %s", w)
	}

	web, err := fs.Sub(smartcheck.Web, "web")
	if err != nil {
		fatal(err)
	}
	srv := &server.Server{Version: Version, DB: db, Web: web, Opener: openExternal, Quit: func() { os.Exit(0) }}
	url, err := srv.Listen()
	if err != nil {
		fatal(err)
	}
	log.Printf("UI available at %s", url)

	if *browser || !runWindow(url, logDir) {
		fmt.Println("Smart Mobile Check is running at", url)
		fmt.Println("Close this window or use the Quit button in the app to exit.")
		_ = openExternal(url)
		select {}
	}
}

func appDataDir() string {
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		return filepath.Join(d, "SmartMobileCheck")
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "SmartMobileCheck")
	}
	return filepath.Join(os.TempDir(), "SmartMobileCheck")
}

func fatal(err error) {
	log.Printf("fatal: %v", err)
	showError("Smart Mobile Check could not start:\n\n" + err.Error())
	os.Exit(1)
}
