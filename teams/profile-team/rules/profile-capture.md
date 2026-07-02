# Profile capture (inline marker protocol)

## Who runs this

**You (the agent) drop `profile-fact` markers inline as you speak** — the same
fire-and-forget reflex as learning capture, on a separate channel. A marker is a silent
HTML comment: invisible in rendered output, preserved in the transcript. ATL's automation
does the rest — `atl tick` transfers your markers into the durable queue exactly once, and
the `/profile-drain` skill (profile-team's `profile-curator`) folds each into the right
person profile at `~/.atl/profiles/`. You never track state or write profile files inline.

Profiles are **global** — the same person is one profile across every project. Capture is
cheap (~30 tokens); free to skip when nothing durable about a person came up.

## What counts as a profile fact

A **durable** fact about a real person the user has a personal bond with:

- **Identity / relation** — who they are to the user (mother, friend, manager), their name, aliases.
- **Anchors** — birthday, anniversary, a date that matters.
- **Traits** — fears, what they enjoy, what they excel at or struggle with, values, character, communication or conflict style, skills.
- **State** — current emotional state, goals, financial situation (these are sensitive — see tiers).
- **Relationships** — how they relate to another person in the user's world.

Do **NOT** mark: public figures or fictional characters (unless directly, personally
relevant); transient small talk ("Alex was tired today" is not durable); the assistant's
own reasoning. Don't mark a fact you already recorded — queue dedup makes it a no-op, but
save the tokens.

## How to mark

Multi-line YAML body, one marker per entity, multiple fields at once:

```html
<!-- profile-fact:
  entity: alex
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

- **`entity`** — a canonical lowercase-kebab slug. Reuse the same slug for the same person
  every time; put alternate names in `identity.aliases` so the curator can match them.
- **`fields`** — a map of `field-path: value` (paths follow the person interface, e.g.
  `traits.fears`, `state.goals`, `anchors.birthday`). Group everything you learned about
  one person in one marker.
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
