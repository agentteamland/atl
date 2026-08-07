---
knowledge-base-summary: "How a BREAKING interface change (major bump) is applied to existing profiles on touch: the `_interfaces/migrations/<type>/<from-major>.x-to-<to>.md` file format (rename/remove/transform/remap-values ops), keyed by the MAJOR boundary alone so a shipped migration still matches a store whose interface I evolved in place, the apply algorithm with its gate-never-weakens + source-preserved + _sources-atomic invariants, ordered multi-major sequencing, the present/missing decision (missing → leave-on-old-schema + note), how a curator-authored migration is stamped agent-<date>, and the four canonical 1.x→2.0.0 migrations shipped for person/project/animal/object. The breaking-change sibling of interface-creation.md."
---

# Schema Migration (breaking-change apply-on-touch)

`interface-model.md` covers the *common* case: an **add-only** interface growth (minor
bump) is caught up by the changelog-driven lazy fill. This file covers the *breaking*
case: a **major bump** that renames, removes, re-shapes, or re-values a field. A breaking
bump cannot be filled — it must be **migrated**, and a migration mutates existing user
data irreversibly, so it is the most load-bearing thing I do to a profile. I never guess
one: I apply a migration only from an explicit **migration file**, and if none exists I
leave the profile untouched on its old schema and say so.

This is the breaking-change sibling of `interface-creation.md`. Where that authors a new
*interface*, this authors + applies a migration *between* interface versions.

## The migration file

`~/.atl/profiles/_interfaces/migrations/<type>/<from-major>.x-to-<to>.md` — one file per
type, per major-version jump (e.g. `migrations/person/1.x-to-2.0.0.md`). It is **per-type**
because version numbers are type-local: `person` 2.0.0 and `object` 2.0.0 are unrelated
histories. Frontmatter declares the field-level operations; the body is prose the curator
reads for judgment on anything the ops can't fully encode.

### A migration is keyed by the MAJOR boundary, never by an exact from-version

The `from` side is written `<major>.x` and matched on the **major alone**: a profile at
`1.0.0` and one at `1.7.3` are both migrated by `1.x-to-2.0.0.md`. Its minor and patch never
participate in selecting the file.

This is not cosmetic. **An upstream-shipped migration cannot know which minor of the previous
major a given store sits at**, because I author add-only bumps in place: on a live store the
`project` interface had been evolved to `1.2.0` and `place` to `1.1.0` while the shipped
canonical copies were still `1.0.0`. A file keyed `1.0.0-to-2.0.0` would match neither, every
lookup would fall through to the missing-file branch, and every profile of that type would
stall permanently on its old schema — a silent, store-wide freeze that looks exactly like the
mechanism working. Keying on the major boundary is what makes a shipped migration reach the
stores it was written for.

The consequence for the author: a `1.x-to-2.0.0` migration must be written to be correct for
**any** 1.x profile, not just the shipped 1.0.0 — so its ops must name only paths the whole
major shares, and must tolerate a field being absent.

```markdown
---
type-id: person
from: 1.x
to: 2.0.0
# authored: agent-2026-08-01   # present ONLY on a curator-authored migration (see "Where
#                              # migration files come from"); upstream-shipped files omit it
operations:
  - rename: { from: traits.weak-at, to: traits.struggles-with }
  - remove: { field: traits.legacy-note, into: null }
  - transform:
      from: [identity-extension.demographics.location]
      to:   [identity-extension.demographics.city, identity-extension.demographics.country]
      via: "Split a free-text location into city + country; ambiguous → city only, country null."
  - remap-values:
      field: relation-to-user.kind
      map: { romantic: partner }
      drop: []
---
# person 1.4.0 → 2.0.0
Prose intent for each op — enough that a later reviewer (or I, re-touching) understands
*why*, and enough to disambiguate anything the structured op leaves open.
```

### The four ops

| Op | Shape | Use |
|---|---|---|
| `rename` | `{ from, to }` | 1:1 path move — the common, grep-stable case. |
| `remove` | `{ field, into: <path\|null> }` | Drop a field; optionally merge its value into `into`. |
| `transform` | `{ from: [paths], to: [paths], via: <prose> }` | N:M reshape — value-shape change, 1→N split, N→1 merge, per-element array rewrite. The escape hatch for anything `rename` can't say. |
| `remap-values` | `{ field, map: {<old>: <new>}, drop: [<old>] }` | Re-map the *values* of an enum/list field (the old allowed-values are gone from the new interface, so they MUST be carried here). |

