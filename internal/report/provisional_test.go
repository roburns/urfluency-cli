package report

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/roburns/urfluency-cli/analysis"
	"github.com/roburns/urfluency-cli/facts"
	"github.com/roburns/urfluency-cli/rules"
)

// What a provisional rule finds is listed in the language's report and
// changes nothing else: every score, the summary and combined result an
// upload carries, and the facts are the same with and without it.
func TestProvisionalFindingsAreShownAndNotScored(t *testing.T) {
	rules.Register(rules.Rule{ID: "go.trial-check", Category: rules.CleanCode, Languages: []string{"go"},
		Title: "Trial check", Why: "It is being tried.", Focus: "f", Provisional: true})
	now := time.Now()
	plain := result(500, "go.ignored-error")
	with := result(500, "go.ignored-error")
	with.Languages[0].Provisional = []analysis.Finding{{Rule: "go.trial-check", Path: "b.go", Line: 7}, {Rule: "go.trial-check", Path: "b.go", Line: 9}}

	build := func(res analysis.Result) Report {
		r := Build(res, now, true)
		r.Combined = BuildCombined(res, analysis.Result{})
		Summarize(&r)
		return r
	}
	a, b := build(plain), build(with)
	want := []ProvisionalResult{{ID: "go.trial-check", Category: "clean-code", Title: "Trial check", Count: 2,
		Why: "It is being tried.", Locations: []string{"b.go:7", "b.go:9"}}}
	if got := b.Languages[0].Provisional; !reflect.DeepEqual(got, want) {
		t.Errorf("Provisional = %+v, want %+v", got, want)
	}
	b.Languages[0].Provisional = nil
	if !reflect.DeepEqual(a.Languages, b.Languages) {
		t.Errorf("languages differ apart from the provisional list:\n%+v\n%+v", a.Languages, b.Languages)
	}
	for name, pair := range map[string][2]any{
		"summary":  {a.Summary, b.Summary},
		"combined": {a.Combined, b.Combined},
		"facts":    {facts.New(plain, analysis.Result{}, nil), facts.New(with, analysis.Result{}, nil)},
	} {
		x, _ := json.Marshal(pair[0])
		y, _ := json.Marshal(pair[1])
		if string(x) != string(y) {
			t.Errorf("%s differs with a provisional finding:\n%s\n%s", name, x, y)
		}
		if strings.Contains(string(y), "trial-check") {
			t.Errorf("%s names the provisional rule: %s", name, y)
		}
	}

	var out strings.Builder
	if err := WriteText(&out, build(with)); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Provisional checks (still being benchmarked; not scored, not uploaded)", "? Trial check [clean-code]: 2", "at b.go:9"} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("output missing %q:\n%s", text, out.String())
		}
	}
}
