## What this changes

## For a new or changed check

- Issue where it was agreed:
- Rule ID:
- [ ] A new rule has `Provisional: true` and is not added to `rules/scored.txt`
- [ ] Tests cover the target and each look-alike that must not be flagged
- [ ] `rules/trials/<rule-id>.txt` lists the public projects I tried it on
- What the trial found, and any hit on a well-known project I judged correct:

## Checks run

- [ ] `gofmt -l .` prints nothing
- [ ] `go vet ./...`
- [ ] `go test -race ./...`

## Contribution terms

Keep this line to record your agreement, as CONTRIBUTING.md requires:

I agree to the contribution terms in CONTRIBUTING.md.
