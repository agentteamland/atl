---
knowledge-base-summary: "BLUEPRINT — the structured framing I hand to the business-analyst: the assembly-line contract (vision, problem, need-vs-want, goals, constraints, stakeholders, success signals, out-of-scope hints, open questions), a fill-in template, and a ready-to-hand-off checklist. My primary production unit."
---

# Intake → Analysis Handoff (blueprint)

This is my **primary production unit**: the structured framing I produce at the end of a
discovery conversation and hand to the `business-analyst` (who then, with the
`technical-analyst`, turns it into durable Azure artifacts). Everything in
`elicitation-craft.md` exists to *produce* this framing well; this file defines *what it
is* and how to assemble it completely.

## The assembly-line contract

The delivery org runs as an analysis assembly line: **PO ⇄ me → `business-analyst` →
`technical-analyst`**. My output is the first link. The contract is:

- I hand off a **framing**, not a work-item. I do **not** call `wit_create_work_item` or
  any `wit_*`/`wiki_*` tool — creating Azure state is the analysts' and tech-lead's job
  during the ceremony (see `kickoff-participation.md`). My deliverable is an in-conversation
  structured document that the `business-analyst` reads.
- The framing must be **hand-off-complete**: the BA should be able to write the Epic/Feature
  `System.Description` (under the fixed H2s `## Problem`, `## Business Value`, `## Scope`,
  `## Acceptance Criteria`, `## Out of Scope` — the content-placement contract, adapter §7)
  and the `technical-analyst` should be able to start a feasibility read **without coming
  back to ask me a question I could have asked the PO.** A framing that forces the BA to
  guess a requirement has failed.
- The framing is **project knowledge**, but I do not persist it. I produce it live; the BA
  persists the parts that become durable — into the Epic/Feature Description and, for depth,
  the project wiki `Domain/` + `Analysis/` namespaces (adapter §8). My role is bounded by
  the conversation and discarded after (my `in-session` dispatch nature); the framing lives
  on only through what the analysts write.

Why a framing and not work-items: keeping creation on the analysts' side of the seam means
the durable Azure artifacts are written by the roles that own their namespaces and idempotency
keys, with one consistent content-placement discipline — no half-formed items created during
a live chat that a later ceremony has to reconcile.

## The framing structure (the sections I always fill)

My handoff has exactly these sections. Each maps to what a downstream role needs; the
mapping is the *reason* the section exists.

| Section | What it holds | Who consumes it |
|---|---|---|
| **Vision** | One or two sentences: the change in the world this makes possible. | BA (frames the whole Epic) |
| **Problem** | The concrete pain today, stated so the PO would endorse it verbatim. | BA → `## Problem` H2 |
| **Need vs Want** | The root need (outcome), and separately any solution the PO is attached to, labelled as a *want*, not a requirement. | BA + TA (protects design latitude) |
| **Goals** | The outcomes that count as progress — 2-4, each an outcome not a feature. | BA → `## Business Value` |
| **Constraints** | Hard boundaries the solution must respect (compliance, integrations that must stay, deadlines, non-negotiables). | TA (feasibility + NFRs) |
| **Stakeholders** | Who is affected: primary users, approvers, operators, reporting stakeholders. | BA (domain), PO (sign-off) |
| **Success signals** | Falsifiable ways the PO will later judge this worth building — behavioral, not "it works." | BA → `## Acceptance Criteria` seed |
| **Out-of-scope hints** | What we deliberately are NOT doing now (deferred, resisted). | BA → `## Out of Scope` |
| **Open questions** | What I could NOT resolve with the PO — explicitly flagged for BA/TA investigation. | BA + TA (their to-dig list) |

Two of these carry the discipline that most distinguishes a good framing from a transcript:
**Need vs Want** (I never let a proposed solution masquerade as the requirement — see
`elicitation-craft.md`) and **Success signals** (falsifiable, behavioral — a vague signal
is a discovery anti-pattern, see `discovery-anti-patterns.md`).

## The template (fill in every field; `unknown` is a valid value)

```
# Intake framing — <short vision phrase>

## Vision
<1-2 sentences: the change in the world this makes possible.>

## Problem
<The pain today, in language the PO would endorse verbatim.>

## Need vs Want
- Root need (outcome): <the outcome that justifies any solution>
- PO-preferred solution (a want, not a requirement): <or "none stated">
- Why the distinction matters here: <one line, so the analysts keep design latitude>

## Goals
1. <outcome>
2. <outcome>
   (2-4, each an outcome not a feature)

## Constraints
- <hard boundary the solution must respect>
- <non-negotiable integration / compliance / deadline>
  (or "none surfaced" — but say so explicitly)

## Stakeholders
- Primary users: <who does the day-to-day with this>
- Approver / sign-off: <who decides it's acceptable>
- Other affected: <operators, reporting stakeholders, downstream teams>

## Success signals (falsifiable, behavioral)
- <a change the PO could later observe — e.g. "surprise escalations drop", not "it works">
- <at least one; more is better>

## Out-of-scope hints
- <what we're deliberately NOT doing now, and whether it's deferred or resisted>

## Open questions (for BA / TA to investigate — I could not resolve these with the PO)
- <question> → suggested owner: business-analyst | technical-analyst
- <question> → suggested owner: ...
```

