// Package report renders a scan result into a standalone, printable HTML file.
package report

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/mo3bdlaa/smart-mobile-check/internal/iphone"
)

type group struct {
	Name   string
	Checks []iphone.Check
}

type view struct {
	R        *iphone.Result
	Groups   []group
	Date     string
	Ring     float64
	Color    string
	Notes    string
	Hands    bool
	Unreadab []string
}

// HTML renders the result.
func HTML(r *iphone.Result, buyerNotes string) ([]byte, error) {
	v := view{R: r, Date: r.GeneratedAt.Format("2 Jan 2006, 15:04"), Notes: buyerNotes}
	if r.GeneratedAt.IsZero() {
		v.Date = time.Now().Format("2 Jan 2006, 15:04")
	}
	for _, cat := range iphone.Categories {
		g := group{Name: cat}
		for _, c := range r.Checks {
			if c.Category == cat {
				g.Checks = append(g.Checks, c)
			}
		}
		if len(g.Checks) > 0 {
			v.Groups = append(v.Groups, g)
			if cat == iphone.CatHands {
				v.Hands = true
			}
		}
	}
	v.Ring = 326.7 * float64(r.Score) / 100
	switch r.Verdict {
	case "good":
		v.Color = "#16a34a"
	case "caution":
		v.Color = "#d97706"
	default:
		v.Color = "#dc2626"
	}
	if r.Snapshot != nil {
		for k, e := range r.Snapshot.Errors {
			v.Unreadab = append(v.Unreadab, k+": "+e)
		}
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FileName builds a safe report file name.
func FileName(r *iphone.Result, ext string) string {
	model := strings.NewReplacer(" ", "-", "/", "-").Replace(r.ModelName)
	serial := ""
	if r.Snapshot != nil {
		if s, ok := r.Snapshot.Lockdown["SerialNumber"].(string); ok {
			serial = "_" + s
		}
	}
	return fmt.Sprintf("%s%s_%s.%s", model, serial, time.Now().Format("2006-01-02_1504"), ext)
}

var funcs = template.FuncMap{
	"icon": func(s iphone.Status) string {
		switch s {
		case iphone.Pass:
			return "✔"
		case iphone.Warn:
			return "!"
		case iphone.Fail:
			return "✖"
		case iphone.Skip:
			return "–"
		}
		return "i"
	},
	"pct":   func(f float64) string { return fmt.Sprintf("%.0f%%", f) },
	"upper": strings.ToUpper,
	"cnt":   func(m map[iphone.Status]int, k string) int { return m[iphone.Status(k)] },
}

var tpl = template.Must(template.New("r").Funcs(funcs).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Inspection report — {{.R.ModelName}}</title>
<style>
:root{--bg:#f6f7fb;--card:#fff;--ink:#0f172a;--mute:#64748b;--line:#e2e8f0;--pass:#16a34a;--warn:#d97706;--fail:#dc2626;--info:#2563eb;--skip:#94a3b8}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:14px/1.5 "Segoe UI",system-ui,-apple-system,sans-serif}
.wrap{max-width:960px;margin:0 auto;padding:32px 20px}
header{display:flex;gap:24px;align-items:center;background:var(--card);border:1px solid var(--line);border-radius:18px;padding:24px}
h1{margin:0;font-size:26px}h2{font-size:16px;margin:28px 0 10px;text-transform:uppercase;letter-spacing:.06em;color:var(--mute)}
.sub{color:var(--mute)}.verdict{display:inline-block;margin-top:8px;padding:6px 12px;border-radius:999px;font-weight:600;color:#fff;background:{{.Color}}}
.ring{flex:none}.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:8px 24px;background:var(--card);border:1px solid var(--line);border-radius:14px;padding:16px 20px}
.kv{display:flex;justify-content:space-between;gap:12px;border-bottom:1px dashed var(--line);padding:4px 0}.kv span:first-child{color:var(--mute)}.mono{font-family:Consolas,ui-monospace,monospace}
table{width:100%;border-collapse:collapse;background:var(--card);border:1px solid var(--line);border-radius:14px;overflow:hidden}
td{padding:10px 12px;border-top:1px solid var(--line);vertical-align:top}tr:first-child td{border-top:none}
.st{width:34px}.b{display:inline-flex;width:24px;height:24px;border-radius:50%;align-items:center;justify-content:center;color:#fff;font-weight:700;font-size:12px}
.pass{background:var(--pass)}.warn{background:var(--warn)}.fail{background:var(--fail)}.info{background:var(--info)}.skip{background:var(--skip)}
.t{font-weight:600}.d{color:var(--mute);font-size:13px}.ea{font-size:12px;color:var(--mute);margin-top:4px}.adv{font-size:12px;margin-top:4px;color:#334155}
.bat{display:grid;grid-template-columns:repeat(4,1fr);gap:12px}.tile{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:14px}.tile b{font-size:22px;display:block}
footer{margin-top:32px;color:var(--mute);font-size:12px;text-align:center}
@media print{body{background:#fff}.wrap{padding:0}header,table,.grid,.tile{break-inside:avoid}}
</style></head><body><div class="wrap">
<header>
 <svg class="ring" width="130" height="130" viewBox="0 0 130 130"><circle cx="65" cy="65" r="52" fill="none" stroke="#e2e8f0" stroke-width="12"/>
 <circle cx="65" cy="65" r="52" fill="none" stroke="{{.Color}}" stroke-width="12" stroke-linecap="round" stroke-dasharray="{{.Ring}} 400" transform="rotate(-90 65 65)"/>
 <text x="65" y="72" text-anchor="middle" font-size="30" font-weight="700" fill="#0f172a">{{.R.Score}}</text></svg>
 <div><div class="sub">Smart Mobile Check · Inspection report</div><h1>{{.R.ModelName}}</h1>
 <div class="sub">{{if eq .R.Mode "new"}}New-iPhone purchase check{{else}}Used-iPhone purchase check{{end}} · {{.Date}}{{if .R.Snapshot}}{{if .R.Snapshot.Demo}} · DEMO DATA{{end}}{{end}}</div>
 <div class="verdict">{{.R.VerdictText}}</div>
 <div class="sub" style="margin-top:8px">{{cnt .R.Counts "pass"}} passed · {{cnt .R.Counts "warn"}} warnings · {{cnt .R.Counts "fail"}} failed{{if not .Hands}} · hands-on tests not recorded{{end}}</div></div>
</header>
<h2>Device</h2><div class="grid">{{range .R.Identity}}<div class="kv"><span>{{.Label}}</span><span{{if .Mono}} class="mono"{{end}}>{{.Value}}</span></div>{{end}}</div>
{{if .R.Battery.Available}}<h2>Battery</h2><div class="bat">
<div class="tile"><span class="sub">Health</span><b>{{pct .R.Battery.HealthPercent}}</b></div>
<div class="tile"><span class="sub">Cycles</span><b>{{.R.Battery.CycleCount}}</b><span class="sub">of {{.R.Battery.RatedCycles}} rated</span></div>
<div class="tile"><span class="sub">Capacity</span><b>{{.R.Battery.FullChargeCap}} mAh</b><span class="sub">design {{.R.Battery.DesignCapacity}} mAh</span></div>
<div class="tile"><span class="sub">Temperature</span><b>{{printf "%.1f" .R.Battery.TemperatureC}} °C</b></div></div>{{end}}
{{range .Groups}}<h2>{{.Name}}</h2><table>{{range .Checks}}<tr><td class="st"><span class="b {{.Status}}">{{icon .Status}}</span></td>
<td><div class="t">{{.Title}}</div><div class="d">{{.Summary}}</div>
{{if or .Expected .Actual}}<div class="ea">{{if .Expected}}Expected: {{.Expected}}{{end}}{{if and .Expected .Actual}} · {{end}}{{if .Actual}}Found: <span class="mono">{{.Actual}}</span>{{end}}</div>{{end}}
{{if .Advice}}<div class="adv">→ {{.Advice}}</div>{{end}}</td><td style="width:90px;text-align:right" class="d">{{if or (eq (printf "%s" .Status) "fail") (eq (printf "%s" .Status) "warn")}}{{upper (printf "%s" .Severity)}}{{end}}</td></tr>{{end}}</table>{{end}}
{{if .Notes}}<h2>Notes</h2><div class="tile">{{.Notes}}</div>{{end}}
{{if .Unreadab}}<h2>Items that could not be read</h2><div class="tile d">{{range .Unreadab}}<div>{{.}}</div>{{end}}</div>{{end}}
<footer>Generated by Smart Mobile Check v{{.R.AppVersion}}. Automated checks read data reported by the device; always combine them with the hands-on tests and verify the serial/IMEI with the carrier and Apple before paying.</footer>
</div></body></html>`))
