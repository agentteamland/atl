# Testing — table tests over pure helpers, the code-surface gate

A Go CLI is verified entirely on the **code surface**: `go build ./... && go vet ./... &&
go test ./...`. There is no browser or emulator. The craft is putting the coverage where it's cheap
and meaningful — a **table test over the extracted pure helper** — not fighting to test the cobra
closure.

## The wide base: a table test over the pure helper

The decision lives in a pure function ([code-conventions.md](code-conventions.md)); the test drives it
with a table of `name → input → want` cases. This is fast, deterministic, and exhaustive over the
branches that matter. Use stdlib `testing` + `t.Run` subtests — no third-party assertion library is
needed.

```go
func TestUpToDateMessage(t *testing.T) {
    for _, tc := range []struct {
        name    string
        offline bool
        want    string
    }{
        {"online, nothing changed", false, "atl update: everything up to date"},
        {"offline, refresh could not run", true, "atl update: up to date (offline — using cached index)"},
    } {
        t.Run(tc.name, func(t *testing.T) {
            if got := upToDateMessage(tc.offline); got != tc.want {
                t.Errorf("upToDateMessage(%v) = %q, want %q", tc.offline, got, tc.want)
            }
        })
    }
}
```

Cover the branches that carry meaning: the empty/zero case (a `--json` helper returns `"{}"`, not
`null`), each flag-conflict combination (every pair that must error, plus the single-mode cases that
must *not*), and the honest-vs-misleading output split (n==0 vs n>0). Name each subtest for the case
it locks, so a failure reads like a spec.

## What is and isn't testable

- **Testable:** the pure helpers — message selection, flag-conflict logic, value mapping, JSON
  shaping. If the important logic is in a helper, the table test *is* the coverage.
- **Not worth fighting:** the cobra `RunE` closure that opens a real store or hits the network. Don't
  build elaborate mocks to drive it end-to-end; if you feel that urge, the decision hasn't been
  extracted yet — extract it, then table-test the extract.
- **Negative assertions matter too:** if a change is "this no longer hardcodes X" or "the human path
  is byte-identical when `--json` is off", assert the *absence*/equality explicitly so a regression
  trips the test.

## The gate

- `go build ./...` — first: a green suite over non-compiling code is a false green.
- `go vet ./...` — catches printf/format mismatches and other non-compiler smells.
- `go test ./...` — the full module, not just the touched package. (Scope to one package —
  `go test ./cmd/<tool>/commands/` — for a fast inner loop, but the **gate** is `./...`, so a change
  can't silently break a sibling package.)

All three green is the developer's Level-1 self-test. The tester re-runs them independently (Level-2)
and probes edges/regression before gating green; the run output is the evidence attached to the
work-item. A surface you couldn't run is UNVERIFIED — block, never fake a green.
