package rules

import "testing"

// A provisional rule can be looked up and listed, but is not one of the
// rules its language is scored on.
func TestProvisionalRulesAreNotScored(t *testing.T) {
	t.Cleanup(func() {
		delete(catalog, "trial.unproven")
		delete(catalog, "trial.proven")
	})
	Register(
		Rule{ID: "trial.unproven", Category: CleanCode, Languages: []string{"trial"}, Title: "Unproven", Why: "w", Focus: "f", Provisional: true},
		Rule{ID: "trial.proven", Category: CleanCode, Languages: []string{"trial"}, Title: "Proven", Why: "w", Focus: "f"},
	)
	if r, ok := Lookup("trial.unproven"); !ok || !r.Provisional {
		t.Errorf("Lookup = %+v, %v, want the provisional rule", r, ok)
	}
	scored := ForLanguage("trial")
	if len(scored) != 1 || scored[0].ID != "trial.proven" {
		t.Errorf("ForLanguage = %+v, want only trial.proven", scored)
	}
	listed := false
	for _, r := range All() {
		listed = listed || r.ID == "trial.unproven"
	}
	if !listed {
		t.Error("All leaves out the provisional rule")
	}
}
