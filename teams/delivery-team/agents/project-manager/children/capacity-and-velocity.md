---
knowledge-base-summary: "The capacityModel as data: velocity = mean StoryPoints over the last velocityWindowN (=3) closed sprints; the cold-start po-seed + seed-decay blend for the first N sprints; the availabilityFactor 0-1 dial for short-staffed sprints. Velocity is read-only, idempotent, client-side arithmetic over wit_* Done queries (resolve the Completed state-category at runtime). Reading work_get_team_capacity as a secondary signal."
---

# Capacity & Velocity

Capacity is the ceiling my sprint-planning blueprint admits against. It is a *number I compute*,
not a policy I hold — every parameter comes from `capacityModel` in `.delivery/methodology.json`
(read as data, see [methodology-as-data.md](methodology-as-data.md)), and every input comes from a
read-only query against the live project. This is craft that travels to any project: the *formula*
lives here, the *numbers* live in Azure and the descriptor.

## The capacityModel — the parameters I read (never hardcode)

```json
"capacityModel": {
  "velocityWindowN": 3,
  "unit": "storyPoints",
  "coldStart": "po-seed",
  "seedVelocity": null,
  "availabilityFactorDefault": 1.0
}
```

| Field | What I do with it |
|---|---|
| `velocityWindowN` | how many closed sprints the velocity mean averages over (default 3). |
| `unit` | the estimation unit (`storyPoints`) — the field I sum is `StoryPoints`. |
| `coldStart` | the strategy when fewer than `velocityWindowN` closed sprints exist (`po-seed`). |
| `seedVelocity` | the PO-set starting velocity, set at `/kickoff`; `null` until seeded. |
| `availabilityFactorDefault` | the default 0–1 availability dial (1.0 = fully staffed). |

I read these values; I never bake `3` or `1.0` into my reasoning. A project that ships a
descriptor with `velocityWindowN: 5` gets a 5-sprint mean with zero change to my craft.

## Velocity — read-only, client-side, idempotent

**Velocity = the mean of completed StoryPoints over the last `velocityWindowN` closed sprints.**
It is pure arithmetic over Done-item queries — no write, no state, no analytics API.

> **WHY client-side arithmetic and not an analytics call.** The adapter has exactly one REST
> carve-out — attachment upload (adapter §9) — and it is deliberately *not* an analytics hatch:
> Resolution #3 removed the analytics transport gap precisely because velocity is expressible as
> `wit_*` queries. Computing it client-side keeps the team on the MCP-first contract and keeps
> velocity **idempotent by nature** (adapter §5): re-running the query sums the same Done items to
> the same number, so a re-plan never corrupts the ceiling.

### How I compute it

For each of the last `velocityWindowN` closed sprints:

1. Read that sprint's completed items. Prefer `wit_get_work_items_for_iteration` for the sprint's
   IterationPath, or a high-`top` `wit_query_by_wiql` filtered to that IterationPath **and** the
   Completed state-category — resolved at runtime via `wit_get_work_item_type`, **never** the
   literal `"Done"` (adapter §6). Different process templates spell completion differently (Scrum
   `Done` vs Agile `Closed`); resolving keeps the math correct on any template.
2. Sum `Microsoft.VSTS.Scheduling.StoryPoints` over those completed items. Only items that
   actually reached the Completed category count — a carried-over or rejected item contributes
   zero to the sprint it *didn't* complete in (see [reject-and-carryover.md](reject-and-carryover.md)).
3. Average the per-sprint sums: `velocity = mean(sprint_points[])`.

**"List means all" is load-bearing here** (adapter §4): a half-read Done set silently *understates*
velocity and would shrink every future sprint. So I read to exhaustion, and I treat a result AT
the WIQL cap as a truncation error to surface — never as a complete read.

### Worked example (generic)

`velocityWindowN = 3`. Last three closed sprints completed `22`, `18`, `26` story points.
`velocity = (22 + 18 + 26) / 3 = 22`. With `availabilityFactor = 1.0`, the sprint-planning ceiling
is `22` points.

## Cold start — the po-seed + seed-decay blend (#6)

