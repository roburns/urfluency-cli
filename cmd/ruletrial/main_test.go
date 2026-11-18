package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sha = "0123456789abcdef0123456789abcdef01234567"

func corpus(lines ...string) string { return strings.Join(lines, "\n") + "\n" }

func TestReadCorpus(t *testing.T) {
	good := corpus("# my picks", "",
		"well-known a/one "+sha, "well-known a/two "+sha, "well-known a/three "+sha,
		"learner b/one "+sha, "learner  b/two\t"+sha, "learner b/three "+sha)
	entries, err := readCorpus(strings.NewReader(good))
	if err != nil || len(entries) != 6 || entries[4] != (entry{learner, "b/two", sha}) {
		t.Fatalf("readCorpus = %+v, %v", entries, err)
	}
	many := []string{}
	for i := range maxEntries + 1 {
		many = append(many, "learner b/r"+string(rune('a'+i%26))+string(rune('a'+i/26))+" "+sha)
	}
	for name, tc := range map[string]struct{ in, want string }{
		"missing field":    {corpus("well-known a/one"), "want a tier"},
		"unknown tier":     {corpus("top a/one " + sha), "tier \"top\""},
		"url, not a name":  {corpus("learner https://github.com/a/one " + sha), "not a GitHub owner/name"},
		"path traversal":   {corpus("learner a/.. " + sha), "not a GitHub owner/name"},
		"option as a name": {corpus("learner --upload-pack=x/y " + sha), "not a GitHub owner/name"},
		"short commit":     {corpus("learner a/one 0123456"), "full 40-character commit ID"},
		"branch name":      {corpus("learner a/one main"), "full 40-character commit ID"},
		"listed twice":     {corpus("learner a/one "+sha, "learner A/One "+sha), "listed twice"},
		"too many":         {corpus(many...), "at most"},
		"one tier only":    {corpus("learner b/one "+sha, "learner b/two "+sha, "learner b/three "+sha), "0 well-known repositories"},
		"too few learners": {corpus("well-known a/one "+sha, "well-known a/two "+sha, "well-known a/three "+sha, "learner b/one "+sha), "1 learner repositories"},
	} {
		if _, err := readCorpus(strings.NewReader(tc.in)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want one containing %q", name, err, tc.want)
		}
	}
}

func TestRuleIDs(t *testing.T) {
	if ids, err := ruleIDs("", "rules/trials/go.some-check.txt"); err != nil || len(ids) != 1 || ids[0] != "go.some-check" {
		t.Errorf("from the file name: %v, %v", ids, err)
	}
	if ids, err := ruleIDs("py.b, go.a,go.a", "x.txt"); err != nil || strings.Join(ids, " ") != "go.a py.b" {
		t.Errorf("from the list: %v, %v", ids, err)
	}
	if _, err := ruleIDs("go.a;rm", "x.txt"); err == nil {
		t.Error("a rule ID with a semicolon was accepted")
	}
}

// A report counts a rule's hits per tier and links each to its line at the
// pinned commit: a path inside a project folder is found under the checkout,
// and one that matches no file, or two, stays text.
func TestReport(t *testing.T) {
	cache := t.TempDir()
	for _, f := range []string{"o__known/app/src/a.go", "o__known/x/dup.go", "o__known/y/dup.go", "o__learn/b.go"} {
		p := filepath.Join(cache, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries := []entry{{wellKnown, "o/known", "aaa"}, {wellKnown, "o/quiet", "bbb"}, {learner, "o/learn", "ccc"}}
	got := map[string]map[string][]string{
		"o/known": {"go.trial": {"src/a.go:3", "dup.go:9", "gone.go:1"}, "go.other": {"src/a.go:1"}},
		"o/learn": {"go.trial": {"b.go:4"}},
	}
	out := report([]string{"go.trial"}, entries, got, cache)
	for _, want := range []string{
		"## `go.trial`",
		"| well-known | 2 | 1 | 3 |",
		"| learner | 1 | 1 | 1 |",
		"- [o/known src/a.go:3](https://github.com/o/known/blob/aaa/app/src/a.go#L3)",
		"- o/known dup.go:9\n",
		"- o/known gone.go:1\n",
		"- [o/learn b.go:4](https://github.com/o/learn/blob/ccc/b.go#L4)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "go.other") {
		t.Errorf("report names a rule that was not asked for:\n%s", out)
	}
}
