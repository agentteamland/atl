# `atl rules`

The **deterministic collect** half of rules-distill: gather the normative / imperative statements across the skill + agent corpus so the [`/rules-distill`](/skills/rules-distill) skill can judge which recurring principles deserve to become a core rule. The judgment — *which* candidate is a real principle — is the skill's; this is the CLI/Skill boundary.

This is a **maintainer-side** surface that runs against the monorepo's `core/` + `teams/`. Outside the monorepo it does nothing and exits 0.

## Usage

```bash
atl rules scan            # print normative statements across the skill corpus
atl rules scan --json     # the same, machine-readable (file, line, text)
atl rules scan --record   # stamp HEAD as the last distill (after a /rules-distill sweep)
```

## `atl rules scan`

Walks the skill + agent markdown in `core/` and `teams/` and prints every line carrying a **strong normative/imperative trigger** (`always`, `never`, `must`, `don't`, `avoid`, the grep-before-edit idiom), each with its `file:line`:

```
core/skills/drain/SKILL.md:49  Be strict — mine only what's worth never-repeating.
teams/software-project-team/agents/api-agent/agent.md:88  Never expose the domain entity directly …
```

It **over-collects on purpose** — the collect step only gathers grounded candidates; the LLM decides which are real recurring principles. `rules/` subtrees are skipped, because rules are the distill *target*, not a source.

`--record` stamps HEAD as the last-distilled commit (in `~/.atl/rules-distill-state.json`), which resets the session-start "a distill is due" runaway-guard for ~1 day. `/rules-distill` calls it at the end of a sweep.

## Related

- [`/rules-distill`](/skills/rules-distill) — the LLM half: cluster the candidates, grep existing rules, propose the recurring ones as core rules (human-confirmed).
- [`/rule`](/skills/rule) — authors one rule you already have in mind; rules-distill discovers *which* rules the corpus is asking for.
- [`atl skills`](/cli/skills) — the sibling deterministic gate (asset content-quality; this one feeds rule discovery).