`rename` and `remove` are the 90% case. `transform` and `remap-values` exist because the
corpus proves the hard cases are real: `traits.struggles-with` is already annotated
`# merge of "weak-at" + "no-skill-at"` (an N→1 merge), and the interface carries real
enums (`allowed-kinds`, `allowed-roles`) whose members a major bump can rename.

## The apply algorithm

Invoked from `marker-drain.md` §4 when a touched profile's `meta.schema-version` P is
behind the interface's `schema-version` I **and the bump crosses a major boundary**
(`I.major > P.major`). Same-major (minor/patch) stays on the add-only lazy fill — this
file is only for major boundaries.

### Step 0 — validate the whole file before applying anything
A migration is applied atomically per hop: validate first, and if it fails, apply
**nothing** (fall to the missing/malformed fallback below). The gate discipline splits by
what is actually computable at apply time.

**Only the *current* interface I is materialized on disk** — the old version P is gone (an
interface is materialized-forward, never kept per-version). So the "never weaken a gate"
invariant cannot be checked by comparing P's and I's `tier-defaults`; it is an **authoring
responsibility** (the migration author has both versions in view and must not move a value
to a more-open path — the body prose justifies any deliberate loosening). What I *can*
enforce at apply time, using only I's `tier-defaults` + the value's `_sources` flag, is the
one genuinely dangerous state — an under-sourced value at a high-privacy path:

- **No `agent-inferred` value may land at — or be raised to — a Tier-3+ path.** After each
  op, re-derive the destination path's tier from I. Independently, for any field whose
  `tier-defaults` entry *tightened* across this major bump (a frontmatter change to I, not
  one of the ops above), re-gate the existing value in place. In either case, if the field
  is now Tier 3+ and its `_sources` is `agent-inferred`, **strip the value** (skip + record)
  — the "Tier-3 rejects inference" gate `marker-drain.md` §3 applies to fresh writes,
  extended to migration.
- **Never fabricate a Tier-3+ value.** A `transform` that *synthesizes* a value not present
  before, landing at a Tier-3+ path, is rejected (skip + record) — a migration may reshape
  a user-confirmed value, never invent a sensitive one.

### Step 1 — apply each op (source + policy preserving)
- **`rename`** — move the value to the new path, and move its provenance with it:
  set `_sources.<to>` = `_sources.<from>`, then **delete `_sources.<from>`**. A pure path
  move is not new inference — the source is carried verbatim, never downgraded.
- **`remove`** — delete the field and **delete `_sources.<field>`**. If `into` is set,
  merge the value into the target *per the target's `change-policy`*: an `overwrite`
  target takes the value + the removed field's `_sources` entry; a `history-tracked`
  target (`state.*`) routes through the same push-current-onto-history contract as a
  normal write (never a raw set that clobbers the `{current, history}` shape). On a
  source conflict, keep the **lower-confidence** source (`agent-inferred` over
  `user-confirmed`) so provenance never over-claims.
