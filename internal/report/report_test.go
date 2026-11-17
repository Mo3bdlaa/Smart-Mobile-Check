package report

import (
	"strings"
	"testing"

	smartcheck "github.com/mo3bdlaa/smart-mobile-check"
	"github.com/mo3bdlaa/smart-mobile-check/internal/iphone"
	"github.com/mo3bdlaa/smart-mobile-check/internal/specs"
)

func TestArabicReport(t *testing.T) {
	db := specs.Load(smartcheck.Specs, "")
	r := iphone.Localize(iphone.EvaluateMode(iphone.DemoSnapshot("new"), db, "test", iphone.ModeNew), "ar")
	out, err := HTML(r, "")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{`dir="rtl"`, "فحص شراء آيفون جديد", "فحوصات الجهاز الجديد", "الجهاز أصلي وسليم"} {
		if !strings.Contains(s, want) {
			t.Errorf("Arabic report missing %q", want)
		}
	}
}

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
