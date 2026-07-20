---
name: rules-distill
description: Mine recurring principles out of the skill + agent corpus and propose them as core rules — the complement of /rule (which authors one rule you already have in mind). Deterministic collect via `atl rules scan`, then an LLM pass that clusters, greps existing rules to avoid duplicates, and proposes candidates. Run when atl signals "a distill is due", or to mine the corpus manually.
argument-hint: "[--all]"
---

# /rules-distill — mine recurring principles into core rules

Corpus-driven rule discovery — the complement of `/rule`. `/rule` authors a single
rule you already have in mind; `/rules-distill` reads the corpus *itself* and
surfaces principles that recur across many skills/agents but aren't yet a rule. It
**proposes, never auto-writes**: a new core rule is structural growth, so the human
confirms it, then it is authored directly into `core/rules/<name>.md` and wired
through the new-rule-shipping-checklist (the coreassets re-sync + the docs page).
(`/rule` is the user-facing sibling — it writes project/global rules, not the
in-monorepo `core/rules/` a distilled principle belongs to.)

Auto-triggered: `atl session-start` signals "a distill is due" once corpus-
affecting commits (`core/` `teams/`) pile up since the last distill, gated by a
~1-day runaway-guard. Non-destructive (propose-only) → fully automatic per the
Lane 3 automation decision.

## When to use it

- When `atl` reports **"a distill is due — run /rules-distill"** at session start.
- Any time you want to mine the corpus on purpose (`--all` forces the whole corpus).

## Procedure

### 1. Collect (deterministic, grounded)
```bash
atl rules scan
```
This prints every normative/imperative line across `core/` + `teams/` skills and
agents (`rules/` is skipped — it's the distill target), each with its `file:line`.
It over-collects on purpose; you do the judging. `--json` gives the machine form.

### 2. Focus on what changed (change-aware)
A full-corpus distill every time is wasteful. Scope to what changed since the last
distill — the cursor is in `~/.atl/rules-distill-state.json` (`lastSHA`):
```bash
git diff --name-only <lastSHA>..HEAD -- core/skills core/rules teams
```
Cross-reference the scan output against the changed files. `--all` forces a
whole-corpus distill.

### 3. Cluster + judge (the LLM half)
Group the collected statements into **recurring principles** — the same discipline
stated in **≥ several** different skills/agents (e.g. "grep before you edit"
appearing across multiple agents). A one-off is not a principle; require genuine
recurrence.
- **Grep the existing rules first** (`core/rules/*.md`): if the principle is
  already a rule, drop it — don't re-propose.
- Ground each candidate with the specific `file:line`s where it recurs.

### 4. Propose — never auto-write
For each surviving candidate, **propose** it via **AskUserQuestion**: "principle X
recurs in A, B, C — promote it to a core rule?" A confirmed candidate is authored
directly into `core/rules/<name>.md` and wired through the new-rule-shipping-checklist
(the coreassets re-sync + the docs page) — **not** through `/rule`, which writes only
project/global rules. Never write a core rule autonomously — that's the reactive-creation
boundary.

### 5. Record the distill
```bash
atl rules scan --record
```
Stamps HEAD as the last-distilled commit, resetting the runaway-guard so
session-start won't re-signal for ~1 day.

### 6. Report
The principles proposed, where each recurs, and each proposal's resolution. Short.

## Notes

- **Deterministic collect, LLM judge.** `atl rules scan` gathers grounded
  candidates (over-collecting); the recurrence + principle-worthiness judgment is
  yours. That's the CLI/Skill boundary.
- **Recurrence is the bar.** A principle earns a rule by recurring across the
  corpus, not by appearing once — require the evidence (the `file:line`s).
- **Propose, don't author.** A new core rule is structural: human-confirmed before it
  is authored into `core/rules/`, never autonomous.
- **Complements `/rule`.** `/rules-distill` mines a corpus-recurring principle into
  `core/rules/`; `/rule` captures a single project/global rule you already have in mind.
- **Monorepo-internal.** Outside the repo's own `core/` + `teams/`, nothing to do.

## Source

- Deterministic half: [`atl rules`](https://github.com/agentteamland/atl/blob/main/docs/site/cli/rules.md)
- CLI: [cli/cmd/atl/commands/rules.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/rules.go)
