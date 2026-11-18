// Command syntaxtree prints the syntax tree an analyzer sees for a source
// file, so the author of a check can find the node kinds and field names to
// match (CONTRIBUTING.md). The language comes from the file's extension.
//
//	go run ./cmd/syntaxtree example.py
//
// Each line is a node: its field name in the parent, if any, its kind, and
// its line. Quoted kinds are keywords and operators such as "is". A node
// without children also shows its text. Go files are not shown: the Go
// analyzer uses go/ast, not this tree.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/roburns/urfluency-cli/parsing"
)

var grammars = map[string]parsing.Grammar{
	".py": parsing.Python,
	".js": parsing.JavaScript, ".jsx": parsing.JavaScript, ".mjs": parsing.JavaScript, ".cjs": parsing.JavaScript,
	".ts": parsing.TypeScript, ".tsx": parsing.TSX,
	".rs": parsing.Rust,
	".c":  parsing.C, ".h": parsing.C,
	".cc": parsing.CPP, ".cpp": parsing.CPP, ".hpp": parsing.CPP,
	".cs":    parsing.CSharp,
	".java":  parsing.Java,
	".kt":    parsing.Kotlin,
	".swift": parsing.Swift,
	".dart":  parsing.Dart,
	".sh":    parsing.Bash, ".bash": parsing.Bash,
	".ps1": parsing.PowerShell,
	".php": parsing.PHP,
	".rb":  parsing.Ruby,
}

func main() {
	if len(os.Args) != 2 {
		fail(fmt.Errorf("usage: syntaxtree file"))
	}
	path := os.Args[1]
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".go" {
		fail(fmt.Errorf("the Go analyzer uses go/ast, not tree-sitter; see analyzers/golang"))
	}
	g, ok := grammars[ext]
	if !ok {
		fail(fmt.Errorf("no grammar for %q files", ext))
	}
	src, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	tree, err := parsing.Parse(g, src)
	if err != nil {
		fail(err)
	}
	defer tree.Close()
	show(os.Stdout, tree, tree.Root(), "", 0)
}

// maxText bounds the text shown for a leaf.
const maxText = 40

// show writes n and its children, one per line, indented by depth.
func show(w io.Writer, t *parsing.Tree, n *ts.Node, field string, depth int) {
	line := strings.Repeat("  ", depth)
	if field != "" {
		line += field + ": "
	}
	if n.IsNamed() {
		line += n.Kind()
	} else {
		line += fmt.Sprintf("%q", n.Kind())
	}
	line += fmt.Sprintf("  (line %d)", parsing.Line(n))
	if n.IsNamed() && n.ChildCount() == 0 {
		text := strings.Join(strings.Fields(t.Text(n)), " ")
		if len(text) > maxText {
			text = text[:maxText] + "…"
		}
		line += "  " + text
	}
	fmt.Fprintln(w, line)
	for i := uint(0); i < n.ChildCount(); i++ {
		show(w, t, n.Child(i), n.FieldNameForChild(uint32(i)), depth+1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "syntaxtree:", err)
	os.Exit(1)
}
