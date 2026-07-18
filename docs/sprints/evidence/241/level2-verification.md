# Level-2 verification evidence — #241 `atl update` offline-honest message

Independent Level-2 verification (tester, micro-loop step 4b) for work-item
[#241](https://github.com/agentteamland/atl/issues/241). Intent re-derived fresh from the
issue body (`## Acceptance Criteria`), the `**[Technical Analysis]**` comment, and the
`**[Canonical Brief]**` comment — not from the developer's self-test. Surface: code (Go CLI);
no web/mobile surface applies, so no emulator lease was needed.

## Verdict: PASS — all three acceptance criteria bound

### AC1 — offline message (exact string)

The offline branch returns `atl update: up to date (offline — using cached index)`. Verified
**byte-for-byte against the spec**, not just against the test (the test's `want` and the code's
return could be wrong together; the spec is the source of truth):

```text
SPEC  (issue AC1) : ...6f 20 64 61 74 65 20 28 6f 66 66 6c 69 6e 65 20 | e2 80 94 | 20 75 73 69 6e 67 ...
CODE  (update.go) : ...6f 20 64 61 74 65 20 28 6f 66 66 6c 69 6e 65 20 | e2 80 94 | 20 75 73 69 6e 67 ...
TEST  (update_test): ...6f 20 64 61 74 65 20 28 6f 66 66 6c 69 6e 65 20 | e2 80 94 | 20 75 73 69 6e 67 ...
diff spec vs code  → identical
diff code vs test  → identical
```

`e2 80 94` is UTF-8 for U+2014 EM DASH — the exact character the spec requires (not a hyphen or
en-dash). spec == code == test, byte-identical.

### AC2 — online no-op message unchanged

The online branch returns `atl update: everything up to date`, unchanged from before this change
(the diff only adds the offline branch; the online literal is preserved verbatim).

### AC3 — pure helper + table test, full gate green

- Message selection is the pure helper `upToDateMessage(offline bool) string` (mirrors `drainSignal`).
- `TestUpToDateMessage` is a table test covering both branches (mirrors `TestAutoDrainNotice`).
- The exact brief gate, fresh (`-count=1`, chained): `cd cli && go build ./... && go vet ./... && go test ./...` → **exit 0, all packages `ok`.**

## Anti-theatre — sabotage confirms the test binds behavior

A test that stays green when the code is broken is theatre. Each production branch was temporarily
mutated in `update.go` (then restored via `git checkout`; the working tree was left clean):

| Mutation | Expected | Result |
|---|---|---|
| Corrupt the OFFLINE return string | offline case fails | ✅ `--- FAIL` on `upToDateMessage(true)` |
| Corrupt the ONLINE return string | online case fails | ✅ `--- FAIL` on `upToDateMessage(false)` |
| Invert the `offline` selector (`if offline` → `if !offline`) | both cases fail | ✅ both sub-tests fail (strings swap) |

All three mutations were caught → the table test genuinely binds both branches and the selector.

## Blast radius / regression

- `refreshErr` (previously discarded as `_ =`) is captured and used **only** in the no-op
  `default` branch's message selection (`upToDateMessage(refreshErr != nil)`, update.go:58). It
  does **not** alter the cache/embedded fallback, `updateTeams`, `fanOut`, `reflectCore`, or
  `suggestPublishable` — control flow is unchanged.
- `go vet ./...` is clean → no unused variable; the captured error is genuinely consumed.
- The offline line prints **only** on a true no-op (`upgraded == 0 && refreshed == 0`), matching
  AC1's "and there is nothing new"; the `refreshed > 0` / `upgraded > 0` branches are untouched.
- Out of Scope respected: the refresh logic itself is unchanged (`cli/internal/index/index.go`
  not modified), and no team-update / self-update path was touched.

## Premise chain (offline ⟹ the offline message), by inspection

1. Offline ⟹ `Fetch(url)` errors ⟹ `RefreshCache` returns non-nil (`index.go:181-183`).
2. `refreshErr != nil` ⟹ `upToDateMessage(true)` (`update.go:58`).
3. `upToDateMessage(true)` ⟹ the exact AC1 offline string (verified above).
4. The `default` switch arm is the genuine no-op case (`update.go:57`), so step 2 fires only
   when nothing changed.

Note: the pure helper is unit-tested; the single-line wiring `upToDateMessage(refreshErr != nil)`
is verified by inspection (AC3 asks for exactly a pure helper + table test, which is satisfied).

## Command output (Level-2, verbatim)

```text
$ cd cli && go build ./... ; echo exit=$?
exit=0

$ go vet ./... ; echo exit=$?
exit=0

$ go test ./cmd/atl/commands/ -run TestUpToDateMessage -v -count=1
=== RUN   TestUpToDateMessage
=== RUN   TestUpToDateMessage/online_refresh,_nothing_changed
=== RUN   TestUpToDateMessage/offline,_refresh_could_not_run
--- PASS: TestUpToDateMessage (0.00s)
    --- PASS: TestUpToDateMessage/online_refresh,_nothing_changed (0.00s)
    --- PASS: TestUpToDateMessage/offline,_refresh_could_not_run (0.00s)
PASS
ok  	github.com/agentteamland/atl/cli/cmd/atl/commands

$ ( cd cli && go build ./... && go vet ./... && go test ./... -count=1 ) ; echo chain=$?
# all 35 packages: ok / [no test files], zero FAIL
chain=0

# sabotage (offline string corrupted), restored after:
--- FAIL: TestUpToDateMessage (0.00s)
    update_test.go:39: upToDateMessage(true) = "atl update: SABOTAGED-offline", want "atl update: up to date (offline — using cached index)"

# sabotage (online string corrupted), restored after:
--- FAIL: TestUpToDateMessage (0.00s)
    update_test.go:39: upToDateMessage(false) = "atl update: SABOTAGED-online", want "atl update: everything up to date"

# sabotage (offline selector inverted), restored after:
--- FAIL: TestUpToDateMessage (0.00s)
    update_test.go:39: upToDateMessage(false) = "atl update: up to date (offline — using cached index)", want "atl update: everything up to date"
    update_test.go:39: upToDateMessage(true) = "atl update: everything up to date", want "atl update: up to date (offline — using cached index)"
```
