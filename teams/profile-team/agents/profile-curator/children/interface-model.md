---
knowledge-base-summary: "How interfaces evolve and profiles stay current: the profile.md layout, schema-version + changelog diff, changelog-driven lazy fill (inference tolerated + source-flagged, Tier-3+ inference rejected), the override/history policy, and BC via semver."
---

# Interface Model

The mechanism that lets the schema grow without ever breaking an existing profile:
*schema-as-evolving-interface + lazy migration + opportunistic enrichment on touch.* A
profile is never batch-migrated; each one catches up to the current interface the next
time drain touches it.

## The profile.md layout

Each person is a directory `~/.atl/profiles/people/<slug>/`:

```
people/<slug>/
├── profile.md        # frontmatter = the schema fields (per person-interface.md); body = free narrative
├── wiki/             # topic-organized current truth for this person (overwritable pages)
└── learnings/        # pattern-organized, KB-rebuilt (recurring patterns about this person)
```

`profile.md` frontmatter holds the structured fields; the markdown body holds the human
narrative (the story a lens reads for nuance the fields can't carry). `wiki/` and
`learnings/` are the person's *internal* organization — they do not blur with the
project-scoped `.atl/wiki/` and `.atl/journal/`; the two worlds cross-reference by free
relative markdown links only.

## Schema-version + changelog diff

Every profile records `meta.schema-version` — the interface version it was last grown
against (e.g. `1.0.0`). The interface (`_interfaces/person.md`) carries its own
`schema-version` plus a `changelog` list of `{ version, added: [...] }` (and, on a major
bump, `breaking: [...]`).

**The diff is deterministic:** when the profile is behind, the fields to fill = the union
of every `added` entry in the changelog *after* the profile's version. No guessing which
fields are new — the changelog is the authority. Example: profile at `1.2.0`, interface
at `1.4.0` → fill the `added` fields from the `1.3.0` and `1.4.0` changelog entries.

## Lazy fill (enrichment on touch)

When drain touches a profile whose `meta.schema-version` is behind the interface:

1. Compute the missing fields from the changelog diff (above).
2. For each missing field, **attempt to fill from what is already known** — the
   conversation context being drained, the profile's own narrative body, its `wiki/`
   pages. What can be filled is filled; what can't stays absent.
3. **Inference is tolerated but flagged.** A value inferred rather than user-stated is
   written with `source: agent-inferred-<date>`. Mesut's rule: *"Is there a hallucination
   risk? Not very important — if it's wrong it gets corrected in the next conversation."*
   Source-flagging is what makes that self-correction possible; a later `user-confirmed`
   value overwrites the inference cleanly.
4. **Privacy gate still applies.** A Tier-3 field (`state.emotional`, `state.goals`,
   `relationships.*.user-perceives`) is **never** filled by inference — only a
   `user-confirmed` fact writes it. A Tier-4 field (`state.financial`) needs
   `meta.consent.<field>: true`. Lazy fill obeys the same tiers as a normal write (see
   `curation-charter.md`).
5. Set `meta.schema-version` to the interface version. The profile is now current.

Lazy fill never removes or rewrites a field the evidence doesn't touch — it only *adds*
what the new interface introduced, filling opportunistically.

## Override & history policy

Per-field `change-policy` (declared in the interface frontmatter). The default is
**overwrite** — Mesut: *"it gets overwritten automatically ... keeping the current data
is more important."* History is opt-in, and only on the fields PAT's temporal lens needs:

| Field | Policy |
|---|---|
| `identity.*`, `anchors.*`, `relation-to-user.*`, `traits.*` | overwrite (latest wins) |
| `state.emotional`, `state.goals`, `state.financial` | history-tracked (`current` moves; prior value appends to `history` with its date) |

A history-tracked write pushes the old `current` (with its date) onto the `history` array
and sets the new `current`. This is what lets a lens ask "they were low in March, better
in June — is the pattern repeating?"

## Backwards compatibility (semver)

The interface evolves under standard semver, protected by the consumer contract
(`requires: profile-team@^1.0.0`):

- **Add-only field expansion = minor bump.** BC preserved; lazy fill covers it end-to-end.
  This is the common case and fully implemented.
- **Rename / remove / type change = major bump.** BC broken; the changelog gets a
  `breaking: [...]` entry, and the change is applied by a **migration**, not a fill — see
  `schema-migration.md` for the file format (`_interfaces/migrations/<type>/<from>-to-<to>.md`)
  and the apply-on-touch algorithm. A migration is applied only from an explicit migration
  file: **present** → apply its ops (gate-never-weakened, `_sources` carried, add-only fill
  on the remaining span); **missing or malformed** → leave the profile on its old schema and
  note it, rather than guessing. v1 ships no breaking change, so no canonical migration file
  exists yet — the mechanism is defined and ready for the first one.

## Type detection

Each seeded interface carries `matches` / `examples-*` self-descriptions and a
`thresholds.type-match` (0.80): the curator scores an entity against every interface, reuses
the best fit at/above threshold, and below it either authors a new interface for a coherent
novel kind or holds an `unknown` stub — see `type-detection.md` and `interface-creation.md`.
Type *detection* and interface *evolution* are two halves of one self-describing-interface
model: types are detected, and an interface then **grows** (add-only lazy fill, above) or
**migrates** (a breaking change, `schema-migration.md`).
