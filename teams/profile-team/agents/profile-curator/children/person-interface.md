---
knowledge-base-summary: "The canonical person interface — the seed schema I materialize to ~/.atl/profiles/_interfaces/person.md: self-describing frontmatter (matches/examples/schema-version/changelog/tier-defaults/thresholds) + the core + person field shape."
---

# Person Interface (canonical seed)

This is the one interface v1 ships. It is *self-describing*: its own frontmatter carries
what it is (`matches` + examples), its version and change history (`schema-version` +
`changelog`), its privacy tiers (`tier-defaults`), its tunables (`thresholds`), and its
allowed enums — so type detection, lazy fill, and privacy gating all read from one place.

**How this reaches disk.** I keep the canonical copy here (in my knowledge). On drain I
ensure `~/.atl/profiles/_interfaces/person.md` exists; if it is absent I materialize it
verbatim from the block below. If a newer canonical version ships (higher
`schema-version` than the materialized file), I bring the on-disk interface forward and
the changelog drives the lazy fill of every person profile (see `interface-model.md`).
Thresholds live here — **not** in a config system (v2 has none by design); they are
type-specific, so the interface is their natural home.

## The canonical interface

```markdown
---
type-id: person
schema-version: 1.0.0
matches: |
  A human — a living individual the user has a personal or emotional bond with
  (family, friend, colleague, partner, mentor, rival, neighbour, acquaintance).
  NOT a public figure unless directly and personally relevant to the user.
  NOT a fictional character.
examples-positive:
  - "I talked with Alex today"
  - "I had a fight with my mom on the phone"
  - "My manager Deniz gave me tough feedback"
examples-negative:
  - "Steve Jobs supposedly used to say this"
  - "Sherlock Holmes always did it this way"
  - "The mayor announced a new policy"
changelog:
  - version: 1.0.0
    added: [everything]   # initial person interface
thresholds:
  # Fit score at or above which an existing interface is reused instead of a new one
  # being created. Person-only in v1, so this governs v2 auto-creation; kept here so it
  # travels with the type. Mesut's scenario: "couldn't find a fit of 75% or higher ...
  # we could even push this to 80%."
  type-match: 0.80
  # relation-to-user.salience is auto-computed from marker count over the window.
  salience:
    window-days: 30
    high: 10     # >= 10 markers in the window
    medium: 3    # 3..9
    # low: 0..2 (the remainder)
tier-defaults:
  identity.*: 1
  anchors.*: 1
  relation-to-user.kind: 1
  relation-to-user.role: 1
  relation-to-user.sentiment: 2
  traits.*: 2
  state.emotional: 3
  state.goals: 3
  state.financial: 4
  relationships.*.user-perceives: 3
allowed-kinds: [family, friend, colleague, romantic, mentor, professional, neighbor, acquaintance]
allowed-roles: [mother, father, sister, brother, partner, spouse, child, friend, colleague, manager, mentor, mentee, rival, neighbor]
fields:
  # ---- common core (every type has these) ----
  meta:
    type-id: <person | ... | unknown>
    schema-version: <semver of the interface this profile was last grown against>
    created: <ISO date>              # set once, on creation
    last-updated: <ISO date>         # drain maintains
    confidence: <high | medium | low>  # profile-wide summary of how much rests on inference
    is-self: <bool>                  # default false; true only for the user's own profile
    perception-flag: <bool>          # auto-true on is-self:false once a Tier-2 field is written
    consent: {}                      # per-field opt-in for Tier 4, e.g. { state.financial: true }
  identity:
    name: <canonical display name>
    aliases: [<other names/readings used in conversation, for marker matching>]
  relation-to-user:
    kind: <one of allowed-kinds>
    role: <one of allowed-roles>
    sentiment: <positive | negative | mixed | neutral>
    first-noticed: <ISO date>        # first marker date
    salience: <high | medium | low>  # auto from marker count; manual override respected
    salience-source: <drain-auto-<date> | user-set | lens-set>
  emotional-tags: [<free strings — "cherished", "estranged", "safe person", ...>]

  # ---- person type-extension ----
  identity-extension:
    demographics: { age: <int|null>, gender: <str|null>, occupation: <str|null>, location: <str|null> }
  anchors:
    birthday: <ISO date|null>
    anniversary: <ISO date|null>
    work-start: <ISO date|null>
    custom: [ { label: <str>, date: <ISO date>, note: <str|null> } ]
  traits:
    # can / can't axis
    fears: [<list>]
    enjoys: [<list>]
    excels-at: [<list>]
    struggles-with: [<list>]          # merge of "weak-at" + "no-skill-at"
    # values / character axis
    values: [<list>]                  # principles they care about
    triggers: [<list>]                # concrete triggers (distinct from fears)
    character:
      strengths: [<list>]
      weaknesses: [<list>]
    # style axis
    communication-style: <str|null>
    conflict-style: <str|null>
    # skills (with self-awareness, decision 4c)
    skills:
      - { name: <str>, aware-of-it: <bool|null>, level: <novice|intermediate|expert|null> }
  state:                              # current + optional history (change-policy below)
    emotional: { current: <str|null>, history: [ { value: <str>, date: <ISO date> } ] }
    goals:     { current: <str|null>, history: [ { value: <str>, date: <ISO date> } ] }
    financial: { current: <str|null>, history: [ { value: <str>, date: <ISO date> } ] }  # Tier 4
  relationships:                      # person-to-person graph (this person's ties)
    - { to: <slug>, kind: <str>, user-perceives: <str|null> }
change-policy:
  # default overwrite (decision D); history opt-in only where PAT's temporal lens needs it
  identity.*: overwrite
  anchors.*: overwrite
  relation-to-user.*: overwrite
  traits.*: overwrite
  state.emotional: history-tracked
  state.goals: history-tracked
  state.financial: history-tracked
---

# Person

Notes about this person live in the narrative body of each person's profile.md, not here.
This interface file is the shared schema every person profile is grown against.
```

## Reading the schema

- **Core vs extension** (hybrid-C architecture): `meta` / `identity` / `relation-to-user`
  / `emotional-tags` are the common core every future type shares; everything from
  `identity-extension` down is person-specific and evolves independently of the core.
- **No required fields.** The schema is a *target to fill toward*, never a validator. A
  person with only `identity.name` is a valid profile. This is what lets the interface
  grow (add a field → old profiles stay valid → lazy-filled on next touch).
- **`tier-defaults`** feeds the privacy gating in `curation-charter.md`; **`change-policy`**
  feeds the override policy in `interface-model.md`; **`thresholds`** feed salience
  auto-compute and (in v2) type detection. All three read from this one frontmatter — the
  reason the interface is "self-describing."
- **`_sources` parallel namespace.** Each written field gets an entry under a parallel
  `_sources.<path>` key in the profile (e.g. `_sources.identity.name: user-confirmed`) —
  the field/source split from `curation-charter.md`. It is not part of `fields:` above; it
  is maintained alongside, per write.
