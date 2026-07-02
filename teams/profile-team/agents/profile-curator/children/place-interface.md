---
knowledge-base-summary: "The canonical place interface — the seed schema I materialize to ~/.atl/profiles/_interfaces/place.md: a real place (country/city/district/neighbourhood/village/home/spot) the user is emotionally bonded to, with its bond kind, anchors, and the sensory memories/associations/feelings it evokes."
---

# Place Interface (canonical seed)

This is a type-extension interface built on the shared common core. It is *self-describing*:
its own frontmatter carries what it is (`matches` + examples), its version and change history
(`schema-version` + `changelog`), its privacy tiers (`tier-defaults`), its tunables
(`thresholds`), and its allowed enums — so type detection, lazy fill, and privacy gating all
read from one place.

**How this reaches disk.** I keep the canonical copy here (in my knowledge). On drain I
ensure `~/.atl/profiles/_interfaces/place.md` exists; if it is absent I materialize it
verbatim from the block below. If a newer canonical version ships (higher `schema-version`
than the materialized file), I bring the on-disk interface forward and the changelog drives
the lazy fill of every place profile (see `interface-model.md`). Thresholds live here — **not**
in a config system (v2 has none by design); they are type-specific, so the interface is their
natural home.

## The canonical interface

```markdown
---
type-id: place
schema-version: 1.0.0
matches: |
  A real place the user has a personal or emotional bond with — a country, city,
  district, neighbourhood, village, home, or a specific spot (a café, a bench, a
  childhood street). NOT a place merely mentioned as neutral fact (a news dateline,
  a weather location). NOT a fictional or mythical place.
examples-positive:
  - "I still miss the village where my grandmother lived"
  - "Moda is the one place in Istanbul where I feel like myself"
  - "The old apartment on Bahariye smelled like my whole childhood"
examples-negative:
  - "There was an earthquake in Japan this morning"
  - "The capital of Portugal is Lisbon"
  - "Rivendell was hidden deep in the valley"
changelog:
  - version: 1.0.0
    added: [everything]   # initial place interface
thresholds:
  # Fit score at or above which this existing interface is reused instead of a new type
  # being created. Kept here so it travels with the type. Mesut's scenario: "couldn't find
  # a fit of 75% or higher ... we could even push this to 80%."
  type-match: 0.80
  # relation-to-user.salience is auto-computed from marker count over the window.
  salience:
    window-days: 30
    high: 10     # >= 10 markers in the window
    medium: 3    # 3..9
    # low: 0..2 (the remainder)
tier-defaults:
  identity-extension.kind: 1
  identity-extension.location: 1
  anchors.*: 1
  relation-to-user.bond: 1
  traits.sensory-memories: 2
  traits.associations: 2
  traits.feelings-evoked: 2
  associated-people.*: 1
allowed-kinds: [country, city, district, neighbourhood, village, home, spot]
allowed-bond-kinds: [hometown, childhood-home, lived, visited, longed-for, dreamed-of]
fields:
  # ---- common core (inherited — see interface-model.md) ----
  # meta / identity / relation-to-user (incl. salience) / emotional-tags come from the
  # shared core; only relation-to-user gains a place-specific `bond` sub-field below.

  # ---- place type-extension ----
  identity-extension:
    kind: <one of allowed-kinds>       # country | city | district | neighbourhood | village | home | spot
    location: <str|null>               # where it sits — parent place / coords / free description
  relation-to-user:
    bond: <one of allowed-bond-kinds>  # hometown | childhood-home | lived | visited | longed-for | dreamed-of
  anchors:
    first-visited: <ISO date|null>
    lived-from: <ISO date|null>
    lived-until: <ISO date|null>
    left: <ISO date|null>              # when the user last parted from it
  traits:                              # the user's felt sense of the place (Tier 2)
    sensory-memories: [<list>]         # sounds, smells, light, textures tied to it
    associations: [<list>]             # what it stands for — people, eras, ideas
    feelings-evoked: [<list>]          # what it makes the user feel now
  associated-people:                   # free links to person profiles bound to this place
    - { to: <person-slug>, note: <str|null> }
change-policy:
  # default overwrite (decision D); the place is mostly stable, so nothing is
  # history-tracked in v1 — the user's felt sense is overwritten, not versioned.
  identity-extension.*: overwrite
  relation-to-user.bond: overwrite
  anchors.*: overwrite
  traits.*: overwrite
  associated-people.*: overwrite
---

# Place

Notes about this place live in the narrative body of each place's profile.md, not here.
This interface file is the shared schema every place profile is grown against.
```

## Reading the schema

- **Core vs extension** (hybrid-C architecture): `meta` / `identity` / `relation-to-user`
  / `emotional-tags` are inherited from the common core (see `interface-model.md` and
  `person-interface.md`) and are *not* repeated above; everything from `identity-extension`
  down is place-specific and evolves independently of the core. `relation-to-user.bond` is
  the one place-specific addition onto a core field.
- **No required fields.** The schema is a *target to fill toward*, never a validator. A place
  with only `identity.name` + `identity-extension.kind` is a valid profile. This is what lets
  the interface grow (add a field → old profiles stay valid → lazy-filled on next touch).
- **`tier-defaults`** feeds the privacy gating in `curation-charter.md`; **`change-policy`**
  feeds the override policy in `interface-model.md`; **`thresholds`** feed salience
  auto-compute and (in v2) type detection. All three read from this one frontmatter — the
  reason the interface is "self-describing." Only this type's own fields are tiered here; the
  core tiers are inherited.
- **`_sources` parallel namespace.** Each written field gets an entry under a parallel
  `_sources.<path>` key in the profile (e.g. `_sources.identity-extension.kind: user-confirmed`)
  — the field/source split from `curation-charter.md`. It is not part of `fields:` above; it
  is maintained alongside, per write.
