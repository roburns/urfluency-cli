# Contributing

Thank you for helping urFluency rate code more accurately. This guide covers
what you can contribute, the standard a check has to meet, and how a new check
reaches people's ratings.

## Before you start

Three things to know, whatever language your check is for:

1. **Detection is written in Go.** urFluency's analyzers are Go programs, so
   a check for Python, TypeScript, or any other language is Go code that
   reads that language's syntax tree. You do not write the check in the
   language it checks.
2. **You do not need to be a Go expert.** Most checks are 5 to 20 lines
   modeled on a check that is already there, and a tool in this repository
   shows you the syntax tree to match. The
   [worked example](#worked-example-a-python-check-from-start-to-finish)
   goes through every file you touch.
3. **You can contribute without writing code.** If you know the practice but
   not Go, open a "Propose a check" issue with the flagged example, the
   corrected one, and the look-alikes. That knowledge is the hard part of a
   check. A proposal may be picked up by a maintainer or another contributor,
   who writes the detection; it is not a promise that it will be. If it is
   built, you are credited as its author.

To write a check you need Git, Go at the version in `go.mod`, and a C
compiler (Xcode Command Line Tools on macOS, `build-essential` on Debian and
Ubuntu), because the parsers are C. You do not need an urFluency account.

## What you can contribute

- **A wrong finding.** If a check flags code that is correct and deliberate,
  open an issue with the rule ID, the language, and a short example that
  reproduces it. Use code you are free to share, never private code.
- **A fix to a check.** Narrow a rule that misfires, or widen one that misses
  an obvious case, with a test for each case you change.
- **A new check.** A problem or a strength that an experienced reviewer would
  point out in that language.
- **Documentation and CLI fixes.**

For anything larger than a fix, open an issue first so we can agree on the
approach before you write it.

Do not report a security problem in a public issue or pull request. Report
it privately with "Report a vulnerability" on the repository's
[Security tab](https://github.com/roburns/urfluency-cli/security).

## How a check is built

A check is three small pieces, in the same places for every language:

| Piece | File | What it holds |
| --- | --- | --- |
| The rule | `rules/<language>.go` | The ID, category, title, and the `Why` and `Focus` texts people read in their report. Strengths (the `idiom` and `construct` categories) have no `Focus` |
| The detection | `analyzers/<language>/` | Go code that reports a finding with the rule's ID at each place it applies |
| The tests | `analyzers/<language>/*_test.go` | Source that must be flagged, and look-alikes that must not |

The folders are `c`, `cpp`, `csharp`, `dart`, `golang`, `java`, `javascript`
(which also holds TypeScript), `kotlin`, `php`, `powershell`, `python`,
`ruby`, `rust`, `shell`, and `swift`.

Go is the one language analyzed differently: `analyzers/golang` uses Go's
own parser (`go/ast`). Every other language is parsed with tree-sitter
through the shared `parsing` package, and `analyzers/python` is the
reference to copy from.

Analyzers parse code and never run it. They read only the files they are
given and make no network calls.

## Worked example: a Python check from start to finish

This follows a check that is already in the repository,
`py.is-literal-comparison`, so you can read the real code beside it. It flags
`x is 1`, which compares identity where the author meant value.

**1. See the syntax tree.** Put the code you want to flag in a file and
print what the analyzer sees:

```sh
printf 'def f(a):\n    return a is 1\n' > example.py
go run ./cmd/syntaxtree example.py
```

```
        comparison_operator  (line 2)
          identifier  (line 2)  a
          operators: "is"  (line 2)
          integer  (line 2)  1
```

Each line is a node: its kind, its line, and its text when it has no
children. Quoted kinds are keywords and operators. So the target is a
`comparison_operator` that has an `"is"` token and a literal, here an
`integer`, as an operand. Print the look-alikes too (`a is None`, `a == 1`)
to see what tells them apart.

**2. Add the rule** to `rules/python.go`. A new rule also sets
`Provisional: true`; this one is older and already counts.

```go
Rule{
	ID:        "py.is-literal-comparison",
	Category:  LanguageProficiency,
	Languages: py,
	Title:     "Identity comparison with a literal",
	Why:       "is compares object identity, not value; x is 1 or x is \"a\" depends on interpreter caching and can be false for equal values.",
	Focus:     "Use == for values; keep is for None, True, and False.",
},
```

**3. Add the detection** to `analyzers/python/python.go`. The analyzer
visits every node; you add a case for the kind you found and report the rule:

```go
case "comparison_operator":
	if parsing.HasToken(n, "is", "is not") {
		for _, operand := range parsing.NamedChildren(n) {
			if literalKinds[operand.Kind()] {
				c.report("py.is-literal-comparison", n)
				break
			}
		}
	}
```

`parsing.HasToken`, `parsing.NamedChildren`, and `c.tree.Text(n)` cover most
checks. `None`, `True`, and `False` are not in `literalKinds`, which is how
the look-alike `a is None` is left alone.

**4. Add a test** to the table in `analyzers/python/python_test.go`: source
with the target and the look-alikes, and the findings expected as
`rule:line`.

```go
{
	name: "identity comparisons with literals but not None",
	path: "app/a.py",
	src: `def f(a: int) -> bool:
    return a is 1 or a is not "x" or a is None or a is True
`,
	want: []string{"py.is-literal-comparison:2", "py.is-literal-comparison:2"},
},
```

Two findings, for `is 1` and `is not "x"`; none for `is None` or `is True`.

**5. Run the tests.**

```sh
go test ./rules ./analyzers/python
```

**6. Try it on real projects**, as step 3 of
[How a new check reaches ratings](#how-a-new-check-reaches-ratings)
describes, then open the pull request.

## The standard a check must meet

1. **It names a real practice.** A mistake an expert would point out in that
   language, or a strength that shows command of it. Cite where the practice
   comes from: the language's documentation, a well-known linter's rule, or
   examples in widely used projects.
2. **It does not flag correct code.** Run it on several well-maintained public
   projects in the language and read every hit. A hit on correct, deliberate
   code is a false alarm; narrow the check until there are none. The steps
   below say how.
3. **It finds what it claims to.** Tests cover the target and each look-alike
   that must not be flagged, including every false alarm you fixed.
4. **Its texts help.** `Why` says what goes wrong, and `Focus` says what to do
   instead, in plain words.

## How a new check reaches ratings

Ratings appear on profiles and badges, so a check does not affect them until
it has been measured. The steps, and who does each:

1. **Propose it (you).** Open a "Propose a check" issue with the practice,
   its source, a flagged and a corrected example, and the look-alikes. We
   agree on it, or say why not, before you write anything.
2. **Write it (you).** Add the rule with `Provisional: true`, the detection,
   and tests. Leave `rules/scored.txt` alone: a test fails if a new rule is
   neither provisional nor listed there, and only maintainers add to it.
3. **Try it on real code (you).** You choose the repositories. Add
   `rules/trials/<rule-id>.txt` listing public GitHub projects in the
   language, each at a full commit ID: at least three widely used,
   well-maintained ones (`well-known`) and three tutorials, course projects,
   or first applications (`learner`), up to 30 in all.
   [rules/trials/README.md](rules/trials/README.md) has the format. Then:

   ```sh
   go build -o urfluency ./cmd/urfluency
   go run ./cmd/ruletrial -cli ./urfluency rules/trials/<rule-id>.txt
   ```

   The report counts the check's hits per tier and links each one. Read
   every hit on a well-known project and narrow the check until none is a
   false alarm. When you open the pull request, the Rule trial workflow runs
   the same trial and puts the report in the run's summary, so reviewers see
   what you saw. Choose projects that test the check, not ones that flatter
   it: step 5 runs it on repositories you did not choose.
4. **Review (maintainers).** We read the code first. It runs on our
   benchmark machines in the next step, so an analyzer that does anything
   but parse the files it is given is declined.
5. **Benchmark (maintainers).** We run the reviewed commit on urFluency's
   benchmark: a few hundred public repositories at fixed commits, from widely
   used projects to beginner ones. The benchmark itself is not public, but
   its result for your check is. We post it on the pull request: how often
   the check fired on well-known projects and on learner ones, and a link to
   every hit on a well-known project.
6. **Triage (together).** Every hit on a well-known project claims a mistake
   in expert code, so each is read. Where the code is correct and deliberate,
   you narrow the check and add that case as a test, and we run the
   benchmark again. A check for a problem should fire rarely on well-known
   projects and more on learner ones; one that fires alike on both is
   measuring style, not skill.
7. **Merge and release as provisional (maintainers).** People see what the
   check finds in their local report. It counts toward no score, and nothing
   about it is uploaded.
8. **Promote (maintainers).** After a release as provisional with no false
   alarms reported, a maintainer removes `Provisional`, adds the rule to
   `rules/scored.txt`, and the next release counts it. The release notes
   name you as its author unless you ask us not to.

A check that cannot pass triage is closed with the hits that decided it, so
the reason is on record. A fix to an existing, scored check goes through
steps 2 to 6; it has no provisional stage, because the rule already counts.

One limit: the trial lists where problems and provisional checks fire, but a
strength that already counts (an `idiom` or `construct` rule) reports only
how often it was used, not where. A fix to one of those is judged from its
tests, so cover every case you change.

Checks for the safety scanner (`urfluency check`), including secret
detection, are the exception: they are never provisional. A safety verdict is
shown for other people's repositories, so those checks are benchmarked before
they are merged. Open an issue first if you want to add one.

The scoring models themselves are not changed through contributed checks. The
local model in this repository is open, and changes to it are discussed in an
issue first. The official model, which scores uploads, is maintained by
urFluency.

## Before you open a pull request

```sh
gofmt -l .                 # must print nothing
go vet ./...
go test -race ./...        # needs a C compiler for tree-sitter
```

Keep the change focused on one check or one fix. Follow the conventions in
[PROJECT_CONTEXT.md](PROJECT_CONTEXT.md): errors are wrapped and returned,
tests are table-driven, and test fixtures are synthetic. New dependencies need
a stated reason.

## Contribution terms

The CLI is released under [FSL-1.1-MIT](LICENSE.md), and RB Creations also
ships it inside urFluency's official build and services. A license from you
on FSL terms alone would not clearly allow that, so contributions are
accepted on the terms below. They follow the usual shape of a contributor
license agreement: you keep ownership of your work and grant the rights the
project needs.

By submitting a contribution (code, tests, documentation, or any other
material) to this repository, you agree that:

1. **You can submit it.** You wrote it, or you have the right to submit it on
   these terms. If your employer has rights to what you write, you have its
   permission. Where part of it is someone else's work, you say so in the
   pull request and name its license.
2. **You keep your copyright.** These terms do not transfer ownership.
3. **Copyright license.** You grant Robert Burns (RB Creations) a perpetual,
   worldwide, non-exclusive, royalty-free, irrevocable license to use, copy,
   modify, publish, and distribute your contribution and works based on it,
   and to sublicense those rights, under any license terms. This covers
   releasing it under FSL-1.1-MIT and later MIT, as [LICENSE.md](LICENSE.md)
   describes, and including it in urFluency's products.
4. **Patent license.** You grant RB Creations, and everyone who receives the
   software from it, a perpetual, worldwide, non-exclusive, royalty-free,
   irrevocable license under any patent claims you can license that your
   contribution necessarily infringes, alone or combined with the software
   it was submitted to.
5. **No warranty.** You provide your contribution as it is, without warranty
   of any kind.

To record your agreement, include this line in the pull request description:

> I agree to the contribution terms in CONTRIBUTING.md.

A pull request without it cannot be merged.
