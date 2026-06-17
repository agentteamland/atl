---
name: publish
description: Share your global-layer gains for a team upstream — propose a contribution (team you don't own) or re-publish (team you own). Reads `atl publish`'s plan, judges what's worth sharing, writes the PR body, opens the contribution. Run when atl suggests "gains in X not yet upstream", or to share gains manually.
---

# /publish — share your gains upstream

The LLM half of `atl publish` (ring 2→3 of gain circulation). The CLI computes
the plan (which global-layer files differ from the published version + whether
you own the repo) and does the git/PR plumbing; this skill supplies the
judgment and the prose the CLI can't: *what* is worth proposing and the PR body.

**publish is deliberate by design** — it crosses the author boundary, so it
never runs automatically. You invoke it (your consent to share outward); the
owner accepts the PR (their consent). Your own local + global gains never depend
on the owner accepting anything.

## Procedure

### 1. Get the plan
Run, in any project:
```
atl publish <handle>/<team>
```
It lists the publishable gains (each `modified` or `new`) and whether you own
the repo. If it says "nothing to publish", report that and stop.

### 2. Judge what's worth sharing
Not every divergence belongs upstream. **Keep** gains that are *general* — a
clearer instruction, a better pattern, a reusable learning that helps anyone
using the team. **Drop** anything *project-* or *user-specific* — that belongs
at project scope (which shadows global), not in the shared team. When a gain is
borderline, ask the user before including it.

### 3a. Team you DON'T own → propose upstream
The CLI does the mechanics (fork + branch + apply the kept gains + push + open a
PR against the source repo). You write the **PR body**:
- **What changed and WHY** — one short section per gain, leading with the reason.
- Frame it as a best-effort contribution from real usage.
- No pressure: the owner accepts or not; either way the contributor keeps the
  gains locally + globally.

### 3b. Team you OWN → re-publish
The CLI commits the kept gains to your repo, bumps the team version, and tags;
topic-discovery (index CI) reindexes from there. You write the **commit
message** (conventional, with the why).

### 4. Report
Surface the PR link (propose-upstream) or the new tag (re-publish), plus a
one-line summary of what was shared and what was deliberately left out.

## Notes
- **Whole-file gains** (consistent with promote/fanout) — you propose each file
  as it stands in your global layer, not a hand-merge of fragments.
- **CLI/Skill boundary:** the deterministic apply (fork/branch/push/PR-open, or
  commit/tag) is the CLI's job; the judgment (step 2) + the prose (PR body /
  commit message) are this skill's. The skill drives the CLI's apply primitive
  and hands it the authored body.
- **Suggestion, not automation:** atl may surface "gains in X not yet upstream"
  during a throttled network check — that's the prompt to run this; the act
  stays yours.
