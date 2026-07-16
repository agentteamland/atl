---
knowledge-base-summary: "How I write the canonical brief a developer worker reads — the artifact that bounds a fresh, isolated worker's context. It restates the unit's goal + acceptance, names the area (→ knowledge-pack) and EMBEDS the exact Architecture/ + Conventions/ durable-knowledge page paths for that area (concept #9 read contract) so the worker pulls the right project knowledge from the durable-knowledge store, and lists the unit's dependencies. What a good brief contains, and what it deliberately leaves out."
---

# Canonical Brief

The developer worker in this team is a **fresh, isolated `claude -p` per work-unit** (concept #9,
config-and-methodology `roles[].dispatch: worker`): its own git worktree, its own context, **no
carry-over** from any other unit or ceremony. That isolation is a feature — it keeps workers
parallel and context clean — but it means a worker knows *nothing* except what its context is
assembled from. The read contract (concept #9) says that context is:

> stack-pack + durable-knowledge store + task + **the tech-lead's canonical brief**.

The **canonical brief is mine.** It is the piece that turns four disconnected inputs into a
worker that behaves as if it had sat in the `/refine` room. Getting it right is what makes an
isolated worker produce work that fits the architecture and obeys the conventions — without a
human re-explaining the project every time.

## What a good brief contains

A brief is short and pointed — it *bounds* context, it does not dump it. The worker will read the
durable-knowledge pages I point it at; the brief's job is to point precisely and add only what the
store can't.

1. **The unit's goal, restated in one or two sentences** — what "done" means for *this* unit,
   traced to the Feature's Acceptance Criteria (which I read from the Feature spec field, concept
   #2, at decomposition). The worker should not have to reconstruct intent from the raw work-item.
2. **The area** — the `area:<name>` tag I applied at decomposition. This binds the worker to its
   knowledge-pack (`packs/<area>/`, stone #5) and tells it which slice of the durable-knowledge
   store to load.
3. **The embedded durable-knowledge page paths (the load-bearing part).** I name the *exact* pages
   for this unit's area (concept #9 read contract):
   - the `Architecture/` slice relevant to the area (boundaries, the module the unit touches, any
     ADR that constrains it — from [architecture-and-adr.md](architecture-and-adr.md)),
   - the `Conventions/` page (project rules layered on the pack's generics — from
     [conventions-craft.md](conventions-craft.md)).
   The worker reads these from the durable-knowledge store; a store search is the fallback when a
   path isn't pre-named. Because I *embed the paths*, the worker loads the right knowledge
   deterministically instead of guessing or scanning the whole store.
4. **Dependencies** — the sibling units this one builds on (the `Dependency` links I added). The
   engine won't dispatch this unit until its prerequisites merged, but the worker still needs to
   know what surface it's building against and must not re-implement a sibling's contract.
5. **The test-evidence expectation** — a pointer to what must be attached before review (code +
   web + mobile-emulator evidence where the surface applies), so the worker knows the review gate
   ahead of time (see [review-craft.md](review-craft.md), which is the other side of this
   contract).

## What a good brief deliberately leaves out

Bounding context is as much about exclusion as inclusion:

- **The whole durable-knowledge store.** I point at the *relevant* pages, not "read
  `Architecture/`." A brief that says "load everything" defeats the isolation and burns the
  worker's context on irrelevant areas.
- **Other units' internals.** The worker sees its dependencies' *contracts* (via the
  durable-knowledge store / merged code), not their implementation reasoning. Cross-unit coherence
  is my job at the
  integration checkpoint ([integration-checkpoint.md](integration-checkpoint.md)), not a thing I
  push into every brief.
- **Stack how-to.** That is the knowledge-pack's job (stone #5). The brief names the area so the
  pack loads; it does not restate the stack's idioms. Keeping stack out of the brief is the same
  stack-agnostic discipline that keeps it out of my `children/`.
- **Methodology mechanics.** The worker doesn't need the sprint model; it needs *this unit*. Cadence
  lives in `methodology.json`, read by ceremonies, not repeated to a worker.

## Where the brief lives — the storage contract (concept #3)

The brief is **one labeled comment on the work-unit** (concept #3), whose **first line is the
exact sentinel** — nothing before it:

```
**[Canonical Brief]**
```

then the fixed H2s `## Goal`, `## Area`, `## Load These Pages`, `## Depends On`,
`## Evidence Before Review`. It is a **work-item comment, never a durable-knowledge page** — the
same placement discipline the `**[Technical Analysis]**` comment follows (concept #3). The
sentinel is what makes the brief machine-locatable: a spawned `developer` (and the `tester`)
locates it by listing the comments (concept #3) filtered to the `**[Canonical Brief]**` sentinel —
a **sentinel match, never "the newest comment,"** so a later human comment never shadows it.

**Idempotency by sentinel.** A re-refined unit must not stack a second brief. Before I add, I list
the comments (concept #3) and sentinel-match: **found** → I update that comment in place (keyed
by its `atl-key`); **not found** → I add a comment (concept #3) with the sentinel as line one. The
sentinel is the *locator*; the `atl-key` is the *convergence* guard. Drop the sentinel or rename a
heading and I silently break every downstream reader.

## Worked example (generic)

```
**[Canonical Brief]**

## Goal
Implement the credential-validation path for the auth surface so a submitted credential is
checked against the store and yields an authenticated session or a typed failure. Done = the
Feature's AC "invalid credentials are rejected with a typed error" holds, plus the self-test +
tester gates pass.

## Area
area:auth  → knowledge-pack packs/auth/

## Load These Pages
- Architecture/Auth-surface       (module boundary + the write-path owner; see ADR-3)
- Conventions/                    (project error-handling agreement; naming scheme)

## Depends On
Task #2085 (ordinal 1, the auth surface shell) — build against its session contract; do not
re-declare it.

## Evidence Before Review
unit/integration for the validation path; web evidence for the sign-in flow; mobile-emulator
evidence if this surface renders on mobile.
```

(Illustrative — the paths, ADR number, and IDs are examples; the real ones come from the project's
durable-knowledge store and the decomposition plan. The `**[Canonical Brief]**` sentinel + the
five H2s are fixed.)

## Why the brief is a tech-lead artifact, not a developer one

I write the brief because I hold the highest context in the team: I did the decomposition, I own
`Architecture/` and `Conventions/`, and I know which pages a given unit needs. A worker cannot
write its own brief — it starts with nothing. And I am the review gate ([review-craft.md](review-craft.md)),
so the brief and the review criteria come from **one owner**, which keeps "what I asked for" and
"what I'll accept" aligned. The brief is the contract; the review is me holding the worker to it.

## Checklist

- [ ] Goal restated in 1–2 sentences, traced to the Feature's Acceptance Criteria.
- [ ] `area:<name>` named → binds the knowledge-pack.
- [ ] **Exact** `Architecture/` slice + `Conventions/` page paths embedded (concept #9 read
      contract) — not "read the store," specific paths.
- [ ] Any constraining ADR referenced by number.
- [ ] Dependencies named (the `Dependency`-linked prerequisites), with "build against, don't
      re-declare" guidance.
- [ ] Test-evidence expectation stated (code + web + mobile-emulator where the surface applies).
- [ ] Nothing dumped that the worker doesn't need — the whole durable-knowledge store, other
      units' internals, stack how-to, and methodology mechanics all left out.
