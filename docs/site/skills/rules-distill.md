# `/rules-distill`

Mine recurring principles out of the skill + agent corpus and propose them as core rules — the **complement of [`/rule`](/skills/rule)**. `/rule` authors a single rule you already have in mind; `/rules-distill` reads the corpus *itself* and surfaces principles that recur across many skills/agents but aren't yet a rule.

It is **both** manually callable and auto-triggered. [`atl session-start`](/cli/setup-hooks) signals **"a distill is due"** once corpus-affecting commits (`core/` `teams/`) pile up since the last distill, gated by a ~1-day runaway-guard. Non-destructive (propose-only), so it auto-signals per the Lane 3 automation decision.

## When to use it

- When `atl` reports **"a distill is due — run /rules-distill"** at session start.
- Any time you want to mine the corpus on purpose (`--all` forces the whole corpus).

## How it works

### Deterministic collect

The skill runs [`atl rules scan`](/cli/rules), which prints every normative/imperative line across the skill + agent corpus (`always`, `never`, `must`, `don't`, `avoid`, the grep-before-edit idiom) with its `file:line`. It over-collects on purpose — the collect only gathers grounded candidates; the skill judges.

### Change-aware

A full-corpus distill every time is wasteful, so it scopes to what changed since the last distill (the cursor lives in `~/.atl/rules-distill-state.json`). `--all` forces the whole corpus.

### Cluster + judge

The LLM groups the collected statements into **recurring principles** — the same discipline stated across several skills/agents (e.g. "grep before you edit" appearing in multiple agents). A one-off is not a principle; genuine recurrence is the bar. It **greps the existing rules first** so it never re-proposes what's already a rule, and grounds each candidate with the `file:line`s where it recurs.

### Proposes — never auto-writes

Each surviving candidate is **proposed** via AskUserQuestion ("principle X recurs in A, B, C — promote it to a core rule?"). A new core rule is structural growth — the human confirms it, and a confirmed candidate is authored through [`/rule`](/skills/rule), which carries the new-rule-shipping-checklist. `/rules-distill` never writes a core rule autonomously.

### Records the distill

On completion the skill stamps the cursor (`atl rules scan --record`), which resets the runaway-guard so session-start won't re-signal for ~1 day.

## The CLI / Skill split

`/rules-distill` is the LLM half of rule discovery. The deterministic half — collecting the grounded candidate statements — is [`atl rules scan`](/cli/rules). The skill never re-derives the collect; it spends LLM effort only on clustering, recurrence-judgment, and the propose step. distill says *which* rule the corpus is asking for; `/rule` says *how to ship it*.

## Related

- [`atl rules`](/cli/rules) — the deterministic collect this skill builds on.
- [`/rule`](/skills/rule) — authors a confirmed candidate (rules-distill discovers, /rule ships).
- [`/skill-stocktake`](/skills/skill-stocktake) — the sibling corpus-hygiene backstop (skill quality; this one is rule discovery).

## Source

- Spec: [core/skills/rules-distill/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/rules-distill/SKILL.md)
