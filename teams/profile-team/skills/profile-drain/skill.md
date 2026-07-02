---
name: profile-drain
description: Fold pending profile-fact queue items into the global person profiles at ~/.atl/profiles — the profile-curator resolves each fact to a person, gates it by privacy tier, applies it, lazily fills the schema, rebuilds the index, and acks it. Run when atl reports "N profile-fact(s) pending", or to process the profile queue manually.
---

# /profile-drain — fold the profile-fact queue into the person profiles

This is the **consuming half** of the profile loop, the sibling of core `/drain`. Capture
(silent `<!-- profile-fact: … -->` markers → `atl tick` → the bbolt queue's `profile-fact`
channel) is automatic and deterministic; this skill is the LLM half. Core `/drain` handles
the `learning` channel only — this skill owns `profile-fact`.

The actual curation (parse, resolve, privacy-gate, apply, lazy-fill, index) is the
`profile-curator` agent's job — its knowledge lives in its `children/`. This skill is the
thin trigger: check for work, hand it to the curator, relay the result.

## Procedure

### 1. Check for pending facts

Run in the current project directory (the queue is keyed by project — a fact is queued
where it was observed, even though profiles are global):

```bash
atl learnings peek --channel profile-fact --json
```

Each item is `{id, channel, payload, enqueued_at}`; `payload` is the raw marker body (the
`entity` / `fields` / `source` YAML). If the list is **empty**, report "no profile facts
to drain" and stop — do not spawn the curator for nothing.

### 2. Hand the facts to the profile-curator

Spawn the **`profile-curator`** agent (the agent named in profile-team's `team.json`
`capabilities.profile.curator`) with the task:

> Drain the `profile-fact` queue for this project into the global profiles at
> `~/.atl/profiles/`. For each pending item: parse the body, resolve the entity, and
> apply it per your charter — honor the 4-tier privacy gate and `source` flags, follow
> the change-policy (overwrite vs history-tracked), and run the schema-version lazy-fill.
> Create a new profile (person interface) when the entity is unknown. After each item is
> integrated, `atl learnings ack <id>` it. When all items are done, rebuild `_index.md`.
> Return a short report: per entity what changed, any new profiles, and anything skipped
> by a privacy gate.

The curator has `Read`/`Write`/`Edit`/`Glob`/`Grep`/`Bash`, so it does the full
peek → apply → ack loop itself. Its `children/` (`marker-drain`, `type-detection`,
`interface-model`, `person-interface`, `curation-charter`, `index-rebuild`) are the
detailed playbook.

### 3. Relay the report

Surface the curator's summary to the user: which people were updated or created, and
anything a privacy tier held back (so a consent-gated fact isn't silently dropped without
the user knowing it was seen).

## Boundaries

- **Never write profile files from this skill directly** — the curator is the single
  writer, so privacy gating and source discipline live in exactly one place.
- **Ack only after a fact is integrated.** A fact the curator could not place (e.g. an
  unresolvable entity) is left un-acked and reported, not dropped.
- This skill does not read or interpret profiles for advice — that is a consuming team's
  lens, not the drain.
