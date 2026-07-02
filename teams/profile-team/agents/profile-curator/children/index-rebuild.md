---
knowledge-base-summary: "How I rebuild ~/.atl/profiles/_index.md after a drain — the on-demand discovery file a lens reads to answer 'who exists in the user's world?'. Grouped by type; one line per entity (slug — name | salience | role). Derived wholesale from the profiles; loaded on demand, never into CLAUDE.md."
---

# Index Rebuild

`~/.atl/profiles/_index.md` is the discovery layer: the fast answer to "who and what
exists in the user's world?" A consuming team's lens loads it **on demand** to orient,
then direct-reads the specific `profile.md` it needs. It is **never** injected into
CLAUDE.md — hundreds of profiles would bloat every session's context; it is pulled only
when a lens is actually reasoning about the user's people.

I rebuild it once **after a drain batch** (not per item), so it reflects every change made
in the run.

## Format

```markdown
---
last-rebuilt: 2026-07-02
count: 13
---

## people (12)
- alex — Alex Doe | salience: high | role: friend
- mom — Jane Doe | salience: high | role: mother
- deniz — Deniz Yilmaz | salience: medium | role: manager

## unknown (1)
- the-cabin — The Cabin | salience: low
```

- **Grouped by `meta.type-id`** (v1: `people` + any `unknown` stubs). Each group header
  carries its count.
- **One line per entity:** `<slug> — <identity.name> | salience: <…> | role: <relation-to-user.role>`.
  Omit `role` when empty; a lens gets enough to decide whether to open the full profile.
- **`is-self`** — mark the self profile (e.g. a `| self` tag) so a lens can find "the user"
  quickly.

## Rebuild procedure

1. Glob `~/.atl/profiles/people/*/profile.md` (and any other type dirs once v2 adds them).
2. For each, read `meta.type-id`, `identity.name`, `relation-to-user.salience`,
   `relation-to-user.role`, `meta.is-self`.
3. Group by type, sort each group by salience (high → low) then name.
4. Write the file wholesale — it is **derived**, so I overwrite it entirely rather than
   patching; the profiles are the source of truth. Set `last-rebuilt` = today and `count`
   = the total.

## Discipline

- The index is derived — never hand-edit it, always regenerate from the profiles.
- It carries only the low-cost orientation fields (name, salience, role) — no sensitive
  state. A lens reads the index to find *who*, then opens the profile (subject to the
  tier + capability gates) to learn *what*.
