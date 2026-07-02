---
knowledge-base-summary: "How I decide an entity's type: score it against every seeded interface's matches + examples, reuse the best fit at/above the 0.80 threshold, else hold it as a minimal unknown stub. Six types are seeded (person, org, animal, place, object, project); an explicit type: marker hint short-circuits scoring."
---

# Type Detection

When a `profile-fact` names an entity I have no profile for, I decide which interface to
create it against before scaffolding it. Six interfaces are seeded:

| type-id | is | directory |
|---|---|---|
| `person` | a person the user has a bond with | `people/` |
| `org` | an organization/company/community they're tied to | `orgs/` |
| `animal` | a pet or animal they bonded with | `animals/` |
| `place` | a place loaded with meaning | `places/` |
| `object` | a meaningful physical thing | `objects/` |
| `project` | an endeavour they're invested in | `projects/` |

## The procedure

1. **Explicit hint wins.** If the marker carries a `type:` field (the assistant's hint) and
   it names a seeded type, use it — no scoring needed. The assistant usually knows what it
   just recorded.
2. **Otherwise, fit-score.** Each interface is self-describing: I read its `matches` (a
   description of what fits + what does NOT) and its `examples-positive` / `examples-negative`,
   and I derive a fit score for the entity against **each** of the six from the conversation
   context. I take the best fit.
3. **Reuse at/above threshold.** If the best fit is ≥ the interface's `thresholds.type-match`
   (0.80), the entity is that type — create its profile against that interface, in that
   type's directory.
4. **Below threshold → `unknown` stub.** If nothing fits well, I do **not** force the entity
   into the closest type. I create a minimal `unknown`-type stub: `meta.type-id: unknown` +
   `identity.name` + whatever Tier-1 identity/relation/emotional-tags facts the marker
   carries, under `~/.atl/profiles/unknown/<slug>/`. It is a valid, honest placeholder — the
   entity and how the user feels about it are still remembered, just without a rich typed
   schema. (Authoring a brand-new interface for a genuinely novel type is the
   `interface-creation` job.)

## Reading the fit

- A person fits `person`'s matches (a living individual the user bonds with); a departed pet
  fits `animal`; a childhood home fits `place`; a cherished mug fits `object`; a side project
  fits `project`; an employer fits `org`.
- The `examples-negative` are the guard: "Apple announced a phone" is not the user's `org`;
  "there was an earthquake in Japan" is not their `place`; a public figure or fictional
  character is neither profiled nor stubbed — I skip it (the `profile-capture` rule already
  steers the assistant away from those; this is the backstop).
- **Ambiguity is fine.** If an entity plausibly fits two types (a family farm is both a
  `place` and, via its animals, near `animal`), pick the type the marker's facts most belong
  to, or the assistant's `type:` hint. The core (`identity`, `relation-to-user`,
  `emotional-tags`) is shared, so a mis-typed profile still holds its most important facts and
  can be re-typed later.

## Why scoring, not hard-coding

Every interface carries its own detection signal (`matches` + examples) in its frontmatter,
so adding a new type never touches this logic — it just adds another interface to score
against. That is what makes the type set open-ended (and what `interface-creation` builds on
when no seeded type fits well enough).
