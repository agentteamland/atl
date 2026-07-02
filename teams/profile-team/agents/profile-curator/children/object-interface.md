---
knowledge-base-summary: "The canonical object interface — the seed schema I materialize to ~/.atl/profiles/_interfaces/object.md: a specific physical thing loaded with meaning for the user (beloved toy, meaningful mug, heirloom, missed car) — core + object-extension (identity-extension/provenance/state/story pointer), history-tracked status."
---

# Object Interface (canonical seed)

An object is a specific physical thing the user is emotionally bound to — a cherished
toy, a meaningful mug, an heirloom, a car they miss. Like every type it is
*self-describing*: its own frontmatter carries what it is (`matches` + examples), its
version and change history (`schema-version` + `changelog`), its privacy tiers
(`tier-defaults`), its tunables (`thresholds`), and its allowed enums — so type
detection, lazy fill, and privacy gating all read from one place.

**How this reaches disk.** I keep the canonical copy here (in my knowledge). On drain I
ensure `~/.atl/profiles/_interfaces/object.md` exists; if it is absent I materialize it
verbatim from the block below. If a newer canonical version ships (higher
`schema-version` than the materialized file), I bring the on-disk interface forward and
the changelog drives the lazy fill of every object profile (see `interface-model.md`).
Thresholds live here — **not** in a config system (v2 has none by design); they are
type-specific, so the interface is their natural home.

## The canonical interface

```markdown
---
type-id: object
schema-version: 1.0.0
matches: |
  A specific physical object loaded with personal meaning for the user — a beloved toy,
  a meaningful mug, an heirloom, a missed car; something cherished, feared, missed, or
  otherwise emotionally charged. NOT a generic product mention. NOT a commodity the user
  merely owns and does not care about.
examples-positive:
  - "I still have the mug my grandmother gave me"
  - "I miss my old car, we went everywhere in it"
  - "That worn teddy bear got me through my childhood"
examples-negative:
  - "I need to buy a new phone charger"
  - "The dishwasher broke and I ordered a replacement"
  - "This laptop is fine, nothing special about it"
changelog:
  - version: 1.0.0
    added: [everything]   # initial object interface
thresholds:
  # Fit score at or above which an existing interface is reused instead of a new one
  # being created. Kept here so it travels with the type. Mesut's scenario: "couldn't
  # find a fit of 75% or higher ... we could even push this to 80%."
  type-match: 0.80
  # relation-to-user.salience is auto-computed from marker count over the window.
  salience:
    window-days: 30
    high: 10     # >= 10 markers in the window
    medium: 3    # 3..9
    # low: 0..2 (the remainder)
tier-defaults:
  identity-extension.kind: 1
  identity-extension.description: 1
  provenance.acquired-from: 1
  provenance.acquired-when: 1
  provenance.occasion: 1
  state.status: 1
  state.condition: 1
  affect: 2
  story: 2
allowed-kinds: [gift, heirloom, possession, lost-item, keepsake, purchase, handmade]
allowed-affect: [cherished, feared, missed, meaningful]
allowed-status: [have, lost, broke, gave-away, seeking]
fields:
  # ---- common core (inherited — see interface-model.md) ----
  # meta, identity, relation-to-user (incl. salience), emotional-tags are the shared
  # core every type carries; their shape + tiers are defined in person-interface.md /
  # interface-model.md and are not repeated here.

  # ---- object type-extension ----
  identity-extension:
    kind: <one of allowed-kinds>       # what kind of meaningful thing it is
    description: <str>                  # what the object physically is, plainly
  affect: <one of allowed-affect>       # the emotional charge; also surfaces in emotional-tags
                                        # and relation-to-user.sentiment
  provenance:
    acquired-from: <person-slug|null>   # who it came from, if a person in the profile graph
    acquired-when: <ISO date|null>      # when it entered the user's life
    occasion: <str|null>                # the occasion it marks (a birthday, a farewell, ...)
  state:                                # current standing of the object (change-policy below)
    status: <one of allowed-status>     # have | lost | broke | gave-away | seeking
    condition: <str|null>               # worn, pristine, restored, faded, ...
  story: <pointer>                      # the narrative of why it matters lives in the
                                        # profile.md body, not here; this is a pointer to it
change-policy:
  # default overwrite (decision D); history opt-in only where temporal evolution matters
  identity-extension.*: overwrite
  affect: overwrite
  provenance.*: overwrite
  state.status: history-tracked         # a thing's status evolves: have -> lost -> seeking -> have
  state.condition: overwrite
  story: overwrite
---

# Object

The narrative of why this object matters — its story — lives in the body of each object's
profile.md, not here. This interface file is the shared schema every object profile is
grown against.
```

## Reading the schema

- **Core vs extension** (hybrid-C architecture): `meta` / `identity` / `relation-to-user`
  / `emotional-tags` are the common core inherited from the interface model (see
  `interface-model.md`); everything from `identity-extension` down is object-specific and
  evolves independently of the core. Only the object extension is detailed above — the
  core is inherited, not re-specified.
- **No required fields.** The schema is a *target to fill toward*, never a validator. An
  object with only `identity.name` + `identity-extension.description` is a valid profile.
  This is what lets the interface grow (add a field → old profiles stay valid →
  lazy-filled on next touch).
- **`state.status` is history-tracked.** An object's standing moves over time (a lost
  thing is found, a kept thing breaks, a given thing is sought again), so `status` keeps a
  dated history like the person type's `state.*`; everything else overwrites.
- **`story` is a pointer, not a field to stuff.** The reason an object matters is prose —
  it lives in the profile.md body and reads at Tier 2; the frontmatter only points at it.
- **`tier-defaults`** feeds the privacy gating in `curation-charter.md`; **`change-policy`**
  feeds the override policy in `interface-model.md`; **`thresholds`** feed salience
  auto-compute and (in v2) type detection. All three read from this one frontmatter — the
  reason the interface is "self-describing."
