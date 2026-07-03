---
knowledge-base-summary: "The primary production unit: process one profile-fact into the right profile. Parse the body, resolve the entity, gate each field by tier + source, apply per change-policy with a source flag, run lazy-fill, ack. Create a new profile when the entity is unknown — but only past a reality gate that drops documentation-example / placeholder payloads (never a real entity). Three terminal states: integrated→ack, dropped→ack+report, un-placeable→un-acked. Then rebuild the index."
---

# Marker Drain (blueprint)

This is my primary production unit — turning one `profile-fact` queue item into a durable,
tier-respecting, honestly-sourced profile change. Every item goes through this procedure.
It is invoked by the `/profile-drain` skill, which peeks the queue and hands me the batch.

## Input

A queue item's `payload` is a marker body authored per the `profile-capture` rule:

```yaml
entity: alex
type: person            # optional hint: person|org|animal|place|object|project
is-self: false          # optional
kind: friend            # optional
role: null              # optional
fields:
  identity.name: Alex Doe
  traits.fears: [confrontation, abandonment]
  state.emotional: anxious about the new job
source: user-confirmed  # optional; default user-confirmed
```

## Per-item procedure

### 1. Parse
Read `entity` (the slug), the optional `type:` hint + `is-self`/`kind`/`role` hints, the
`fields` map, and `source` (default `user-confirmed`). If the body doesn't parse:
- if it is recognizably an **illustrative placeholder** — literal skeleton tokens
  (`entity`/`field`/`value` as the actual values, `serbest metin`, `<...>`, `…`), or a bare
  `entity:`/`fields:` husk with no real values — **Drop** it (ack + report, per the reality
  gate §5.0). This is documentation shrapnel the capture scan swept up, not a fact.
- otherwise it looks like a real fact but is corrupt → leave it **un-acked** and report it;
  never guess a malformed fact into a profile, and a corrupt-looking real fact is worth a
  human's eye.

### 2. Resolve the entity
Look for `<entity>/` across the type directories under `~/.atl/profiles/` (`people/`,
`orgs/`, `animals/`, `places/`, `objects/`, `projects/`, `unknown/`) — `_index.md` is the
fast lookup. Also match against existing profiles' `identity.aliases` (the same entity under
a different slug). Then:
- **Found** → existing profile, go to §3.
- **Not found** → new profile, go to §5.

### 3. Apply each field (existing profile)
For every `field-path: value` in `fields`:

1. **Tier** — look up the field's tier from the interface `tier-defaults` (e.g.
   `state.emotional` → 3), honoring any field-level override.
2. **Gate** (per `curation-charter.md`):
   - **Tier 1** → write.
   - **Tier 2** → write; if `is-self: false`, set `meta.perception-flag: true`.
   - **Tier 3** → write **only if** `source == user-confirmed`; otherwise **skip + record**
     it for the report (an `agent-inferred` Tier-3 fact is rejected, not downgraded).
   - **Tier 4** (`state.financial`) → write **only if** `meta.consent.<field>: true`;
     otherwise skip + record.
3. **Change-policy** (from the interface):
   - `overwrite` → set the field to the new value.
   - `history-tracked` (`state.emotional`/`goals`/`financial`) → push the old
     `current` + its date onto `history`, then set the new `current`.
4. **Source flag** — set `_sources.<field-path>` to the effective source
   (`user-confirmed` | `agent-inferred-<today>` | `lens-set`).
5. Update `meta.last-updated` to today.

### 4. Lazy-fill / migrate (existing profile)
Compare `meta.schema-version` P against the interface's `schema-version` I. If behind,
branch on whether the bump crosses a major boundary:
- **Same major** (minor/patch, `I.major == P.major`) → the changelog-driven **add-only
  fill** from `interface-model.md`: add the missing fields, fill what the evidence supports
  (inference tolerated + flagged; Tier-3+ inference still rejected), set `meta.schema-version`
  to I.
- **Major boundary** (`I.major > P.major`) → a **breaking migration**, per
  `schema-migration.md`: apply the `_interfaces/migrations/<type>/<from>-to-<to>.md` file
  (validated so no op weakens a tier gate; `_sources` carried atomically), then the add-only
  fill on the remaining span, advancing one major hop at a time. If the migration file is
  missing or malformed, leave the profile on P and note it — never guess a breaking change.

Then go to §6.

### 5. Create a new profile
A new profile is the **one place a fabricated entity can enter the store**, so creation runs
a reality gate first. An *existing* entity (§2 Found → §3) is already corroborated-real by
its own profile and is **never** reality-gated — this gate guards only new-entity creation.
(A stray archetypal-filler field on an existing real profile is not scrubbed here either:
§3 applies it tier+source-gated, and its source flag + next-conversation overwrite are the
tolerance — consistent with the never-validate discipline.)

**5.0 Reality gate — is this a real entity, or a documentation example?**
The `profile-fact` queue can carry markers that never came from a deliberate `profile-capture`
emission: the capture scan reads the assistant's own prose, and an assistant *illustrating*
the marker format writes example markers the scan can't tell from real ones (this is why the
queue sometimes holds `entity: ahmet`/`emre` with only a stock trait, `serbest metin`,
`entity/field/value`). Before creating a new profile, judge:

