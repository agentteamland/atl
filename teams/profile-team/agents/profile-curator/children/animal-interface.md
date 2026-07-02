---
knowledge-base-summary: "The canonical animal interface — the seed schema I materialize to ~/.atl/profiles/_interfaces/animal.md: self-describing frontmatter (matches/examples/schema-version/changelog/tier-defaults/thresholds) + the inherited common core + the animal type-extension (species/breed, adopted/birthday/passed anchors, temperament/quirks/favorites, history-tracked health + status)."
---

# Animal Interface (canonical seed)

This is one of the extension interfaces the type family ships. Like every interface it is
*self-describing*: its own frontmatter carries what it is (`matches` + examples), its
version and change history (`schema-version` + `changelog`), its privacy tiers
(`tier-defaults`), its tunables (`thresholds`), and its allowed enums — so type detection,
lazy fill, and privacy gating all read from one place.

**How this reaches disk.** I keep the canonical copy here (in my knowledge). On drain I
ensure `~/.atl/profiles/_interfaces/animal.md` exists; if it is absent I materialize it
verbatim from the block below. If a newer canonical version ships (higher `schema-version`
than the materialized file), I bring the on-disk interface forward and the changelog drives
the lazy fill of every animal profile (see `interface-model.md`). Thresholds live here —
**not** in a config system (v2 has none by design); they are type-specific, so the interface
is their natural home.

The animal type shares the common core (`meta`, `identity`, `relation-to-user` incl.
salience, `emotional-tags`) with every other type — that core is defined once in
`interface-model.md` / `person-interface.md` and inherited here, not re-specified. Only the
animal-specific extension is detailed below. This is the most emotionally loaded type: an
animal is often a departed companion, so `anchors.passed` and the history-tracked `health` /
`status` carry real weight — handle them with the same tenderness a lens will.

## The canonical interface

```markdown
---
type-id: animal
schema-version: 1.0.0
matches: |
  A real animal the user personally cares about — an own, family, or childhood pet, a
  working animal, or an animal they formed an emotional bond with. Living, passed, lost,
  or rehomed all fit; the bond is what qualifies it.
  NOT wildlife or a species in general (a fox they saw once, "I love elephants").
  NOT a fictional or anthropomorphized animal (a cartoon pet, a mascot).
examples-positive:
  - "My dog Zeytin passed away last spring and I still miss her"
  - "We adopted a kitten this weekend, she's already ruling the house"
  - "I grew up with a horse named Duman on my grandfather's farm"
examples-negative:
  - "Foxes are such clever animals"
  - "Garfield hates Mondays"
  - "There was a stray cat outside the cafe today"   # no bond formed
changelog:
  - version: 1.0.0
    added: [everything]   # initial animal interface
thresholds:
  # Fit score at or above which this interface is reused instead of a new one being
  # created. Kept here so it travels with the type (governs v2 auto-creation).
  type-match: 0.80
  # relation-to-user.salience is auto-computed from marker count over the window.
  salience:
    window-days: 30
    high: 10     # >= 10 markers in the window
    medium: 3    # 3..9
    # low: 0..2 (the remainder)
tier-defaults:
  # Only this type's own fields are listed; the common-core tiers are inherited.
  identity-extension.*: 1
  anchors.adopted: 1
  anchors.birthday: 1
  anchors.passed: 2        # date of loss — a sensitive fact, not a bare anchor
  relation-to-user.kind: 1
  relation-to-user.role: 1
  traits.temperament: 2
  traits.quirks: 2
  traits.favorites: 2
  state.health.*: 2        # how the animal is doing — perception/wellbeing
  state.status: 1          # alive/passed/lost/rehomed — a factual anchor
allowed-kinds: [companion, family-pet, childhood-pet, working-animal, wild-encountered]
allowed-roles: [companion, guard, herding, service, therapy, working, hunting]
allowed-status: [alive, passed, lost, rehomed]
fields:
  # ---- common core (inherited — see interface-model.md) ----
  # meta / identity / relation-to-user (incl. salience) / emotional-tags are the shared
  # core every type carries; their shape and tiers live in person-interface.md.

  # ---- animal type-extension ----
  identity-extension:
    species: <str|null>              # dog, cat, horse, rabbit, ...
    breed: <str|null>
    gender: <male | female | unknown | null>
    age: <int|null>                  # years; null when unknown or not meaningful
  anchors:
    adopted: <ISO date|null>         # when they came into the user's life
    birthday: <ISO date|null>
    passed: <ISO date|null>          # date of loss — emotionally critical, kept even when status flips
  traits:
    temperament: <str|null>          # gentle, skittish, fiercely loyal, ...
    quirks: [<list>]                 # the little behaviors the user remembers them by
    favorites: [<list>]             # favorite spots, foods, toys, people
  state:                             # current + optional history (change-policy below)
    health:
      current: <str|null>            # a current wellbeing note
      history: [ { value: <str>, date: <ISO date> } ]
    status: <one of allowed-status>  # alive | passed | lost | rehomed
change-policy:
  # default overwrite (decision D); history opt-in only where temporal evolution matters
  identity-extension.*: overwrite
  anchors.*: overwrite
  relation-to-user.*: overwrite
  traits.*: overwrite
  state.health: history-tracked      # a pet's health arc over time is what a lens reads
  state.status: history-tracked      # alive -> passed is a transition worth keeping dated
---

# Animal

Notes about this animal live in the narrative body of each animal's profile.md, not here.
This interface file is the shared schema every animal profile is grown against.
```

## Reading the schema

- **Core vs extension** (hybrid-C architecture): `meta` / `identity` / `relation-to-user`
  / `emotional-tags` are inherited from the common core (see `interface-model.md`);
  everything from `identity-extension` down is animal-specific and evolves independently of
  the core.
- **No required fields.** The schema is a *target to fill toward*, never a validator. An
  animal with only `identity.name` and `state.status` is a valid profile. This is what lets
  the interface grow (add a field → old profiles stay valid → lazy-filled on next touch).
- **`passed` is never dropped.** When `state.status` flips to `passed`, `anchors.passed`
  records the date of loss and stays — the bond outlives the animal, and a lens must be able
  to speak to it with care.
- **`tier-defaults`** feeds the privacy gating in `curation-charter.md`; **`change-policy`**
  feeds the override policy in `interface-model.md`; **`thresholds`** feed salience
  auto-compute and (in v2) type detection. All three read from this one frontmatter — the
  reason the interface is "self-describing."
- **`_sources` parallel namespace.** Each written field gets an entry under a parallel
  `_sources.<path>` key in the profile (e.g. `_sources.state.status: user-confirmed`) — the
  field/source split from `curation-charter.md`. It is not part of `fields:` above; it is
  maintained alongside, per write.
