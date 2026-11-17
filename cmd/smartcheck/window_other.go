//go:build !windows

package main

import (
	"log"
	"os/exec"
	"runtime"
)

// runWindow has no native window outside Windows; the UI opens in the browser.
func runWindow(url, dataDir string) bool { return false }

func openExternal(target string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	return exec.Command(name, target).Start()
}

func showError(msg string) { log.Println(msg) }
