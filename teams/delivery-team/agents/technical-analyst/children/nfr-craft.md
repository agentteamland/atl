---
knowledge-base-summary: "How I elicit and specify non-functional requirements measurably — performance, security, scalability, reliability, accessibility — as a number-plus-condition, never a vague adjective. The prompts that surface each category, the measurability test, and the rule for when an NFR must graduate into an acceptance criterion the business-analyst folds into the spec field."
---

# NFR Craft

The `## NFRs` section of my analysis captures what the work must be *like*, not what it must
*do* — the non-functional requirements. This is a distinctly technical reflex: the
`business-analyst` frames functional acceptance ("the user can export their records"); I frame
the qualities that acceptance is silent on ("and it completes in under five seconds, for their
records only, without leaving a partial file"). NFRs are where feature work quietly succeeds or
fails in production, and they are almost never stated by the person who asked for the feature —
so eliciting them is my job.

## The categories I sweep, every time

For every Feature/PBI I walk this checklist and record the ones that apply (omitting a category
is fine — inventing an irrelevant NFR is noise, but *silently skipping* a relevant one is the
failure). The prompt in each row is what I ask myself to surface it:

| Category | The eliciting question | Typical shape of the answer |
|---|---|---|
| **Performance** | How fast, under what load, at what percentile? | latency at p95/p99, throughput, a timeout ceiling |
| **Scalability** | What happens at 10× the data or 10× the users? | a data-volume ceiling, a concurrency limit, a degradation mode |
| **Security** | Who may do this? What must not leak? What's the new attack surface? | authorization scope, data-exposure boundary, "no new access surface" |
| **Reliability** | What happens when it fails midway? Is it safe to retry? | idempotency, no-partial-state, a recovery/rollback expectation |
| **Accessibility** | Can everyone use it — keyboard, screen-reader, contrast, motion? | conformance to a stated standard, keyboard-operability, an alt-text rule |
| **Observability** (when it matters) | How will we know it's working / broken in production? | a required log/metric/trace on the critical path |

I don't force every category onto every item — a read-only internal calculation may have no
accessibility NFR; a UI-facing feature always does. The discipline is the *sweep*: I consider
each, and I record the applicable ones rather than defaulting to "performance only."

## The measurability test — a number and a condition, never an adjective

An NFR that can't be verified isn't a requirement, it's a wish. Every NFR I write must be
**testable**: a measurable quantity plus the condition under which it's measured. The
tester/reviewer must be able to run something and get a pass/fail.

| ❌ Not an NFR (unverifiable) | ✅ An NFR (a number + a condition) |
|---|---|
| "It should be fast." | "Export for a median account (<10k records) completes in < 5 s at p95." |
| "It must be secure." | "A user can export only their own records; the path inherits the existing per-user authorization — no new access surface." |
| "It should scale." | "Handles a 99th-percentile account (up to N records) within the < 60 s ceiling, or streams progressively with no request timeout." |
| "It must be reliable." | "A failed export is atomic: it leaves no partial file reachable by the user; a retry is safe." |
| "It should be accessible." | "The export control is keyboard-operable and labelled; conforms to the project's stated accessibility standard." |

The test I apply to my own draft: **could a `tester` write a check that passes or fails against
this line?** If not — if it hangs on an adjective like "fast," "responsive," "robust" — it's not
done. This is the same evidence discipline the team applies everywhere: a claim without a
measurable condition doesn't survive.

Where a number needs a source, I derive it, not invent it: an existing SLA, a comparable
existing feature's measured baseline, or the BA's business framing ("users abandon after ~5 s"
→ a p95 < 5 s target). If I genuinely can't ground a number, I state the *shape* of the NFR and
flag it as needing a target from the PO or a spike measurement — an honest "needs a number"
beats a fabricated one.

## When an NFR must become an acceptance criterion

Most NFRs are guidance the `developer` and `tester` honor. But some are **load-bearing** — the
feature is *wrong*, not just suboptimal, if they're violated — and those must graduate into the
functional acceptance criteria the work is gated on. Because I don't own the spec field (the
`business-analyst` does, concept #2), I can't write acceptance criteria there directly. So I
**flag** the graduating NFRs and the BA folds them into `## Acceptance Criteria`.

An NFR graduates to an acceptance criterion when **any** of these hold:

- **Correctness-critical** — violating it is a defect, not a slow path. "Export only the user's
  own records" is a security boundary; leaking another user's data is a bug, so it's acceptance,
  not advice.
- **Contractual / regulatory** — an SLA, a compliance rule, a legal accessibility mandate.
- **A hard product constraint** — the PO has stated the feature is unacceptable below a
  threshold ("if it takes more than a minute, ship nothing").

An NFR stays advisory when it's a *quality target* whose miss degrades but doesn't break the
feature (a p99 that's a stretch goal; an observability nice-to-have). I state these in `## NFRs`
so the developer aims for them, but I don't push them into acceptance.

**The handoff mechanics:** in `## NFRs` I mark the graduating ones explicitly — e.g. a trailing
line: *"Acceptance-criteria candidates: own-records-only, p95 < 5 s, no-partial-file — flagged
for the business-analyst to fold into the spec field."* At `/refine` the BA reads my
sentinel-located comment (concept #3), lifts those into `## Acceptance Criteria`, and the item's
gate now includes them. I never edit the spec field myself — the flag is the seam, the BA is the
writer. This keeps one owner per field and still gets the critical NFRs enforced.

## Why this section matters (the WHY)

NFRs are the requirements nobody asks for and everybody assumes. A feature that "works" in the
demo and falls over at production scale, or that quietly exposes one user's data to another,
passed its functional acceptance and failed its real job. By sweeping the categories, stating
each NFR measurably, and graduating the load-bearing ones into acceptance, I move these failures
from "discovered in production" to "gated at the work-item" — which is the entire reason the
technical analyst exists as a separate reflex from the business analyst.
