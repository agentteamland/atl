# `atl setup-hooks`

Install the ATL automation hooks into Claude Code so the platform stays self-maintaining — with zero manual action from you.

In v2 automation is **mandatory, not opt-in**: `atl install` binds these hooks for you. You only run `atl setup-hooks` directly if you want to (re)install them on their own or change the throttle interval.

## Usage

```bash
atl setup-hooks                    # install with the default 10m tick throttle
atl setup-hooks --throttle=5m      # more aggressive in-session tick (every 5 minutes of activity)
atl setup-hooks --throttle=1h      # less aggressive
```

`--throttle` only affects the `atl tick` `UserPromptSubmit` hook; `atl retrieve` (the per-prompt retrieval hook) is unthrottled and cheap, and `SessionStart` always runs in full.

## What it does

Writes five entries into `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [
          { "type": "command", "command": "atl session-start" }
      ]}
    ],
    "UserPromptSubmit": [
      { "hooks": [
          { "type": "command", "command": "atl tick --throttle=10m" }
      ]},
      { "hooks": [
          { "type": "command", "command": "atl retrieve" }
      ]}
    ],
    "PreToolUse": [
      { "matcher": "Bash|Edit|Write",
        "hooks": [
          { "type": "command", "command": "atl guard" }
      ]}
    ],
    "Stop": [
      { "hooks": [
          { "type": "command", "command": "atl retrieve turn-end" }
      ]}
    ]
  }
}
```

Claude Code runs these automatically:

### `SessionStart` — boot-time maintenance

Runs once when you open a new Claude Code session. `atl session-start` performs the boot-time work in order:

1. **Reflect platform core** — refreshes the in-binary rules + skills into the global `~/.claude` layer so it stays in lockstep with the installed `atl` version, and reflects any rules you authored via `/rule` into the Claude load surface (`~/.claude/rules/`).
2. **Drain the previous session** — discovers this project's transcripts modified since the last drain, extracts the assistant text, and transfers any inline `<!-- learning: ... -->` markers into the durable queue at `~/.atl/queue.db` (exactly once). The same pass runs the **capture watchdog**: if the newest session ended in a substantive marker-less stretch, it prints the one-line review nudge (see [`atl tick`](/cli/tick), step 5) — the catch-all for a session that closed before its stretch was noticed.
3. **Doctor self-check** — runs the queue-health + asset-integrity checks and surfaces (or auto-heals) anything not OK.
4. **Signal pending learnings** — if the queue holds unprocessed learnings, prints a one-line `atl: N learning(s) pending — auto-drain them now in a background subagent (per the learning-capture rule)`; Claude then spawns one background `/drain` subagent that folds them in automatically.
5. **Auto-update, throttled (background)** — at most once a day, checks for a newer `atl` release and, if there is one, spawns a detached [`atl upgrade`](/cli/upgrade); and, once a day per project, spawns a detached [`atl update`](/cli/update) to pull newer *published* team versions. Both run in the background so they never block boot, and the next session runs on the fresh binary / teams. Set `ATL_NO_SELF_UPDATE` or `ATL_NO_TEAM_UPDATE` to opt out.
6. **Retrieval index refresh (background)** — when this project's knowledge corpus (wiki + journal, plus a delivery project's `docs/`) changed since the last build, spawns a detached `atl retrieve index` so the per-prompt retrieval hook has a fresh index. Throttled, skipped inside git worktrees, and disabled with `ATL_NO_RETRIEVE_INDEX`.
7. **Retrieval translation credential notice** — when this project has a retrieval index but the translator has no credential, prints what that costs you: your knowledge base is English, so a non-English prompt searches on one arm instead of two. Two places do **not** deliver the credential to a hook, and the notice names them first — `~/.claude/settings.json`'s `env` block (Claude Code holds the variable in its own process and strips it from every child it spawns) and `~/.zshrc` (zsh reads it for interactive shells only). What works: `~/.atl/claude-token`, a plain file holding just the token, or `CLAUDE_CODE_OAUTH_TOKEN` / `ANTHROPIC_API_KEY` exported from `~/.zshenv`, which zsh reads on every invocation. If a credential *is* configured but its last three translation attempts failed to run, a different notice reports the likely expiry and names the source you actually configured. Information, never a gate — retrieval works without it — and it reprints every session because a setup step that announces itself once is one that gets missed. See the [knowledge-system guide](/guide/knowledge-system).

