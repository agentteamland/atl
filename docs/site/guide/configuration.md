# Configuration

A short guide to `atl`'s own configuration system — what it tunes, how it relates to Claude Code's `settings.json`, and how to inspect or edit it.

## Two configuration surfaces

The atl ecosystem has two distinct configuration files. They do not overlap.

| Surface | Managed by | Default location | Edit command | Schema validation | Project-level support |
| --- | --- | --- | --- | --- | --- |
| **Claude Code settings** | Claude Code itself | `~/.claude/settings.json` | `claude` `/config` slash command | None (free-form) | `.claude/settings.local.json` |
| **atl config** | atl CLI | `~/.atl/config.json` | `atl config edit` | JSON Schema (`atl-config.schema.json`) | `./.atl/config.json` |

**Claude Code's `settings.json`** governs Claude Code's behaviour: which hooks fire, editor shortcuts, IDE integration, theme, the model in use.

**atl's `config.json`** governs atl's own behaviour: auto-update throttling, learning-capture lookback, brainstorm marker cap, locale.

The two intersect at exactly one point: atl's auto-update flow is wired into Claude Code via `SessionStart` and `UserPromptSubmit` hooks, registered by [`atl setup-hooks`](/cli/setup-hooks) inside `~/.claude/settings.json`. The hooks ALWAYS call `atl session-start` / `atl update` if installed; `~/.atl/config.json` then tunes what those invocations actually do (whether to scan markers, throttle window, etc.).

## "Which one am I editing?"

- "Silence atl's upgrade-available banner." → **atl config** — set `autoUpdate.selfCheckEnabled = false`.
- "Disable Claude Code's editor auto-completion." → **Claude Code settings** (atl does not control IDE behaviour).
- "Have atl scan transcripts less aggressively on first run." → **atl config** — lower `learningCapture.firstRunLookbackDays`.
- "Stop hooks from firing every prompt." → **both**: `atl setup-hooks --remove` (removes them from `settings.json`), or keep them registered and set `autoUpdate.promptSubmitEnabled = false` in atl config (the hook still fires but no-ops).
- "Get atl prompts in Turkish." → **atl config** — set `cli.locale = "tr"`. (Currently behaves as `en` until the cli-localization work ships.)

## The 9 user-tunable keys

### `cli.locale`

`enum["en", "tr"]` · default `"en"`

User-facing language for atl prompts. v1 only behaves as English; `"tr"` is reserved for the cli-localization work and currently behaves identically to `"en"`.

### `autoUpdate.sessionStartEnabled`

`boolean` · default `true` (recommended)

When the SessionStart hook fires, run `atl session-start` (cache pull + previous-transcript marker scan + atl-version check). Set to `false` to opt out without removing the hook.

### `autoUpdate.promptSubmitEnabled`

`boolean` · default `true`

When the UserPromptSubmit hook fires, run `atl update` (throttled). Keeps the cache continuously fresh during long sessions; small per-message git-fetch cost.

### `autoUpdate.throttleMinutes`

`integer` in `[1, 1440]` · default `30`

Minimum minutes between consecutive prompt-submit auto-updates. Lower values keep the cache fresher at the cost of more git fetches; the default 30 balances both.

### `autoUpdate.selfCheckEnabled`

`boolean` · default `true`

Have `atl session-start` poll GitHub releases for a newer atl binary. Disable on machines where brew/scoop manage upgrades centrally.

### `autoUpdate.selfCheckHours`

`integer` in `[1, 168]` · default `24`

Minimum hours between self-check polls. Default matches the binary release cadence.

### `learningCapture.autoScanEnabled`

`boolean` · default `true`

Have `atl session-start` scan previous-session transcripts for `<!-- learning -->` markers and report unprocessed ones in the new session's `additionalContext`.

### `learningCapture.firstRunLookbackDays`

`integer` in `[1, 365]` · default `7`

On the first scan for a project (no state file yet), how many days of transcripts to consider. Subsequent scans use the per-project `lastProcessedAt` timestamp from `~/.atl/state/learning-capture-state.json`.

### `brainstorm.markerBulletCap`

`integer` in `[1, 50]` · default `8`

Maximum number of active brainstorm bullets pinned to a scope's `CLAUDE.md` / `README` at once. When the count exceeds this, the oldest bullets are dropped from the marker block (the brainstorm files themselves are never deleted).

## Commands

### Inspect

```bash
atl config show              # effective config (defaults <- global <- project) as JSON
atl config show --table      # key / value / source table
atl config show --global     # raw ~/.atl/config.json contents
atl config show --project    # raw ./.atl/config.json contents
```

### Edit

```bash
atl config init              # first-time welcome + Q&A → write ~/.atl/config.json
atl config edit              # Q&A on existing global config
atl config edit --project    # Q&A on the project's ./.atl/config.json
```

The Q&A walks the 9 keys one screen at a time, with a summary at the end. Boolean questions accept `Y`/`N` shortcuts; integer questions validate the range inline. Press `Esc` to step back to the previous question; `Ctrl+C` opens a discard-confirm prompt.

### Reset

```bash
atl config reset              # interactive: confirm + write defaults to global
atl config reset --yes        # script-safe: skip confirmation
atl config reset --project    # interactive: delete the project overlay
```

## File layout

```
~/.atl/                                # global atl directory
├── config.json                        # global user config
├── state/
│   ├── learning-capture-state.json    # scan progress + processed-marker hashes
│   └── docs-sync-state.json           # docs-sync skill state
├── cache/
│   ├── last-repo-check                # auto-update throttle stamp
│   └── last-self-check                # binary version-check throttle stamp
└── install-marker.json                # first-install onboarding gate

./.atl/                                # project overlay (optional)
└── config.json                        # project-specific overrides
```

The merge order is `defaults <- global <- project`, deep-merged at the field level. A project file containing only `{"schemaVersion": 1, "cli": {"locale": "tr"}}` overrides only `cli.locale`; every other key falls through to global, then to defaults.

## Schema versioning

The on-disk file format is versioned via the required `schemaVersion` field (currently `1`). Future bumps run silent auto-migrations with a backup written to `~/.atl/config.json.bak.v<N>`. The canonical schema lives at [`core/schemas/atl-config.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/atl-config.schema.json) and is auto-applied by `atl migrate` (also auto-triggered from `atl update` / `atl session-start`).

## Decision context

The two-surface split, the keystone test that defines what counts as config (vs rule, vs hardcode), and the migration mechanics were settled in the [`atl-config-system`](https://github.com/agentteamland/workspace/blob/main/.atl/docs/atl-config-system.md) brainstorm. The keystone test in one sentence: *if two users configured this differently, would they still be using the same atl?* — yes for config, no for rule.
