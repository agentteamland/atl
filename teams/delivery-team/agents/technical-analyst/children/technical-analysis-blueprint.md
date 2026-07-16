---
knowledge-base-summary: "My primary production unit: the single labeled Technical Analysis comment on a Feature/PBI. First line is the exact sentinel **[Technical Analysis]**, then the five fixed H2s (## Approach, ## Feasibility & Risks, ## NFRs, ## Dependencies, ## Suggested Areas), added as a comment through the active backend (concept #3). Why a comment (not the spec field), the read-back-by-sentinel contract, a completion checklist, and a generic worked example."
---

# Technical Analysis Blueprint

This is my primary production unit — the artifact I create over and over. For each Feature
or PBI the `business-analyst` has already framed, I attach **one** technical-analysis comment
to the work-item. It is the durable, machine-locatable record of the "what & why" from the
technical side: the approach, what's feasible and where the risk sits, the non-functional
requirements, the technical dependencies, and the areas I *suggest*. Every consumer — the
`tech-lead` at decomposition, the `developer` in a worker, the `project-manager` sizing the
work — reads it back by location, never by guessing.

## The exact contract (concept #3)

I write **one comment** on the work-item — add a comment through the active backend (concept #3).
The comment's **first line is the exact sentinel** — nothing before it:

```
**[Technical Analysis]**
```

Then, in this fixed order, five H2 sections — always all five, never renamed, never
reordered:

```markdown
**[Technical Analysis]**

## Approach
<how I'd build this, at the design/shape level — not code>

## Feasibility & Risks
<can we do it? what's hard, what's unknown, what could go wrong>

## NFRs
<the measurable non-functional requirements this work must meet>

## Dependencies
<technical dependencies: other work-items, external systems, sequencing>

## Suggested Areas
<the area candidates I propose for the tech-lead to bind — suggestions only>
```

The sentinel + the fixed headings are the **whole point**: they make the analysis a
deterministic read-back target (concept #3). At `/refine`, the tech-lead lists the comments
(concept #3) and filters to the comment whose first line matches the
`**[Technical Analysis]**` sentinel — a **sentinel match, never "the newest comment."** A
human can comment after me, a later revision can add another comment, and the tech-lead
still finds the analysis by the sentinel. If I renamed a heading or dropped the sentinel, I
would silently break every downstream reader.

## Why a comment, not the spec field

The Epic/Feature **spec field** belongs to the `business-analyst` — it holds the
business-owned "what & why" under *its* fixed H2s (`## Problem`, `## Business Value`,
`## Scope`, `## Acceptance Criteria`, `## Out of Scope`). If I wrote my analysis into the
spec field I would either overwrite the BA's content or fight them for the same field on
every refine. A **comment** keeps the board's spec field business-owned and single-authored,
while the sentinel keeps my analysis just as findable. One field, one owner — that is how the
team avoids write races on a work-item (the same discipline the durable-knowledge namespaces enforce for
pages, concept #9).

## One comment, written idempotently — not a comment per run

A re-run of the analysis ceremony (a re-plan, a crash-resume) must **not** stack a second
`**[Technical Analysis]**` comment on the item. The comment channel is **append-only**
(concept #3) — there is **no update-comment operation** to guess, so I never invent one. My
idempotency guard is the sentinel itself: before I add, I list the comments (concept #3) and
sentinel-match.

- **Found** → the analysis is already recorded on this item, so I **do not add again** — the
  sentinel match *is* the convergence check, and skipping the add keeps exactly one analysis
  comment. If the analysis genuinely changed (a real re-plan), I add **one** fresh sentinel
  comment stating it supersedes the earlier one; downstream read-back still resolves the
  analysis by sentinel — a later comment never shadows it, it *is* it.
- **Not found** → I add a comment (concept #3) with the sentinel as line one.

This mirrors the team's idempotency discipline (concept #10): the same logical artifact maps
to the same target across re-runs, so a resume is convergent, not duplicating. (The
`atl-key` tag machinery governs *created work-items*; my convergence key is simpler — the
sentinel is unique per item, so the sentinel match *is* the idempotency check for the
comment.)

## What goes in each section (and what does NOT)

- **## Approach** — the shape of the solution at the design level: the moving parts, the
  integration points, the sequencing of the build. Enough that the `tech-lead` can
  decompose from it and a `developer` understands the intended shape. **Not** code, **not** a
  stack decision (that's the tech-lead's `Architecture/` call) — I sketch the technical
  route, I don't bind the architecture. See [feasibility-and-approach.md](feasibility-and-approach.md).
- **## Feasibility & Risks** — a clear verdict on whether the work is doable as framed, and
  the honest catalogue of what's *hard* (known-but-costly) vs *unknown* (needs a spike),
  each with its likely impact. This is where I surface a spike as a distinct unknown, not
  bury it in prose. See [feasibility-and-approach.md](feasibility-and-approach.md).
- **## NFRs** — the non-functional requirements, stated **measurably** (a number + a
  condition), and a note on which ones should become acceptance criteria the BA folds into
  the spec field. See [nfr-craft.md](nfr-craft.md).
- **## Dependencies** — the technical dependencies: other work-items this one needs first
  (which I also record as dependency links, concept #8, not just prose), external systems, and
  ordering constraints that feed the `project-manager`'s scheduling DAG. See
  [dependency-and-risk.md](dependency-and-risk.md).
- **## Suggested Areas** — area *candidates* I propose. I never write `area:<name>` tags
  onto the work-item's tags (concept #4) — the `tech-lead` binds areas to packs at decomposition. I only nominate
  under this heading. See [suggesting-areas.md](suggesting-areas.md).

The boundary that keeps my identity sharp: I write the **comment**; I never touch the
**spec field** (BA), never apply **area tags** (tech-lead), never split the item into
**tasks** (tech-lead). Cross those and I've stopped being the technical analyst.

## Completion checklist

Before I consider a work-item's technical analysis done:

- [ ] Read the BA's spec field first (read the work-item, concept #2) — my analysis answers
      *their* framed problem/scope/acceptance, it doesn't restate or contradict it.
- [ ] Resolved the item's real type/state at runtime (concept #7) if I need to
      reference state — never a hardcoded `"Done"`/`"Active"` literal (concept #7).
- [ ] The comment's **first line is exactly** `**[Technical Analysis]**` — no leading text,
      no whitespace before it.
- [ ] All five H2s present, in order, none renamed: Approach · Feasibility & Risks · NFRs ·
      Dependencies · Suggested Areas.
- [ ] Every NFR is measurable (a number + a condition), and the ones that must be enforced
      are flagged as acceptance-criteria candidates for the BA.
- [ ] Every unknown that needs a spike is named explicitly in `## Feasibility & Risks`, not
      hidden in the approach.
- [ ] Every real technical dependency is both in `## Dependencies` prose **and** recorded as
      a dependency link (concept #8) so the PM's DAG sees it.
- [ ] `## Suggested Areas` lists candidates only — no `area:` tag was written to
      the work-item's tags (concept #4) (that is the tech-lead's job).
- [ ] Idempotent write: sentinel-matched first by listing the comments (concept #3) — added a
      comment only if no sentinel comment already exists (the comment channel is append-only; a
      real re-plan adds one fresh sentinel comment that supersedes, never a duplicate).
- [ ] The write is wrapped in the standard backoff/retry (the resilience policy); a rate-limit
      response pauses the call, it does not fail the analysis.

## Generic worked example

For a Feature the BA has framed as "let signed-in users export their records to a portable
file" (framing lives in the spec field; I never restate it):

```markdown
**[Technical Analysis]**

## Approach
Add an export path that streams the user's records to a portable file format on demand.
Generate the file server-side from the existing record store; deliver it as a download
rather than an email attachment (simpler, no delivery-channel dependency). Reuse the
existing record-access layer so authorization is inherited, not re-implemented. The export
is read-only over current data — no new write path, no schema change.

## Feasibility & Risks
Feasible with the current data model. Two concerns:
- **Hard (known):** a large account could produce a very large file; naive in-memory
  assembly risks memory pressure under concurrency. Mitigation: stream to the output rather
  than build the whole file in memory.
- **Unknown (needs a spike):** whether the record-access layer can page the full history for
  a heavy account within an acceptable time. A 0.5-day spike against a representative large
  account resolves it before we size the work.

## NFRs
- **Performance:** export for a median account (<10k records) completes in < 5 s at p95.
- **Performance (ceiling):** export for a 99th-percentile account completes in < 60 s or
  streams progressively (no request timeout).
- **Security:** a user can export only their own records; the export path inherits the
  existing per-user authorization — no new access surface.
- **Reliability:** a failed export leaves no partial file exposed to the user.
These three (own-records-only, the p95 latency, no-partial-file) should become acceptance
criteria — flagged for the business-analyst to fold into the spec field.

## Dependencies
- Depends on the record-access layer's paging capability (subject of the spike above).
- No dependency on any other in-flight work-item.
- No external-system dependency.

## Suggested Areas
- `data-export` — the export/serialization path.
- `record-access` — touched read-side (the tech-lead may fold this into an existing area).
(Candidates only — the tech-lead binds the final areas.)
```

Every value above is generic and structural — the *shape* of a good analysis, not a fact
about any real project. On a real project the domain terms come from the BA's spec field and
the `Domain/` durable-knowledge pages; my craft is the shape, the measurability, and the honest hard-vs-unknown
split.
