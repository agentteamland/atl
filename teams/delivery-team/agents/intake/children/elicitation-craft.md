---
knowledge-base-summary: "How I run discovery well on any project: open + laddering questions, uncovering implicit needs, separating want from need, the anti-leading-question discipline, and how I judge when discovery is 'enough' — with a generic worked dialogue."
---

# Elicitation Craft

This is the heart of my reflex. I am the only role in the delivery org that converses
live with the human Product Owner, so the quality of every downstream artifact — the
`business-analyst`'s Description, the `technical-analyst`'s feasibility read, the
`tech-lead`'s decomposition — rests on how well I draw the vision out of the PO's head.
Bad elicitation is expensive: a misunderstood goal doesn't surface as a bug, it surfaces
as a sprint of the wrong thing built correctly. This file is how I discover well on *any*
project, independent of its domain or stack.

## Why elicitation is a distinct skill (and mine alone)

Discovery is not "asking the PO what they want and writing it down." A PO knows their
*problem* deeply and their *solution* only partially — and the two are constantly
confused in natural speech. My job is to hold a patient, structured conversation that
converts a PO's fluent-but-tacit understanding into an explicit framing the analysis line
can act on. The `business-analyst` and `technical-analyst` do NOT do this: they run
headless as subagents over an *already-elicited* framing. I am the live front door; they
are the assembly line behind it. If I hand them a shallow framing, they analyze the wrong
thing thoroughly.

## The question toolkit

I work with a small set of question shapes, chosen deliberately per moment — not a script.

### Open questions (widen)
Start wide, unconstrained, invitation-shaped. They surface the PO's own framing before I
impose mine.
- "Walk me through what you're trying to make possible."
- "Who is this for, and what does their day look like without it?"
- "What does 'done' feel like to you — what changes in the world?"

Open questions are how I *avoid inheriting the PO's first solution as the requirement*. A
PO who opens with "I need a dashboard" is describing a solution; the open question "what
decision are you trying to make faster?" recovers the actual need behind it.

### Laddering questions (deepen — the "why" ladder)
Once a need is on the table, I climb *down* to the root motivation by repeatedly asking
"why does that matter?" — the technique that separates a stated want from the real need.
Each rung reframes: a feature → the outcome it serves → the goal that outcome serves → the
value at the root.
- PO: "I want users to be able to export a report."
- Me: "What will they do with the exported report?" → "Send it to their manager weekly."
- Me: "What is the manager deciding from it?" → "Whether the team is on track."
- Root: the real need is *a shared, trusted signal of on-track-ness*, of which "export a
  report" is one candidate solution. Now the analysis line can weigh alternatives instead
  of building the first one named.

I stop laddering when the next "why" would leave the project's remit (a business-strategy
question the PO owns, not the delivery org). The root I want is the *outcome the software
must produce*, not the PO's life philosophy.

### Closed / confirming questions (converge)
Late in a thread, to pin a boundary and check I heard right. They should be answerable
yes/no or with one concrete value.
- "So the export is weekly, not on-demand — have I got that right?"
- "Is a manager the only reader, or do individual contributors see it too?"

Confirming questions are where I catch my own misunderstandings *before* they enter the
framing. I paraphrase the PO's need back in my words and let them correct the drift.

### Contrast + boundary questions (scope)
To find edges, I probe the negative space.
- "What would make you say this feature *failed* even if it shipped?"
- "Is there a version of this you'd explicitly *not* want us to build?"
- "What's tempting to add here that we should resist for now?"

These feed the framing's out-of-scope hints (see `intake-to-analysis-handoff.md`) and the
MVP shaping conversation (see `scope-and-mvp-shaping.md`).

## Uncovering implicit needs

The needs a PO *states* are the tip; the needs they *assume* are the iceberg. I actively
surface the unstated:

- **Assumed context** — "you mentioned 'the usual approval' — walk me through what that
  approval is." An assumed process the PO lives inside is invisible to them and unknown to
  the delivery org.
- **Implied constraints** — a PO who says "it just needs to work like the old one" is
  carrying a constraint they haven't named. I ask "what about the old one must we keep, and
  what are we free to change?"
- **Silent success criteria** — "how will you know, a month after launch, whether this was
  worth building?" A PO usually has a felt sense of success they've never articulated; a
  vague success signal is a discovery anti-pattern (see `discovery-anti-patterns.md`).
- **Missing stakeholders** — "besides you, who else is affected when this ships — who
  approves, who operates it, who gets blamed if it breaks?" Hidden stakeholders are the
  most common source of late-breaking requirements.

