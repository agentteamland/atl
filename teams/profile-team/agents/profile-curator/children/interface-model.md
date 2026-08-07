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
| `identity.*`, `anchors.*`, `relation-to-user.*` (scalars) | overwrite (latest wins) |
| `traits.*`, `person.state.goals`, `project.state.motivation`, and every other **list-valued** field | overwrite with **merge** — an arriving value joins the set, it does not replace it |
| `state.emotional`, `state.financial`, and the enum `state.status` / `state.standing` fields | history-tracked (`current` moves; prior value appends to `history` with its date) |

A history-tracked write pushes the old `current` (with its date) onto the `history` array
and sets the new `current`. This is what lets a lens ask "they were low in March, better
in June — is the pattern repeating?"

**History-tracked is a supersession policy, so it is only sound where the values are
mutually exclusive.** On a field that legitimately holds several values at once it does
not merge — it files a still-true value as *past* because an unrelated one arrived, which
is a silent deletion. The reliable tell is the type: every sound history-tracked field
here is **enum-constrained** (a status, a standing), and the free-text ones
(`state.emotional`, `state.financial`) are sound only under the composite-snapshot reading
their field comments now state explicitly. Concurrent things — goals, motives, traits —
are **list-valued and merged**, never versioned. See `interface-creation.md` for the
cardinality-first authoring rule and `marker-drain.md` for the write semantics.

A list merge defaults to **join**, and only the incoming fact itself may retire a standing
entry. Join is the safe error: a wrong join is additive and visible, and the next
confirmation corrects it; a wrong replace deletes something true and leaves no trace.

Every shipped interface now obeys this rule. `person.state.goals` and
`project.state.motivation` were re-shaped to lists, and `animal.state.status` /
`object.state.status` were given the paired `{current, history[]}` their policy needs — the
`2.0.0` bump in each, with its migration in `schema-migration.md`. `person.state.emotional`,
`person.state.financial` and `animal.state.health` remain history-tracked and are sound only
under the composite-snapshot reading their field comments state — one overall mood, one
overall standing, one overall wellbeing note — not as itemised lists.

The write-side guard in `marker-drain.md` stays in force regardless: on a field that plainly
holds several values at once, **merge instead of displacing, and report the interface as
needing correction.** It now protects against a mis-shaped *agent-authored* interface rather
than a shipped one.

## Backwards compatibility (semver)

The interface evolves under standard semver, protected by the consumer contract
(`requires: profile-team@^1.0.0`):

- **Add-only field expansion = minor bump.** BC preserved; lazy fill covers it end-to-end.
  This is the common case and fully implemented.
- **Rename / remove / type change = major bump.** BC broken; the changelog gets a
  `breaking: [...]` entry, and the change is applied by a **migration**, not a fill — see
  `schema-migration.md` for the file format
  (`_interfaces/migrations/<type>/<from-major>.x-to-<to>.md`, keyed by the major boundary
  alone) and the apply-on-touch algorithm. A migration is applied only from an explicit
  migration file: **present** → apply its ops (gate-never-weakened, `_sources` carried,
  add-only fill on the remaining span); **missing or malformed** → leave the profile on its
  old schema and note it, rather than guessing. `person`, `project`, `animal` and `object`
  are at `2.0.0` and ship canonical migrations; `org` and `place` have never had a breaking
  bump.

### Bringing a newer canonical interface forward over one I evolved in place

An interface on disk is not always the shipped one. I author **add-only** bumps into it
myself (`interface-creation.md` step 2 applied to an existing type) — so a store can sit at
`1.2.0` while the shipped canonical is still `1.0.0`, carrying fields, `tier-defaults` and
`change-policy` entries that exist nowhere upstream and hold real user content.

So when a shipped canonical outranks the materialized copy, **materializing the canonical
block verbatim would delete those entries from the schema** while their values stay in the
profiles — orphaning them: no tier to gate a later write by, no policy to apply, and no
changelog recording that they were ever declared. Nothing errors; the loss is silent.

**Compare the two changelogs before overwriting anything.** An on-disk entry is *mine* when
its `version` is absent from the canonical's changelog, or present there declaring a
different `added:` set. Decide on that comparison alone — **not** on the `authored:` stamp,
which is real evidence when present but is not reliably written on an in-place bump, so a
stamp test under-fires exactly where the drift is quietest.

- No such entry → bring the canonical forward normally.
- One or more → **do not overwrite the interface.** Leave it and its profiles exactly as they
  are and report it: `"<type> interface on disk at <P> carries entries (<versions>) absent
  from the shipped <I>; left as-is — the shipped bump cannot be applied without dropping
  them."`

This is the **same posture as a missing migration file**, this file's existing discipline for
"I cannot do this correctly": a stalled type is a visible, fixable state a human can
reconcile, and a silently narrowed schema is not. Reconciling the two changelogs is an open
design question — there is no decided merge model, so do not invent one on the fly.

The stall is worst precisely where I did the most work, since the entries at risk are the
ones I added because a real fact had nowhere else to land. That is the argument for
reconciling it soon, not for guessing now.

## Type detection

Each seeded interface carries `matches` / `examples-*` self-descriptions and a
`thresholds.type-match` (0.80): the curator scores an entity against every interface, reuses
the best fit at/above threshold, and below it either authors a new interface for a coherent
novel kind or holds an `unknown` stub — see `type-detection.md` and `interface-creation.md`.
Type *detection* and interface *evolution* are two halves of one self-describing-interface
model: types are detected, and an interface then **grows** (add-only lazy fill, above) or
**migrates** (a breaking change, `schema-migration.md`).
