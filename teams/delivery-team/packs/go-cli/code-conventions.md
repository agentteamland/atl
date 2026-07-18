# Code conventions — helpers, errors, packages, embedded-asset gates

The craft here is what makes a Go CLI **testable and honest** below the command layer: extract the
decisions, never swallow a load-bearing error, keep packages small, and respect any embedded-asset
re-sync gate the repo enforces.

## Extract a pure helper for every decision worth testing

The cobra closure is not unit-testable (it opens files, reads a queue, hits the network). So the
*decision* it makes — which message, whether inputs conflict, how a value maps — moves into a small
pure function: inputs in, value out, **no I/O**. That function gets a table test; the closure just
calls it. This is the single highest-leverage habit for a testable CLI (see [testing.md](testing.md)).

- A pure helper takes plain values (or already-loaded data) and returns a value or `(value, error)`.
- It never reads `os`, the filesystem, the network, or `time.Now` directly — if it needs the time or
  a path, take it as a parameter, so the test controls it.

## Never discard an error that changes behavior

`_ = doThing()` is acceptable *only* when the outcome is genuinely irrelevant. If the error should
change what the tool prints or does, **capture and branch on it**:

```go
// wrong — the tool later claims success it can't vouch for
_ = index.RefreshCache(url)

// right — capture, then let the honest-output helper use it
refreshErr := index.RefreshCache(url)
// …
fmt.Println(upToDateMessage(refreshErr != nil))
```

Grep a command you're touching for `_ = ` before you trust its output lines.

## Wrap errors with context; don't panic in library code

Return errors up the stack; add context with `fmt.Errorf("doing X for %q: %w", name, err)` (the `%w`
preserves the chain for `errors.Is`/`errors.As`). Reserve `panic` for truly-impossible states, never
for a bad flag or a missing file — those are ordinary returned errors.

## Package layout

- User-facing command wiring lives near the entrypoint (e.g. `cmd/<tool>/commands/`); the reusable
  logic lives under `internal/<topic>/` so it can't be imported outside the module and stays a clean,
  separately-testable unit.
- One concern per package; a package that opens a store, formats output, *and* parses flags is three
  packages. Small packages keep the pure helpers pure.

## Match the repo's own conventions doc

A mature Go repo often ships a contributor doc (e.g. `cli/CLAUDE.md`) naming its exact build/test
commands, its commit format, and gotchas. The tech-lead's brief points at it — **read it and follow
it**; it overrides the generic advice here where they differ.

## Embedded assets — re-sync + commit in the same change

If the repo embeds files into the binary via `go:embed` from a canonical source tree (a common
pattern: a `core/` or `assets/` dir mirrored into an `embed/` copy the binary bakes in), an
edit to the source **must** re-run the repo's sync step and commit the regenerated `embed/` copy in
the *same* commit — a guard test (e.g. `TestEmbedMatchesCore`) fails the build otherwise. The brief
or the repo's conventions doc names the exact command; don't hand-edit the embedded copy.
