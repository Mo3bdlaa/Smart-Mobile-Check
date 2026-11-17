// Package server is the local HTTP backend of the app. The UI (embedded web
// files) talks to it over 127.0.0.1 only.
package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mo3bdlaa/smart-mobile-check/internal/iphone"
	"github.com/mo3bdlaa/smart-mobile-check/internal/report"
	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

// Server holds application state.
type Server struct {
	Version string
	DB      *specs.DB
	Web     fs.FS
	Opener  func(target string) error
	Quit    func()

	mu     sync.Mutex
	scan   scanState
	manual map[string]iphone.ManualResult
	notes  string
}

type scanState struct {
	ID     int            `json:"id"`
	UDID   string         `json:"udid,omitempty"`
	Status string         `json:"status"` // idle | running | done | error
	Step   int            `json:"step"`
	Total  int            `json:"total"`
	Label  string         `json:"label"`
	Error  string         `json:"error,omitempty"`
	Result *iphone.Result `json:"result,omitempty"`
	base   *iphone.Result
}

// Listen starts the server on a random loopback port and returns its URL.
func (s *Server) Listen() (string, error) {
	s.scan.Status = "idle"
	s.manual = map[string]iphone.ManualResult{}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/env", s.handleEnv)
	mux.HandleFunc("/api/ios/devices", s.handleDevices)
	mux.HandleFunc("/api/ios/pair", s.handlePair)
	mux.HandleFunc("/api/ios/scan", s.handleScan)
	mux.HandleFunc("/api/ios/manual", s.handleManual)
	mux.HandleFunc("/api/ios/manual-tests", s.handleManualTests)
	mux.HandleFunc("/api/demo/profiles", s.handleDemoProfiles)
	mux.HandleFunc("/api/specs", s.handleSpecs)
	mux.HandleFunc("/api/report/save", s.handleSave)
	mux.HandleFunc("/api/open", s.handleOpen)
	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || s.Quit == nil {
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
		go func() { time.Sleep(200 * time.Millisecond); s.Quit() }()
	})
	mux.Handle("/", noCache(http.FileServer(http.FS(s.Web))))
	srv := &http.Server{Handler: localOnly(mux), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("server stopped: %v", err)
		}
	}()
	return "http://" + ln.Addr().String() + "/", nil
}

func localOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host != "127.0.0.1" && host != "::1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		// Reject cross-site requests to the API (DNS-rebinding / CSRF guard).
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if h := r.Host; !strings.HasPrefix(h, "127.0.0.1:") && !strings.HasPrefix(h, "localhost:") {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if o := r.Header.Get("Origin"); o != "" && !strings.Contains(o, r.Host) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
}

func (s *Server) handleEnv(w http.ResponseWriter, r *http.Request) {
	err := iphone.ServiceAvailable()
	env := map[string]any{
		"version":       s.Version,
		"os":            runtime.GOOS,
		"serviceOK":     err == nil,
		"specCount":     len(s.DB.All()),
		"specWarnings":  s.DB.Warnings,
		"externalSpecs": s.DB.External,
		"reportsDir":    reportsDir(),
	}
	if err != nil {
		env["serviceError"] = err.Error()
	}
	writeJSON(w, env)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	list, err := iphone.ListDevices()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	for i := range list {
		if d := s.DB.ByProductType(list[i].ProductType); d != nil {
			list[i].ModelName = d.Name
		}
	}
	writeJSON(w, map[string]any{"ok": true, "devices": list})
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UDID string `json:"udid"`
	}
	if err := readJSON(r, &req); err != nil || req.UDID == "" {
		http.Error(w, "udid required", http.StatusBadRequest)
		return
	}
	state, msg := iphone.Pair(req.UDID)
	writeJSON(w, map[string]any{"state": state, "message": msg})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.mu.Lock()
		st := s.scan
		s.mu.Unlock()
		st.Result = iphone.Localize(st.Result, r.URL.Query().Get("lang"))
		writeJSON(w, st)
		return
	}
	var req struct {
		UDID string `json:"udid"`
		Demo string `json:"demo"`
		Mode string `json:"mode"`
	}
	if err := readJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.scan.Status == "running" {
		s.mu.Unlock()
		http.Error(w, "a scan is already running", http.StatusConflict)
		return
	}
	id := s.scan.ID + 1
	s.scan = scanState{ID: id, UDID: req.UDID, Status: "running", Label: "Connecting", Total: 9}
	s.manual = map[string]iphone.ManualResult{}
	s.notes = ""
	s.mu.Unlock()

	go s.runScan(id, req.UDID, req.Demo, req.Mode)
	writeJSON(w, map[string]any{"id": id})
}

