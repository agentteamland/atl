---
knowledge-base-summary: "How I design a test strategy for one work-unit: risk-based prioritization (test where a failure hurts most × is most likely), coverage thinking against the acceptance criteria, the test pyramid applied at the unit level (many fast checks, few slow end-to-end ones), and the discipline of what to test vs what to trust (the pack/framework/library boundary)."
---

# Test Strategy

Before I run anything, I decide *what* to test, *at what level*, and *in what order*. A work-unit
is small — one task — so the strategy is not a document, it is a short, deliberate plan I form in
my head and record in my verdict comment. The point is to spend my limited effort where a failure
would hurt most and is most likely, not to test everything equally (that's slow) or to test the
happy path only (that's the developer's self-test again, adding nothing).

## The spec is the acceptance criteria — start there

My spec is not the code; it is the `## Acceptance Criteria` list in the work-item's
**spec field** (concept #2), plus the constraints the `technical-analyst` recorded under the
`## NFRs` heading of the `**[Technical Analysis]**` comment. I read these *before* I look at how the
developer implemented anything, so my expectations are derived from intent, not from the code (code
I read to find seams to attack — not to learn what "correct" means).

For each acceptance criterion I ask: **is there a test that would fail if this criterion were
violated?** If not, that criterion is uncovered and I add one. `## Out of Scope` is the mirror
image: it bounds what I must *not* flag — behavior explicitly excluded is not a defect, and flagging
it wastes the developer's re-work cycle.

## Risk-based prioritization — the ranking that decides effort

I rank each thing-to-verify by **risk = impact × likelihood**, and I spend my effort top-down:

- **Impact** — how bad is it if this is wrong? A miscomputed money value, a lost user record, a
  security boundary, an irreversible action: high impact. A cosmetic label: low.
- **Likelihood** — how probable is a bug here? New/changed code is likelier to be wrong than code
  the change didn't touch. Complex branching, concurrency, and boundary arithmetic are likelier
  than a straight-line pass-through. A seam the developer's self-test obviously exercised is less
  likely to hide a bug than one it obviously didn't.

The high-impact × high-likelihood cell gets my deepest probing; low × low may get a single
smoke-check or a documented "trusted, not tested." Writing this ranking down (in the verdict) makes
my coverage decisions auditable — a reviewer can see *why* I probed X hard and Y lightly.

**Worked example (generic).** A work-unit adds a "transfer an amount between two accounts" action
with criteria: (a) the amount moves, (b) you can't transfer more than the source holds, (c)
concurrent transfers don't corrupt balances.

| Verify | Impact | Likelihood | Rank |
|---|---|---|---|
| Balance conservation under concurrency (c) | high (data corruption) | high (concurrency is bug-dense) | **1** |
| Over-balance transfer rejected (b) | high (money invented) | medium (a boundary) | **2** |
| A valid transfer moves the amount (a) | high | low (the happy path, self-tested) | 3 |
| The success message wording | low | low | 4 |

I spend most of my time on rank 1–2 (the concurrency race and the boundary), smoke-check rank 3
(the developer already self-tested it), and glance at rank 4. That is risk-based effort, not
uniform effort.

## Coverage thinking — what "covered" means

Coverage is not a line-percentage number; it is **whether the behaviors that matter each have a
test that would catch their breakage**. I think in terms of:

- **Criteria coverage** — every acceptance criterion has at least one verification. This is
  non-negotiable; an uncovered criterion is a hole in the gate.
- **Path coverage** — the happy path, plus each error/rejection path the change introduces.
- **Boundary coverage** — the edges of each input range (see
  [`edge-case-and-regression.md`](edge-case-and-regression.md)).
- **Regression coverage** — the behaviors *near* the change that it could have broken, even though
  they aren't in the acceptance criteria.

A criterion "covered" by a test that would pass even when the criterion is violated is not covered
— it's theatre. When I'm unsure a test really binds the behavior, I mentally (or actually) break the
code and confirm the test goes red; a test that stays green when I sabotage the thing it claims to
protect is worthless and I replace it.

## The test pyramid, at the unit level

Even within one small work-unit, the pyramid still applies to *how* I verify:

- **Many fast, narrow checks at the bottom** — unit/logic-level assertions on the changed functions
  and their branches. Cheap, deterministic, run at full concurrency, no shared resource. This is
  where most of my coverage lives.
- **Fewer integration checks in the middle** — the changed unit talking to its immediate
  collaborators (a data layer, an adapter), exercised through a real-ish seam.
- **The fewest, slowest end-to-end checks at the top** — a web surface driven through the
  preview/chrome-devtools MCP, or a mobile surface driven through the serialized emulator lease
  (see [`mobile-and-web-surfaces.md`](mobile-and-web-surfaces.md)). These are expensive (the mobile
  one *serializes* against a single-slot lease), so I use them to confirm the criteria end-to-end,
  not to exhaustively probe logic that a bottom-level check covers far more cheaply.

**Why the shape matters here specifically.** The top of the pyramid contends for the shared
mobile-emulator slot; the more I push exhaustive logic-probing up to the end-to-end level, the more
I serialize the whole team behind that slot. Keeping logic-probing at the bottom (full concurrency)
and reserving the emulator for genuine end-to-end criteria confirmation is not just classic pyramid
hygiene — it's how the team's throughput survives.

## What to test vs what to trust — the boundary

I do not test everything the code touches; I test **what this change is responsible for** and
**trust what it stands on**:

- **Trust the pack/framework/library.** The stack's runtime, the framework's routing, a mature
  third-party library's documented behavior — these have their own tests and I don't re-verify them.
  (The stack-specific knowledge is a stone-#5 `packs/<area>/` pack the *developer* loads, not
  something I bake in; my strategy is stack-agnostic, so "trust the framework" is a principle, not a
  framework-specific rule.)
- **Test the seam the change owns.** The new logic, the new branch, the new boundary, the new
  integration point — that's mine.
- **Distrust the boundary between trusted and owned.** The bug is rarely inside the trusted library
  or inside the well-worn logic; it's at the *seam* — the wrong argument passed to the trusted call,
  the unhandled error the library can raise, the assumption about the framework's behavior that
  doesn't hold. That seam is where I aim.

The discipline that keeps this honest: when I decide "trusted, not tested," I say so in the verdict.
A silent decision to skip a surface is indistinguishable from an oversight; an explicit "I trusted
the framework's X here" lets a reviewer disagree if my trust was misplaced.

## Recording the strategy

The strategy lives in my verdict comment on the work-item, in one or two lines: the risk ranking,
what I covered at each pyramid level, and what I deliberately trusted rather than tested. This is
project-agnostic *craft* — but the *instance* of it (the actual criteria, the actual risks) is
project-specific, so it goes in the work-item comment, **not** in this file and **not** in the
durable-knowledge store (I don't write it — concept #9). A durable *lesson* about strategy itself ("for
this class of change, concurrency is always rank 1") routes to this child via `/drain`, generalized.
