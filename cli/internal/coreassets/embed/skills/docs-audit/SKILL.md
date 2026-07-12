---
name: docs-audit
description: Sweep the whole docs site for drift the change-time checks missed — deterministic findings via `atl docs check`, then a grep-grounded + adversarially-verified semantic pass over each section, written up as draft fixes in a PR. Run when atl signals "a full audit is due", or to audit the docs site manually.
argument-hint: "[--external]"
---

# /docs-audit — the docs-correctness backstop

The full-site backstop of docs-sync v2. Change-time (`/create-pr`'s docs-impact
pass + the CI gate) catches drift as it forms; this skill is the net for what they
miss — external-world rot, a skipped docs-pass, accumulated drift. It is **both**
manually callable and auto-triggered: `atl session-start` signals "a full audit is
due" once doc-affecting commits have piled up since the last audit (gated by a
~1-day runaway-guard), and you run it then.

Same discipline as `/publish`: the deterministic half is the CLI's; the judgment
and prose are this skill's. Findings are **grep-grounded** (no claim without a
verbatim source quote) and **adversarially verified** (each semantic finding is
challenged before it's kept) — that's how the ~40% multi-agent-audit hallucination
rate is held down.

## When to use it

- When `atl` reports **"a full audit is due — run /docs-audit"** at session start.
- Any time you want to sweep the whole docs site on purpose.

## Procedure

### 1. Pre-flight
Run from the repo that holds the docs site. If `atl docs check` prints "no docs
site here", stop. Otherwise note its failure + warning counts — they seed step 2.

### 2. Deterministic pass (free, zero-FP)
```bash
atl docs check
```
Fix every **FAIL** (a missing page, an absent TR mirror, a stale install
instruction). These are mechanical — just apply them. Warnings are advisory: judge
each (an illustrative `{placeholder}` link is fine; a genuinely broken link is not).

If invoked with **`--external`**, also run `atl docs check --external` to include
external-world link-rot (HTTP dead-link probing — slow + networked, so it's opt-in);
this is the "external-world rot" the skill's intro promises to cover.

### 3. Semantic sweep (the LLM half)
Split the site into sections (`cli/`, `guide/`, `skills/`, `teams/`, `authoring/`,
`contributing/`, `reference/`) and work them in parallel where you can:
- **Find** — read each page against the code / `SKILL.md` / `team.json` it
  describes. A finding is "this prose says X; the code does Y." **Quote the source
  verbatim (grep) before recording a finding** — if you can't ground it, drop it.
- **Verify (adversarial)** — for each candidate, try to *refute* it: is the prose
  actually right and you misread? Is it deliberate historical contrast ("v1 did
  X")? Default to dropping unless it survives the challenge.
- Apply the surviving fixes to the EN page and regenerate the TR mirror from it.

### 4. Open a PR (autonomous)
Stage every fix and open a PR (via `/create-pr`, or directly). Title
`docs: audit sweep`; body lists what was fixed per section and what was
deliberately left (historical contrast, illustrative links). This is the
autonomous draft — the maintainer reviews and merges; you don't ask first.

### 5. Record the audit
Once the sweep is done (clean, or fixes staged), stamp the cursor so the backstop
knows it ran:
```bash
atl docs check --record-audit
```
This resets the runaway-guard, so session-start won't re-signal for ~1 day.

### 6. Report
Per section: findings kept / refuted, files changed, the PR link. Keep it short.

## Notes

- **Deterministic-first.** Never hand-audit what `atl docs check` already proves —
  run it first, fix its FAILs, then spend LLM effort only on semantic
  prose-vs-code drift.
- **Grep-grounded + adversarial.** The two guards against false positives. A
  finding with no verbatim source quote, or that doesn't survive a refute attempt,
  is dropped — the cost of "fixing" already-correct prose outweighs catching one
  more real drift.
- **EN canonical, TR derived.** Fix the EN page, regenerate the TR mirror from it;
  never let them drift structurally.
- **Monorepo-internal.** The target is the atl docs site; outside a repo with a
  docs site this skill has nothing to do.

## Source

- CLI: [cli/cmd/atl/commands/docs.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/docs.go)
- The deterministic half: [`atl docs`](https://github.com/agentteamland/atl/blob/main/docs/site/cli/docs.md)
