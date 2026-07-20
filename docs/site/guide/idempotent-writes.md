# Idempotent writes

ATL agents and ceremonies **re-run** constantly — on a restart, a retry, a resumed sprint. This core rule makes those re-runs safe: a write to any durable store must **converge** on the same end-state rather than duplicate work or clobber a newer edit. This page is the user side of that discipline.

## What's happening under the hood

The [`idempotent-writes` rule](https://github.com/agentteamland/atl/blob/main/core/rules/idempotent-writes.md) auto-loads in every session (and into the autonomous `claude -p` workers the delivery-team spawns, via the same global rule reflection). It codifies one habit: **check-before-create by a stable key, overwrite current-truth in place, never blind-write.**

It was distilled from the corpus itself — the same principle was independently re-derived in the delivery loop (its cross-backend "concept #10"), in `/brainstorm`, in `/create-code-diagram`, and in `/profile-restore`. When a discipline recurs in that many unrelated places, it belongs in one rule.

## What it means in practice

**Check-before-create by a stable key.** Before creating a durable item — a work-item, a page, a config — the agent searches for it by a key derived from stable inputs (a parent + ordinal, a content hash), never a per-run id. Found → it reuses and updates; not-found → it creates and stamps the key. A collision resolves *to* the existing item instead of erroring.

**Overwrite current-truth in place.** A store that holds "what is true now" — a wiki page, a generated diagram, a review report, a config file — is replaced on re-run, not appended to. Running the same ceremony twice leaves one converged result, not two.

**Never blind-clobber a newer edit.** When an overwrite could destroy data written since the agent last read the target, it guards the write — comparing timestamps/versions, or diffing and confirming — because losing a newer edit is worse than pausing. This is exactly what `/profile-restore` does before it touches your global memory.

## Why it's a core rule

ATL's **autonomous delivery** depends on it: a resumed sprint must converge on the durable state, not re-create it — otherwise a restart means a duplicate work-item, a doubled PR, or a lost edit. Making convergence the default (not a special recovery mode) is what lets an unattended `claude -p` worker retry and resume safely. The one intentional exception is **append-only** stores — a journal or audit log, where adding a new dated entry each run is the point, not a duplicate.
