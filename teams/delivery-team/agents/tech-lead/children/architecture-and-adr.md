---
knowledge-base-summary: "How I own the project's Architecture/ and Architecture/ADR/ wiki namespaces (adapter §8): keeping the Architecture/ page a current-truth upsert of system shape / module boundaries / area vocabulary, deciding when a decision earns an ADR (significant AND hard-to-reverse), the ADR page format, the one-owner-no-write-races discipline, and how project facts a worker surfaces get promoted up to these pages by me."
---

# Architecture & ADR

I am the **sole owner** of two project-wiki namespaces (adapter §8): `Architecture/` — the
system's current-truth shape — and `Architecture/ADR/` — one page per architecture decision.
One owner per namespace is the rule that stops two roles from racing on the same page: the
`business-analyst` owns `Domain/`, the analysts share `Analysis/`, the `project-manager` owns the
sprint-review pages; **architecture is mine.** I write these at `/refine` and at the integration
checkpoint (see [integration-checkpoint.md](integration-checkpoint.md)).

This is **project knowledge** — it lives in the Azure project wiki, not in these `children/`
files, because it is specific to one system. My `children/` teach me *how* to produce these pages
well on any project; the pages themselves I write at runtime.

## The `Architecture/` namespace — current truth, always upsert

`Architecture/` holds the system's shape as it is **now**: the module/area boundaries, how the
areas relate, the cross-cutting decisions in force, and the **area vocabulary** the decomposition
uses (the `area:<name>` tags I apply in
[decomposition-blueprint.md](decomposition-blueprint.md) trace to the areas listed here). It is
the wiki-side counterpart of ATL's own wiki: **current truth, replaced not appended.**

- I write it with `wiki_create_or_update_page`, which is an **idempotent upsert** (adapter §8) —
  safe to re-run, safe under the §5 re-run guard. When the system shape changes I *update the
  page*, I do not append a new note; a stale line on this page is a bug because everyone reads it
  as present-tense truth.
- I resolve the target `wikiId` from `config.json` (cached once at `/delivery-init`, adapter §8);
  I **never re-resolve** it. I verify the namespace exists with `wiki_list_pages` before a first
  write.
- The page is **lean and current** — it is a map, not a history. The *why* of a specific hard
  decision goes to an ADR (below); the *narrative* of what happened goes nowhere here — the
  journal-shaped record is the sprint-review pages the `project-manager` owns. `Architecture/` is
  strictly "what the system's shape is today."

A good `Architecture/` page answers, for a fresh `developer` worker loaded with only its brief:
what areas exist, where the boundaries are, what module owns what responsibility, and which
system-wide decisions constrain a change. My canonical brief
([canonical-brief.md](canonical-brief.md)) points each worker at the slice of this page relevant
to its area.

## The `Architecture/ADR/` namespace — one page per decision

An **ADR** (Architecture Decision Record) is a durable page at
`Architecture/ADR/ADR-<n>-<slug>` capturing a single decision, its context, and its
consequences. `<n>` is a monotonically increasing integer; `<slug>` is a short kebab identifier.
One decision, one page — never merged, never renumbered.

### When a decision earns an ADR

Not every choice is an ADR. The bar is **significant AND hard-to-reverse** — both, not either:

- **Significant** — it constrains multiple areas, shapes an interface many units depend on, or a
  future maintainer would be surprised by it without the reasoning.
- **Hard-to-reverse** — undoing it later is expensive (a data shape, a boundary, an integration
  contract, a cross-cutting concern), so the *why* must survive the people who made it.

A choice that is significant but trivially reversible is a line on the `Architecture/` page, not
an ADR. A choice that is hard-to-reverse but locally scoped (a single unit's internal approach)
is the worker's business, not an ADR. Only the intersection earns the ceremony of a record.
Erring toward *fewer* ADRs keeps them high-signal — an ADR log nobody trusts because it's full of
trivia is worse than none.

### The ADR page format

I keep every ADR in the same shape so they read uniformly and a reader knows exactly where the
reasoning is:

```markdown
# ADR-<n>: <short title>

- **Status:** Accepted            (or: Proposed / Superseded by ADR-<m>)
- **Date:** <YYYY-MM-DD>
- **Areas affected:** <area:...>, <area:...>

## Context
<the forces at play — the constraint, the NFR, the problem that forced a choice.>

## Decision
<what we decided, stated as a present-tense fact. One decision per ADR.>

## Consequences
<what this makes easy, what it makes hard, what it forecloses. The honest trade-off —
including the downside we accepted.>

## Alternatives considered
<the options rejected, and the one-line reason each lost. This is what stops the
decision being relitigated six months later.>
```

**Current-truth via supersession, not edit.** An ADR is immutable current-truth of a *past
decision*. When a later decision reverses it, I do not rewrite the old ADR — I write a **new**
ADR and set the old one's `Status: Superseded by ADR-<m>`, and the new one's Context references
what it replaces. This preserves the reasoning trail (why we once chose the other way) while the
`Architecture/` page always reflects the *live* decision. Editing an accepted ADR in place would
destroy exactly the history the ADR exists to keep.

### Worked example (generic)

> `ADR-3: single write-path for the shared surface` — Context: two areas both needed to mutate
> the same surface and were racing. Decision: one area owns the write-path; the other reads and
> requests changes through it. Consequences: removes the race and gives one review owner; costs a
> hop of indirection for the reading area. Alternatives: dual write-paths with locking (rejected
> — the lock discipline is a standing footgun); merge the two areas (rejected — they diverge
> elsewhere). Areas affected: `area:X`, `area:Y`.

(Illustrative — the real decision comes from the project.)

## One-owner discipline — why I hold the pen

Because I am the only writer of `Architecture/` and `Architecture/ADR/`, there is no write race
on these pages and no divergent "two versions of the truth." Concretely:

- **Developer/tester workers do NOT write the wiki** (adapter §8). When a worker surfaces a real
  project fact — "this boundary is actually leaky," "this area has a hidden dependency" — that
  fact reaches these pages **through me**: I promote it at `/refine` or at the integration
  checkpoint. The worker's own durable *role-craft* learnings go to *its* `children/` via
  `/drain`; project facts route up to my pages. This asymmetry keeps write-authority clean.
- The `technical-analyst` produces the first technical read (feasibility, NFRs, suggested areas)
  as the sentinel comment — I **consume** that and turn the durable parts into the
  `Architecture/` page and, where warranted, an ADR. The analyst does not write `Architecture/`;
  I do. That is the boundary between "first analysis" (analyst) and "architecture of record" (me).

## Checklist

- [ ] `Architecture/` page kept current via upsert; stale lines removed, not appended over.
- [ ] Area vocabulary on the page matches the `area:<name>` tags I apply at decomposition.
- [ ] `wikiId` read from `config.json` cache; namespace existence verified (`wiki_list_pages`)
      before first write.
- [ ] An ADR written **only** for a significant AND hard-to-reverse decision.
- [ ] ADR follows the fixed format (Status / Context / Decision / Consequences / Alternatives).
- [ ] A reversed decision → **new ADR + supersede the old**, never an in-place edit.
- [ ] Worker-surfaced project facts promoted to these pages by me — workers never wrote the wiki.
