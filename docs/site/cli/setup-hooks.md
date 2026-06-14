# `atl setup-hooks`

Configure Claude Code hooks so that teams, global repos (core, brainstorm, rule, team-manager), and the `atl` binary stay auto-checked for updates — AND so that inline learning markers dropped during a conversation get captured at the *next* session start.

All with zero manual action from you.

Requires `atl` ≥ 1.1.0.

## Usage

```bash
atl setup-hooks                    # install with 30m throttle (recommended default)
atl setup-hooks --throttle=5m      # more aggressive update check (every 5 minutes of activity)
atl setup-hooks --throttle=1h      # less aggressive update check
atl setup-hooks --remove           # uninstall the atl hooks
```

On first `atl install`, you're asked whether to enable this automatically. Saying yes runs `atl setup-hooks` for you. If you declined then, or are opting in later, run it manually.

## What it does

Writes two entries into `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [
          { "type": "command", "command": "atl session-start --silent-if-clean" }
      ]}
    ],
    "UserPromptSubmit": [
      { "hooks": [
          { "type": "command", "command": "atl update --silent-if-clean --throttle=30m" }
      ]}
    ]
  }
}
```

Pre-`v1.1.0` installs that registered four entries (`SessionStart` + `UserPromptSubmit` + `SessionEnd` + `PreCompact`) get the legacy `SessionEnd` and `PreCompact` atl entries silently dropped on the next `atl setup-hooks` run. Their commands kept firing under the old setup but their output never reached Claude (see [the History note below](#history-from-four-hooks-to-two)), so removing them is safe and lossless.

Claude Code runs these automatically:

### `SessionStart` — composite boot-time wrapper (new shape in v1.1.0)

Runs once when you open a new Claude Code session. The single command `atl session-start` performs three boot-time tasks in order:

1. **Auto-update**: `atl update --silent-if-clean` — pulls every cached repo under `~/.claude/repos/agentteamland/`. Updates surface as a one-line `🔄 <team> <oldVer> → <newVer>` block in Claude's `additionalContext`.
2. **Previous-transcript marker scan**: `atl learning-capture --previous-transcripts` — scans every transcript file for the current project that was modified since the last successful `/save-learnings` run (state tracked in `~/.claude/state/learning-capture-state.json`, with a 7-day cap on first use). When markers are found, prints a single `🧠 learning-capture: N unprocessed markers across M transcripts → /save-learnings --from-markers --transcripts ...` block.
3. **atl version check**: hits the GitHub Releases API at most once per 24h. When a newer atl is available, surfaces a `⬆ atl X.Y.Z → X.Y.Z+1 available` line.

When nothing changed and no markers exist, output is empty (zero token cost).

### `UserPromptSubmit` — throttled per-message refresh

Runs before every message you send to Claude. Throttled to once per `<duration>` (30m default) so the per-message cost is a single file-stat call (~1ms). Slow path (actual git fetch + pull) runs at most twice an hour.

When something changed, Claude sees the same `🔄 <team> ...` line in its context and can mention it briefly. When nothing changed, you see nothing.

## How marker-driven learning processing reaches Claude

The flow is end-to-end automatic except for one Claude turn:

```
[you close session N]   markers sit in transcript file
        ↓
[you open session N+1]
        ↓
SessionStart hook fires → atl session-start --silent-if-clean
        ↓
   step 2: atl learning-capture --previous-transcripts
        → reads ~/.claude/state/learning-capture-state.json
        → enumerates project transcripts modified after the cutoff
        → grep-scans for <!-- learning --> blocks (assistant-role only)
        → prints `🧠 learning-capture: N markers ... → Run: /save-learnings ...`
        ↓
Claude Code injects stdout into Claude's first additionalContext
        ↓
[your first turn in session N+1]
        ↓
Claude sees the report, invokes /save-learnings --from-markers --transcripts <paths>
        ↓
/save-learnings persists markers to journal + wiki + agent children + skill learnings
        + advances the state file's lastProcessedAt → next session sees 0 markers
```

See [`atl learning-capture`](/cli/learning-capture) for the marker format and the noise-filter details, and the [learning-capture rule](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) / [docs-sync rule](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md) in core for the behavior spec.

## Why two hooks (and not four)

| Hook | Answers |
|---|---|
| `SessionStart` (via `atl session-start`) | "I'm just opening Claude Code fresh, what changed upstream + what learnings did the previous session leave behind + is there a newer atl?" |
| `UserPromptSubmit` (via `atl update`) | "I've been in this session for hours, is there a newer release?" |

Two hooks cover the whole guarantee. The pre-v1.1.0 4-hook design separated update and learning-capture across `SessionStart` / `SessionEnd` / `PreCompact`, but `SessionEnd` and `PreCompact` never delivered hook stdout to Claude's `additionalContext` — see [the History note below](#history-from-four-hooks-to-two). Folding everything into `SessionStart` via a composite wrapper preserves all the behavior and ensures the output actually reaches Claude.

## Idempotency — safe to re-run

The merge preserves any other hooks you have. Re-running `atl setup-hooks` only touches atl-owned entries (any command prefixed with `atl `). All other hooks, permissions, model settings, and `extraKnownMarketplaces` in `settings.json` are left untouched.

`--remove` similarly only strips atl-owned hook entries, leaving everything else in place. Both commands also drop legacy `SessionEnd` / `PreCompact` atl entries from prior installs.

## When you should run this

- **Always recommended** for interactive Claude Code users.
- **Not recommended** for CI / scripted `atl install` (the hooks would fire in CI unnecessarily). The first-install opt-in prompt already skips in non-interactive contexts.

## What exactly gets checked

### Each `atl session-start` run

1. **Auto-update** (step 1): iterates `~/.claude/repos/agentteamland/*/`, parallel `git fetch origin main`, fast-forward pull when behind, parses `team.json` before+after to surface `<oldVer> → <newVer>`.
2. **Marker scan** (step 2): reads `~/.claude/state/learning-capture-state.json` for the per-project `lastProcessedAt` cutoff (or last 7 days on first run), enumerates transcripts modified after that, scans for `<!-- learning -->` blocks emitted by assistant turns only (the v1.1.1 noise filter rejects prose mentions, tool inputs/outputs, summary events, and topics that fail the kebab-case regex). No network calls.
3. **atl self-check** (step 3): one HTTPS GET to `api.github.com/repos/agentteamland/cli/releases/latest`, throttled to once per 24h. Surfaces a `⬆` line when a newer release exists.

Slow path: ~2-3s for typical setups (5-10 cached repos). Fast path (after the throttle window has elapsed and no transcripts changed): ~1ms.

### Each `atl update --silent-if-clean --throttle=30m` run

Same as `atl session-start`'s step 1, but skipped if the last successful run was less than 30 minutes ago (configurable via `--throttle`).

## Offline behavior

If you're offline, each repo's `git fetch` fails silently, that repo gets a `⚠` warning line (if not silenced), and the rest continue. Marker scanning doesn't need the network at all — it just reads local files. Claude Code's prompt still proceeds normally; hooks failing don't block your work.

## History — from four hooks to two

`atl v0.2.0` (2026-04-24) shipped four hooks: `SessionStart` + `UserPromptSubmit` for auto-update, `SessionEnd` + `PreCompact` for learning-capture. The capture half **looked like it worked** (binary ran, scanned markers, printed reports) but per Claude Code v2.1.x, `SessionEnd` and `PreCompact` hook stdout is NOT delivered to Claude's `additionalContext`. 324 markers across 9 maintainer-workspace sessions in the month after v0.2.0 produced **zero** auto-processing — every actual `/save-learnings` run came from manual user invocation.

`atl v1.1.0` (2026-05-02) restructured the flow:

- New `atl session-start` composite wrapper combines update + previous-transcript marker scan + atl self-check.
- New `atl learning-capture --previous-transcripts` mode reads transcripts modified since the state-file cutoff (instead of needing the *current* session's `SessionEnd` to fire).
- `atl setup-hooks` v1.1.0 silently drops legacy `SessionEnd` / `PreCompact` atl entries from prior installs. Other people's hooks under those events are untouched.

The marker protocol itself is unchanged — the v0.2.0 marker format still works. Only the trigger path moved.

`atl v1.1.1` (2026-05-02) added a noise filter to the marker scanner: assistant-role only + kebab-case topic regex. Closes a SessionStart over-report bug where any session that *discussed* the marker format inflated the next session's count by 10-25× (149 raw substring hits → 16 real markers across 5 workspace transcripts in the validation sweep).

## Related

- [`atl update`](/cli/update) — manual update (what the auto-update hook calls silently)
- [`atl learning-capture`](/cli/learning-capture) — manual scanner (what `atl session-start` calls silently)
- [`atl install`](/cli/install) — first install (includes opt-in prompt)
- [Install the CLI](/guide/install) — getting atl on your machine