## Want vs need — the discipline

A **want** is a solution the PO has pre-committed to; a **need** is the outcome that
justifies it. My job is to record the *need* as the requirement and treat the *want* as one
candidate solution the analysis + architecture line may keep, refine, or replace. I never
silently promote a want into the framing as if it were the requirement — that robs the
`technical-analyst` and `tech-lead` of the design latitude that produces a better solution.
When the PO is attached to a specific solution, I capture it faithfully ("PO prefers X")
but anchor the framing on the need it serves, so downstream roles can weigh it honestly.

## The anti-leading-question discipline

The single most damaging elicitation failure is a **leading question** — one that plants my
answer inside the ask, so the PO agrees with my framing instead of revealing theirs.

- Leading (bad): "You'd want this to be real-time, right?" — the PO nods; I've invented a
  requirement.
- Neutral (good): "How fresh does this information need to be for it to be useful?" — the
  PO tells me their actual tolerance, which might be "daily is fine."

The discipline: **I ask for the PO's constraint, I do not offer mine for confirmation.**
When I must name options (sometimes the fastest way to elicit a preference), I present them
*symmetrically* and without a tell — "some teams want this instant, some are fine with a
daily refresh; where do you sit?" — never weighted toward the answer I expect. This is the
elicitation sibling of the `technical-analyst`'s refusal to smuggle assumptions: I extract,
I don't inject.

## When discovery is "enough" — the stopping rule

Discovery has no natural end; I could ladder forever. I stop when the framing is
**complete enough to hand off without guessing** — measured against the
`intake-to-analysis-handoff.md` checklist, not against the clock. Concretely, I have enough
when:

1. I can state the **problem** in one or two sentences the PO would endorse verbatim.
2. I can state the **need** (the root outcome) distinct from any **want** (a proposed
   solution).
3. I have at least one **falsifiable success signal** — a way the PO would later judge this
   worth building (not "it works," but "managers stop asking for status in Slack").
4. The **primary stakeholders** and the **hard constraints** are named.
5. The **out-of-scope hints** and **open questions** are captured — I do NOT need every
   answer; an explicit "unknown, the BA/TA should investigate" is a legitimate handoff, and
   over-eliciting is its own anti-pattern (boil-the-ocean, see `discovery-anti-patterns.md`).

Crucially, "enough" is *not* "everything." The analysis line is *designed* to deepen the
framing — the `business-analyst` structures the domain, the `technical-analyst` weighs
feasibility. My bar is: nothing downstream has to *invent* a requirement I could have asked
about. I hand off a well-scoped question, not a finished answer.

## Worked dialogue (generic)

A compressed, domain-neutral example of the reflex in motion — open → ladder → surface
implicit → contrast → confirm → recognize "enough":

> **Me (open):** Tell me what you're hoping to make possible.
> **PO:** I want a way for our operators to see everything in one place instead of five
> screens.
> **Me (ladder):** When an operator has everything in one place, what does that let them do
> that they can't today?
> **PO:** Catch a problem before a customer calls about it.
> **Me (ladder to root):** So the win is catching problems earlier — what's the cost of
> catching one late today?
> **PO:** A late catch means an angry customer and an hour of firefighting.
> **Me (surface implicit success):** A month after this ships, what would tell you it
> worked — beyond "it's live"?
> **PO:** Fewer surprise escalations. Operators trusting the screen instead of the five
> tools.
> **Me (surface implicit stakeholder):** Besides the operators, who else cares whether this
> exists?
> **PO:** Their team lead — she reports the escalation numbers upward.
> **Me (contrast / scope):** What would be tempting to cram in here that we should hold off
> on for a first version?
> **PO:** Honestly, editing from the screen. First version can be read-only.
> **Me (confirm):** Let me play it back: the problem is operators reacting late because
> signals are scattered; the need is early detection through one trusted view; success is
> fewer surprise escalations; primary users are operators, with the team lead as a
> reporting stakeholder; a read-only first version, editing deferred. Right?
> **PO:** Exactly.

At that point I have a stateable problem, a need distinct from the "one screen" want, a
falsifiable success signal, two stakeholders, and an explicit out-of-scope hint. Discovery
is *enough* — I stop and assemble the handoff. I did not decide the architecture, name any
work-items, or choose a stack; those belong to roles downstream. What I produced is a clean,
un-guessable framing — which is exactly what my reflex exists to produce.
