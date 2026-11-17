package report

import (
	"strings"
	"testing"

	smartcheck "github.com/mo3bdlaa/smart-mobile-check"
	"github.com/mo3bdlaa/smart-mobile-check/internal/iphone"
	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

func TestHTMLReport(t *testing.T) {
	db := specs.Load(smartcheck.Specs, "")
	for _, p := range []string{"healthy", "tampered"} {
		r := iphone.Evaluate(iphone.DemoSnapshot(p), db, "test")
		r = iphone.ApplyManual(r, map[string]iphone.ManualResult{"touch": {ID: "touch", Status: iphone.Pass}})
		out, err := HTML(r, "note <b>escaped</b>")
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		s := string(out)
		if !strings.Contains(s, r.ModelName) || !strings.Contains(s, "&lt;b&gt;escaped") {
			t.Errorf("%s: report missing content", p)
		}
		if name := FileName(r, "html"); !strings.HasSuffix(name, ".html") || strings.ContainsAny(name, `/\ `) {
			t.Errorf("bad file name %q", name)
		}
	}
}
