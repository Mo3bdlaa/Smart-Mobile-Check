//go:build windows

package main

import (
	"log"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/jchv/go-webview2"
)

// runWindow shows the UI in a native WebView2 window. It returns false when
// the WebView2 runtime is not available so the caller can fall back to the
// default browser.
func runWindow(url, dataDir string) bool {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  filepath.Join(dataDir, "WebView2"),
		WindowOptions: webview2.WindowOptions{
			Title:  "Smart Mobile Check",
			Width:  1280,
			Height: 840,
			IconId: 1,
			Center: true,
		},
	})
	if w == nil {
		log.Printf("WebView2 runtime not available, falling back to browser")
		return false
	}
	defer w.Destroy()
	w.SetSize(1040, 700, webview2.HintMin)
	w.Navigate(url)
	w.Run()
	return true
}

func openExternal(target string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func showError(msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	box := user32.NewProc("MessageBoxW")
	text, _ := syscall.UTF16PtrFromString(msg)
	title, _ := syscall.UTF16PtrFromString("Smart Mobile Check")
	_, _, _ = box.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10)
}
