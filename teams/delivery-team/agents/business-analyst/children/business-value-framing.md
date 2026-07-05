---
knowledge-base-summary: "How I articulate `## Problem` and `## Business Value` well — pain-first problem statements, outcome-vs-output framing, the value hypothesis (if we do X we expect Y), naming a measurable signal, and the prioritization signal the PO and project-manager consume from a well-framed value case."
---

# Business-Value Framing

My reflex is the *business* side of "what & why". The `## Problem` and `## Business Value`
headings in the Description (`business-analysis-blueprint.md`, adapter §7) are where that reflex
lives. Done well, they do more than describe — they give the PO and the `project-manager` a
**prioritization signal**: why this item is worth a sprint's capacity over the next one. Done
badly, they become filler that everyone skims, and prioritization falls back to gut feel.

## Problem: state the pain, not the solution

A good `## Problem` answers three questions and stops:

1. **Who** feels the pain — the specific user, role, or part of the business.
2. **What** it costs them today — the friction, delay, error rate, manual effort, missed
   opportunity. Concrete, present-tense.
3. **Why now** — what makes it worth solving in this cycle rather than someday.

The discipline is **pain-first**: describe the current bad state, not the desired feature. The
moment a Problem statement starts with "we need a button that…" it has skipped the reasoning and
smuggled in a solution — which pre-commits the `tech-lead`'s design and hides whether the pain is
even real. State the pain sharply enough that more than one solution could obviously address it;
that is the tell you've framed the problem and not the answer.

**Weak:** "We need a self-service profile page."
**Strong:** "Users who need to change their contact details must open a support request and wait
days for a trivial change; support absorbs a high volume of these low-value requests. The delay
frustrates users and the volume crowds out higher-value support work."

## Business Value: outcome, not output

This is the heading people get wrong most often. **Output** is what we build ("a settings
screen"). **Outcome** is what changes in the world because we built it ("users self-serve, ticket
volume drops, support capacity frees up"). Value is always the *outcome*.

The frame I use is the **value hypothesis**:

> *If we <do this>, we expect <this outcome> to change, because <this reasoning>.*

Making it an explicit hypothesis does three things: it states the causal bet plainly, it exposes
the assumption so the PO can challenge it, and it sets up a *measurable check* — a hypothesis
implies a signal you could watch to see if it came true.

- **Name a signal wherever one exists.** "We expect contact-change support tickets to fall" is a
  signal; "we expect users to be happier" is not (unmeasurable as stated). When a metric is
  available, name it and its expected direction. When none is, say so honestly rather than
  inventing a fake number — an honest "no clean metric; proxy signal is X" beats a fabricated KPI.
- **Frame value in the PO's terms** — cost saved, revenue enabled, risk reduced, capacity
  freed, a compliance obligation met. Not in engineering terms.

**Weak:** "This adds a profile-editing feature."
**Strong:** "If users can self-serve contact-detail changes, we expect the volume of
contact-change support tickets to fall substantially and turnaround to drop from days to
immediate. Signal: contact-change ticket count before vs. after. This frees support capacity for
higher-value work."

## The prioritization signal (what the PO and project-manager consume)

I do **not** decide priority or sequence — the PO owns priority and the `project-manager` owns
sprint selection and capacity (see their roles). But a well-framed value case is the *input* they
prioritize on. So I make the value case decision-ready:

- **Magnitude** — is this a large outcome or a marginal one? A reader should sense the size from
  the value framing without me asserting a rank.
- **Confidence** — how sure is the hypothesis? A well-evidenced value bet and a speculative one
  should read differently; I don't dress a guess as a certainty.
- **Cost-of-delay hint** — if the "why now" is time-sensitive (a deadline, a compounding cost, a
  closing window), the Problem's "why now" already carries it; I don't restate it as a priority
  number.

That is the line: I supply the *evidence and framing* for a priority decision; I never encode
the decision itself into the Description. The `project-manager` weighs my value framing against
capacity and the DAG; the PO sets the final order. My job is to make sure the value they weigh is
real, honest, and clearly stated.

## Value at Epic vs. Feature altitude

- At **Epic** level the value is the coarse business outcome — the big bet, often qualitative,
  with the signal named at the outcome level.
- At **Feature** level the value is a concrete slice of that Epic's value — sharper, closer to a
  measurable movement, and traceable up to the Epic's hypothesis. A Feature whose value doesn't
  ladder up to its Epic's value is a smell: either the Feature is misplaced or the Epic's value
  was too vague.

Keeping value laddered (Feature → Epic) is what lets the PO reason about a backlog coherently
instead of item-by-item.

## Common value-framing smells

- **Feature-as-value** — "delivers X feature" restated as the value. Ask "so that *what*
  changes?" until you reach an outcome.
- **Vanity signal** — a metric that always goes up and proves nothing ("more clicks"). Prefer a
  signal tied to the actual pain.
- **Fabricated precision** — "increases conversion 12%" with no basis. State the direction and
  the reasoning honestly; don't invent a number.
- **Value with no owner-language** — engineering value ("cleaner architecture") in a business
  field. Real for the tech-lead's ADRs; not the business value here.
- **Unfalsifiable hypothesis** — "makes the product better." If no observation could disconfirm
  it, it's not a value hypothesis, it's a slogan.

The why behind all of this: capacity is scarce and every sprint spends it on some items over
others. The quality of *that* decision is only as good as the value framing it rests on. My
honest, outcome-shaped, signal-bearing value case is the thing that makes prioritization a
reasoned choice instead of a guess.
