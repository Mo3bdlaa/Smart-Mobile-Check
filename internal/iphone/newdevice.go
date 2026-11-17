package iphone

import (
	"fmt"
	"strings"
)

// newDeviceChecks verifies that a phone sold as brand new really is new:
// a retail unit that has not been charged, set up or repaired before.
func (e *evaluator) newDeviceChecks(b BatteryReport) {
	// Retail unit (part number prefix M).
	if mn := e.ld("ModelNumber"); mn != "" {
		full := mn + e.ld("RegionInfo")
		prefix := strings.ToUpper(mn[:1])
		if prefix == "M" || prefix == "P" {
			e.add(Check{ID: "new-retail", Category: CatNew, Title: "New retail unit", Status: Pass, Actual: full,
				Summary: "Part number starts with \"" + prefix + "\" — sold new by Apple or an authorised reseller."})
		} else {
			e.add(Check{ID: "new-retail", Category: CatNew, Title: "New retail unit", Status: Fail, Severity: Critical, Actual: full,
				Summary: "Not a new retail unit: " + strings.ToLower(e.db.Reference.PartPrefixes[prefix]) + ".",
				Advice:  "A phone sold as \"new\" must have a part number starting with M. Ask for a refund or a lower refurbished price."})
		}
	}

	// Battery: ~0 cycles and full health.
	if b.Available {
		c := Check{ID: "new-cycles", Category: CatNew, Title: "Battery never used", Actual: fmt.Sprintf("%d cycles", b.CycleCount), Expected: "0 – 5 cycles"}
		switch {
		case b.CycleCount <= 5:
			c.Status, c.Summary = Pass, "Battery has (almost) never been cycled — consistent with a new phone."
		case b.CycleCount <= 20:
			c.Status, c.Severity, c.Summary = Warn, Major, "More charge cycles than a new phone should have — it may have been used as a display or for testing."
		default:
			c.Status, c.Severity, c.Summary = Fail, Critical, "This battery has been through many charge cycles — the phone has been used before."
		}
		e.add(c)
		if b.HealthPercent > 0 {
			h := Check{ID: "new-health", Category: CatNew, Title: "Battery at full capacity", Actual: fmt.Sprintf("%.0f%%", b.HealthPercent), Expected: "100%"}
			if b.HealthPercent >= 98 {
				h.Status, h.Summary = Pass, "Battery is at full factory capacity."
			} else {
				h.Status, h.Severity, h.Summary = Fail, Major, "A new battery should be at 100% — this one is already worn."
			}
			e.add(h)
		}
	} else {
		e.add(Check{ID: "new-cycles", Category: CatNew, Title: "Battery never used", Status: Skip,
			Summary: "Battery gauge not readable — check Settings › Battery › Battery Health (100%, cycle count 0)."})
	}

	// No third-party apps / user data.
	if e.s.AppsScanned {
		var third []string
		for _, a := range e.s.Apps {
			if strings.EqualFold(a.Type, "User") && !strings.HasPrefix(strings.ToLower(a.BundleID), "com.apple.") {
				third = append(third, firstNonEmpty(a.Name, a.BundleID))
			}
		}
		if len(third) == 0 {
			e.add(Check{ID: "new-apps", Category: CatNew, Title: "No previous owner's apps", Status: Pass,
				Summary: "Only Apple's pre-installed apps are present."})
		} else {
			e.add(Check{ID: "new-apps", Category: CatNew, Title: "No previous owner's apps", Status: Warn, Severity: Major,
				Summary: fmt.Sprintf("%d third-party app(s) installed — the phone has been set up and used.", len(third)),
				Actual:  strings.Join(third, ", "),
				Advice:  "Ignore this only if you installed these apps yourself after setting the phone up."})
		}
	}

	// Stored user data.
	data, _ := num(e.dom("com.apple.disk_usage", "TotalDataCapacity"))
	avail, _ := num(e.dom("com.apple.disk_usage", "TotalDataAvailable"))
	if data > 0 && avail > 0 {
		used := data - avail
		c := Check{ID: "new-data", Category: CatNew, Title: "No stored user data", Actual: humanBytes(used) + " used"}
		if used < 25e9 {
			c.Status, c.Summary = Pass, "Storage usage is consistent with a fresh phone."
		} else {
			c.Status, c.Severity, c.Summary = Warn, Major, "A lot of storage is already used — photos, apps or data from someone else."
		}
		e.add(c)
	}

	// Crash history.
	if e.s.CrashesRead {
		if len(e.s.Panics) == 0 {
			e.add(Check{ID: "new-panics", Category: CatNew, Title: "Clean crash history", Status: Pass,
				Summary: "No kernel panics recorded."})
		} else {
			e.add(Check{ID: "new-panics", Category: CatNew, Title: "Clean crash history", Status: Warn, Severity: Major,
				Summary: fmt.Sprintf("%d kernel panic report(s) on a phone sold as new.", len(e.s.Panics))})
		}
	}

	// Activation (informational).
	switch st := e.ld("ActivationState"); st {
	case "Unactivated":
		e.add(Check{ID: "new-activation", Category: CatNew, Title: "Activation", Status: Pass, Actual: st,
			Summary: "Never activated — as expected for a sealed phone. Activate it now in front of the seller."})
	case "":
	default:
		e.add(Check{ID: "new-activation", Category: CatNew, Title: "Activation", Status: Info, Actual: st,
			Summary: "Already activated. Fine if you just set it up yourself; otherwise confirm on checkcoverage.apple.com that the warranty starts today."})
	}
}
