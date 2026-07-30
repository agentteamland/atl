---
name: profile-drain
description: Fold pending profile-fact queue items into the global entity profiles at ~/.atl/profiles — mine the conversation for facts no marker recorded, then the profile-curator resolves each fact to an entity, gates it by privacy tier, applies it, lazily fills the schema, rebuilds the index, and acks it. Run when atl reports "N profile-fact(s) pending" or fires the capture-watchdog (profile-fact) signal (then the queue may be empty — the mining step recovers the marker-less stretch), or to process the profile queue manually.
---

# /profile-drain — fold the profile-fact queue into the person profiles

This is the **consuming half** of the profile loop, the sibling of core `/drain`. Capture
(silent `<!-- profile-fact: … -->` markers → `atl tick` → the bbolt queue's `profile-fact`
channel) is automatic and deterministic; this skill is the LLM half. Core `/drain` handles
the `learning` channel only — this skill owns `profile-fact`.

The actual curation (parse, resolve, privacy-gate, apply, lazy-fill, index) is the
`profile-curator` agent's job — its knowledge lives in its `children/`. This skill stays
thin: recover what went unmarked, check for work, hand it to the curator, relay the
result. It never writes a profile itself.

## Procedure

### 1. Mine the conversation for unmarked facts

Before peeking the queue, harvest what the agent forgot to mark — the same recovery
pass core `/drain` runs for its channel. Without it a `profile-fact` that was never
written is invisible to the whole pipeline: the queue only ever holds what someone
remembered to mark, so the marker is the one non-deterministic link and this step is
its safety net.

Read the recent conversation flow (prose only — tool calls/results are stripped):

```
atl learnings transcript
```

Scan it for **durable** facts about entities in the user's inner world that no marker
recorded — the same bar the `profile-capture` rule sets: a person, org, animal, place,
object, or project they have a real personal bond with, and a fact about it that will
still be true next month. Identity and relation, an anchor date, a trait, a link to
another entity.

**This is the capture-watchdog case.** When the drain was triggered by an
`atl: capture-watchdog (profile-fact)` signal (a stretch with no profile markers), the
queue may be **empty and that's expected** — this mining step IS the run's purpose.

For each one, write the marker body and enqueue it exactly as a marker would:

```
atl learnings _enqueue profile-fact "<the marker body: entity / type / fields / source>"
```

Use the same YAML body shape the `profile-capture` rule defines, so the curator parses
a mined fact and a marked one identically.

Two disciplines that matter more here than on the learning channel:

- **Be honest about `source`.** A fact the user plainly stated in the transcript is
  `user-confirmed`; anything you are concluding *from* the flow is `agent-inferred`.
  Tier-3 fields are written only from `user-confirmed`, so a mislabeled inference
  hardens a guess into fact — do not launder one to make it land.
- **Skip the session's own examples.** Documentation, tests, and marker-format samples
  in the conversation are not real entities. The curator's reality gate is the backstop,
  but mining them wastes a round-trip and, when ATL is being developed under ATL, floods
  the queue with fixture names.

Be **strict** — mine a durable fact, not small talk. The queue dedups by content hash,
so re-mining the same fact is a safe no-op; don't lean on that to lower the bar. If
nothing qualifies, enqueue nothing and move on.

### 2. Check for pending facts

Run in the current project directory (the queue is keyed by project — a fact is queued
where it was observed, even though profiles are global):

```bash
atl learnings peek --channel profile-fact --json
```

Each item is `{id, channel, payload, enqueued_at}`; `payload` is the raw marker body (the
`entity` / `fields` / `source` YAML). If the list is **empty** — mining found nothing and
nothing was pending — report "no profile facts to drain" and stop; do not spawn the curator
for nothing. On a watchdog-triggered run that is a correct outcome, not a failure: the
stretch genuinely held no durable entity fact.

### 3. Hand the facts to the profile-curator

Spawn the **`profile-curator`** agent (the agent named in profile-team's `team.json`
`capabilities.profile.curator`) with the task:

> Drain the `profile-fact` queue for this project into the global profiles at
> `~/.atl/profiles/`. For each pending item: parse the body, resolve the entity, and
> apply it per your charter — honor the 4-tier privacy gate and `source` flags, follow
> the change-policy (overwrite vs history-tracked), and run the schema-version lazy-fill.
> Create a new profile when the entity is unknown — but first apply the **reality gate**
> (`marker-drain.md` §5.0): if a new-entity payload is a documentation example or format
> placeholder (a bare name with only a stock trait, `serbest metin`, `entity/field/value`),
> **drop** it (ack + report) instead of materializing a fabricated person; an existing
> profile is proof-of-realness and is never reality-gated. After each item is integrated
> **or dropped**, `atl learnings ack <id>` it; leave only an un-placeable item un-acked.
> When all items are done, rebuild `_index.md`. Return a short report: per entity what
> changed, any new profiles, anything a privacy gate skipped, and anything the reality gate
> dropped as a non-real example.

The curator has `Read`/`Write`/`Edit`/`Glob`/`Grep`/`Bash`, so it does the full
peek → apply → ack loop itself. Its `children/` (`marker-drain`, `type-detection`,
`interface-model`, `person-interface`, `curation-charter`, `index-rebuild`) are the
detailed playbook.

### 4. Relay the report

Surface the curator's summary to the user: which people were updated or created, anything a
privacy tier held back (so a consent-gated fact isn't silently dropped without the user
knowing it was seen), and anything the **reality gate dropped as a non-real example** (so a
real fact wrongly dropped is visible, never silent).

## Boundaries

- **Never write profile files from this skill directly** — the curator is the single
  writer, so privacy gating and source discipline live in exactly one place.
- **Three terminal states, not two.** A fact *integrated* into a profile is acked. A payload
  the reality gate judges *not a real entity* (a documentation example / placeholder) is
  **acked + reported as dropped** — it was processed, it just wasn't real. Only a fact the
  curator *could not place* (an unresolvable entity, a corrupt-looking body) is left
  **un-acked** and reported for a human. Ack deletes the item, so an example the assistant
  keeps writing in the current session may re-surface + re-drop until its transcript ages out
  — bounded and visible, never a fabricated profile.
- This skill does not read or interpret profiles for advice — that is a consuming team's
  lens, not the drain.
