---
knowledge-base-summary: "What I own under ~/.atl/profiles/, the self/third-party distinction, the 4-tier privacy framework, and the source-flag discipline every write obeys."
---

# Curation Charter

The always-applicable ground rules for every profile I touch. The mechanics of *how* I
drain, detect types, and fill fields live in their own children; this file is *what I am
allowed to record and how I attribute it* — the discipline that applies before any of
that.

## The world I own

Everything under the global layer at `~/.atl/profiles/`:

```
~/.atl/profiles/
├── _index.md                     # discovery: who/what exists, salience, role (I rebuild it)
├── _interfaces/                  # the self-describing schemas — six types seeded
│   ├── person.md                 #   (+ agent-authored interfaces for novel types)
│   ├── org.md · animal.md · place.md · object.md · project.md
│   └── migrations/<type>/        # breaking-change migration files (<from>-to-<to>.md);
│                                 #   applied on touch, see schema-migration.md
└── <type-dir>/                   # one directory per type: people · orgs · animals ·
    └── <slug>/                   #   places · objects · projects (+ unknown/ for stubs)
        ├── profile.md            # frontmatter core + narrative body
        ├── wiki/                 # topic-organized current truth (overwritable)
        └── learnings/            # pattern-organized, KB-rebuilt
```

An entity's `meta.type-id` (singular) maps to its directory (plural): person→`people`,
org→`orgs`, animal→`animals`, place→`places`, object→`objects`, project→`projects`;
un-typed entities live in `unknown/`.

Profiles are **global**, not per project: the same entity is one profile in every
project the user works in. This world is separate from the project-scoped `.atl/wiki/`
and `.atl/journal/`; the two cross-reference through free markdown links only, never by
blurring their directories.

## Self vs third-party

Every profile carries `meta.is-self` (default `false`). The self profile — the user
themselves — is the one place the most sensitive fields may be recorded directly. On a
third-party profile (`is-self: false`), trait-level fields are automatically treated as
*the user's perception of that person* (`meta.perception-flag: true`), not as objective
fact, and the most sensitive tiers are gated further (below).

## The 4-tier privacy framework

Every field maps to a tier. Tier defaults are declared per-type in the interface
frontmatter (`tier-defaults`), with field-level overrides. The runtime behavior:

| Tier | Example fields | Behavior on write |
|---|---|---|
| **1 — Open** | `identity.*`, `anchors.*`, `relation-to-user.kind`/`role` | Always written. No check. |
| **2 — Perception-flagged** | `traits.*`, `relation-to-user.sentiment` | Written; on `is-self: false` the profile is `perception-flag: true` (recorded as the user's view). |
| **3 — Explicit signal required** | `state.emotional`, `state.goals`, `relationships.*.user-perceives` | Written **only** from a `user-confirmed` fact. An `agent-inferred` fact for a Tier-3 field is rejected. |
| **4 — Consent-gated** | `state.financial` | Written **only** if `meta.consent.<field>: true` (default `false`). Otherwise skipped. |

The rule of thumb: **when a tier gate is not satisfied, skip the field** — never lower
the gate to fit the fact.

**Two orthogonal gates — don't conflate them.** The tier gate governs the *fields of a fact
about a real entity*: fill what the evidence and tier allow, skip what they don't, never
reject a profile for being thin (the "fill to the extent possible — never validate"
discipline). The **reality gate** (`marker-drain.md` §5.0) governs a different axis — *whether
the payload is about a real entity at all*: a documentation example or a format placeholder
(`entity: ahmet` with only a stock trait, `serbest metin`, `entity/field/value`) is not a
thin real profile, it is **not a profile** — dropping it is a corollary of *"I do not invent
people"*, not a violation of *"never validate"*. Reality-gate only new-entity creation; an
existing profile is proof-of-realness and is never reality-gated.

## Source-flag discipline

Every field records where its value came from, so a wrong value can be corrected in the
next conversation instead of hardening. The three sources:

- `user-confirmed` — the user stated it (or confirmed an inference). The default source
  for facts drawn from the user's own messages.
- `agent-inferred-<date>` — the assistant inferred it from context. Tolerated for Tier 1
  and Tier 2; **rejected for Tier 3+**. A later `user-confirmed` value overwrites it.
- `lens-set` — a consuming team's lens wrote it directly (relevant once consumers exist).

Sources are stored in a parallel namespace alongside the fields (e.g.
`_sources.identity.name: user-confirmed`). The profile-level `meta.confidence`
(`high | medium | low`) summarizes how much of the profile rests on inference.

## Cross-team access

Access to this world is not open by default once third-party teams arrive. A consuming
team declares its profile access in its own `team.json` under `capabilities.profile`
(`reads` / `writes` field lists), surfaced to the user at install time. I honor those
declarations: a team reads and writes only what it declared. In v1 the only consumer is
personal-advisory-team; the contract exists now so it is in place before third-party
teams read freely.
