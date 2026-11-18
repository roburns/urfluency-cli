// Command ruletrial tries rules on public repositories the author of a
// check chose, and reports where they fire: how many repositories and places
// per tier, and a link to each hit. A check for a problem should fire rarely
// on well-known projects and more on learner ones (CONTRIBUTING.md).
//
//	go build -o urfluency ./cmd/urfluency
//	go run ./cmd/ruletrial -cli ./urfluency rules/trials/go.some-check.txt
//
// The file lists one repository per line: its tier (well-known or learner),
// owner/name on GitHub, and a full commit ID, so a trial can be repeated on
// the same code. The rules tried are the file's name, or -rules. Each
// repository is fetched once, at that commit only, into -cache, and scanned
// without running anything in it.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const (
	wellKnown = "well-known"
	learner   = "learner"
	// minPerTier and maxEntries bound a trial: enough of each tier to compare,
	// and few enough to fetch in one CI job.
	minPerTier = 3
	maxEntries = 30
	// maxListed bounds the hits linked per rule and tier.
	maxListed = 200
)

type entry struct{ tier, repo, sha string }

var (
	repoRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)
	shaRe  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	ruleRe = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)
)

func main() {
	cli := flag.String("cli", "./urfluency", "the build to scan with")
	cache := flag.String("cache", filepath.Join(os.TempDir(), "urfluency-ruletrial"), "where repositories are fetched")
	ruleList := flag.String("rules", "", "comma-separated rule IDs; the corpus file's name by default")
	flag.Parse()
	if flag.NArg() != 1 {
		fail(fmt.Errorf("usage: ruletrial [-cli path] [-rules ids] corpus-file"))
	}
	path := flag.Arg(0)
	f, err := os.Open(path)
	if err != nil {
		fail(err)
	}
	entries, err := readCorpus(f)
	f.Close()
	if err != nil {
		fail(fmt.Errorf("%s: %w", path, err))
	}
	ids, err := ruleIDs(*ruleList, path)
	if err != nil {
		fail(err)
	}
	got := map[string]map[string][]string{}
	for _, e := range entries {
		fmt.Fprintln(os.Stderr, "scanning", e.repo)
		found, err := scan(*cli, *cache, e, ids)
		if err != nil {
			fail(fmt.Errorf("%s: %w", e.repo, err))
		}
		got[e.repo] = found
	}
	fmt.Print(report(ids, entries, got, *cache))
}