func (s *Server) runScan(id int, udid, demo, mode string) {
	progress := func(step, total int, label string) {
		s.mu.Lock()
		if s.scan.ID == id {
			s.scan.Step, s.scan.Total, s.scan.Label = step, total, label
		}
		s.mu.Unlock()
	}
	var snap *iphone.Snapshot
	var err error
	if demo != "" {
		labels := []string{"Reading device identity", "Reading battery gauge", "Running hardware diagnostics", "Measuring storage",
			"Auditing installed apps", "Checking management & profiles", "Probing for jailbreak services", "Analysing crash & panic logs", "Evaluating results"}
		for i, l := range labels {
			progress(i+1, len(labels), l)
			time.Sleep(380 * time.Millisecond)
		}
		snap = iphone.DemoSnapshot(demo)
	} else {
		snap, err = iphone.Collect(udid, progress)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan.ID != id {
		return
	}
	if err != nil {
		s.scan.Status = "error"
		s.scan.Error = err.Error()
		return
	}
	res := iphone.EvaluateMode(snap, s.DB, s.Version, mode)
	s.scan.base = res
	s.scan.Result = iphone.ApplyManual(res, s.manual)
	s.scan.Status = "done"
}

func (s *Server) handleManualTests(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, iphone.ManualTests)
}

func (s *Server) handleManual(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Results []iphone.ManualResult `json:"results"`
		Notes   string                `json:"notes"`
	}
	if err := readJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manual = map[string]iphone.ManualResult{}
	for _, m := range req.Results {
		s.manual[m.ID] = m
	}
	s.notes = req.Notes
	if s.scan.base != nil {
		s.scan.Result = iphone.ApplyManual(s.scan.base, s.manual)
	}
	writeJSON(w, iphone.Localize(s.scan.Result, r.URL.Query().Get("lang")))
}

func (s *Server) handleDemoProfiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, iphone.DemoProfiles)
}

func (s *Server) handleSpecs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"devices": s.DB.All(), "warnings": s.DB.Warnings, "external": s.DB.External})
}

func reportsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "Smart Mobile Check Reports")
	}
	docs := filepath.Join(home, "Documents")
	if st, err := os.Stat(docs); err != nil || !st.IsDir() {
		docs = home
	}
	return filepath.Join(docs, "Smart Mobile Check", "Reports")
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Format string `json:"format"`
		Open   bool   `json:"open"`
		Lang   string `json:"lang"`
	}
	_ = readJSON(r, &req)
	s.mu.Lock()
	res := iphone.Localize(s.scan.Result, req.Lang)
	notes := s.notes
	s.mu.Unlock()
	if res == nil {
		http.Error(w, "no scan result yet", http.StatusBadRequest)
		return
	}
	dir := reportsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	var data []byte
	var err error
	ext := "html"
	if req.Format == "json" {
		ext = "json"
		data, err = json.MarshalIndent(map[string]any{"result": res, "notes": notes}, "", "  ")
	} else {
		data, err = report.HTML(res, notes)
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	path := filepath.Join(dir, report.FileName(res, ext))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if req.Open && s.Opener != nil {
		_ = s.Opener(path)
	}
	writeJSON(w, map[string]any{"ok": true, "path": path})
}

// handleOpen opens an allow-listed external link or the reports folder.
func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
	}
	if err := readJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	allowed := map[string]string{
		"apple-devices":     "ms-windows-store://pdp/?ProductId=9NP83LWLPZ9K",
		"apple-devices-web": "https://apps.microsoft.com/detail/9np83lwlpz9k",
		"itunes":            "https://www.apple.com/itunes/",
		"coverage":          "https://checkcoverage.apple.com/",
	}
	target, ok := allowed[req.Target]
	if req.Target == "reports" {
		dir := reportsDir()
		_ = os.MkdirAll(dir, 0o755)
		target, ok = dir, true
	}
	if !ok || s.Opener == nil {
		http.Error(w, "unknown target", http.StatusBadRequest)
		return
	}
	if err := s.Opener(target); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprint(err)})
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}