- **`transform`** — apply the `via` prose to reshape/split/merge. Carry `_sources` per the
  prose (union on merge, partition on split). A merge's result takes its **destination
  path's tier under I** (the Step-0 gate then strips it if it lands `agent-inferred` at
  Tier 3+); when the inputs carried different protections, the migration **body prose** is
  the authority on which the merged value must keep — the old input paths' tiers are no
  longer on disk to compare against. For an **array-of-objects** element change, the
  prose iterates per element (e.g. "for each element of `traits.skills`, rename sub-key
  `X`→`Y`, preserving `_sources.traits.skills[].X`") — transform the elements, don't
  clobber the list wrapper. A transform preserves the original source unless it genuinely
  *synthesizes* a value not present before; a synthesized **Tier-3+** value is **rejected**
  (skip + record), never persisted.
- **`remap-values`** — for each stored value, apply `map` (a clean 1:1 remap keeps the
  original source). A value in `drop` (no target in the new vocabulary) is removed; if it
  was the field's only value, the field empties + its `_sources` entry is deleted. Note
  each drop in the report.
- **Core/extension boundary moves.** A `rename`/`transform` that moves a field across the
  hybrid-C core↔extension boundary (per `person-interface.md` "Core vs extension") must
  **also relocate the field's companion `tier-defaults` and `change-policy` entries** — a
  bare path move that forgets them is latent privacy/policy drift. The body prose flags
  any such op.

### Step 2 — add-only fill for the same span, on the set-difference
After the ops complete, run the normal changelog-driven add-only fill
(`interface-model.md`) for the `added` fields of this major span — but **skip any path a
migration op already wrote this span**. A field a rename/transform/remove-into just
populated is not "missing"; the two passes must never contend for the same path (that
would double-write, or clobber a migrated value with an inference).

### Step 3 — advance the version
Set `meta.schema-version` to this hop's ceiling. **Multi-major jumps (e.g. 1.x → 3.x) are
an ordered sequence, not one collapsed apply:** for each major boundary in order —
(1) apply migration file `<b-1>.x-to-<b>.0.0`, (2) run the add-only fill up to b's ceiling,
(3) set `meta.schema-version` to b's ceiling — then advance to the next boundary. The
profile passes through each real intermediate version so the next hop's `from` paths match
what the previous hop produced. If **any** migration file in the chain is missing, halt at
the last good version and note it (the fallback below, applied **per hop**, not just at the
entry gate).

### The missing / malformed fallback
If the required migration file is absent, or fails Step-0 validation, **do not guess.**
Leave the profile on its current `meta.schema-version`, and note it in the drain report:
`"<slug> at <P>, <type> interface at <I>, migration <from>-to-<to> missing/invalid — left
as-is."` A stalled profile is a visible, fixable state; a guessed breaking migration is
silent data corruption.

## Where migration files come from

Two sources, mirroring how interfaces themselves arrive (`person-interface.md` "How this
reaches disk"):

1. **Upstream-shipped (canonical).** A future profile-team version that breaking-bumps an
   interface ships the canonical migration file *in my knowledge* (a block in this child,
   alongside the bumped interface). On drain I ensure
   `_interfaces/migrations/<type>/<from-major>.x-to-<to>.md` exists, materializing it verbatim
   from that block if absent — exactly as I materialize a `<type>.md` interface. These carry
   **no** `authored:` stamp (they are canonical, like a shipped interface). The four blocks
   under "Canonical migrations shipped with this version" below are the first of these.
2. **Curator-authored (provisional).** When I myself perform a breaking bump (e.g. while
   evolving an agent-authored interface), I author the migration file on the fly and stamp
   it **`authored: agent-<today>`** in the frontmatter — the same provisional/reviewable
   marker `interface-creation.md` puts on an agent-authored interface, because a
   curator-authored breaking migration mutates existing user data irreversibly and must be
   at least as guardrailed as a new-interface authoring. The stamp makes it discoverable,
   reviewable, and promotable to canonical.

## Canonical migrations shipped with this version

The first breaking bump the team has ever shipped: four interfaces go to `2.0.0` because
their `state` fields were declared with the wrong *shape*. Two kinds of defect, and they are
not the same fault — read each block's body before applying it.

Nothing below invents a value. Every op is a **reshape of a value already stored**, which
matters at Step 0: the "never fabricate a Tier-3+ value" check rejects a `transform` that
*synthesizes*, and splitting one stored string into the strands it already contains is not
synthesis. Where a path is unchanged, its `_sources` entry is unchanged too — there is
nothing to move.

### `migrations/person/1.x-to-2.0.0.md`

```markdown
---
type-id: person
from: 1.x
to: 2.0.0
operations:
  - transform:
      from: [state.goals]
      to:   [state.goals]
      via: "Single {current, history[]} slot -> a list of { goal, noted }. Absent or null current and empty history -> leave the field absent; do not write an empty list. Otherwise: (a) split `current` into one element per goal the stored text ALREADY separates — an explicit domain label, an enumerated strand, a sentence naming a distinct aim — carrying each strand's wording VERBATIM; if the text does not plainly separate, emit exactly ONE element holding it whole. Never paraphrase, summarize, or complete a strand. (b) `noted` takes a date the strand itself states, else null — never today's date, which would claim the goal was stated now. (c) Append each `history` entry as a further element { goal: <value>, noted: <date> }: those values were filed as past by the very policy this bump removes, so under the concurrent reading they were never shown to have ended. Name every carried history entry and every split in the drain report so a wrong split is visible and correctable."
---
# person 1.x -> 2.0.0

`state.goals` was named plural and typed as one scalar slot under `history-tracked`, so an
arriving goal in one life domain filed a still-live goal in another as *past*. Goals are
concurrent; the field is now list-valued and merged.

**Why history is carried in rather than dropped.** A `history` entry here is not evidence a
goal ended — it is evidence the broken policy displaced it. Join is the safe error: a wrong
join is additive and visible on the page and the next confirmation corrects it, while a wrong
drop deletes something true and leaves no trace it was ever there.

**Splitting is judgment, so it is bounded.** Split only where the stored text separates
itself; when in doubt keep it whole. One over-large element is a cosmetic flaw the next write
can refine; a split that invents a boundary puts words in the user's mouth on a Tier-3 field.

**Tier.** `state.goals` stays Tier 3 and the path does not move, so no gate is weakened and
`_sources.state.goals` carries over untouched. If that entry reads `agent-inferred`, Step 0's
Tier-3 rule applies exactly as it would to a fresh write.
```

### `migrations/project/1.x-to-2.0.0.md`

```markdown
---
type-id: project
from: 1.x
to: 2.0.0
operations:
  - transform:
      from: [state.motivation]
      to:   [state.motivation]
      via: "Single {current, history[]} slot -> a list of { motive, noted }. Same rules as person/state.goals: absent or null and empty history -> leave absent; split `current` only where the stored text already separates distinct motives, carrying wording verbatim, else ONE element holding it whole; `noted` from a date the text states, else null; each `history` entry appended as a further element { motive: <value>, noted: <date> }. Report every split and every carried history entry."
---
# project 1.x -> 2.0.0

`state.motivation` held one slot under `history-tracked`, so a newly stated motive filed a
still-operating one as past. Motives stack — money, craft and obligation coexist on the same
endeavour — so the field is now list-valued and merged.

`state.status` is deliberately **untouched**: its `allowed-statuses` enum makes two values
structurally impossible at once, so supersession is sound there, and it already declared the
paired `{current, history[]}`. This bump changes one field only.

**Tier.** `state.motivation` is Tier 2 and the path does not move — no gate moves, and
`_sources.state.motivation` carries over untouched.
```

### `migrations/animal/1.x-to-2.0.0.md`

```markdown
---
type-id: animal
from: 1.x
to: 2.0.0
operations:
  - transform:
      from: [state.status]
      to:   [state.status]
      via: "Bare enum scalar -> the paired { current, history[] }. If a value is stored, set current to that exact value and start history as an empty list. If the field is absent or null, leave it absent — do not write a { current: null, history: [] } husk. No value is read, invented, or dropped; this only gives the existing value the container the write contract already assumed."
---
# animal 1.x -> 2.0.0

`state.status` declared `history-tracked` against a bare scalar, so the push-old-onto-history
write had no array to push onto. Cardinality was never the problem — `allowed-status` is an
enum, so the values are genuinely mutually exclusive and `history-tracked` is the right
policy. Only the shape was wrong, and it now matches `state.health` directly above it.

`history` starts **empty**, not seeded: the pre-2.0.0 store has no record of what the status
used to be, and inventing one would fabricate a dated fact about a departed animal. The arc a
lens reads begins at the first post-migration transition.

**Tier.** `state.status` stays Tier 1 and the path does not move.
```

### `migrations/object/1.x-to-2.0.0.md`

```markdown
---
type-id: object
from: 1.x
to: 2.0.0
operations:
  - transform:
      from: [state.status]
      to:   [state.status]
      via: "Bare enum scalar -> the paired { current, history[] }. If a value is stored, set current to that exact value and start history as an empty list. If the field is absent or null, leave it absent — do not write a { current: null, history: [] } husk. No value is read, invented, or dropped."
---
# object 1.x -> 2.0.0

Identical in kind to the animal bump: `state.status` declared `history-tracked` against a
bare scalar and had no target for the history push. `allowed-status` is an enum, so the
policy was right and only the shape was wrong.

`history` starts **empty** for the same reason — the pre-2.0.0 store never recorded a prior
standing, so there is nothing truthful to seed it with.

**Tier.** `state.status` stays Tier 1 and the path does not move.
```

## Completion discipline

- [ ] Step-0 validation passed (no op weakens a tier gate; tier-raises stripped) — else the
      whole file refused + noted.
- [ ] Each op applied with `_sources` moved atomically (`<to>` set, `<from>` deleted);
      **no `_sources.<path>` references a field absent from the new schema.**
- [ ] `history-tracked` merge targets routed through push-current-onto-history, not raw set.
- [ ] Add-only fill ran on the set-difference (skipped every migrated path).
- [ ] `meta.schema-version` advanced per hop (multi-major = ordered sequence).
- [ ] Missing/invalid migration → profile left on old schema + noted (never guessed).
- [ ] Any drop / strip / synthesized-Tier-3-rejection surfaced in the drain report.
