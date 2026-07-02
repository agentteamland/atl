---
knowledge-base-summary: "The canonical project interface — the seed schema I materialize to ~/.atl/profiles/_interfaces/project.md: self-describing frontmatter (matches/examples/schema-version/changelog/tier-defaults/thresholds) + the inherited core + the project type-extension (kind/domain, status/motivation/stakes, anchors, investment, associated-people)."
---

# Project Interface (canonical seed)

This is a non-person type interface. Like every interface it is *self-describing*: its own
frontmatter carries what it is (`matches` + examples), its version and change history
(`schema-version` + `changelog`), its privacy tiers (`tier-defaults`), its tunables
(`thresholds`), and its allowed enums — so type detection, lazy fill, and privacy gating all
read from one place. It inherits the common core defined in `person-interface.md` /
`interface-model.md` and adds only the project-specific extension.

**How this reaches disk.** I keep the canonical copy here (in my knowledge). On drain I
ensure `~/.atl/profiles/_interfaces/project.md` exists; if it is absent I materialize it
verbatim from the block below. If a newer canonical version ships (higher `schema-version`
than the materialized file), I bring the on-disk interface forward and the changelog drives
the lazy fill of every project profile (see `interface-model.md`). Thresholds live here —
**not** in a config system (v2 has none by design); they are type-specific, so the interface
is their natural home.

## The canonical interface

```markdown
---
type-id: project
schema-version: 1.0.0
matches: |
  A real endeavour the user is personally invested in and driving — a dream, a side
  project, a work initiative, a creative work — that they are considering, planning,
  building, or have finished. NOT a task or todo item. NOT someone else's project the
  user only observes or comments on without personal stake.
examples-positive:
  - "I've been thinking about starting a podcast about urban gardening"
  - "Shipped the v2 of my budgeting app tonight, finally"
  - "The kitchen renovation is on hold until spring"
examples-negative:
  - "I need to reply to that email before 5pm"
  - "My coworker is building some internal dashboard, not my problem"
  - "Read an article about someone who founded a startup"
changelog:
  - version: 1.0.0
    added: [everything]   # initial project interface
thresholds:
  # Fit score at or above which this existing interface is reused instead of a new type
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
  identity-extension.domain: 1
  anchors.*: 1
  associated-people.*.to: 1
  associated-people.*.kind: 1
  relation-to-user.investment: 2
  state.status: 1
  state.progress: 2
  state.motivation: 2
  state.stakes: 2
allowed-kinds: [personal, work, creative, side]
allowed-statuses: [considered, planned, in-progress, paused, done, abandoned]
allowed-investments: [dream, obligation, experiment]
allowed-associated-kinds: [collaborator, stakeholder, mentor]
fields:
  # ---- common core (inherited — see interface-model.md) ----
  # meta / identity / relation-to-user (incl. salience) / emotional-tags come from the core.
  # relation-to-user.investment below is the project-specific extension of the core relation.

  # ---- project type-extension ----
  identity-extension:
    kind: <one of allowed-kinds>        # personal | work | creative | side
    domain: <str|null>                  # subject area, e.g. "software", "writing", "home"
  state:                                # current + optional history (change-policy below)
    status:     { current: <one of allowed-statuses>, history: [ { value: <str>, date: <ISO date> } ] }
    progress: <str|null>                # free-text sense of how far along
    motivation: { current: <str|null>, history: [ { value: <str>, date: <ISO date> } ] }  # why the user is (or isn't) pushing on it now
    stakes: <str|null>                  # what riding on it — consequences of success/failure
  anchors:
    started: <ISO date|null>
    target: <ISO date|null>             # aimed-for completion
    completed: <ISO date|null>
  relation-to-user:
    # extends the core relation-to-user; investment is why THIS project matters to the user
    investment: <one of allowed-investments>   # dream | obligation | experiment
  associated-people:                    # people tied to this project (each `to` a person-slug)
    - { to: <person-slug>, kind: <one of allowed-associated-kinds> }
change-policy:
  # default overwrite (decision D); history opt-in only where temporal evolution matters
  identity-extension.*: overwrite
  anchors.*: overwrite
  relation-to-user.investment: overwrite
  associated-people.*: overwrite
  state.progress: overwrite
  state.stakes: overwrite
  state.status: history-tracked
  state.motivation: history-tracked
---

# Project

Notes about this project live in the narrative body of each project's profile.md, not here.
This interface file is the shared schema every project profile is grown against.
```

## Reading the schema

- **Core vs extension** (hybrid-C architecture): `meta` / `identity` / `relation-to-user` /
  `emotional-tags` are the common core inherited from `interface-model.md` (detailed in
  `person-interface.md`); everything from `identity-extension` down is project-specific and
  evolves independently of the core. `relation-to-user.investment` is the one place this type
  extends a core field rather than adding a new one.
- **No required fields.** The schema is a *target to fill toward*, never a validator. A
  project with only `identity.name` + `state.status` is a valid profile. This is what lets the
  interface grow (add a field → old profiles stay valid → lazy-filled on next touch).
- **`tier-defaults`** feeds the privacy gating in `curation-charter.md` — it lists only this
  type's own fields; the core tiers are inherited. **`change-policy`** feeds the override
  policy in `interface-model.md`; **`thresholds`** feed salience auto-compute and (in v2) type
  detection. All read from this one frontmatter — the reason the interface is "self-describing."
- **`_sources` parallel namespace.** Each written field gets an entry under a parallel
  `_sources.<path>` key in the profile (e.g. `_sources.state.status: user-confirmed`) — the
  field/source split from `curation-charter.md`. It is not part of `fields:` above; it is
  maintained alongside, per write.
