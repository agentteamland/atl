---
name: work-new
description: /work-new "<title>" — open one work item by hand, in the right place in the hierarchy. Infers the type, finds or proposes a parent, elicits only what the item genuinely lacks, writes the fixed-heading spec field, and stamps a manual-provenance key so a re-run converges instead of duplicating. The light way to get a card onto the board; `/request` stays the heavy PO-gated intake for a request that needs a feasibility verdict. Mutating and interactive; run it explicitly.
disable-model-invocation: true
---

# /work-new — one card, in the right place

The everyday way to put work on the board. Something surfaced, it needs to exist as an item, and
it does not need a triage ceremony to get there.

**This is not `/request`.** That skill carries an unplanned ask through a business-analyst →
technical-analyst → tech-lead assessment and an honest PO gate, because a request needs a *verdict*
before it earns a place. `/work-new` is for work whose place is not in question — a bug you just
found, a task the sprint's work implies, a chore. If you find yourself wanting a feasibility
opinion, you wanted `/request`.

Reads its contracts from [`config-and-methodology.md`](../../knowledge/config-and-methodology.md),
[the backend interface](../../knowledge/backend-interface.md), and the active
`backends/<backend>/adapter.md`. **Never invent a tool name.**

## Backend support

**GitHub is bound; Azure halts** — same reason as the rest of the drive loop.

## Procedure

### 1. Infer the type, and confirm rather than assume

The hierarchy is `Epic → Feature → PBI → Task | Bug`, and the concrete type names are resolved at
run time from the methodology's artifact hierarchy — never hardcoded, because one project's `PBI`
is another's `User Story`.

Infer from how the title reads: implementation-unit phrasing (a conventional-commit style subject,
"fix", "add", "wire") is a **Task**; a named regression with a symptom is a **Bug**; outcome
phrasing that describes a capability rather than a change is a **PBI** or above.

When it is ambiguous, **default to the smallest unit and confirm**. The failure that costs is the
other direction: an implementation task filed as a PBI acquires children it should not have and has
to be unwound by hand later.

### 2. Place it under a parent — never silently

`Bug` and `Task` sit under a **PBI**; a PBI sits under a Feature; a Feature under an Epic. They do
not attach directly to a Feature.

Search for a plausible parent by the title's primary domain noun, offer the best few, and always
offer "none of these — open a new one". **Never silently adopt the first match**: a wrong parent is
invisible on the item itself and only surfaces later as a sprint report that does not add up.

When a level genuinely does not exist yet, create it in the same pass, top-down, so the chain is
whole before the leaf is written.

On this backend a child is **two calls**: create the issue, then attach it via the sub-issues
endpoint using the child's REST id — not its issue number, and there is no `--parent` flag on
`gh issue create` to shortcut it.

### 3. Elicit only what is missing

Write the spec field with the fixed headings — `## Problem`, `## Business Value`, `## Scope`,
`## Acceptance Criteria`, `## Out of Scope` — and ask, one question at a time, only for the ones
you cannot honestly fill from what you were given.

**Do not invent an acceptance criterion.** An item whose acceptance criteria were guessed reads
exactly like one whose criteria were stated, and the difference only surfaces when the work is
reviewed against them. If you cannot get a verifiable criterion, say the item is not ready and
leave the heading empty rather than filling it plausibly.

A bug needs its reproduction. There is **no native repro field bound** on this backend, so it goes
under the spec field's own heading (`## Repro`) rather than reaching for a provider field the
adapter has not verified. Say so if you add it, and register the heading in the adapter so the next
reader finds a contract instead of a precedent.

### 4. Stamp provenance — a manual key, not a plan key

Do **not** stamp `atl-key`. That key is `hash(parent-id + plan-ordinal)`, and its convergence
property rests entirely on the ordinal being assigned by a durable decomposition plan and never
reused. A hand-created item has no plan and no ordinal, so any `atl-key` it carried would be a
fabricated one that a later `/refine` could collide with.

Stamp `atl-manual:<slug>` instead — the sibling of the `atl-brainstorm:<slug>` and
`atl-request:<slug>:<initiator>` keys that already exist for exactly this reason: items that are
real but have no plan position.

**Check that key before creating.** Found ⇒ update the existing item toward the intended state;
not found ⇒ create, then stamp. A create that collides with an existing item is *resolved to it*,
not surfaced as an error.

### 5. Do not assign a sprint, and do not assign a person

A new item belongs in the backlog. Admitting it into a sprint is a planning decision that
`/sprint-plan` makes — or that `/work-start` makes explicitly, by asking, at the moment someone
actually picks it up. Stamping a carrier here quietly grows the sprint after it was planned.

Leave it unassigned for the same reason: whoever starts it claims it.

### 6. Report

```
#<id> — <title>   [<type>]
  parent:  #<pid> <parent title>[ (new)]
  chain:   <every level created in this pass>
  key:     atl-manual:<slug>
  sprint:  backlog (not admitted)

Next: /work-start <id> when you pick it up.
```

## Idempotent re-run

Convergence is by the `atl-manual:<slug>` key, checked before every create: a second run finds the
item and updates it rather than opening a duplicate.

Note the honest bound — that only holds for the item this skill *stamped*. If a re-run derives a
different slug from a reworded title, the key changes and the check misses. Derive the slug from
stable words in the title, and when in doubt search for a near-duplicate before creating: the
board's readability is worth one extra query.
