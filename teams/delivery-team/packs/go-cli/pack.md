---
area: go-cli
stack: "Go + Cobra CLI"
---

# Go-CLI pack — Go + Cobra command-line tools

This pack covers **command-line tool** work in Go: cobra commands and their flags, the pure
helpers behind them, and the code-surface gate (`go build`/`vet`/`test`) that verifies them. The
tech-lead tags a work-unit `area:go-cli` when its acceptance criteria live in a Go CLI — a command,
a flag, an output line, a validation, an internal package — with no rendered UI, no browser or
emulator surface (it is verified entirely on the code surface / CI). A developer worker that loads
this pack is building or extending a Go cobra tool.

This pack is the **generic stack craft** for that concern — how to build and test a Go cobra CLI
*anywhere*. The **project's** specific choices (its actual package layout, its command tree, its
error conventions, any repo-specific gate) live in the project's durable-knowledge store
(`Conventions/`, `Architecture/`) layered ATOP this pack, and the tech-lead's canonical brief names
the exact pages — including, for a repo that ships one, its contributor doc (e.g. a `cli/CLAUDE.md`).
See [`knowledge/pack-format.md`](../../knowledge/pack-format.md) for the three-layer read contract
(pack = generic stack / durable-knowledge = project-specific / brief = the bridge).

## Topics

- [command-conventions.md](command-conventions.md) — cobra command structure, `RunE` (return
  errors, never `os.Exit`), flag/arg validation incl. mutual-exclusivity, and honest output through
  a pure helper.
- [code-conventions.md](code-conventions.md) — Go idioms for a CLI: extract a pure helper for every
  decision worth testing, wrap errors with context, never discard an error that changes behavior,
  package layout, and the embedded-asset re-sync gate.
- [testing.md](testing.md) — the table test over the extracted pure helper (the wide base of the
  pyramid), what is and isn't unit-testable in a cobra command, and the code-surface gate commands.

## Test commands

From the module root (commonly `cli/`):

- build (a green test suite over code that doesn't compile is a false green — build first):
  `go build ./...`
- vet (catches the mistakes the compiler doesn't — printf mismatches, unreachable code):
  `go vet ./...`
- unit tests: `go test ./...` (or scope to the touched package, e.g.
  `go test ./cmd/atl/commands/`, for a fast inner loop — but the **gate** is the full `./...`)

There is no web or mobile surface command — a `go-cli` unit is verified on the **code surface** (CI)
only. If the change is user-facing (a new command/flag/output the tool documents), the project may
run a docs gate too (e.g. `atl docs check`) — the brief names it when it applies.

## Key conventions

- **`RunE`, not `Run`; return errors, never `os.Exit` inside a command.** A command's action is a
  `RunE func(...) error` that *returns* its failures; cobra prints them and sets the exit code once,
  centrally. A helper that calls `os.Exit` can't be unit-tested and bypasses cobra's error handling.
  See [command-conventions.md](command-conventions.md).
- **Extract a pure helper for every decision worth testing.** The cobra closure opens files, reads a
  queue, touches the network — it is not unit-testable as-is. So the *decision* (which message to
  print, whether flags conflict, how a value maps) moves into a small pure function with no I/O,
  which a table test then pins. This is the wide base of the test pyramid.
  See [code-conventions.md](code-conventions.md), [testing.md](testing.md).
- **Never discard an error that changes behavior.** `_ = doThing()` is fine only when the outcome is
  genuinely irrelevant; if the error should change what the tool prints or does, capture it and
  branch. (The gold-standard fix: `atl update` printed "everything up to date" even offline because
  it dropped the refresh error — captured, it now tells the honest truth.)
- **Honest, single-sourced output.** A user-facing line that can vary (success vs. offline vs.
  no-op) is produced by one pure helper so every variant is testable and consistent — the tool never
  asserts something it didn't do (no "reversible" when nothing was removed, no "up to date" when the
  check couldn't run).
- **Validate flags/args up front, fail loudly on contradictions.** Mutually exclusive modes are
  rejected with a clear error before any work — not silently precedence-ordered (especially when one
  branch is irreversible). See [command-conventions.md](command-conventions.md).
- **The code surface is the whole gate.** `go build ./... && go vet ./... && go test ./...` all green
  is the developer's Level-1 self-test; the tester's Level-2 pass gates the green. Evidence (the
  run output) attaches to the work-item.

## Dependency baseline

Go CLIs lean on the standard library; the one near-universal third-party dependency is the command
framework. A real project pins exact versions in `go.mod`/`go.sum` and records its own baseline in
the durable-knowledge store.

- **Go** — a current toolchain (`go 1.2x` in `go.mod`); the standard library (`flag` is superseded by
  cobra here, but `encoding/json`, `os`, `fmt`, `errors` are the everyday tools).
- **spf13/cobra** `^1.8` — the command/subcommand/flag framework; the reference for `Command`, `RunE`,
  and `Flags()`. Its companion **spf13/pflag** provides the POSIX flags.
- **testing** (stdlib) — table tests via `t.Run` subtests; no third-party assertion library is
  required (and often deliberately avoided). See [testing.md](testing.md).