`SessionStart` is the one Claude Code event that delivers hook stdout to Claude's context, so whatever `session-start` prints reaches Claude — including a gc orphan-awareness line (`atl: N orphaned file(s) beside installed units — run atl gc to review`) when unowned files sit beside installed units. It stays quiet when there's nothing worth surfacing, so a boring boot costs nothing.

### `UserPromptSubmit` — throttled in-session tick

Runs before every message you send to Claude. `atl tick --throttle=10m` does the cheap work on every prompt and the heavier work at most once per throttle window:

- **Fan-out** (every call, generation-guarded) — when the global layer changed since this project last fanned out, it pulls the updated assets down. Otherwise it's a single small file read, cheap enough to ride every prompt.
- **Drain + doctor** (throttled) — re-scans this project's transcripts for new markers, runs the **capture watchdog** on the live session (a one-line nudge, once per marker-less dry stretch — see [`atl tick`](/cli/tick), step 5), and runs the doctor self-check. Skipped if the last tick was within the throttle window, so the per-prompt cost stays a single file-stat call.
- **Promote gains** (throttled) — lifts this project's accumulated gains to the global layer (additive, conflict-archived, pinnable), so they circulate without waiting for a manual `atl promote`.

When something surfaces, Claude sees the corresponding line in its context and can mention it. When nothing changed, you see nothing.

### `UserPromptSubmit` — per-prompt knowledge retrieval

Alongside the tick, a second `UserPromptSubmit` entry runs `atl retrieve`: it ranks this project's knowledge pages (wiki + journal) against each prompt — BM25 fused with a local semantic embedder — and surfaces the top matches as context, so Claude consults the most relevant pages before answering. A delivery project (one with a `.delivery/config.json`) also indexes its in-repo `docs/` tree alongside wiki + journal. It's fail-open: any error prints nothing and never blocks the prompt. See the [knowledge-system guide](/guide/knowledge-system).

### `Stop` — record that the turn completed

Runs when a turn ends. `atl retrieve turn-end` writes one line to the retrieval fire log and
prints nothing, so it never reaches Claude's context and never affects a reply.

It exists to make one question answerable: **does anyone act on what retrieval surfaces?**
Without a turn marker the fire log records what was offered and nothing about what happened
next, so "is this channel worth improving?" can only be answered by a forensic pass over the
whole transcript. `atl retrieve stats` reads this to report the consult rate.

### `PreToolUse` — the enforcement guard

