---
knowledge-base-summary: "How I own and maintain the durable-knowledge store's `Domain/` namespace (concept #9) — the glossary, entities, and business rules that are the project's shared vocabulary. Covers what belongs there vs. in a work-item, keeping it current-truth via idempotent upsert, the one-owner discipline that avoids write races, and generic page shapes."
---

# Domain Modeling — the `Domain/` durable-knowledge namespace

Work-items are **transient execution state**; the **durable-knowledge store** is the **durable current-truth**
(concept #9 — the current-truth-vs-history split, expressed in the active backend). Of its namespaces, I own
**`Domain/`** outright: the project's shared business vocabulary — its glossary, its entities,
and its business rules. This is the one namespace that answers "what do the words mean here?" for
the whole team, human and agent.

**This is a runtime action, not content I author now.** My `children/` teach me *how* to build a
good `Domain/` page on any project; the actual glossary/entities/rules are project-specific and
get written into the durable-knowledge store when I run inside a ceremony. Everything below is the reusable
craft — the examples are deliberately generic.

## Why `Domain/` exists (and why I own it)

Every project develops its own language — a word that means one precise thing here and something
else elsewhere. Left implicit, that language drifts: the `technical-analyst` assumes one meaning,
a `developer` worker assumes another, the `tester` tests a third, and the defect is a definition
mismatch nobody wrote down. `Domain/` is the antidote: **one authoritative place** where the
business vocabulary lives as current-truth.

I own it because domain vocabulary is a **business-analysis** artifact — it is the structured
"what" of the problem space, my reflex. One owner per namespace is the rule (concept #9) and it
is load-bearing: it prevents two roles fighting over the same page (a write race). The
`technical-analyst` reads `Domain/` and builds on it in `Analysis/` (co-owned); the `tech-lead`
reads it and builds `Architecture/`; nobody else *writes* `Domain/`. If a `developer`/`tester`
worker surfaces a project fact that belongs in the domain, it is **promoted to the durable-knowledge
store by the tech-lead** at `/refine`/integration review — workers never write the durable-knowledge
store themselves (concept #9).

## What belongs in `Domain/` (and what does not)

**In `Domain/`:**
- **Glossary** — each term the project uses in a specific sense, with a crisp definition and, where
  it helps, what it is *not*.
- **Entities** — the core business objects, their meaning, their key attributes at the business
  level (not a database schema — that's the tech-lead's `Architecture/`), and how they relate.
- **Business rules** — the invariants and policies that hold regardless of implementation ("an
  order cannot be cancelled after it ships"). Rules that are true about the *domain*, stated so a
  human and an agent read them the same way.

**Not in `Domain/`:**
- Per-Epic/Feature analysis (personas, scenarios, deep reasoning) → the `Analysis/` namespace
  (`analysis-store-craft.md`).
- System shape, module boundaries, data schema, stack decisions → the tech-lead's `Architecture/`.
- Anything stack- or implementation-specific → not my namespace at all.

The test: `Domain/` holds what is true about **the business**, phrased so it would still be true
if the whole system were rebuilt on a different stack.

## Keeping it current-truth (the durable-knowledge discipline)

The durable-knowledge store is **current-truth, not history** — when a fact changes, I *update the page*, I do not
append a new definition beside the old one. A glossary with two contradictory definitions of the
same term is worse than none.

- **Read before I write.** Before adding or changing a term/entity/rule, I read the existing page
  (read the durable-knowledge store) so I update in place rather than duplicate. Discovery when I don't
  know the path: search the durable-knowledge store (concept #9).
- **Idempotent upsert.** A write to the durable-knowledge store is an idempotent upsert (concept #9) — safe
  to re-run, safe under a ceremony re-run. I write the whole page; a second run with the same
  content is a no-op.
- **Verify the namespace exists** before a first write (list the durable-knowledge pages); the store's
  location is resolved **once** at `/delivery-init` and cached in `.delivery/config.json` — I read that
  cached location, I never re-resolve it (concept #9).
- **Resilience** — durable-knowledge writes wrap the same backoff + jitter as any backend write (the
  resilience policy); and I never silently truncate a durable-knowledge-store listing — I **loop to
  exhaustion** (the "list means all" policy).

## Generic page shapes (the craft, not the content)

A `Domain/Glossary` page — one entry per term:

```markdown
# Glossary

## <Term>
<A one-to-three-sentence definition in business language. What it means here,
precisely. Where useful: "Not to be confused with <near-term>.">

## <Term>
...
```

A `Domain/Entities` page — one section per entity:

```markdown
# Entities

## <Entity>
<What this business object represents.>
- Key attributes (business-level): <attr — meaning>, ...
- Relationships: <relates to <other entity> because ...>
- Governing rules: <see Business Rules #n>
```

A `Domain/Business-Rules` page — numbered so criteria and analysis can cite them:

```markdown
# Business Rules

1. <An invariant/policy stated as a testable business truth.>
2. <...>
```

**Why numbered rules:** my acceptance criteria (`acceptance-criteria-craft.md`) and the
`Analysis/` pages can then *reference* a rule by number instead of restating it — one
authoritative statement, many citations, no drift.

## The one-owner discipline, restated

The value of `Domain/` is that it is *authoritative*. That property only survives if exactly one
role writes it. So: I keep the definitions honest and current; I resist letting implementation
detail creep in (that dilutes the business truth); and when I learn the domain was wrong or has
evolved, I **correct the page**, I don't fork it. A `Domain/` namespace that everyone trusts is
what lets the rest of the team stop re-litigating what the words mean and get on with the work.
