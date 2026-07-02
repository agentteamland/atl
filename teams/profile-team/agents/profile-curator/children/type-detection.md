---
knowledge-base-summary: "How I decide an entity's type. v1 is person-only: every entity resolves to person, a clearly non-person entity is held as a minimal unknown stub (never fabricated into a person). The fit-scoring mechanism (matches + examples) exists for v2 multi-type + auto-creation."
---

# Type Detection

When a `profile-fact` names an entity I have no profile for, I decide which interface to
create it against before scaffolding it.

## v1: person-only

v1 seeds a single interface — `person`. So type detection is nearly trivial:

- **Almost everything is a person.** The profile world's north-star consumer
  (personal-advisory-team) is about the people in the user's life; a named entity in a
  `profile-fact` is a person unless it clearly isn't.
- **A clearly non-person entity** (a place, a pet, an organization, an object) is **not**
  forced into the person shape. I create a minimal `unknown`-type stub:
  `meta.type-id: unknown` + `identity.name` + whatever Tier-1 identity/relation facts the
  marker carries. It is a valid, honest placeholder. It gets a real interface in v2, when
  the other types + auto-creation ship. I never invent person fields (traits, state) for a
  non-person.
- I do **not** author a brand-new interface from scratch in v1 — that is the highest-risk
  piece (an LLM generating a schema) and is deferred to v2 by the 8f scope decision.

## How I recognize a person

The `person` interface is self-describing — I read its own frontmatter:

- **`matches`** — "a living individual the user has a personal/emotional bond with; not a
  public figure unless personally relevant; not fictional."
- **`examples-positive`** — "I talked with Alex today", "I had a fight with my mom".
- **`examples-negative`** — "Steve Jobs used to say…", "Sherlock Holmes always…".

If the entity fits `matches` and looks like the positive examples (and not the negative
ones), it is a person. A public figure mentioned in passing, or a fictional character, is
neither profiled nor stubbed — I skip it (the `profile-capture` rule already tells the
agent not to mark those, so this is a backstop).

## The fit-scoring mechanism (present, mostly v2)

The interface carries `thresholds.type-match` (0.80). The mechanism, which becomes load-
bearing once v2 adds more types: score the entity's mentions against **each** interface's
`matches` + `examples-*`, take the best fit; at/above threshold reuse that interface, below
it (v2) author a new one. In v1 there is only `person` to score against, so this reduces to
"person or `unknown` stub." Keeping the mechanism now means v2 adds types without reworking
detection — it just adds interfaces to score against.
