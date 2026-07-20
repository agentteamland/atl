# `/publish`

Share your global-layer gains for a team upstream — propose a contribution to a team you don't own, or re-publish a team you do. The LLM half of [`atl publish`](/cli/publish) (ring 2→3 of gain circulation).

`/publish` supplies the judgment and prose the CLI can't: *what* is worth sharing and the PR body or commit message. The CLI computes the plan (which global-layer files differ from the published version, and whether you own the repo) and does the git/PR plumbing.

**publish is deliberate by design** — it crosses the author boundary, so it never runs automatically. You invoke it (your consent to share outward); the owner accepts the PR (their consent). Your own local and global gains never depend on the owner accepting anything.

## When to use it

- When `atl` surfaces **"gains in X not yet upstream"** during a throttled network check.
- Any time you want to share a team's accumulated gains manually.

## Procedure

### 1. Get the plan

```bash
atl publish <handle>/<team>
```

It lists the publishable gains (each `modified` or `new`) and whether you own the repo. If it says "nothing to publish", report that and stop.

### 2. Judge what's worth sharing

Not every divergence belongs upstream. **Keep** gains that are *general* — a clearer instruction, a better pattern, a reusable learning that helps anyone using the team. **Drop** anything *project-* or *user-specific* — that belongs at project scope (which shadows global), not in the shared team. When a gain is borderline, ask the user before including it.

### 3a. Team you DON'T own → propose upstream

The CLI does the mechanics (fork + branch + apply the kept gains + push + open a PR against the source repo). Drive the apply with:

```bash
atl publish <handle>/<team> --apply --body-file <file> --only <path> [--only <path> …]
```

`--only` restricts the apply to the subset your step-2 judgment kept — **without it every gain is published, including the ones you decided to drop**, so the keep/drop judgment must feed the `--only` list. `--body-file` hands the CLI your authored PR body; add `--dry-run` to preview the plan first.

You write the **PR body**:

- **What changed and WHY** — one short section per gain, leading with the reason.
- Frame it as a best-effort contribution from real usage.
- No pressure: the owner accepts or not; either way you keep the gains locally and globally.

### 3b. Team you OWN → re-publish

The CLI commits the kept gains to your repo, bumps the team version, and tags it; topic-discovery (the index CI) reindexes from there. You write the **commit message** (conventional, with the why).

### 4. Report

Surface the PR link (propose-upstream) or the new tag (re-publish), plus a one-line summary of what was shared and what was deliberately left out.

## Notes

- **Whole-file gains** (consistent with [`atl promote`](/cli/promote) and fan-out) — you propose each file as it stands in your global layer, not a hand-merge of fragments.
- **CLI / Skill boundary** — the deterministic apply (fork/branch/push/PR-open, or commit/tag) is the CLI's job; the judgment (step 2) and the prose (PR body or commit message) are this skill's. The skill drives the CLI's apply primitive and hands it the authored body.
- **Suggestion, not automation** — atl may surface "gains in X not yet upstream" during a throttled network check; that's the prompt to run this, but the act stays yours.

## Related

- [`atl publish`](/cli/publish) — the deterministic half this skill drives.
- [`atl promote`](/cli/promote) — the ring 1→2 step (project → global) that publish builds on.
- [Skills overview](/skills/drain)

## Source

- Spec: [core/skills/publish/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/publish/SKILL.md)
- CLI: [cli/cmd/atl/commands/publish.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/publish.go)
