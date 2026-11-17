package iphone

import "time"

// AppEntry is an installed application reported by installation_proxy.
type AppEntry struct {
	BundleID string `json:"bundleId"`
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Type     string `json:"type,omitempty"`
	Signer   string `json:"signer,omitempty"`
}

// ProfileEntry is an installed configuration profile.
type ProfileEntry struct {
	Identifier        string `json:"identifier"`
	DisplayName       string `json:"displayName"`
	Description       string `json:"description,omitempty"`
	RemovalDisallowed bool   `json:"removalDisallowed"`
}

// PanicLog summarises a kernel panic report found on the device.
type PanicLog struct {
	File           string   `json:"file"`
	Timestamp      string   `json:"timestamp,omitempty"`
	OSVersion      string   `json:"osVersion,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	MissingSensors []string `json:"missingSensors,omitempty"`
}

// FSInfo is filesystem information reported by the AFC service.
type FSInfo struct {
	Model      string `json:"model"`
	TotalBytes uint64 `json:"totalBytes"`
	FreeBytes  uint64 `json:"freeBytes"`
}

// Snapshot is everything that was read from a device during a scan. It is
// the single input of the check engine, which keeps the checks testable and
// lets the demo mode reuse the same pipeline.
type Snapshot struct {
	CollectedAt    time.Time                 `json:"collectedAt"`
	UDID           string                    `json:"udid"`
	ConnectionType string                    `json:"connectionType"`
	Lockdown       map[string]any            `json:"lockdown"`
	Domains        map[string]map[string]any `json:"domains"`
	Battery        map[string]any            `json:"battery,omitempty"`
	BatterySource  string                    `json:"batterySource,omitempty"`
	Diagnostics    map[string]any            `json:"diagnostics,omitempty"`
	Gestalt        map[string]any            `json:"gestalt,omitempty"`
	FS             *FSInfo                   `json:"fs,omitempty"`
	Apps           []AppEntry                `json:"apps,omitempty"`
	AppsScanned    bool                      `json:"appsScanned"`
	Profiles       []ProfileEntry            `json:"profiles,omitempty"`
	ProfilesRead   bool                      `json:"profilesRead"`
	CloudConfig    map[string]any            `json:"cloudConfig,omitempty"`
	CrashReports   []string                  `json:"crashReports,omitempty"`
	CrashesRead    bool                      `json:"crashesRead"`
	Panics         []PanicLog                `json:"panics,omitempty"`
	AFC2Open       bool                      `json:"afc2Open"`
	AFC2Probed     bool                      `json:"afc2Probed"`
	Errors         map[string]string         `json:"errors,omitempty"`
	Demo           bool                      `json:"demo,omitempty"`
}

// Status of a single check.
type Status string

const (
	Pass Status = "pass"
	Warn Status = "warn"
	Fail Status = "fail"
	Info Status = "info"
	Skip Status = "skip"
)

// Severity tells how much a failed/warned check matters for the verdict.
type Severity string

const (
	Critical Severity = "critical"
	Major    Severity = "major"
	Minor    Severity = "minor"
	Note     Severity = "info"
)

// Check is the result of one validation.
type Check struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Title    string   `json:"title"`
	Status   Status   `json:"status"`
	Severity Severity `json:"severity"`
	Summary  string   `json:"summary"`
	Expected string   `json:"expected,omitempty"`
	Actual   string   `json:"actual,omitempty"`
	Advice   string   `json:"advice,omitempty"`
}

// KV is a labelled value for display.
type KV struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Mono  bool   `json:"mono,omitempty"`
}

// BatteryReport is the computed battery picture.
type BatteryReport struct {
	Available         bool    `json:"available"`
	HealthPercent     float64 `json:"healthPercent,omitempty"`
	CycleCount        int     `json:"cycleCount"`
	RatedCycles       int     `json:"ratedCycles,omitempty"`
	DesignCapacity    int     `json:"designCapacity,omitempty"`
	FullChargeCap     int     `json:"fullChargeCapacity,omitempty"`
	ExpectedDesign    []int   `json:"expectedDesign,omitempty"`
	TemperatureC      float64 `json:"temperatureC,omitempty"`
	VoltageV          float64 `json:"voltageV,omitempty"`
	ChargePercent     int     `json:"chargePercent,omitempty"`
	IsCharging        bool    `json:"isCharging"`
	Serial            string  `json:"serial,omitempty"`
	HealthSource      string  `json:"healthSource,omitempty"`
	ExternalConnected bool    `json:"externalConnected"`
}

// Result is the full outcome of a scan.
type Result struct {
	Snapshot    *Snapshot      `json:"snapshot"`
	Spec        any            `json:"spec"`
	ModelName   string         `json:"modelName"`
	Identity    []KV           `json:"identity"`
	Checks      []Check        `json:"checks"`
	Battery     BatteryReport  `json:"battery"`
	Score       int            `json:"score"`
	Verdict     string         `json:"verdict"`
	VerdictText string         `json:"verdictText"`
	Counts      map[Status]int `json:"counts"`
	Storage     map[string]any `json:"storage"`
	GeneratedAt time.Time      `json:"generatedAt"`
	AppVersion  string         `json:"appVersion"`
}
