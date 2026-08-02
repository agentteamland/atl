---
name: work-list
description: /work-list [mine|active|sprint|blocked|backlog|<type>] — read-only view of the board for a human working through a queue. Filters server-side where the backend supports it, excludes candidates from the ready frontier, and renders a compact grouped table rather than raw JSON. Writes nothing, so it is safe to run at any time.
---

# /work-list — what is on the board, readably

The browse step of the drive loop. It answers "what is mine / what is in flight / what is in this
sprint" without opening a board in a browser, and it writes nothing at all — which makes it the one
skill in the loop with no preconditions and no way to do damage.

Reads its contracts from [`config-and-methodology.md`](../../knowledge/config-and-methodology.md),
[the backend interface](../../knowledge/backend-interface.md), and the active
`backends/<backend>/adapter.md`. **Never invent a tool name.**

## Backend support

**GitHub is bound; Azure halts.** The Azure query substrate is bound (WIQL) but its state literals
must be resolved at run time and the operation map is self-disclaimed, so the filters this skill
offers cannot be expressed there without guessing. A read-only skill halting is cheap; a read-only
skill quietly returning the wrong set is not.

## Filters

Tokens combine, space-separated, case-insensitive. No argument ⇒ **mine**, open only.

| Token | Means |
|---|---|
| `mine` | assigned to you |
| `active` | Status is In Progress |
| `sprint` | in the current sprint — the `sprint:<n>` label under `flow`, the Iteration field under `scrum` |
| `blocked` | carries the `blocked` label |
| `backlog` | open, in no sprint |
| a type name | that type only |

## Procedure

### 1. Resolve the mode before the sprint filter

`sprint` means different things under different methodologies, and the mode is **read from
`.delivery/methodology.json`, never inferred from the board**. Under `flow` the carrier is a label
and the filter is server-side; under `scrum` it is a project field, and Projects v2 has **no
server-side field filter** — so that path lists the project's items and filters client-side. Say
which path ran when the result is large enough that the difference matters.

Under `flow`, compare sprint ordinals **as integers**. `sprint:10` outranks `sprint:9`, and a
lexical "highest" quietly returns a stale sprint.

### 2. Query — server-side where possible, and page to exhaustion

Prefer the search surface, which filters by label, state, type and assignee on the server. Two
traps the adapter names and that are easy to hit backwards:

- **`--state all` is rejected by the issue search** — omit the flag to get both open and closed.
  The plain issue list *does* accept it. Getting this backwards produces an empty result that looks
  like an empty board.
- **A list means all.** Page until the cursor is exhausted; a first page silently presented as the
  whole answer is how a count comes out wrong by a factor nobody notices.

### 3. Exclude candidates from the ready frontier

An item flagged `candidate` is a request that has not been accepted yet. It is deliberately outside
the ready-to-pull set, so it must not appear in `mine`, `active`, `sprint` or `backlog`.

Show candidates only when asked for explicitly, and label them as such — a candidate rendered like
ordinary backlog invites someone to start work the PO has not agreed to.

### 4. Render for a human, not for a parser

Group by type, newest-relevant first, one line per item:

```
#<id>  [<status>]  <title>   <area>  <sprint>  <assignee if not you>
```

Never dump raw JSON. Keep each group under roughly twenty rows; past that, truncate with an
explicit `(+N more)` and a hint at the tighter filter, so a truncated view can never be mistaken
for a complete one.

Close with a count line — total, how many are yours, how many are in flight — because that is the
number the driver actually acts on.

### 5. Say when the answer is empty and why

An empty result has more than one cause and they call for opposite reactions: nothing matched, or
the filter combination cannot match anything (`blocked backlog` on a board where blocked items are
always in a sprint), or a query failed. Distinguish them. "No items" presented for a failed query
is a wrong answer delivered confidently.

## Idempotent re-run

Total — it is a pure read. Nothing is written, cached, or stamped, so there is no re-run behaviour
to reason about and no reason to hesitate before running it again.
