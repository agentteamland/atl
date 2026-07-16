---
knowledge-base-summary: "How I write testable, unambiguous acceptance criteria for the `## Acceptance Criteria` H2 — Given/When/Then and checklist styles, the INVEST-style qualities good AC carry, how they feed the tester's coverage and the tech-lead's decomposition, and the common AC smells I refuse to ship."
---

# Acceptance-Criteria Craft

Acceptance criteria (AC) are the single most load-bearing part of my spec field. `## Problem`
and `## Business Value` motivate the work; the AC are what actually gets **verified**. The
`tester` turns each criterion into a pass/fail check; the `tech-lead` sizes and decomposes
against them; the PO reads them to confirm we understood the ask. A vague criterion silently
weakens all three consumers, so I hold a hard bar: **every criterion must be independently
testable and unambiguous, or it does not ship.**

## What an acceptance criterion is (and is not)

A criterion is a **binary, business-observable condition** that decides whether the
Epic/Feature delivered its intent. It is:

- **Testable** — someone could execute it and get an unambiguous pass or fail, with no
  judgement call.
- **Business-observable** — phrased in terms of what a user or the system does/shows, not how
  it is built.
- **Unambiguous** — one reading only. No "fast", "easy", "properly", "as expected" — words that
  mean different things to different readers.
- **Independent** — each criterion stands on its own; failing one does not entangle another.

It is **not** a task ("build the form"), **not** a design ("use a modal"), and **not** a
non-functional target owned elsewhere (performance/security budgets are the `technical-analyst`'s
NFRs, in their comment — I reference a business-observable *effect* if one exists, I do not
re-author the NFR).

## Two styles — pick per criterion, mix freely

### Given / When / Then (behavioural)
The default for anything with a precondition, an action, and an expected outcome. It forces me
to state all three, which is where ambiguity usually hides.

```
Given <a specific, testable starting state>
When  <a single, concrete action>
Then  <a single, observable, verifiable outcome>
```

Rules that keep it clean: one Given-state, one When-action, one Then-outcome per criterion (a
second "And Then…" that tests a *different* outcome should be its own criterion — it keeps each
line independently pass/fail). The Given must be *reachable and specific* ("a signed-in user
with no saved address", not "a user"). The Then must be *observable* ("a confirmation is shown"
/ "the value persists after reload"), never internal ("the record is updated" — updated where,
observed how?).

### Checklist (declarative)
For a set of independent conditions that don't each need a scenario — constraints, presence
checks, boundary rules. Each bullet is one verifiable statement.

```
- The list shows at most <N> items per page.
- An empty result shows the empty-state message, not a blank area.
- Submitting with a required field missing is rejected with a field-specific reason.
```

Use whichever style makes the criterion **clearest to verify** — a boundary rule reads better as
a checklist line than as a forced Given/When/Then. Mixing both in one spec field is normal and
good.

## The qualities I check every criterion against

- **Verifiable** — can the `tester` write a pass/fail check from this sentence alone, with no
  clarifying question? If not, rewrite.
- **Atomic** — one condition per criterion. A compound criterion hides a failure (which half
  failed?).
- **Positive AND negative** — the happy path *and* the rejection/edge path. A Feature with only
  happy-path AC is under-specified; the invalid-input, empty, and boundary cases are where real
  behaviour is pinned down. Missing negatives is the single most common gap the `tester` finds.
- **Boundary-aware** — where a limit exists (max length, count, range), name the exact boundary
  and what happens at, below, and above it.
- **Stack-free** — no framework, no UI-widget name, no data-store detail. "A confirmation is
  shown", not "a toast appears"; "the change persists", not "the row is committed".

## How good AC feed the downstream roles

- **The `tester`** builds Level-2 verification (strategy / edge / regression) directly from my
  AC. Each criterion becomes at least one test case; each negative criterion becomes an
  edge-case check. Weak AC ⇒ thin coverage ⇒ escaped defects.
- **The `tech-lead`** decomposes and sizes against the AC — a Feature's AC bound its task
  breakdown and its "done". Ambiguous AC inflate estimates (padding for unknowns) and cause
  re-work when the ambiguity resolves the wrong way.
- **The PO** approves at sprint-review by reading whether the AC were met — so the AC are also
  the acceptance contract with the human. If the PO can't sign off from the AC alone, the AC
  weren't specific enough.

Because the AC are consumed from a **fixed location** (`## Acceptance Criteria` in the
**spec field**, concept #2), every consumer parses them the same way — another reason the
heading is a contract, not a style choice.

## Common AC smells (I refuse these)

| Smell | Why it's bad | Fix |
|---|---|---|
| "works correctly / as expected / properly" | Not testable — no defined outcome. | State the exact observable outcome. |
| "fast / quick / responsive" | Subjective; also usually an NFR, not an AC. | If business-observable, name the effect; otherwise route to the `technical-analyst`'s NFRs. |
| Compound "…and… and…" in one criterion | A partial pass is unrepresentable. | Split into one criterion per condition. |
| Only happy-path criteria | Under-specified; the tester will surface the gaps late. | Add the invalid, empty, and boundary cases. |
| Solution baked in ("use a dropdown") | Constrains the tech-lead's design; not a business condition. | State the business need; let design own the how. |
| Unreachable Given ("a user with an expired legacy token") | Can't be set up ⇒ can't be tested. | Use a Given the tester can actually construct. |
| Restating the title as a criterion | Adds no verifiable condition. | Delete it or make it a concrete, observable check. |

## The discipline, and the why

I would rather ship **fewer, sharper** criteria than a long list of soft ones. Each criterion is
a promise the whole downstream chain relies on: the tester's coverage, the tech-lead's estimate,
the PO's sign-off. A soft criterion doesn't just under-specify — it *misleads*, because it looks
like a covered case while covering nothing. When I can't make a criterion testable, that is a
signal the requirement itself is unclear, and I resolve the ambiguity with the PO (through the
ceremony) rather than paper over it with vague words.
