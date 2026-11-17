// Package inspect holds the platform-neutral pieces of an inspection: check
// results, scoring and the verdict. The iPhone and Android engines both build
// on it.
package inspect

import "math"

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

// KV is a labelled value for display. Key is a stable English identifier.
type KV struct {
	Key   string `json:"key,omitempty"`
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
	CyclesKnown       bool    `json:"cyclesKnown,omitempty"`
}

// Categories shared by every platform, in display order.
const (
	CatNew      = "New-device checks"
	CatIdentity = "Identity & authenticity"
	CatOwner    = "Ownership & locks"
	CatSoftware = "Software integrity"
	CatBattery  = "Battery"
	CatHardware = "Hardware & diagnostics"
	CatHands    = "Hands-on tests"
)

// Categories lists the categories in display order.
var Categories = []string{CatNew, CatIdentity, CatOwner, CatSoftware, CatBattery, CatHardware, CatHands}

// Purchase modes.
const (
	ModeUsed = "used"
	ModeNew  = "new"
)

// Outcome is the scored verdict for a set of checks.
type Outcome struct {
	Counts  map[Status]int
	Score   int
	Verdict string // good | caution | risk
}

// Evaluate scores a set of checks.
func Evaluate(checks []Check) Outcome {
	o := Outcome{Counts: map[Status]int{}}
	score := 100.0
	critical, majorFail := false, false
	for _, c := range checks {
		o.Counts[c.Status]++
		switch c.Status {
		case Fail:
			switch c.Severity {
			case Critical:
				score -= 45
				critical = true
			case Major:
				score -= 20
				majorFail = true
			case Minor:
				score -= 8
			}
		case Warn:
			switch c.Severity {
			case Critical:
				score -= 15
			case Major:
				score -= 8
			case Minor:
				score -= 3
			}
		}
	}
	if score < 0 {
		score = 0
	}
	o.Score = int(math.Round(score))
	switch {
	case critical:
		o.Verdict = "risk-critical"
	case o.Score >= 85 && !majorFail:
		o.Verdict = "good"
	case o.Score >= 60:
		o.Verdict = "caution"
	default:
		o.Verdict = "risk"
	}
	return o
}

// VerdictText returns the verdict label and the human sentence for it.
func VerdictText(v, lang string) (string, string) {
	ar := lang == "ar"
	switch v {
	case "risk-critical":
		if ar {
			return "risk", "لا تشتري — توجد مشكلة خطيرة"
		}
		return "risk", "Do not buy — a critical problem was found"
	case "good":
		if ar {
			return "good", "الجهاز أصلي وسليم"
		}
		return "good", "Looks genuine and healthy"
	case "caution":
		if ar {
			return "caution", "اشترِ بحذر — راجع التحذيرات"
		}
		return "caution", "Buy with caution — review the warnings"
	default:
		if ar {
			return "risk", "خطر — توجد مشاكل كثيرة"
		}
		return "risk", "High risk — several problems were found"
	}
}
