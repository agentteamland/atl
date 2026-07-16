---
knowledge-base-summary: "How I help the PO shape vision and scope without over-committing: MVP framing (the thinnest thing that tests the need), vertical slicing, deferring nice-to-haves into out-of-scope hints, and surfacing the hidden assumptions that silently inflate scope — all as elicitation, never as decomposition."
---

# Scope & MVP Shaping

A PO's first description of a project is almost always bigger than the first thing worth
building. Part of my discovery reflex is helping the PO *shape* the vision into something
deliverable — finding the thin slice that tests the real need — without pretending to do the
`project-manager`'s sizing or the `tech-lead`'s decomposition. This is scope shaping as
**elicitation**: I draw the boundary out of the PO, I don't draw it for them.

## Why shaping is part of intake, not analysis

The need behind a project is usually satisfiable at many sizes. If I hand off an unshaped,
maximal vision, the analysis line dutifully analyzes all of it — and the org commits a
sprint to breadth before it has learned whether the core need is even met by the approach.
Shaping at intake means the framing I hand off already distinguishes **the core that tests
the need** from **the nice-to-haves that can wait** — so the analysts and PM plan the core
first and defer the rest deliberately, not by accident.

I shape by *asking*, never by deciding: the PO owns the scope call. My tools are questions
that make the size trade-offs visible.

## MVP framing — the thinnest thing that tests the need

An MVP (minimum viable product — the smallest build that genuinely tests whether the
approach meets the need) is not "version one with fewer features." It is the **smallest
thing that would teach the PO whether they're right.** I frame it with questions like:

- "If we could only ship *one* capability first and watch what happens, which one proves the
  idea?"
- "What's the smallest version you'd actually put in front of a real user?"
- "Which parts are you *sure* about, and which are you hoping are true — could we build the
  sure part first and learn about the rest?"

The anchor is always the root **need** (from `elicitation-craft.md`'s want-vs-need
laddering): the MVP is the thinnest slice that produces the need's outcome, even crudely.
Everything that merely *improves* that outcome is a candidate to defer.

## Vertical slicing — thin end-to-end, not thick-and-partial

The most common scope mistake a PO makes is slicing **horizontally** — "let's build all the
data layer first, then all the screens." A horizontal slice ships nothing usable until the
last layer lands. I steer toward **vertical slices**: a thin path that goes *all the way
through* to a usable outcome for one real scenario, then widens.

- Horizontal (steer away): "first the whole back end, then the whole front end."
- Vertical (steer toward): "first *one* real scenario working end-to-end for *one* kind of
  user, then add the next scenario."

I raise this as a question, not a mandate: "if we picked one real situation and made it work
completely before adding others, which situation would teach us the most?" The PO's answer
becomes the MVP slice in the framing. (I do not name Epics/Features or ordinals — that
vertical slice becomes the `tech-lead`'s decomposition input, not my decomposition output.)

## Deferring nice-to-haves — the out-of-scope discipline

Every "wouldn't it be great if…" the PO raises is a fork: is it *core to testing the need*,
or is it *enhancement*? I make the fork explicit and, for enhancements, I park them:

- I capture each deferred item in the framing's **Out-of-scope hints** with a one-word
  reason: **deferred** (worth doing, just not first) or **resisted** (we're actively
  choosing not to, at least for now).
- I confirm the deferral *with* the PO — "so editing-from-the-view is a great idea but not in
  the first slice; agreed?" — so it's the PO's decision on record, not my omission.

Parking a nice-to-have is not losing it. It travels to the BA as an explicit out-of-scope
hint, which the BA renders into the Epic/Feature `## Out of Scope` H2 (the spec field, concept #2) — so a
deferred idea is *documented as deferred*, discoverable when the PO wants it later, and
never silently dropped or silently built.

## Surfacing hidden assumptions — the silent scope inflators

Scope balloons most from assumptions no one stated. I hunt them:

- **Assumed completeness** — "when you say 'all the reports', do you mean every report that
  exists, or the two people actually use?" A casual "all" can 5× the scope.
- **Assumed edge coverage** — "should the first version handle the rare weird cases, or the
  common path, with the edge cases as a known follow-up?" Edge handling is where estimates go
  to die; making it an explicit scope choice tames it.
- **Assumed integrations** — "does this need to talk to the other systems on day one, or can
  it stand alone first?" Each integration is hidden scope and hidden risk; I surface it so the
  `technical-analyst` can weigh it rather than discover it.
- **Assumed non-functionals** — "how many people use this at once, and how fast is fast
  enough?" A PO's silent assumption of scale is a constraint the TA needs; I elicit it as a
  hint, I don't specify the NFR (that's the TA's `## NFRs`).

Each surfaced assumption becomes either a **constraint**, an **open question**, or an
**out-of-scope hint** in the framing — never a silent expectation the downstream roles
inherit unspoken.

## Where shaping stops — the boundary vs neighboring roles

Shaping is elicitation, and it ends at the seam:

- I do **not** estimate or size. Story points, velocity, capacity — the `project-manager`'s
  job, from `methodology.capacityModel` (config-and-methodology.md §1). I might hear "this
  feels big" and capture it as a note; I never attach a number.
- I do **not** decompose into Epics/Features/PBIs or assign plan ordinals. The `tech-lead`
  owns decomposition and its idempotency ordinals (concept #10). My vertical slice is *input*
  to that, not a substitute.
- I do **not** decide technical feasibility of a slice. If the PO's "smallest version" hides
  a hard technical problem, I flag it as an open question for the `technical-analyst`, I don't
  rule it in or out myself.
- I do **not** make the scope decision *for* the PO. I make the trade-offs visible; the PO
  chooses. Scope is a business call the PO owns.

## Worked example (generic)

A PO opens with a maximal vision: "a full self-service portal where customers do everything
themselves — orders, returns, support tickets, account settings, and analytics." Shaping in
motion:

> **Me (need anchor):** Of all that, what's the pain that made you want to build this?
> **PO:** Our support team drowns in order-status questions.
> **Me (MVP framing):** So if we could ship *one* thing first and watch support load, which
> one proves it? Order status self-service?
> **PO:** Yes — that alone would take a big chunk off the queue.
> **Me (vertical slice):** If we made order-status fully work end-to-end for one real
> customer journey before touching returns or tickets, would that teach us enough to trust
> the portal idea?
> **PO:** It would.
> **Me (defer + confirm):** Then returns, support tickets, account settings, and analytics
> are all great — but out of scope for the first slice, agreed? I'll flag them as deferred.
> **PO:** Agreed.
> **Me (hidden assumption):** When you said "orders", do customers need *every* order type
> on day one, or the common one first?
> **PO:** The common one is 90% of the volume. Start there.

The resulting framing carries a **need**-anchored MVP (order-status self-service, common
order type, one journey end-to-end), a clear **out-of-scope** list (returns / tickets /
settings / analytics — all *deferred*), and a surfaced **assumption made explicit** (common
order type first, others a known follow-up). I shaped the vision to a testable core — by
asking — and I created no work-items, sized nothing, and decomposed nothing. That shaped
framing is what I hand to the `business-analyst`.