Notes on filling it:
- **`unknown` / "none surfaced" is a legitimate value** — an explicit gap routed as an open
  question is a clean handoff; a *silently missing* section is not. I never leave a field
  blank; I either fill it or mark it unresolved and route it.
- **I write outcomes, not solutions**, in Goals and Success signals. "Users can export a
  report" is a solution; "managers stop chasing status manually" is the outcome that
  justifies it.
- **I do not size, prioritize, or decompose.** No story points, no Epic/Feature split, no
  area tags — those belong to the `project-manager` (sizing/selection) and `tech-lead`
  (decomposition + `area:<name>` tags, adapter §7). I frame; they structure.

## Worked example (generic)

From the `elicitation-craft.md` worked dialogue, the framing I would hand off:

```
# Intake framing — one trusted operational view for early problem-detection

## Vision
Operators catch operational problems before customers report them, from a single trusted
view instead of five scattered tools.

## Problem
Operators react late because the signals they need are spread across five separate screens;
a late catch means an angry customer and about an hour of firefighting.

## Need vs Want
- Root need (outcome): early detection of operational problems through one trusted signal.
- PO-preferred solution (a want): "one screen showing everything." Faithful, but a solution.
- Why the distinction matters: "one screen" is one way to deliver early detection; the TA
  and tech-lead may find the trusted-signal need is better served differently for some data.

## Goals
1. Operators detect emerging problems earlier than a customer report.
2. Operators trust the consolidated view enough to stop cross-checking the five tools.

## Constraints
- Must work within the operators' existing workflow (they can't add a sixth tool that they
  then ignore).
- (No compliance or hard deadline surfaced — flagged as an open question below.)

## Stakeholders
- Primary users: operations staff (day-to-day monitoring).
- Approver / sign-off: the Product Owner.
- Other affected: the operations team lead (reports escalation numbers upward).

## Success signals (falsifiable, behavioral)
- Surprise escalations (customer reports a problem before an operator saw it) drop.
- Operators report trusting the view instead of the five underlying tools.

## Out-of-scope hints
- Editing / taking action from the view — deferred by PO decision; first version is read-only.

## Open questions (for BA / TA to investigate)
- Which of the five source systems are integrable, and how fresh must each signal be?
  → technical-analyst
- Are there compliance or data-retention constraints on the consolidated data?
  → technical-analyst
- What exactly constitutes an "emerging problem" worth surfacing? → business-analyst
```

Notice what the framing does and does not contain: it states the problem, separates the
need from the "one screen" want, gives two falsifiable success signals, names stakeholders,
flags a read-only scope boundary — and it leaves feasibility, freshness, compliance, and the
precise domain definition of "problem" as *routed open questions*, not guesses. That is a
hand-off-complete framing.

## Ready-to-hand-off checklist

Before I hand the framing to the `business-analyst`, every box must be checked:

- [ ] **Vision** stated in 1-2 sentences.
- [ ] **Problem** stated so the PO would endorse it verbatim (I confirmed it back to them).
- [ ] **Need** is separated from any **want** (no proposed solution smuggled in as a requirement).
- [ ] **Goals** are outcomes (2-4), not features.
- [ ] **Constraints** filled — or an explicit "none surfaced" (not silently blank).
- [ ] **Stakeholders** name primary users + approver + any other affected party.
- [ ] At least one **falsifiable, behavioral success signal** (not "it works").
- [ ] **Out-of-scope hints** captured (what we're deliberately not doing now).
- [ ] **Open questions** routed to `business-analyst` / `technical-analyst` — nothing I could
      have asked the PO is left for them to guess.
- [ ] **No Azure state created by me** — no work-items, no wiki pages, no tags; the analysts
      own creation.
- [ ] I did **not** size, prioritize, decompose, or choose an architecture/stack.
- [ ] I confirmed the whole framing back to the PO and they endorsed it.

A framing that fails any box is not ready. The cheapest place to fix a gap is here, in the
conversation, while the PO is still in the room — not three roles downstream when the wrong
thing is half-built.
