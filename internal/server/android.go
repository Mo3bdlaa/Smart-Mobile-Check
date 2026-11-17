package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/mo3bdlaa/smart-mobile-check/internal/adb"
	"github.com/mo3bdlaa/smart-mobile-check/internal/android"
)

type androidState struct {
	mu          sync.Mutex
	id          int
	serial      string
	status      string // idle | running | done | error
	step, total int
	label       string
	err         string
	snap        *android.Snapshot
	mode        string
	manual      map[string]android.ManualResult
	notes       string
	downloading bool
}

func (s *Server) registerAndroid(mux *http.ServeMux) {
	s.and.status = "idle"
	s.and.manual = map[string]android.ManualResult{}
	mux.HandleFunc("/api/android/tools", s.handleAndroidTools)
	mux.HandleFunc("/api/android/tools/download", s.handleAndroidDownload)
	mux.HandleFunc("/api/android/devices", s.handleAndroidDevices)
	mux.HandleFunc("/api/android/scan", s.handleAndroidScan)
	mux.HandleFunc("/api/android/manual", s.handleAndroidManual)
	mux.HandleFunc("/api/android/manual-tests", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, android.ManualTests) })
	mux.HandleFunc("/api/android/demo/profiles", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, android.DemoProfiles) })
	mux.HandleFunc("/api/android/specs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"devices": s.DB.Android()})
	})
}

func (s *Server) handleAndroidTools(w http.ResponseWriter, r *http.Request) {
	t := adb.Ensure(s.AppData)
	s.and.mu.Lock()
	dl := s.and.downloading
	s.and.mu.Unlock()
	writeJSON(w, map[string]any{"serverRunning": t.ServerRunning, "adbPath": t.ADBPath, "error": t.Error, "downloading": dl, "downloadUrl": adb.DownloadURL()})
}

func (s *Server) handleAndroidDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	s.and.mu.Lock()
	if s.and.downloading {
		s.and.mu.Unlock()
		writeJSON(w, map[string]any{"ok": false, "error": "download already running"})
		return
	}
	s.and.downloading = true
	s.and.mu.Unlock()
	err := adb.Download(s.AppData)
	s.and.mu.Lock()
	s.and.downloading = false
	s.and.mu.Unlock()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	t := adb.Ensure(s.AppData)
	writeJSON(w, map[string]any{"ok": t.ServerRunning, "error": t.Error, "adbPath": t.ADBPath})
}

func (s *Server) handleAndroidDevices(w http.ResponseWriter, r *http.Request) {
	list, err := android.ListDevices()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	for i := range list {
		if d := s.DB.AndroidByModel(list[i].Model, list[i].Device); d != nil {
			list[i].ModelName = d.Name
		}
	}
	writeJSON(w, map[string]any{"ok": true, "devices": list})
}

// androidResult evaluates the current snapshot in the requested language.
func (s *Server) androidResult(lang string) (*android.Result, string) {
	s.and.mu.Lock()
	snap, mode, notes := s.and.snap, s.and.mode, s.and.notes
	manual := map[string]android.ManualResult{}
	for k, v := range s.and.manual {
		manual[k] = v
	}
	s.and.mu.Unlock()
	if snap == nil {
		return nil, ""
	}
	return android.Evaluate(snap, s.DB, s.Version, mode, lang, manual), notes
}

func (s *Server) handleAndroidScan(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.and.mu.Lock()
		out := map[string]any{"id": s.and.id, "status": s.and.status, "step": s.and.step, "total": s.and.total, "label": s.and.label, "error": s.and.err}
		done := s.and.status == "done"
		s.and.mu.Unlock()
		if done {
			res, _ := s.androidResult(r.URL.Query().Get("lang"))
			out["result"] = res
		}
		writeJSON(w, out)
		return
	}
	var req struct {
		Serial string `json:"udid"`
		Demo   string `json:"demo"`
		Mode   string `json:"mode"`
	}
	if err := readJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.and.mu.Lock()
	if s.and.status == "running" {
		s.and.mu.Unlock()
		http.Error(w, "a scan is already running", http.StatusConflict)
		return
	}
	s.and.id++
	id := s.and.id
	s.and.serial, s.and.status, s.and.step, s.and.total, s.and.label, s.and.err = req.Serial, "running", 0, android.StepCount, "", ""
	s.and.snap, s.and.mode = nil, req.Mode
	s.and.manual, s.and.notes = map[string]android.ManualResult{}, ""
	s.and.mu.Unlock()
	go s.runAndroidScan(id, req.Serial, req.Demo)
	writeJSON(w, map[string]any{"id": id})
}

func (s *Server) runAndroidScan(id int, serial, demo string) {
	progress := func(step, total int, label string) {
		s.and.mu.Lock()
		if s.and.id == id {
			s.and.step, s.and.total, s.and.label = step, total, label
		}
		s.and.mu.Unlock()
	}
	var snap *android.Snapshot
	var err error
	if demo != "" {
		for i := 1; i <= android.StepCount; i++ {
			progress(i, android.StepCount, "")
			time.Sleep(380 * time.Millisecond)
		}
		snap = android.DemoSnapshot(demo)
	} else {
		snap, err = android.Collect(serial, progress)
	}
	s.and.mu.Lock()
	defer s.and.mu.Unlock()
	if s.and.id != id {
		return
	}
	if err != nil {
		s.and.status, s.and.err = "error", err.Error()
		return
	}
	s.and.snap, s.and.status = snap, "done"
}

func (s *Server) handleAndroidManual(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Results []android.ManualResult `json:"results"`
		Notes   string                 `json:"notes"`
	}
	if err := readJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.and.mu.Lock()
	s.and.manual = map[string]android.ManualResult{}
	for _, m := range req.Results {
		s.and.manual[m.ID] = m
	}
	s.and.notes = req.Notes
	s.and.mu.Unlock()
	res, _ := s.androidResult(r.URL.Query().Get("lang"))
	writeJSON(w, res)
}
