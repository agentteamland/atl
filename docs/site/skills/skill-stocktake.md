# `/skill-stocktake`

Sweep the skill + agent corpus for content-quality issues a deterministic gate can't judge — the **judgment half of skill-quality** (the docs-sync v2 pattern). [`atl skills check`](/cli/skills) proves the corpus is structurally sound; `/skill-stocktake` judges **obedience** (does a skill follow its own documented flow?) and **redundancy** (do two skills do the same job or contradict each other?).

It is **both** manually callable and auto-triggered. [`atl session-start`](/cli/setup-hooks) signals **"a stocktake is due"** once asset-affecting commits (`core/` `teams/`) have piled up since the last recorded stocktake, gated by a ~1-day runaway-guard — you run the skill then.

## When to use it

- When `atl` reports **"a stocktake is due — run /skill-stocktake"** at session start.
- Any time you want to sweep the skill corpus on purpose (`--all` forces the whole corpus).

## How it works

### Deterministic first

The skill runs [`atl skills check`](/cli/skills) and fixes every **FAIL** (a missing frontmatter block, a team.json↔disk mismatch, a child with no summary) — mechanical, zero-false-positive. It never hand-judges what the CLI already proves.

### Change-aware

A full-corpus sweep every time is wasteful, so it scopes to the skills/agents touched since the last stocktake (the cursor lives in `~/.atl/skill-stocktake-state.json`). Nothing changed and no `--all`? There's nothing to sweep.

### Semantic, grep-grounded, adversarial

Then two passes, each guarded against the ~40% multi-agent-audit hallucination rate:

- **Obedience** — read each in-scope skill's `SKILL.md` against itself: a step referencing a flag it never defines, a contradiction between description and body, a dangling "see step N", a promised output the flow never produces.
- **Redundancy** — compare each in-scope skill against the rest: two skills claiming the same trigger, doing the same job, or giving contradictory instructions.

Every finding is **grep-grounded** (quoted verbatim, or dropped) and **adversarially verified** (challenged, and dropped unless it survives).

### Proposes — never rewrites silently

Surviving findings are **proposed** via AskUserQuestion (fix a broken flow / merge two overlapping skills / reconcile a contradiction). A skill rewrite touches identity, so the human confirms it — this is the one place `/skill-stocktake` is stricter than [`/docs-audit`](/skills/docs-audit), whose prose fixes are safe to auto-apply.

### Records the stocktake

On completion the skill stamps the cursor (`atl skills check --record-stocktake`), which resets the runaway-guard so session-start won't re-signal for ~1 day.

## The CLI / Skill split

`/skill-stocktake` is the LLM half of skill-quality. The deterministic half — frontmatter validity, team.json consistency, agent-KB children — is [`atl skills check`](/cli/skills), which also runs as a CI gate on every PR. The skill never re-derives what the CLI proves; it spends LLM effort only on obedience + redundancy, grep-grounded and adversarially verified.

## Related

- [`atl skills`](/cli/skills) — the deterministic half this skill builds on.
- [`/docs-audit`](/skills/docs-audit) — the same backstop shape for the docs site (its prose fixes are autonomous; this skill's are proposed).

## Source

- Spec: [core/skills/skill-stocktake/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/skill-stocktake/SKILL.md)
