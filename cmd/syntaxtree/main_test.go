package main

import (
	"strings"
	"testing"

	"github.com/roburns/urfluency-cli/parsing"
)

// The tree shows what a check matches on: node kinds, field names, quoted
// keywords and operators, lines, and the text of leaves.
func TestShow(t *testing.T) {
	tree, err := parsing.Parse(parsing.Python, []byte("def f(a):\n    return a is 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	var b strings.Builder
	show(&b, tree, tree.Root(), "", 0)
	for _, want := range []string{
		"module  (line 1)\n",
		"    name: identifier  (line 1)  f\n",
		"        comparison_operator  (line 2)\n",
		"          operators: \"is\"  (line 2)\n",
		"          integer  (line 2)  1\n",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("tree missing %q:\n%s", want, b.String())
		}
	}
}

// Every extension an analyzer reads through tree-sitter has a grammar here.
func TestGrammarsCoverTreeSitterLanguages(t *testing.T) {
	for _, ext := range []string{".py", ".js", ".ts", ".tsx", ".rs", ".c", ".cpp", ".cs", ".java", ".kt", ".swift", ".dart", ".sh", ".ps1", ".php", ".rb"} {
		if _, ok := grammars[ext]; !ok {
			t.Errorf("no grammar for %s", ext)
		}
	}
}
