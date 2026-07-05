---
knowledge-base-summary: "How I run the cross-unit integration checkpoint at sprint-review: verifying that the units merged to dev over the sprint actually cohere as a whole (not just per-unit green), surfacing integration findings the per-unit gate can't see, filing forward-fix Tasks for what doesn't cohere (idempotently, per adapter §5), and PROMOTING the project facts workers surfaced up into my Architecture/ / Conventions/ / ADR wiki pages (adapter §8) — the mechanism that turns worker-surfaced facts into durable current-truth."
---

# Integration Checkpoint

Per-unit review ([review-craft.md](review-craft.md)) proves each unit is green **in isolation**.
It cannot prove the units **cohere as a whole** — a set of individually-correct changes can still
compose into something inconsistent, because each was built by a fresh isolated worker that never
saw the others (the isolation that keeps workers parallel is exactly what hides cross-unit
seams). Closing that gap is the **integration checkpoint** — my cross-unit pass at
`/sprint-review` (the ceremony is stone #6; I describe my contribution). It is the durable,
whole-sprint counterpart to the per-unit gate.

## What I verify — coherence, not correctness

The per-unit gate already established each unit is correct. Here I ask the questions only a
whole-sprint view can answer, over the units that merged to `dev` this sprint:

- **Do the seams line up?** Two units that share a surface (linked by a `Dependency`, or touching
  the same area) — does the consumer actually use the producer's contract as built, or did they
  drift because each worker guessed the seam independently?
- **Do the areas still compose?** A change in one area that a sibling area silently assumed away —
  the kind of break that passes both units' isolated reviews and only shows when they run together.
- **Did the sprint honor the architecture?** The units, taken together, still fit the
  `Architecture/` boundaries and the `Conventions/` in force — or did the aggregate quietly erode
  a boundary I own?
- **Is the Feature's intent actually delivered?** The units collectively satisfy the Feature's
  Acceptance Criteria (from the `System.Description` I read at decomposition), not just each
  Task's local goal.

I read the sprint's merged units with `wit_get_work_items_for_iteration` (batching, per adapter
§4 — **"list means all"**, never a silently-truncated read; if the set could exceed the tool's
return I close the gap with a high-`top` `wit_query_by_wiql` and treat a result *at* the cap as a
truncation error, not a complete read). I read their PRs/threads on the Azure-native surface.

## Surfacing integration findings and filing forward-fixes

An integration finding is, by definition, something no per-unit review could catch — so I don't
re-litigate the merged units; I record what doesn't cohere and route it forward:

- I **file a forward-fix Task** for each real integration break, as a new work-unit the next
  sprint's dispatch will pick up. I create it under the right parent and **idempotently** (adapter
  §5): compute `atl-key = hash(parent-id + plan-ordinal)` with a fresh ordinal in the parent's
  plan, run the check-first WIQL, found → reuse+update, not-found → create-then-stamp. A re-run of
  the checkpoint must not duplicate the forward-fix — the same idempotency discipline as
  decomposition ([decomposition-blueprint.md](decomposition-blueprint.md)).
- I **area-tag** each forward-fix (`area:<name>`, adapter §7) and add any `Dependency` links, so
  the scheduler orders it correctly — same rules as any unit I decompose.
- I apply the **evidence discipline** here too: an integration finding I file names the concrete
  seam (which two units, which surface, what breaks) — not a vague "these don't feel integrated."
  Same standard as the review evidence gate; a finding I can't point at isn't a finding.
- I resolve any work-item state at runtime (`wit_get_work_item_type`, adapter §6) — never a
  hardcoded literal.

## Promoting worker-surfaced project facts up to the wiki

This is the checkpoint's other half, and it is a rule the adapter states explicitly (adapter §8):
**developer/tester workers do NOT write the wiki.** During a sprint, workers *surface* real
project facts — "this boundary is leaky," "this area has a hidden dependency on that one," "the
real contract here differs from what the analysis assumed." Those facts arrive to me through the
sprint's work-items (progress comments, PR threads) and the tester's evidence. Two destinations,
and I route each correctly:

| The worker surfaced… | Goes to… | Who writes it |
|---|---|---|
| a durable **role-craft** learning ("how I test X well," a reusable technique) | that worker-agent's own `children/` via `/drain` | the worker (project-agnostic) |
| a **project fact** ("this system's boundary is actually here," "this area depends on that") | my `Architecture/` / `Conventions/` / an ADR page | **me**, at this checkpoint |

I promote the project facts by **upserting** the relevant page
([architecture-and-adr.md](architecture-and-adr.md),
[conventions-craft.md](conventions-craft.md)): update the `Architecture/` map to reflect the
boundary as it really is, add a `Conventions/` line if the sprint established a cross-area
agreement, and — if a fact revealed a decision that is significant AND hard-to-reverse — write an
**ADR** (or supersede one). This is the mechanism that turns transient, worker-surfaced facts into
**durable current-truth**: work-items are transient execution state; the wiki is the durable
record (adapter §8), and I am the one owner of these namespaces, so promotion goes through me and
there is no write race.

Why this matters: without the promotion step, hard-won knowledge (a boundary a worker discovered
was wrong) evaporates when the isolated worker exits, and the next sprint's workers rediscover it
from scratch. The checkpoint is where that knowledge is *captured into the project's memory* so
the next canonical brief ([canonical-brief.md](canonical-brief.md)) points workers at the *now-correct*
page.

## Checklist

- [ ] Read all units merged to `dev` this sprint (`wit_get_work_items_for_iteration`, batched);
      "list means all" — no silent truncation; cap-at-`top` treated as a truncation error.
- [ ] Verified seams between dependent/same-area units line up as built.
- [ ] Verified the aggregate still fits `Architecture/` boundaries + `Conventions/`, and the
      Feature's Acceptance Criteria are collectively delivered.
- [ ] Each integration finding names the concrete seam (two units + surface + break) — evidenced,
      not vague.
- [ ] Forward-fix Tasks filed **idempotently** (adapter §5: atl-key by parent+fresh-ordinal,
      check-first WIQL), area-tagged, dependency-linked; state resolved at runtime.
- [ ] Worker-surfaced **project facts** promoted to `Architecture/` / `Conventions/` / an ADR by
      me (upsert); worker **role-craft** learnings left to their own `children/` via `/drain`.
- [ ] Pages left as current-truth (updated in place, stale lines removed); ADR added/superseded,
      never edited in place.
