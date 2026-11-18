package analysis

import (
	"context"
	"reflect"
	"testing"

	"github.com/roburns/urfluency-cli/rules"
)

// Findings of a provisional rule on owned lines are kept apart: they are not
// findings, idioms, or AI-assisted findings, whether problems or strengths,
// and Merge carries them with their repository's label.
func TestProvisionalFindingsCountTowardNothing(t *testing.T) {
	rules.Register(
		rules.Rule{ID: "trial.problem", Category: rules.CleanCode, Languages: []string{"trial"}, Title: "t", Why: "w", Focus: "f", Provisional: true},
		rules.Rule{ID: "trial.idiom.strength", Category: rules.Idiom, Languages: []string{"trial"}, Title: "t", Why: "w", Provisional: true},
	)
	group := []ownedFile{{
		file:     File{Path: "a.trial", Content: []byte("one\ntwo\nthree\nfour\n")},
		owned:    map[int]string{1: "c1", 2: "c1", 3: "c1"},
		assisted: map[int]bool{1: true},
	}}
	analyze := func(context.Context, []File) ([]Finding, error) {
		return []Finding{
			{Rule: "trial.idiom.strength", Path: "a.trial", Line: 2, Positive: true},
			{Rule: "trial.problem", Path: "a.trial", Line: 1},
			{Rule: "trial.problem", Path: "a.trial", Line: 4}, // not an owned line
			{Rule: "scored.problem", Path: "a.trial", Line: 3},
		}, nil
	}
	lr, _, err := run(context.Background(), "trial", group, analyze, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(lr.Findings) != 1 || lr.Findings[0].Rule != "scored.problem" {
		t.Errorf("Findings = %+v, want only scored.problem", lr.Findings)
	}
	if len(lr.Idioms) != 0 || lr.AssistedFindings != 0 || len(lr.AssistedByRule) != 0 {
		t.Errorf("provisional findings were counted: idioms %v, assisted %d %v", lr.Idioms, lr.AssistedFindings, lr.AssistedByRule)
	}
	want := []Finding{
		{Rule: "trial.problem", Path: "a.trial", Line: 1},
		{Rule: "trial.idiom.strength", Path: "a.trial", Line: 2, Positive: true},
	}
	if !reflect.DeepEqual(lr.Provisional, want) {
		t.Errorf("Provisional = %+v, want %+v", lr.Provisional, want)
	}

	part := Result{Languages: []LanguageResult{lr}}
	merged := Merge([]string{"one", "two"}, []Result{part, part})
	got := merged.Languages[0].Provisional
	if len(got) != 4 || got[0].Path != "one/a.trial" || got[3].Path != "two/a.trial" {
		t.Errorf("merged Provisional = %+v, want two from each repository under its label", got)
	}
}