- **Drop** (ack + report; create **nothing** — no `profile.md`, no `_interfaces/` file, no
  index entry, and never reaching the interface-authoring path in §5.1) when the payload is
  an illustration/placeholder, not a real entity:
  - a literal format placeholder / skeleton (also caught at §1 — listed here for completeness);
  - a new entity with **no relationship anchor and no situational specificity**: no `kind`/
    `role`, no real situation in the fields, and nothing in the drained conversation that
    references it as a real person — carrying only archetypal filler (a bare name + a stock
    trait like `fears: confrontation`). That bare shape is exactly how the docs illustrate the
    format.
- **Proceed** to 5.1 otherwise.

**The discriminator is the relationship anchor / specificity, NOT the trait's shape.**
`kind: friend` + "just started an anxious new job" is a real, plainly-stated fact and MUST be
created — brevity is not archetypality, and a stated relationship is the tell of a real person
the user actually knows. **When genuinely uncertain whether a new entity is real, Proceed**
(create it, source-flagged): a wrongly-created profile self-corrects (source flag +
next-conversation overwrite), whereas the gate exists to catch *clear* illustrations, not to
adjudicate borderline-real people. Corroboration in the drained conversation
(`atl learnings transcript`) may only **rescue** a would-be-Drop (an entity that turns out to
be a discussed real person → Proceed); its *absence* is never itself grounds to Drop — a fact
mined from a prior session is legitimately absent from the current conversation.

This gate owns exactly one case — **not-a-real-entity** (illustration/placeholder → Drop,
write nothing). It does not touch the two cases `type-detection.md` already owns: a
real-but-untypeable entity still becomes a minimal `unknown` stub, and a public figure /
fictional character is still skipped there. Neither is a reality-gate Drop.

Once past the gate:

1. **Type** — detect the entity type (`type-detection.md`): the marker's `type:` hint, else
   fit-score against the seeded interfaces. Below threshold, judge per `interface-creation.md`:
   author a new interface for a coherent novel kind, else hold a minimal `unknown` stub
   (core-only) — never force it into the closest type.
2. **Interface** — for a seeded type, ensure `~/.atl/profiles/_interfaces/<type>.md` exists,
   materializing it from the matching `<type>-interface.md` child if absent; for a newly
   authored type, `interface-creation.md` has already written the interface; an `unknown`
   stub needs no interface (common core only).
3. **Scaffold** `~/.atl/profiles/<type-dir>/<slug>/profile.md` (the type→dir map is in
   `curation-charter.md`) with `meta` (`type-id`,
   `schema-version` from the interface, `created`/`last-updated` = today, `confidence`,
   `is-self` from the marker (default `false`), `consent: {}`), `identity.name` (from
   `fields.identity.name` or a readable form of the slug), and `relation-to-user`
   (`kind`/`role` from the marker hints, `first-noticed` = today, `salience` initial,
   `salience-source: drain-auto-<today>`).
4. Apply the marker's `fields` exactly as in §3 (same gating), then §6.

### 6. Salience
Recompute `relation-to-user.salience` from recent activity using the interface
`thresholds.salience` (marker/touch count over the window: `high` ≥ 10, `medium` 3–9, else
`low`). Track enough recent-touch dates in `meta` to compute this across drains. **Respect
a manual override:** if `salience-source` is `user-set`/`lens-set`, leave it.

### 7. Ack — three terminal states
Every item ends in exactly one of three states:
- **Integrated** (applied to a profile, §3–§6) → `atl learnings ack <id>`.
- **Dropped** (reality gate §5.0, or a §1 placeholder) → `atl learnings ack <id>` **and**
  name it in the report. A Drop is a real *processing* outcome, not a failure — the item was
  handled, it just wasn't a real fact.
- **Un-placeable** (a corrupt-looking real fact, an unresolvable entity) → leave it
  **un-acked** and report it; a human looks next time.

Ack deletes the item. It leaves no tombstone, and the capture cursor re-scans a
still-growing transcript, so a Dropped illustration the assistant keeps writing in the
*current* session can re-enqueue and re-Drop until that transcript ages behind the cursor —
**bounded, always visible in the report, and never a fabricated profile.** This is not silent
data loss: the gate's promise is that the junk never becomes a profile, not that it never
re-appears in the queue.

## After the batch — rebuild the index
When every item is processed, rebuild `~/.atl/profiles/_index.md` (`index-rebuild.md`).

## Completion checklist
Each item resolves to one of three shapes:

**Dropped** (reality gate §5.0 / §1 placeholder):
- [ ] Verdict = Drop → nothing written (no profile, no interface, no index entry)
- [ ] Acked + named in the report (the resolve/apply/tier/salience lines are N/A)

**Un-placeable** (a corrupt-looking real fact, an unresolvable entity):
- [ ] Left **un-acked** and named in the report (the resolve/apply/tier/salience lines are N/A)

**Integrated:**
- [ ] Body parsed (corrupt-looking → un-acked; illustrative placeholder → Drop)
- [ ] Entity resolved (slug or alias); a **new** entity passed the reality gate (§5.0)
- [ ] Each field tier-gated; Tier-3 inference / un-consented Tier-4 skipped + reported
- [ ] Change-policy applied (overwrite vs history-tracked)
- [ ] `_sources.<path>` set for every written field
- [ ] `meta.schema-version` current (add-only lazy-fill, or a breaking migration per `schema-migration.md`)
- [ ] Salience recomputed (manual override respected)

- [ ] Item acked (integrated **or** dropped); un-acked only if un-placeable; every Drop named in the report; index rebuilt after the batch
