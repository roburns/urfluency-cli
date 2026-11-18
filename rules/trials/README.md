# Rule trials

Each file here is the list of public repositories a check was tried on,
chosen by the check's author: `<rule-id>.txt`, one repository per line.

```
# tier        owner/name          full commit ID
well-known    go-chi/chi          9f8adb1aabaa8a91180a821111d3f8e668d1c0bf
learner       someone/first-api   0123456789abcdef0123456789abcdef01234567
```

- **Tier** is `well-known` for widely used, well-maintained projects, or
  `learner` for tutorials, course projects, and first applications. A list
  needs at least 3 of each and at most 30 repositories in all.
- **The commit ID** is the full 40 characters, so the trial can be repeated
  on the same code and its links keep pointing at the same lines.

A provisional rule must have a list. Run a trial with:

```sh
go build -o urfluency ./cmd/urfluency
go run ./cmd/ruletrial -cli ./urfluency rules/trials/<rule-id>.txt
```

The Rule trial workflow runs the same on every pull request that adds or
changes a list. See [CONTRIBUTING.md](../../CONTRIBUTING.md).
