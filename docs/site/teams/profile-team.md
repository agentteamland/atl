# profile-team

**profile-team** curates a shared, cross-project profile of the people **and things** in your
world — your inner world of the entities that carry meaning: people, organizations, animals,
places, objects, and projects. It is the first rebuilt first-party team and a
**global-scope** team: the same entity is one profile in every project you work in. It is the
foundation the paused personal-advisory stack builds on — advisory lenses read the profiles
it maintains.

```bash
atl install profile-team
```

It installs globally by default (its `team.json` declares `scope: global`), landing the
`profile-curator` agent, the `/profile-drain` skill, and the `profile-capture` rule in
`~/.claude`, with profiles stored under `~/.atl/profiles/`.

## The profile world

Everything lives under the global ATL layer at `~/.atl/profiles/`:

```
~/.atl/profiles/
├── _index.md                     # discovery: who exists, salience, role
├── _interfaces/
│   └── person.md                 # the self-describing schema (person, in v1)
└── people/
    └── <slug>/
        ├── profile.md            # frontmatter core + narrative body
        ├── wiki/                 # topic-organized current truth for this person
        └── learnings/            # pattern-organized, KB-rebuilt
```

This world is **entity-organized** and deliberately separate from a project's
**topic-organized** `.atl/wiki/` and `.atl/journal/`. The two cross-reference by free
relative markdown links only. Because profiles are global, they never live inside a
project — profile-blind projects (a pure software repo) pay zero cost, and a person you
mention in one project is the same profile everywhere.

## How it learns — capture then drain

profile-team reuses ATL's marker → queue → drain machinery on a dedicated `profile-fact`
channel, the sibling of the [learning loop](/guide/learning-marker-lifecycle):

1. **Capture.** The `profile-capture` rule teaches the assistant to drop a silent marker
   when a durable fact about a person comes up:

   ```html
   <!-- profile-fact:
     entity: alex
     kind: friend
     fields:
       identity.name: Alex Doe
       traits.fears: [confrontation]
       state.emotional: anxious about the new job
     source: user-confirmed
   -->
   ```

2. **Queue.** `atl tick` (and session start) transfers markers into the durable queue's
   `profile-fact` channel exactly once — deterministic, no LLM. `atl learnings status`
   counts them; `atl learnings peek --channel profile-fact` inspects them.

3. **Signal.** At session start, `atl` surfaces `N profile-fact(s) pending — run
   /profile-drain` (the sibling of the learning signal).

4. **Drain.** `/profile-drain` hands the pending facts to the `profile-curator` agent,
   which resolves each to the right person, applies it (privacy-gated, source-flagged),
   evolves the schema, rebuilds `_index.md`, and acks it. Core `/drain` stays
   `learning`-only — `profile-fact` is profile-team's channel.

## The interfaces

Each entity **type** has its own **self-describing** interface file (`_interfaces/<type>.md`).
Six are seeded — **person, org, animal, place, object, project** — and the world is type-open
(below). An interface's frontmatter carries what it is (`matches` + examples, for type
detection), its `schema-version` + `changelog` (for evolution), `tier-defaults` (privacy),
`thresholds` (type-match + salience), and its allowed enums. The `fields:` block is a
**hybrid** shape: a common **core** every type shares (`meta`, `identity`, `relation-to-user`
incl. `salience`, `emotional-tags`) plus a **type extension** — person adds nine trait fields
+ skills + `state.{emotional,goals,financial}` + relationships; animal adds species +
`adopted`/`passed` anchors + history-tracked health; place adds a bond + sensory-memories;
object adds provenance + a history-tracked status; project adds status/motivation/stakes; org
adds standing + key-people links.

**Type detection.** When a fact names an entity, the curator takes the marker's optional
`type:` hint, else fit-scores the entity against every interface's `matches` + examples and
reuses the best fit at/above the 0.80 threshold.

**Type-open (auto-creation).** When an entity fits none of the seeded types well and is a
*coherent, recurring kind*, the curator **authors a new interface** for it on the fly —
silent but guardrailed (a small extension over the core, conservative default tiers, stamped
`authored: agent-<date>` so it stays reviewable). A genuine one-off is kept as a light
`unknown` stub. This is what lets the profile world hold kinds it was never pre-taught.

**Interface evolution.** There are no required fields — every profile is always valid, and
the curator's discipline is to *fill fields to the extent the evidence supports*, never to
validate. When the interface grows (a minor version adds fields), older profiles are not
batch-migrated; each catches up **lazily** the next time drain touches it — the changelog's
`added` lists drive a deterministic fill. Inference is tolerated but flagged
(`agent-inferred-<date>`), so a wrong guess self-corrects in a later conversation rather
than hardening into fact. Thresholds live in the interface frontmatter (v2 has no config
system by design — they are type-specific, so the interface is their home).

## Privacy

Every field maps to one of four tiers, gating what the curator writes:

| Tier | Example fields | Behavior |
|---|---|---|
| **1 — Open** | identity, anchors, relation kind/role | Always written. |
| **2 — Perception-flagged** | traits | Written; on a third-party profile (`is-self: false`) recorded as *the user's perception*. |
| **3 — Explicit signal** | state.emotional, state.goals | Written **only** from a `user-confirmed` fact; an inferred value is rejected. |
| **4 — Consent-gated** | state.financial | Written **only** if the user has opted in (`meta.consent.<field>`), default off. |

Every written field also records its `source` (`user-confirmed` / `agent-inferred-<date>` /
`lens-set`), and `meta.is-self` marks the user's own profile — the one place the most
sensitive fields may be recorded directly.

## Reading profiles

A consuming team's lens reads profiles directly: it loads `_index.md` on demand to see who
and what exists, then reads the specific `~/.atl/profiles/<type>/<slug>/profile.md` it needs
(profiles are plain markdown, like the wiki). The index is never injected into `CLAUDE.md`
— it is pulled only when a lens is actually reasoning about the user's world.

**Cross-team access** is declared, not open. Once third-party advisory teams arrive, each
declares its profile access in its own `team.json` under `capabilities.profile` (`reads` /
`writes` field lists), surfaced at install time — so a legal-advice team can be granted
different access than a wellbeing team. In v1 the only consumer is personal-advisory-team;
the contract exists now so it is in place before third-party teams read freely.

## What ships

profile-team ships the full loop over **six entity types** (person, org, animal, place,
object, project), each with its own self-describing interface, plus **auto-creation** of a
new interface for a coherent novel kind and interface **evolution** (changelog-driven
lazy-fill). The whole thing runs on the north-star consumer (personal-advisory-team) it was
built for.

Deferred to a later version (design captured, trigger-gated): **scheduled / interval drain**
(today it runs at session start — gated on a separate ATL scheduling primitive); **structured
cross-links** between the profile and project worlds (free markdown links today); the
lazy-**migration** implementation for breaking schema changes; and the consumer-side pieces
that arrive with the first consuming team — **transitive dependency install** and
**`capabilities.profile` enforcement**.

## See also

- [Learning marker lifecycle](/guide/learning-marker-lifecycle) — the sibling loop `profile-fact` mirrors
- [Concepts: scope](/guide/concepts#scope-global-and-project) — how global and project layers interact
- [`atl learnings`](/cli/learnings) — inspect the queue (`--channel profile-fact`)
- [`atl install`](/cli/install) — how a team resolves and installs
