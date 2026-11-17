package adb

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Tools describes the adb server situation on this computer.
type Tools struct {
	ServerRunning bool   `json:"serverRunning"`
	ADBPath       string `json:"adbPath,omitempty"`
	Error         string `json:"error,omitempty"`
}

func exeName() string {
	if runtime.GOOS == "windows" {
		return "adb.exe"
	}
	return "adb"
}

// ToolsDir is where downloaded platform-tools are kept.
func ToolsDir(appData string) string { return filepath.Join(appData, "platform-tools") }

// FindADB looks for adb next to the app, in the app-data folder, then PATH.
func FindADB(appData string) string {
	var cands []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		cands = append(cands, filepath.Join(dir, exeName()), filepath.Join(dir, "platform-tools", exeName()))
	}
	if appData != "" {
		cands = append(cands, filepath.Join(ToolsDir(appData), exeName()))
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		cands = append(cands, filepath.Join(la, "Android", "Sdk", "platform-tools", exeName()))
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	if p, err := exec.LookPath(exeName()); err == nil {
		return p
	}
	return ""
}

// Ensure makes sure an adb server is running, starting the local adb if needed.
func Ensure(appData string) Tools {
	c := New()
	c.Timeout = 3 * time.Second
	if _, err := c.Version(); err == nil {
		return Tools{ServerRunning: true, ADBPath: FindADB(appData)}
	}
	path := FindADB(appData)
	if path == "" {
		return Tools{Error: "adb not found"}
	}
	cmd := exec.Command(path, "start-server")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Tools{ADBPath: path, Error: strings.TrimSpace(string(out) + " " + err.Error())}
	}
	for i := 0; i < 20; i++ {
		if _, err := c.Version(); err == nil {
			return Tools{ServerRunning: true, ADBPath: path}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return Tools{ADBPath: path, Error: "adb server did not start"}
}

// DownloadURL is Google's official platform-tools package for this OS.
func DownloadURL() string {
	osName := map[string]string{"windows": "windows", "darwin": "darwin"}[runtime.GOOS]
	if osName == "" {
		osName = "linux"
	}
	return "https://dl.google.com/android/repository/platform-tools-latest-" + osName + ".zip"
}

// Download fetches Google's platform-tools and extracts adb (and on Windows
// its two DLLs) into the app-data folder.
func Download(appData string) error {
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Get(DownloadURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	want := map[string]bool{exeName(): true, "AdbWinApi.dll": true, "AdbWinUsbApi.dll": true}
	dir := ToolsDir(appData)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	got := 0
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if !want[base] || !strings.HasPrefix(f.Name, "platform-tools/") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, base), b, 0o755); err != nil {
			return err
		}
		got++
	}
	if got == 0 {
		return fmt.Errorf("adb not found in the downloaded package")
	}
	return nil
}
