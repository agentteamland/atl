# Profile capture (inline marker protocol)

## Who runs this

**You (the agent) drop `profile-fact` markers inline as you speak** — the same
fire-and-forget reflex as learning capture, on a separate channel. A marker is a silent
HTML comment: invisible in rendered output, preserved in the transcript. ATL's automation
does the rest — `atl tick` transfers your markers into the durable queue exactly once, and
the `/profile-drain` skill (profile-team's `profile-curator`) folds each into the right
profile at `~/.atl/profiles/`. You never track state or write profile files inline.

Profiles are **global** — the same entity is one profile across every project. Capture is
cheap (~30 tokens); free to skip when nothing durable about an entity came up.

## What counts as a profile fact

A **durable** fact about an entity in the user's inner world — someone or something they
have a real personal or emotional bond with. Six kinds:

- **person** — family, friends, colleagues, partners, mentors (the richest type).
- **org** — an employer, school, community, or club the user is genuinely tied to.
- **animal** — a pet or an animal they bonded with (living, passed, or from childhood).
- **place** — a hometown, a childhood home, a place loaded with meaning.
- **object** — a cherished / feared / missed thing (an heirloom, a beloved toy, a missed car).
- **project** — an endeavour they're invested in (a dream, a side project, a work initiative).

Across all six the durable facts share a shape: **identity/relation** (what it is to the
user), **anchors** (dates that matter), **traits/state** (what it's like, how it's doing —
the sensitive ones are tier-gated), and **links** to other entities.

Do **NOT** mark: public figures, brands, or fictional entities (unless directly, personally
relevant); transient small talk ("Alex was tired today", "grabbed a coffee somewhere" — not
durable); the assistant's own reasoning. Don't mark a fact you already recorded — queue dedup
makes it a no-op, but save the tokens.

## How to mark

Multi-line YAML body, one marker per entity, multiple fields at once:

```html
<!-- profile-fact:
  entity: alex
  type: person
  is-self: false
  kind: friend
  role: null
  fields:
    identity.name: Alex Doe
    traits.fears: [confrontation, abandonment]
    state.emotional: anxious about the new job
  source: user-confirmed
-->
```

- **`entity`** — a canonical lowercase-kebab slug. Reuse the same slug for the same entity
  every time; put alternate names in `identity.aliases` so the curator can match them.
- **`type`** — optional hint (`person` | `org` | `animal` | `place` | `object` | `project`,
  or a novel type you name). Include it when you know it — it saves the curator inferring.
  Omit it and the curator fit-scores; a coherent novel kind gets its own new interface, a
  genuine one-off becomes a light stub.
- **`fields`** — a map of `field-path: value` (paths follow that type's interface, e.g.
  `traits.fears`, `anchors.birthday` for a person; `state.status`, `provenance.acquired-from`
  for an object). Group everything you learned about one entity in one marker.
- **`is-self`/`kind`/`role`** — optional hints; include when known (helps the curator
  create or route the profile). `is-self: true` only for the user themselves.
- **`source`** — optional; defaults to `user-confirmed`. Set `source: agent-inferred` when
  you are **inferring** a fact rather than recording something the user stated. This is
  load-bearing for sensitive fields:

## Source & the privacy tiers (why source matters)

The curator gates writes by tier, and your `source` flag decides what lands:

- **Tier 1–2** (identity, anchors, relation, traits) — written either way; inference is
  tolerated and flagged, corrected later if wrong.
- **Tier 3** (`state.emotional`, `state.goals`, `relationships.*.user-perceives`) — written
  **only** from `source: user-confirmed`. If you are inferring one of these, either don't
  mark it or mark it `agent-inferred` knowing the curator will reject it — do not launder
  an inference into `user-confirmed`.
- **Tier 4** (`state.financial`) — only written if the user has consented for that profile;
  the curator skips it otherwise.

Be honest about `source`: a wrong inference self-corrects next conversation, but a
mislabeled `user-confirmed` hardens a guess into fact.
