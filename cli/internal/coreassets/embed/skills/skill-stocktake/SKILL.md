---
name: skill-stocktake
description: Sweep the skill + agent corpus for content-quality issues the deterministic gate can't judge — does each skill obey its own documented flow, and do any two skills overlap or contradict? Grep-grounded + adversarially verified, focused on what changed since the last stocktake. Run when atl signals "a stocktake is due", or to sweep the corpus manually.
argument-hint: "[--all]"
---

# /skill-stocktake — the skill-quality backstop

The judgment half of skill-quality (the docs-sync v2 pattern). The deterministic
gate (`atl skills check`) proves the corpus is structurally sound — frontmatter,
team.json consistency, agent-KB children. This skill judges what a gate can't:
**obedience** (does a skill follow its own documented flow?) and **redundancy**
(do two skills do the same job or contradict each other?).

It is auto-triggered: `atl session-start` signals "a stocktake is due" once
asset-affecting commits pile up since the last stocktake (gated by a ~1-day
runaway-guard), and you run it then. It is also manually callable.

Same discipline as `/docs-audit`: deterministic first, then a **grep-grounded**
(no claim without a verbatim quote) + **adversarially-verified** (each finding
challenged before it's kept) LLM pass. But unlike `/docs-audit`, it **proposes**
— it never rewrites a skill silently. A skill rewrite is structural growth; the
human confirms it.

## When to use it

- When `atl` reports **"a stocktake is due — run /skill-stocktake"** at session start.
- Any time you want to sweep the skill corpus on purpose.

## Procedure

### 1. Pre-flight (deterministic, free, zero-FP)
Run from the monorepo. If `atl skills check` prints "no core/ here", stop. Otherwise:
```bash
atl skills check
```
Fix every **FAIL** first (a missing frontmatter block, a team.json↔disk mismatch,
a child with no summary) — these are mechanical. Only then spend LLM effort on the
semantic passes below.

### 2. Focus on what changed (change-aware)
A full-corpus sweep every time is wasteful. Scope to the skills/agents touched
since the last stocktake — the cursor is in `~/.atl/skill-stocktake-state.json`
(`lastSHA`):
```bash
git diff --name-only <lastSHA>..HEAD -- core/skills core/rules teams
```
If nothing changed and you were not asked for `--all`, there's nothing to sweep —
record and stop. `--all` forces the whole corpus.

### 3. Obedience pass (per in-scope skill)
Read each skill's `SKILL.md` against itself:
- Does the documented procedure hang together — no step referencing a flag or
  command it never defines, no contradiction between the `description` and the
  body, no dangling "see step N" that doesn't exist, no promised output the flow
  never produces?
- **Grep-ground every finding** — quote the offending lines verbatim. If you can't
  ground it, drop it.
- **Verify adversarially** — try to refute each: is the skill actually right and
  you misread? Default to dropping unless the finding survives the challenge.

### 4. Redundancy pass (across skills)
Compare each in-scope skill against the rest of the corpus:
- Do two skills claim the same trigger, or do the same job (overlap)?
- Do two skills give contradictory instructions for the same situation?
- Ground each finding with the specific skills + lines; verify adversarially.

### 5. Propose — never rewrite silently
Collect the surviving findings and **propose** each via **AskUserQuestion** (fix a
skill's broken flow / merge two overlapping skills / reconcile a contradiction). A
skill rewrite touches identity — the human confirms it. Do **not** edit a
`SKILL.md` autonomously (this is the one place /skill-stocktake is stricter than
/docs-audit, whose prose fixes are safe to auto-apply).

### 6. Record the stocktake
Once the sweep is done (clean, or proposals resolved):
```bash
atl skills check --record-stocktake
```
This stamps HEAD as the last-stocktaken commit, resetting the runaway-guard so
session-start won't re-signal for ~1 day.

### 7. Report
Per skill: findings kept / refuted, and each proposal's resolution. Keep it short.

## Notes

- **Deterministic-first.** Never hand-judge what `atl skills check` already proves —
  run it, fix its FAILs, then spend LLM effort only on obedience + redundancy.
- **Grep-grounded + adversarial.** The two guards against false positives — the same
  ~40%-multi-agent-hallucination lesson as `/docs-audit`.
- **Propose, don't rewrite.** Obedience/redundancy fixes touch skill identity, so
  they're human-confirmed, never autonomous.
- **Monorepo-internal.** The target is the repo's own `core/` + `teams/`; outside
  the monorepo this skill has nothing to do.

## Source

- Deterministic half: [`atl skills`](https://github.com/agentteamland/atl/blob/main/docs/site/cli/skills.md)
- CLI: [cli/cmd/atl/commands/skills.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/skills.go)
