---
knowledge-base-summary: "How I use the project wiki's `Analysis/` namespace (adapter §8), co-owned with the technical-analyst — the deep per-Epic/Feature analysis that is too long for the work-item Description. Covers the split between the terse business-owned Description and the deep wiki page, the co-ownership discipline that keeps my business layer and the TA's technical layer from colliding, and a generic page shape."
---

# Analysis-Wiki Craft — the `Analysis/` namespace

The Epic/Feature `System.Description` is deliberately **terse** — it is the always-loaded "what &
why" summary, five fixed headings, scannable (`business-analysis-blueprint.md`, adapter §7). But
some Epics and Features need real depth: personas and their contexts, the scenarios and edge
conditions behind the acceptance criteria, the business reasoning that led to the scope
decisions, the open questions and their resolutions. That depth does not belong in the
Description — it goes in the project wiki's **`Analysis/`** namespace (adapter §8).

`Analysis/` is **co-owned** with the `technical-analyst`. Domain reasoning is mine; technical
depth (the approach, feasibility, risks, NFR rationale) is theirs. The discipline below is how we
share the namespace without a write race.

**Runtime action, not pre-authored content.** These children teach me *how* to write a strong
`Analysis/` page on any project; the actual page is project-specific and written when I run inside
a ceremony. The example is generic.

## The split: terse Description vs. deep wiki page

| | Work-item `System.Description` | `Analysis/` wiki page |
|---|---|---|
| **Purpose** | The always-loaded summary — value + testable conditions | The deep reasoning behind that summary |
| **Length** | Terse, five fixed H2s | As long as the analysis needs |
| **Audience** | Every consumer, every read of the item | Whoever needs the depth (refine, hard estimates, disputes) |
| **Owner** | Business-owned (mine) | Co-owned: my business layer + the TA's technical layer |
| **Read via** | `wit_get_work_item` (parse headings) | `wiki_get_page_content` / `search_wiki` |

**The rule of thumb:** if it is a *conclusion* the whole team must always see, it goes in the
Description. If it is the *reasoning* that produced the conclusion — needed sometimes, by some
readers — it goes in `Analysis/`. The Description says "these are the acceptance criteria"; the
`Analysis/` page says "here is why these criteria and not others, here are the scenarios they
cover, here is the edge case we deliberately excluded and why."

**Why keep them separate rather than one big Description:** the Description is loaded on *every*
read of the work-item by *every* consumer — bloating it taxes everyone. The wiki page is pulled
only when the depth is wanted. Keeping the summary terse and the depth linked is what keeps the
board fast to reason about while the reasoning stays recoverable.

## Co-ownership with the technical-analyst (no write race)

`Analysis/` is the one namespace with two owners, so we keep our layers cleanly separated on the
page:

- **My layer (business analysis):** the problem depth, personas and their real contexts, the
  scenarios, the business rules in play (cross-linked to `Domain/`), scope reasoning, open
  business questions.
- **The TA's layer (technical analysis):** approach depth, feasibility and risk reasoning, NFR
  rationale, dependency analysis. This mirrors the TA's work-item comment (the
  `**[Technical Analysis]**` sentinel, adapter §7) but at wiki depth.

The mechanics that prevent a collision:
- **Distinct sections, one page.** A per-Epic/Feature `Analysis/` page carries a **Business
  Analysis** section (mine) and a **Technical Analysis** section (the TA's). We each own our
  section; neither rewrites the other's.
- **Read before write.** Before I update the page I read it (`wiki_get_page_content`) so I revise
  my section in place and leave the TA's intact. `wiki_create_or_update_page` is an idempotent
  upsert (adapter §8), so writing the whole page back with only my section changed is safe.
- **Cross-link, don't duplicate.** My section links to `Domain/` for definitions and to the
  work-item for the terse criteria; the TA's section links to `Architecture/`/ADRs. We reference
  each other's layer instead of restating it.
- **When our layers meet** (a business scope decision that turns on a technical constraint, or
  vice-versa), we note the dependency and cross-reference — the business reason lives in my
  section, the technical reason in theirs, linked. Neither absorbs the other.

## Generic page shape (the craft)

An `Analysis/Feature-<n>-<slug>` page (deliberately generic content):

```markdown
# Analysis — <Feature title>

> Work-item: #<id>   ·   Epic: #<parent-id>   ·   Domain: Domain/Glossary, Domain/Entities

## Business Analysis   <!-- owner: business-analyst -->

### Personas & context
<Who is affected, in what real situation, with what constraint. Not a stereotype — the
actual context that shapes the requirement.>

### Scenarios
<The concrete flows behind the acceptance criteria — happy path, the meaningful variations,
the failure and edge conditions. Each ties to an AC in the work-item Description.>

### Scope reasoning
<Why this boundary. What's deliberately Out of Scope and the business reason. Rejected
alternatives and why they were rejected.>

### Business rules in play
<Cross-links to Domain/Business-Rules by number; any rule this Feature introduces or
tightens, promoted to Domain/ separately.>

### Open questions
<Unresolved business questions + who owns the answer (usually the PO). Resolved ones kept
with their resolution and date, so the reasoning trail survives.>

## Technical Analysis   <!-- owner: technical-analyst -->
<The technical-analyst's depth — I do not write here.>
```

## When to bother with an `Analysis/` page at all

Not every Feature needs one. A small, well-understood Feature is fully served by its Description.
I open an `Analysis/` page when: the Feature has non-obvious scenarios or edge conditions; the
scope boundary needed real reasoning to draw; there are open business questions worth recording;
or the PO/team will plausibly revisit "why did we decide it this way?" later. The heuristic is
**would a future refine or dispute be resolved faster with the reasoning written down?** — if
yes, write the page; if no, the terse Description is enough. Over-documenting a trivial item is as
much a smell as under-documenting a complex one.