Before `velocityWindowN` closed sprints exist, there is no honest empirical mean. The
`coldStart: "po-seed"` strategy blends the PO's seed with the accumulating real data so the
ceiling isn't a blind guess *and* isn't frozen at the guess:

- **Sprint 1** — zero closed sprints. Use `seedVelocity` outright (the PO's estimate from
  `/kickoff`). If `seedVelocity` is `null`, I cannot compute a defensible ceiling: I surface that
  the cold-start seed is missing and ask the PO to set it, rather than inventing a number.
- **Sprints 2 … N-1 (fewer than N closed)** — **blend** the seed with the real closed-sprint
  data, decaying the seed's weight as real sprints accumulate. The seed fills the not-yet-existing
  sprints of the N-window: over a fixed N-denominator, each of the `N − count_closed` missing sprints
  contributes the seed and each closed sprint contributes its actual points —

  ```
  blended = (Σ closed_sprint_points + seed × (N − count_closed_sprints)) / N
  ```

  So the seed's weight is `(N − count_closed) / N`: full at cold-start, `2/3` after one closed sprint
  (N=3), `1/3` after two, and **exactly zero** once `count_closed_sprints == velocityWindowN` — the
  blend retires itself at N with no special-case switch.
- **Sprint N onward** — `count_closed_sprints ≥ velocityWindowN`: the plain N-sprint mean takes
  over; the seed is gone (the formula above already yields it — `seed × 0`). Empirical data has fully
  replaced the guess.

> **WHY blend + decay rather than "seed until N, then switch."** A hard switch makes the ceiling
> lurch at sprint N (the seed's error snaps out all at once). Decaying the seed's weight lets each
> real sprint correct the estimate gradually, so the ceiling converges smoothly onto the team's
> actual throughput instead of jumping.

## The availabilityFactor dial

The **availabilityFactor** (0–1) scales the computed velocity for a sprint the team is
short-staffed for (holidays, a member on leave). The final ceiling is:

```
capacity = velocity × availabilityFactor
```

- `availabilityFactorDefault` (1.0) applies unless the PO signals reduced availability for a
  specific sprint. A half-staffed sprint → `0.5` → half the point ceiling.
- The factor is a **planning dial the PO owns**, not something I infer. I apply the value I'm
  given; I don't guess who's on leave.

> **WHY a multiplicative dial rather than re-estimating velocity.** Velocity is the team's *proven*
> throughput at full staffing — a durable, earned number. Availability is a *this-sprint*
> condition. Keeping them separate means a low-availability sprint doesn't poison the velocity
> history: next sprint's mean still reflects true capability, and only the factor changes.

## `work_get_team_capacity` — the secondary signal

Azure's own capacity model (`work_get_team_capacity`) records
per-member daily capacity and days-off for an iteration. I read it as a **secondary, corroborating
signal**, not as my primary ceiling:

- If the PO (or a `/delivery-init` setup) has populated team capacity in Azure, it can inform the
  `availabilityFactor` — e.g. a member with days-off logged is a real availability reduction I can
  reflect rather than ask about.
- But my ceiling stays **velocity-derived** (empirical throughput), because Azure's capacity is an
  *hours* model and this team estimates in *story points*; the two don't convert cleanly. Azure
  capacity refines the availability dial; it does not replace the velocity mean.
- `work_update_team_capacity` is a write; I use it sparingly and idempotently if a ceremony needs
  to record the sprint's availability back to Azure — the same durable-milestone discipline as any
  other write (adapter §3). Reading is the default; writing team capacity is not part of routine
  planning.

## What velocity is NOT

- Not a per-developer metric. This is a *team* throughput number; the dynamic `developer`
  instances are fungible workers, not tracked individuals.
- Not a commitment or a target imposed on the team — it is a *forecast* ceiling for how much to
  admit. Its only consumer is my cap-admission step
  ([sprint-planning-blueprint.md](sprint-planning-blueprint.md) §4).
- Not affected by items that were *rejected* by the PO or *carried over* incomplete — only items
  that reached the Completed category in a given sprint count toward that sprint's points.
