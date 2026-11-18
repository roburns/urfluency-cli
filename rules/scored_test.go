package rules

import (
	"flag"
	"os"
	"slices"
	"strings"
	"testing"
)

var updateScored = flag.Bool("update", false, "rewrite scored.txt from the catalog")

// scored.txt lists every rule that counts toward a score, and this test keeps
// it in step with the catalog: a rule that is not provisional must be listed.
// Making a rule count is then a change to that file, which a reviewer sees,
// and never a flag left off a new rule (CONTRIBUTING.md, "How a new check
// reaches ratings").
func TestScoredRulesAreListed(t *testing.T) {
	var scored []string
	for _, r := range All() {
		if !r.Provisional {
			scored = append(scored, r.ID)
		}
	}
	const header = "# Rules that count toward scores, one ID per line, sorted.\n" +
		"# Maintainers add a rule here when it is promoted from provisional.\n"
	if *updateScored {
		if err := os.WriteFile("scored.txt", []byte(header+strings.Join(scored, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile("scored.txt")
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			listed = append(listed, line)
		}
	}
	if !slices.IsSorted(listed) {
		t.Error("scored.txt is not sorted")
	}
	for _, id := range scored {
		if !slices.Contains(listed, id) {
			t.Errorf("rule %s is not provisional and not in scored.txt: a new rule starts with Provisional: true", id)
		}
	}
	for _, id := range listed {
		r, ok := Lookup(id)
		switch {
		case !ok:
			t.Errorf("scored.txt lists %s, which is not in the catalog", id)
		case r.Provisional:
			t.Errorf("scored.txt lists %s, which is still provisional", id)
		}
	}
}

// A provisional rule has a list of repositories its author tried it on,
// which the Rule trial workflow runs (rules/trials/README.md).
func TestProvisionalRulesHaveATrial(t *testing.T) {
	for _, r := range All() {
		if !r.Provisional {
			continue
		}
		if _, err := os.Stat("trials/" + r.ID + ".txt"); err != nil {
			t.Errorf("provisional rule %s has no rules/trials/%s.txt", r.ID, r.ID)
		}
	}
}
