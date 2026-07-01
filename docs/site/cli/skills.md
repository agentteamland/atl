# `atl skills`

Deterministic, LLM-free **content-quality checks** for the platform's own skills, agents, and team manifests — the sibling of [`atl docs check`](/cli/docs). Where docs-check validates the docs *site* against the code, skills-check validates the **assets themselves**.

This is a **maintainer-side** gate that runs against the monorepo's `core/` and `teams/` trees. Outside the monorepo it does nothing and exits 0 (the pre-flight skip), so end-user sessions never see it.

## Usage

```bash
atl skills check    # validate frontmatter, team.json consistency, agent-KB children
```

## What it checks

Every check is **zero-false-positive by construction** — a failure is always a real problem, so it's safe to gate a PR on:

| Check | What must hold |
|---|---|
| **frontmatter** | Every skill's `SKILL.md` and every agent's `agent.md` carries a `name` + `description` frontmatter block. |
| **manifest** | Each `team.json`'s `agents[]` / `skills[]` names match the on-disk directories — **both directions** (nothing declared-but-absent, nothing on-disk-but-undeclared). |
| **children** | Every agent-KB child (`agents/<x>/children/*.md`) declares a non-empty `knowledge-base-summary` frontmatter — the KB-rebuild contract. |

`atl skills check` exits non-zero on any failure, so it **gates every PR in CI** alongside the docs-drift gate. The judgment half — does a skill obey its own documented flow? do two skills overlap? — is the job of the companion [`/skill-stocktake`](/skills/skill-stocktake) skill (LLM), not this deterministic net. That split is the CLI/Skill boundary: deterministic checks here, grounded judgment in the skill.

## Related

- [`/skill-stocktake`](/skills/skill-stocktake) — the LLM half: obedience + redundancy, grep-grounded, change-aware
- [`atl docs check`](/cli/docs) — the sibling gate: docs-site drift (this one is asset content-quality)
- [`atl doctor`](/cli/doctor) — the runtime self-heal (this is a build-time quality gate)