Runs before every `Bash`, `Edit`, and `Write` tool call (scoped by the hook's `matcher`). `atl guard` applies ATL's discipline as **deterministic enforcement** rather than prose a model can skip — in two layers, split by reversibility:

- **Catastrophe layer (blocks)** — an irreversible Bash operation is denied outright, with the reason shown to Claude so it can take a safe path instead. The fixed set: `git push --force` (use `--force-with-lease`), `git reset --hard`, `git clean -f`, discarding the working tree (`git checkout .` / `git checkout -- <path>` / `git checkout -f` / `git restore <path>` — a branch switch like `git checkout main` stays permitted, and so does `git restore --staged`, which only unstages), destructive SQL (`DROP TABLE` / `DROP DATABASE` / `TRUNCATE`), and `--no-verify` (which bypasses the commit/push gate). `rm -rf /` and `rm -rf ~` are deliberately left out — Claude Code already blocks those itself, even in bypass mode.
- **Quality layer (never blocks)** — three nudges, all injected as context with no permission decision, so neither interrupts the flow nor changes what you're prompted to approve. The first time you edit an *existing* file in a session, a grep-before-edit reminder (the second edit of the same file is silent; creating a new file is exempt). And when a Bash command reads `$?` after a **pipeline**, a reminder that the status belongs to the last command — the `head`/`tail`/`grep` you piped into — so a failing command whose output you filtered reports success. Set `pipefail` and the nudge stays quiet, since then the pipeline's status is the one you meant. And when a commit reads its message from a **file that was last written more than ten minutes ago**, a reminder that the file may not be this turn's: if the heredoc meant to write it never ran — most often because this guard refused the compound command it was part of — the commit succeeds with the wrong message and the right files, and nothing reports it. A message written moments before the commit stays silent, which is the ordinary case.

The guard fires in every permission mode — including `bypassPermissions` — because a PreToolUse hook is an enforcement layer above the permission prompts. Like the other hooks it never fails your work: on malformed input or any internal error it stays silent and the tool call proceeds.

## How marker-driven learning processing reaches Claude

Capture is automatic; only the *fold-in* needs one Claude turn (the LLM work the CLI can't do itself — the CLI/Skill boundary):

```
[you close session N]   inline learning markers sit in the transcript file
        ↓
[you open session N+1]
        ↓
SessionStart hook fires → atl session-start
        → drains the previous session's transcripts into ~/.atl/queue.db (each marker enqueued exactly once)
        → if the queue holds pending learnings, prints `atl: N learning(s) pending — auto-drain them now in a background subagent (per the learning-capture rule)`
        ↓
Claude Code injects stdout into Claude's first additionalContext
        ↓
[your first turn in session N+1]
        ↓
Claude sees the signal, spawns one background /drain subagent (single-in-flight)
        ↓
/drain folds each queued learning into wiki / journal / agent KB, then acks (deletes) it
```

Within a single session, `atl tick` keeps the queue current between prompts, so the count surfaced at the next `session-start` (or the next `atl learnings`) is always up to date.

See [`atl learnings`](/cli/learnings) for the marker format and the queue's status/peek/ack surface, [`atl tick`](/cli/tick) for the in-session cadence, and the [`/drain` skill](/skills/drain) for how queued learnings get folded into the knowledge base.

## Why these hooks

| Hook | Answers |
|---|---|
| `SessionStart` (via `atl session-start`) | "I'm opening Claude Code fresh — drain what the last session left behind, heal anything broken, and tell me if there are learnings to fold in." |
| `UserPromptSubmit` (via `atl tick`) | "I've been in this session a while — keep the queue current, pull any global-layer changes, and circulate gains, cheaply, between prompts." |
| `UserPromptSubmit` (via `atl retrieve`) | "I just sent a prompt — surface the project knowledge pages most relevant to it, so Claude consults them before answering." |
| `PreToolUse` (via `atl guard`) | "I'm about to run a tool — block the irreversible mistakes outright, and remind me to grep before I first edit a file." |

The first two implement the three-speed in-session cadence (an every-prompt fan-out, a throttled tick, and the boot-time drain); the second `UserPromptSubmit` entry adds per-prompt knowledge retrieval; `PreToolUse` adds the enforcement layer that makes ATL's discipline bite at the moment of action.

## Idempotency — safe to re-run

The merge preserves any other hooks you have. Re-running `atl setup-hooks` (or `atl install`, which binds the same hooks) only replaces atl-owned entries — any command prefixed with `atl `. All other hooks, permissions, model settings, and `extraKnownMarketplaces` in `settings.json` are left untouched. The write is atomic.

## When you should run this

- **Always** for interactive Claude Code users — `atl install` already does it, but you can re-run it to change the throttle.
- **Not recommended** for CI / scripted use (the hooks would fire in CI unnecessarily).

## Offline behavior

The core cadence needs no network — draining transcripts, the bbolt queue, the doctor checks, and the per-prompt fan-out all work fully offline. The only network passes are `session-start`'s throttled, detached auto-updates (the binary self-update and the team update), which are best-effort: they fail quietly offline and, being detached, never block boot. A hook must never block your work, so `session-start` and `tick` never fail the session; if something goes wrong they surface a line (or stay quiet) and the prompt proceeds normally.

## Related

- [`atl tick`](/cli/tick) — the in-session maintenance tick (what the `UserPromptSubmit` hook calls)
- [`atl learnings`](/cli/learnings) — inspect the durable learning queue (status / peek / ack)
- [`atl doctor`](/cli/doctor) — the self-check the hooks run on each pass
- [`atl install`](/cli/install) — first install (binds these hooks for you)
- [Install the CLI](/guide/install) — getting atl on your machine
