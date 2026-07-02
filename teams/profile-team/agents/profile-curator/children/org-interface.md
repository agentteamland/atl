---
knowledge-base-summary: "The canonical org interface — the seed schema I materialize to ~/.atl/profiles/_interfaces/org.md: self-describing frontmatter (matches/examples/schema-version/changelog/tier-defaults/thresholds) + the inherited core + the lean org type-extension (identity-extension, anchors, standing state, key-people links)."
---

# Org Interface (canonical seed)

This is the interface for organizations, companies, institutions, and communities the
user has a genuine tie to. Like every type it is *self-describing*: its own frontmatter
carries what it is (`matches` + examples), its version and change history
(`schema-version` + `changelog`), its privacy tiers (`tier-defaults`), its tunables
(`thresholds`), and its allowed enums — so type detection, lazy fill, and privacy gating
all read from one place. It is the least emotionally loaded of the extension types, so it
stays lean: identity, anchors, standing, and links to the people inside it.

**How this reaches disk.** I keep the canonical copy here (in my knowledge). On drain I
ensure `~/.atl/profiles/_interfaces/org.md` exists; if it is absent I materialize it
verbatim from the block below. If a newer canonical version ships (higher `schema-version`
than the materialized file), I bring the on-disk interface forward and the changelog drives
the lazy fill of every org profile (see `interface-model.md`). Thresholds live here —
**not** in a config system (v2 has none by design); they are type-specific, so the
interface is their natural home.

## The canonical interface

```markdown
---
type-id: org
schema-version: 1.0.0
matches: |
  An organization, company, institution, or community the user has a genuine personal
  relationship with — they work there, study(ied) there, are a member, a client, a partner.
  NOT any company merely mentioned in passing.
  NOT a brand the user only buys from without an actual tie.
examples-positive:
  - "My company reorged our whole team this week"
  - "I graduated from METU back in 2015"
  - "The climbing club I'm in is organizing a trip"
examples-negative:
  - "Apple announced a new phone today"
  - "I ordered a coffee from Starbucks"
  - "The government passed a new tax law"
changelog:
  - version: 1.0.0
    added: [everything]   # initial org interface
thresholds:
  # Fit score at or above which this existing interface is reused instead of a new type
  # being created. Kept here so it travels with the type.
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
  identity-extension.location: 1
  anchors.*: 1
  state.standing: 1
  state.user-relationship-status: 2
  key-people.*: 1
allowed-kinds: [employer, client, alma-mater, member, partner, competitor]
allowed-org-kinds: [company, school, institution, community, government]
allowed-standing: [thriving, stable, struggling, defunct]
fields:
  # ---- common core (inherited — see interface-model.md) ----
  # meta / identity / relation-to-user (incl. salience) / emotional-tags come from the
  # shared core; relation-to-user.kind draws from allowed-kinds above, and role captures
  # what the user is there (employee, customer, member, founder, ...).

  # ---- org type-extension ----
  identity-extension:
    kind: <one of allowed-org-kinds>   # what sort of organization this is
    domain: <str|null>                 # industry / field / focus (e.g. "fintech", "public university")
    location: <str|null>               # HQ / campus / where the user relates to it
  anchors:
    user-joined: <ISO date|null>       # when the user's tie began
    user-left: <ISO date|null>         # when it ended (null = ongoing)
  state:                               # current + optional history (change-policy below)
    standing: { current: <one of allowed-standing|null>, history: [ { value: <str>, date: <ISO date> } ] }
    user-relationship-status: <str|null>   # the user's current standing with it ("active employee", "alum", "on leave")
  key-people:                          # links into the person profiles inside this org
    - { to: <person-slug>, role: <str> }   # e.g. { to: deniz-yilmaz, role: manager }
change-policy:
  # default overwrite (decision D); history opt-in only where temporal evolution matters
  identity-extension.*: overwrite
  anchors.*: overwrite
  state.standing: history-tracked
  state.user-relationship-status: overwrite
  key-people.*: overwrite
---

# Org

Notes about this organization live in the narrative body of each org's profile.md, not
here. This interface file is the shared schema every org profile is grown against.
```

## Reading the schema

- **Core vs extension** (hybrid-C architecture): `meta` / `identity` / `relation-to-user`
  / `emotional-tags` are inherited from the common core (defined in `person-interface.md` /
  `interface-model.md`) and are **not** re-listed here; everything under
  `identity-extension` down is org-specific and evolves independently of the core.
- **No required fields.** The schema is a *target to fill toward*, never a validator. An org
  with only `identity.name` is a valid profile. Adding a field keeps old profiles valid —
  they lazy-fill on next touch.
- **`tier-defaults`** feeds the privacy gating in `curation-charter.md`; **`change-policy`**
  feeds the override policy in `interface-model.md`; **`thresholds`** feed salience
  auto-compute and type detection. This type is deliberately lean — identity/anchors/kind
  and the `key-people` links are Tier 1 (structural, non-sensitive); only
  `user-relationship-status` is Tier 2 (how the user stands with it can be personal). It
  needs no Tier 3-4 field. `key-people` links resolve to person profiles by slug.
- **`_sources` parallel namespace.** Each written field gets an entry under the parallel
  `_sources.<path>` key in the profile (e.g. `_sources.state.standing: user-confirmed`) —
  the field/source split from `curation-charter.md`, maintained alongside, per write.

