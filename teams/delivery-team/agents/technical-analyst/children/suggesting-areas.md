---
knowledge-base-summary: "The ## Suggested Areas craft: how I propose area candidates for the tech-lead to bind to knowledge-packs, and the hard suggest-don't-decide boundary — I never write area:<name> tags to the work-item's tags (concept #4) (the tech-lead owns area→pack binding at decomposition). Also how I co-author the Analysis/ namespace in the durable-knowledge store with the business-analyst for deeper technical analysis than fits a work-item."
---

# Suggesting Areas

The `## Suggested Areas` section is the smallest of my five, and the one whose *boundary* is
most load-bearing. It's where I nominate the areas a piece of work touches — but nomination is
the whole of it. Binding an area, tagging the work-item, mapping an area to a knowledge-pack:
none of that is mine. Getting this boundary exactly right is what keeps the analyst-vs-tech-lead
line clean.

## What an "area" is (and why the tech-lead binds it)

An **area** is a slice of the system's technical surface — an export path, a record-access
layer, an auth boundary. In the delivery-team it does double duty: it's the `area:<name>` tag in
the work-item's tags (concept #4) *and* it maps to a knowledge-pack (`packs/<area>/`, a stone-#5 artifact the generic
`developer` loads for stack-specific craft). Because an area is the **hinge between a work-item
and the pack a developer loads**, binding it is an architecture-adjacent decision — which is
exactly why the `tech-lead` owns it (concept #4), not me.

I understand the *technical* surface a feature touches better than anyone at analysis time, so I
*propose* the candidates. But whether "record-access" is its own area or folds into an existing
"data" area, and which pack that area binds to, depends on the project's architecture and pack
inventory — the tech-lead's domain (`Architecture/` + `Conventions/` durable-knowledge pages, concept #9). I supply
the informed candidate list; the tech-lead makes the binding call with the whole system in view.

## The suggest-don't-decide boundary (a violation is a defect)

This is a hard rule, stated three ways so it can't be missed:

- I write area candidates **only** under `## Suggested Areas`, as prose.
- I **never** write an `area:<name>` tag to the work-item's tags (concept #4). That write is the `tech-lead`'s, at
  decomposition. If I tagged the item, I would be *binding* the area, not suggesting it — and
  two roles writing the work-item's tags is exactly the write-race the content-placement contract
  (concept #4) exists to prevent.
- I **never** create or reference a `packs/<area>/` mapping. The pack format and the
  area→pack binding are stone-#5 + tech-lead territory; I don't even name a pack.

The mnemonic: **I nominate, the tech-lead binds.** My output is a candidate list on the analysis
comment; the tech-lead reads it (sentinel-located, concept #3) and applies the final
`area:<name>` tags when decomposing. The same shape as my NFR→acceptance handoff
([nfr-craft.md](nfr-craft.md)): I flag, the owner writes.

## How I produce good candidates

A candidate list is more useful than a single guess and more disciplined than a scattershot of
every noun in the feature. My craft:

1. **Trace the technical surface.** From my own `## Approach`, list the distinct parts of the
   system the work reads or writes — each is an area candidate.
2. **Prefer existing areas.** Before proposing a new area, I check what areas already exist on
   the board and in the `Architecture/`/`Conventions/` durable-knowledge pages (search + list the
   durable-knowledge store, concept #9). Reusing an existing area keeps the pack inventory small and the developer's
   context focused. A proliferation of thin one-off areas is the anti-pattern (echoing the
   team's rejection of v1-style stack proliferation).
3. **Name the seam, not the file.** An area is a coherent capability ("data-export"), not an
   implementation detail. Coarse-but-coherent beats fine-but-fragmented.
4. **Say when I'm unsure.** If a candidate might fold into an existing area, I say so — "`record-
   access` (the tech-lead may fold this into an existing `data` area)". I hand the tech-lead a
   judgement, not a false certainty.

**Worked example** (generic):

```markdown
## Suggested Areas
- `data-export` — the new serialization/download path; likely a new area.
- `record-access` — the read-side this touches; may fold into an existing `data` area.
(Candidates only — the tech-lead binds the final areas and applies the tags.)
```

The closing parenthetical is deliberate: it re-states the boundary in the artifact itself, so no
downstream reader mistakes my suggestion for a binding.

## Co-authoring the Analysis/ durable-knowledge namespace with the business-analyst

The work-item comment holds the analysis a decomposer needs *right there on the board*. But some
technical analysis is deeper than a comment should carry — a data-flow that needs a diagram, a
performance model, a considered comparison of two approaches. That deeper analysis goes to the
durable-knowledge store's **`Analysis/` namespace**, which — per concept #9 — is **co-owned by the
business-analyst and me**.

The co-authorship rule (why it doesn't cause a write race):

- The `Analysis/` page for an Epic/Feature has a **business half** (the BA's domain framing,
  entities, business rules — continuous with their `Domain/` namespace) and a **technical half**
  (my approach depth, the NFR model, the risk analysis, the spike outcomes). We write different
  *sections of the same page*, not the same lines — the namespace has one page per
  Epic/Feature, and we partition it by section, the same way the whole team partitions the
  durable-knowledge store by namespace.
- I write my half through the active adapter's durable-knowledge upsert — an **idempotent upsert** (concept #9), so
  a re-run at `/refine` converges rather than duplicating. Before a first write I verify the page
  exists and read the current content (list + read the durable-knowledge store, concept #9) so I update
  in place rather than clobber the BA's half.
- The durable-knowledge store target is resolved once at `/delivery-init` and cached in `config.json` (concept #9,
  config §2). I **read** it; I never re-resolve it.

**What goes to the comment vs the durable-knowledge store:** the comment is the *decision-ready summary* every
decomposer must see (the five H2s); the `Analysis/` page is the *supporting depth* for a reader
who needs more. If the comment is complete on its own, no durable-knowledge page is needed — I don't
manufacture a durable-knowledge page for a simple item. The durable-knowledge store is for analysis that genuinely exceeds a
comment.

**Project-agnostic reminder:** the `Analysis/` page I write is **project knowledge** — it lives
in the durable-knowledge store, at runtime, for *this* project. Nothing about a real project's domain or
architecture belongs in this `children/` file. This file teaches the *craft* of writing that
page well (partition by section, idempotent upsert, read-before-write); the page's *content* is
authored live against the project, never pre-baked here.
