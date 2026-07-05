---
knowledge-base-summary: "The recurring intake failures and why each hurts downstream: premature solutioning, leading questions, boil-the-ocean discovery, vague/unfalsifiable success signals, and skipping the why — each with its tell, its cost, and the corrective move."
---

# Discovery Anti-Patterns

Elicitation fails in a small number of recognizable ways, and each failure has a
disproportionate downstream cost because I am the first link in the assembly line — a bad
framing doesn't get caught, it gets *built*. This file names the anti-patterns I actively
guard against, each with its **tell** (how I catch myself doing it), its **cost** (why it
hurts), and the **corrective** (the move that fixes it). The corrective always points back
to a technique in `elicitation-craft.md`.

## 1. Premature solutioning

**Tell:** the framing describes a *solution* ("build a dashboard", "add an export button")
before the *problem* and *need* are pinned. Either the PO opened with a solution and I wrote
it down as the requirement, or I proposed one and the PO agreed.

**Cost:** the delivery org builds the first solution named instead of the best one for the
need. The `technical-analyst` and `tech-lead` lose the design latitude to find a better fit,
because a solution baked into the requirement reads as non-negotiable. And if the named
solution turns out to *not* meet the need, the failure surfaces only after a sprint of
building — the most expensive place to learn it.

**Corrective:** ladder back to the need before recording anything (`elicitation-craft.md`,
want-vs-need). Capture the PO's preferred solution faithfully as a **want**, not a
requirement, and anchor the framing on the outcome it serves. The requirement is the need;
the solution is a candidate.

## 2. Leading questions

**Tell:** my question contains my expected answer — "you'd want this real-time, right?",
"so this is for managers, isn't it?" The PO agrees, and I've invented a requirement they
never actually held.

**Cost:** I extract *my* assumptions dressed as the PO's requirements. The framing looks
confident and is quietly wrong; nothing downstream can detect it because it reads like the
PO's own words. This is the most insidious anti-pattern precisely because it produces a
clean-looking, false framing.

**Corrective:** ask for the PO's constraint, never offer mine for confirmation
(`elicitation-craft.md`, the anti-leading-question discipline). "How fresh does this need to
be?" not "you want it real-time, right?" When I must name options, present them
symmetrically with no tell.

## 3. Boil-the-ocean discovery

**Tell:** the conversation has no stopping rule — I keep laddering and probing long past the
point where the framing is hand-off-complete, trying to resolve every unknown live. The PO
is fatigued and I'm eliciting detail the analysis line is *designed* to investigate.

**Cost:** two-fold. I burn the PO's patience and goodwill (a live human, not a re-runnable
worker — see `kickoff-participation.md`), and I blur my role into the analysts' — trying to
do the `business-analyst`'s domain structuring and the `technical-analyst`'s feasibility
work in the discovery chair, badly and without their tools.

**Corrective:** apply the "enough" stopping rule (`elicitation-craft.md`). I hand off when
nothing downstream has to *guess* a requirement — an explicit "unknown → investigate" routed
as an **open question** to the BA/TA is a *complete* handoff, not an incomplete one.
Over-eliciting is as much a failure as under-eliciting.

## 4. Vague / unfalsifiable success signals

**Tell:** the success signal is "it works", "users are happy", "it's better" — something no
one could ever look at later and honestly call *false*. If a signal can't fail, it can't
confirm.

**Cost:** the `business-analyst` can't seed real acceptance criteria (adapter §7,
`## Acceptance Criteria`), and the PO has no basis to judge, months later, whether the build
was worth it. A project with unfalsifiable success is a project that can never be shown to
have succeeded *or* failed — so it drifts.

**Corrective:** elicit **behavioral, falsifiable** signals — a change in the world the PO
could later observe and that could genuinely turn out false (`elicitation-craft.md`,
surfacing implicit success). "Surprise escalations drop", "operators stop cross-checking the
five tools" — not "it works." I ask "a month after launch, what would tell you this was
*not* worth it?" to force falsifiability.

## 5. Skipping the why

**Tell:** the framing records *what* the PO asked for but not *why* it matters — no root
motivation, just a list of wants. I took the request at face value and never laddered.

**Cost:** without the why, every downstream role is flying blind on trade-offs. When the
`tech-lead` hits a design fork, or the `project-manager` must cut scope for capacity, the
*why* is what tells them which parts are load-bearing and which are negotiable. A framing
with no why forces those decisions to be made by guessing at the PO's priorities — or by
going back to re-elicit, which the live-only nature of my role makes expensive.

**Corrective:** the why is non-optional in the framing — the **Need vs Want** section and
the laddered root outcome exist to carry it (`intake-to-analysis-handoff.md`). Every goal is
an outcome (a why), not a feature (a what). If I can't state why a requested thing matters, I
haven't finished eliciting it.

## The through-line

Four of these five failures share one root: **letting a want, an assumption, or a surface
request stand in for the underlying need.** Premature solutioning promotes a want; leading
questions inject my assumption; vague signals skip the falsifiable outcome; skipping the why
drops the motivation. The single most protective habit is the **why-ladder** — it is the
corrective that, applied consistently, prevents four of the five. Boil-the-ocean is the odd
one out: its cure is the opposite restraint — knowing when the framing is *enough* and
handing the rest to the roles built to investigate it. Between the two — *ladder to the why*
and *stop at enough* — sits a clean, un-guessable, falsifiable framing, which is the entire
point of my reflex.