// ruleIDs are the rules to try: the list given, or the corpus file's name
// without its extension.
func ruleIDs(list, corpusPath string) ([]string, error) {
	if list == "" {
		list = strings.TrimSuffix(filepath.Base(corpusPath), filepath.Ext(corpusPath))
	}
	var ids []string
	for _, id := range strings.Split(list, ",") {
		id = strings.TrimSpace(id)
		if !ruleRe.MatchString(id) {
			return nil, fmt.Errorf("%q is not a rule ID", id)
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return slices.Compact(ids), nil
}

// readCorpus reads "tier owner/name commit" lines, skipping blank lines and
// # comments, and checks the list can support a comparison.
func readCorpus(r io.Reader) ([]entry, error) {
	var out []entry
	perTier := map[string]int{}
	seen := map[string]bool{}
	s := bufio.NewScanner(r)
	for n := 1; s.Scan(); n++ {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		switch {
		case len(p) != 3:
			return nil, fmt.Errorf("line %d: want a tier, owner/name, and a commit ID", n)
		case p[0] != wellKnown && p[0] != learner:
			return nil, fmt.Errorf("line %d: tier %q, want %s or %s", n, p[0], wellKnown, learner)
		case !repoRe.MatchString(p[1]) || strings.Contains(p[1], ".."):
			return nil, fmt.Errorf("line %d: %q is not a GitHub owner/name", n, p[1])
		case !shaRe.MatchString(p[2]):
			return nil, fmt.Errorf("line %d: %q is not a full 40-character commit ID", n, p[2])
		case seen[strings.ToLower(p[1])]:
			return nil, fmt.Errorf("line %d: %s is listed twice", n, p[1])
		}
		seen[strings.ToLower(p[1])] = true
		perTier[p[0]]++
		out = append(out, entry{p[0], p[1], p[2]})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(out) > maxEntries {
		return nil, fmt.Errorf("%d repositories, at most %d", len(out), maxEntries)
	}
	for _, tier := range []string{wellKnown, learner} {
		if perTier[tier] < minPerTier {
			return nil, fmt.Errorf("%d %s repositories, at least %d of each tier are needed to compare them", perTier[tier], tier, minPerTier)
		}
	}
	return out, nil
}

// repoDir is where a repository is fetched under the cache.
func repoDir(cache, repo string) string {
	return filepath.Join(cache, strings.ReplaceAll(repo, "/", "__"))
}

// fetch makes sure dir holds repo at sha, fetching only that commit. The
// user's and the system's Git settings are not read, and hooks cannot run.
func fetch(dir, repo, sha string) error {
	git := func(args ...string) *exec.Cmd {
		c := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null"}, args...)...)
		c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
		return c
	}
	if head, err := git("-C", dir, "rev-parse", "HEAD").Output(); err == nil && strings.TrimSpace(string(head)) == sha {
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	steps := []struct {
		name string
		args []string
	}{
		{"init", []string{"init", "-q", dir}},
		{"fetch", []string{"-C", dir, "fetch", "-q", "--depth", "1", "https://github.com/" + repo, sha}},
		{"checkout", []string{"-C", dir, "checkout", "-q", "FETCH_HEAD"}},
	}
	for _, step := range steps {
		if out, err := git(step.args...).CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %w: %s", step.name, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// located is a rule's findings in a scan's report.
type located struct {
	ID        string   `json:"id"`
	Locations []string `json:"locations"`
}

// scan returns the rules' findings in one repository: problems, and what
// provisional rules found.
func scan(cli, cache string, e entry, ids []string) (map[string][]string, error) {
	dir := repoDir(cache, e.repo)
	if err := fetch(dir, e.repo, e.sha); err != nil {
		return nil, err
	}
	out, err := exec.Command(cli, "scan", "--no-git", "--json", "--locations", "--offline", dir).Output()
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	var r struct {
		Unversioned struct {
			Languages []struct {
				Weaknesses  []located `json:"weaknesses"`
				Provisional []located `json:"provisional"`
			} `json:"languages"`
		} `json:"unversioned"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, fmt.Errorf("scan output: %w", err)
	}
	// A scan that read nothing must not pass as a repository without hits.
	if len(r.Unversioned.Languages) == 0 {
		return nil, fmt.Errorf("the scan read no files")
	}
	found := map[string][]string{}
	for _, l := range r.Unversioned.Languages {
		for _, w := range slices.Concat(l.Weaknesses, l.Provisional) {
			if slices.Contains(ids, w.ID) {
				found[w.ID] = append(found[w.ID], w.Locations...)
			}
		}
	}
	for id := range found {
		sort.Strings(found[id])
	}
	return found, nil
}

// report is the Markdown result: for each rule, a table of repositories and
// hits per tier, then the hits, well-known projects first, since each of
// those claims a mistake in expert code and is to be read.
func report(ids []string, entries []entry, got map[string]map[string][]string, cache string) string {
	type hit struct {
		e   entry
		loc string
	}
	var b strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&b, "## `%s`\n\n| Tier | Repositories | Repositories with hits | Hits |\n| --- | --- | --- | --- |\n", id)
		hits := map[string][]hit{}
		for _, tier := range []string{wellKnown, learner} {
			repos, withHits, total := 0, 0, 0
			for _, e := range entries {
				if e.tier != tier {
					continue
				}
				repos++
				locs := got[e.repo][id]
				if len(locs) > 0 {
					withHits++
				}
				total += len(locs)
				for _, loc := range locs {
					hits[tier] = append(hits[tier], hit{e, loc})
				}
			}
			fmt.Fprintf(&b, "| %s | %d | %d | %d |\n", tier, repos, withHits, total)
		}
		for _, tier := range []string{wellKnown, learner} {
			if len(hits[tier]) == 0 {
				continue
			}
			fmt.Fprintf(&b, "\n**Hits on %s projects**\n\n", tier)
			for i, h := range hits[tier] {
				if i == maxListed {
					fmt.Fprintf(&b, "- and %d more\n", len(hits[tier])-maxListed)
					break
				}
				fmt.Fprintf(&b, "- %s\n", link(cache, h.e, h.loc))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// link is a hit as a Markdown link to its line at the pinned commit. A scan
// reports a path relative to the project it found, which can be a folder
// inside the repository, so the file is looked up under the checkout; a path
// that matches no file, or several, is given as text.
func link(cache string, e entry, loc string) string {
	text := e.repo + " " + loc
	i := strings.LastIndex(loc, ":")
	if i < 0 {
		return text
	}
	path, line := loc[:i], loc[i+1:]
	root := repoDir(cache, e.repo)
	var matches []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if !d.IsDir() && (rel == path || strings.HasSuffix(rel, "/"+path) || strings.HasSuffix(path, "/"+rel)) {
			matches = append(matches, rel)
		}
		return nil
	})
	if len(matches) != 1 {
		return text
	}
	return fmt.Sprintf("[%s](https://github.com/%s/blob/%s/%s#L%s)", text, e.repo, e.sha, matches[0], line)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "ruletrial:", err)
	os.Exit(1)
}
