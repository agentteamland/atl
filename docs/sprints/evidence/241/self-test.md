# Self-test evidence — #241 `atl update` offline-honest message

Level-1 self-test for work-item [#241](https://github.com/agentteamland/atl/issues/241)
("atl update: offline-honest message when the index refresh could not run").

## What was verified

- **AC1 — offline path.** `upToDateMessage(true)` returns
  `atl update: up to date (offline — using cached index)` (covered by the
  `offline, refresh could not run` sub-test). Wired in `update.go`: the previously
  discarded `index.RefreshCache` error is captured into `refreshErr`, and the no-op
  branch selects `upToDateMessage(refreshErr != nil)`.
- **AC2 — online no-op unchanged.** `upToDateMessage(false)` returns the existing
  `atl update: everything up to date` (covered by the `online refresh, nothing
  changed` sub-test).
- **AC3 — pure helper + table test, suite green.** Message selection is the pure
  helper `upToDateMessage(offline bool)` with a table test `TestUpToDateMessage`,
  mirroring the `drainSignal` / `TestAutoDrainNotice` pattern. `go build`, `go vet`,
  and the full `go test ./...` are all green.

## Command output

```text
$ cd cli && go build ./...
(exit 0)

$ go vet ./...
(exit 0)

$ go test ./cmd/atl/commands/ -run TestUpToDateMessage -v
=== RUN   TestUpToDateMessage
=== RUN   TestUpToDateMessage/online_refresh,_nothing_changed
=== RUN   TestUpToDateMessage/offline,_refresh_could_not_run
--- PASS: TestUpToDateMessage (0.00s)
    --- PASS: TestUpToDateMessage/online_refresh,_nothing_changed (0.00s)
    --- PASS: TestUpToDateMessage/offline,_refresh_could_not_run (0.00s)
PASS
ok  	github.com/agentteamland/atl/cli/cmd/atl/commands	0.009s
(exit 0)

$ go test ./...
ok  	github.com/agentteamland/atl/cli/cmd/atl/commands
ok  	github.com/agentteamland/atl/cli/internal/coreassets
ok  	github.com/agentteamland/atl/cli/internal/dispatch
ok  	github.com/agentteamland/atl/cli/internal/docscheck
ok  	github.com/agentteamland/atl/cli/internal/docsstate
ok  	github.com/agentteamland/atl/cli/internal/doctor
ok  	github.com/agentteamland/atl/cli/internal/drain
ok  	github.com/agentteamland/atl/cli/internal/fanout
ok  	github.com/agentteamland/atl/cli/internal/gc
ok  	github.com/agentteamland/atl/cli/internal/generation
ok  	github.com/agentteamland/atl/cli/internal/guard
ok  	github.com/agentteamland/atl/cli/internal/index
ok  	github.com/agentteamland/atl/cli/internal/integrity
ok  	github.com/agentteamland/atl/cli/internal/manifest
ok  	github.com/agentteamland/atl/cli/internal/marker
ok  	github.com/agentteamland/atl/cli/internal/pin
ok  	github.com/agentteamland/atl/cli/internal/promote
ok  	github.com/agentteamland/atl/cli/internal/publish
ok  	github.com/agentteamland/atl/cli/internal/queue
ok  	github.com/agentteamland/atl/cli/internal/rulesscan
ok  	github.com/agentteamland/atl/cli/internal/rulesstate
ok  	github.com/agentteamland/atl/cli/internal/scaffold
ok  	github.com/agentteamland/atl/cli/internal/scope
ok  	github.com/agentteamland/atl/cli/internal/selfupdate
ok  	github.com/agentteamland/atl/cli/internal/semver
ok  	github.com/agentteamland/atl/cli/internal/settings
ok  	github.com/agentteamland/atl/cli/internal/skillcheck
ok  	github.com/agentteamland/atl/cli/internal/skillsstate
ok  	github.com/agentteamland/atl/cli/internal/source
ok  	github.com/agentteamland/atl/cli/internal/teampkg
ok  	github.com/agentteamland/atl/cli/internal/throttle
ok  	github.com/agentteamland/atl/cli/internal/transcript
ok  	github.com/agentteamland/atl/cli/internal/userrules
(exit 0)
```
