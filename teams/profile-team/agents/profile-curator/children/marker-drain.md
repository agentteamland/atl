---
knowledge-base-summary: "The primary production unit: process one profile-fact into the right profile. Parse the body, resolve the entity, gate each field by tier + source, apply per change-policy with a source flag, run lazy-fill, ack. Create a new profile when the entity is unknown. Then rebuild the index."
---

# Marker Drain (blueprint)

This is my primary production unit — turning one `profile-fact` queue item into a durable,
tier-respecting, honestly-sourced profile change. Every item goes through this procedure.
It is invoked by the `/profile-drain` skill, which peeks the queue and hands me the batch.

## Input

A queue item's `payload` is a marker body authored per the `profile-capture` rule:

```yaml
entity: alex
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
Read `entity` (the slug), the optional `is-self`/`kind`/`role` hints, the `fields` map,
and `source` (default `user-confirmed`). If the body doesn't parse, leave the item
**un-acked** and report it — never guess a malformed fact into a profile.

### 2. Resolve the entity
Look for `~/.atl/profiles/people/<entity>/`. If absent, also check `_index.md` and existing
profiles' `identity.aliases` for a match (the same person under a different slug). Then:
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

### 4. Lazy-fill (existing profile)
Compare `meta.schema-version` against the interface's `schema-version`. If behind, run the
changelog-driven fill from `interface-model.md`: add the missing fields, fill what the
evidence supports (inference tolerated + flagged; Tier-3+ inference still rejected), set
`meta.schema-version` to the interface version. Then go to §6.

### 5. Create a new profile
1. **Type** — detect the entity type (`type-detection.md`). In v1 this resolves to
   `person`; a non-person entity is held as a minimal `unknown`-type stub, not fabricated
   into a person.
2. **Interface** — ensure `~/.atl/profiles/_interfaces/person.md` exists; if absent,
   materialize it verbatim from `person-interface.md`.
3. **Scaffold** `~/.atl/profiles/people/<slug>/profile.md` with `meta` (`type-id`,
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

### 7. Ack
Only once the item is fully integrated: `atl learnings ack <id>`. The queue deletes it, so
re-drains are safe (dedup) and nothing re-reports.

## After the batch — rebuild the index
When every item is processed, rebuild `~/.atl/profiles/_index.md` (`index-rebuild.md`).

## Completion checklist
- [ ] Body parsed (malformed → un-acked + reported)
- [ ] Entity resolved (slug or alias) or created
- [ ] Each field tier-gated; Tier-3 inference / un-consented Tier-4 skipped + reported
- [ ] Change-policy applied (overwrite vs history-tracked)
- [ ] `_sources.<path>` set for every written field
- [ ] `meta.schema-version` current (lazy-fill run)
- [ ] Salience recomputed (manual override respected)
- [ ] Item acked **after** integration; index rebuilt after the batch
