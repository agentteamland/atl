---
name: work-start
description: /work-start <id> — claim one work item and build it, in a session the human is steering. Resolves the item over the active backend, refuses if the autonomous engine already owns it, resolves the sprint carrier, cuts the engine's own `delivery/<slug>/<id>` branch off a freshly fetched integration branch, moves the item to In Progress, briefs from the item's own spec field, its Canonical Brief and the nearest ancestor's Technical Analysis, loads the area's stack pack, then proceeds with implementation by default — `--prep-only` stops after the briefing. Mutating and interactive — run it explicitly; it is not something to invoke in passing. Hand-driven half of the drive loop; `/work-finish` closes the unit.
---

# /work-start — claim a unit and build it

The drive loop's entry point. Where `atl work dispatch` spawns N isolated workers with nobody
watching, this takes **one** unit in the session the human is steering: same board, same card, same
branch grammar, same PR contract. Only the supervision differs.

**"Hand-driven" names who is steering, not who is typing.** The human is present, one card is in
flight, and every step is visible and interruptible — that is the whole of it. Stopping to ask
"shall I begin?" after a briefing the human just requested buys none of that and costs a round trip
on every single card, so implementation proceeds by default. `--prep-only` is there for the times
you want to read the brief and decide.

That symmetry is the design, not a coincidence. The branch this cuts is the engine's own
`delivery/<slug>/<id>`, so a hand-driven unit is indistinguishable from an engine-driven one
downstream — `MergedToBase`'s merge verification, the worktree reconcile, and the unmerged-branch
quarantine all still apply to it.

Everything about *how this project works* is data:
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md) for the `.delivery/`
config + methodology descriptor, [the backend interface](../../knowledge/backend-interface.md) for
the provider-neutral concepts, and the active backend's `backends/<backend>/adapter.md` for the one
contract every operation follows. **Never invent a tool name** — if an operation is not in the
adapter, it is not bound, and the honest move is to say so rather than to guess a call.

## When to run

- **Any time you are about to work a card by hand.** It is not tied to a ceremony and not part of
  the dispatch loop.
- **After** the unit exists on the board and has been refined enough to work from — `/kickoff`,
  `/refine`, or `/work-new` put it there.
- **Not** for a unit the engine is driving. Step 2 refuses that case; see the plan guard.

## Arguments

`/work-start <id> [--prep-only]`

| | |
|---|---|
| `<id>` | the work item to claim. Required — this skill never picks for you. |
| `--prep-only` | claim, cut the branch, brief — then **stop**. Also accepted: `--no-implement`, or the same intent stated in prose ("just prep it", "brief me first, don't implement yet"), **in whatever language the user is writing in** — match the intent, not an English phrase. |

The flag is an **opt-out**, so an invocation that says nothing about implementation gets it. Read
the intent from the invocation as a whole rather than matching the literal flag: a human who asks
for the briefing is asking for the briefing.

## Backend support

**GitHub is bound. Azure is not, and this halts there rather than guessing.**

The Azure adapter's operation map carries its own disclaimer — every row except the two most
recently verified "still name the **pre-consolidation** surface and have **not** been re-verified"
against the live MCP. Writing a mutating skill against tool names the adapter itself calls
unverified is precisely what the never-invent-a-tool-name rule exists to prevent, so this skill
does not do it.

On a project whose `config.backend` is not `github`, stop with:

```
/work-start: backend "<backend>" is unbound for the drive loop.
The Azure operation map is self-disclaimed as unverified against the live MCP,
so this skill will not issue writes against it. Drive this unit by hand, or
re-verify the Azure tool names and bind them in backends/azure/adapter.md first.
```

That mirrors `atl work promote`, which already returns a `backend-unbound` hold on Azure rather
than re-deriving a path it cannot verify.

## Procedure

### 1. Resolve the project's data — never assume any of it

Read, in this order, and halt on anything missing rather than defaulting:

| Source | What you need | On absence |
|---|---|---|
| `.delivery/config.json` | `backend`, `branchPair.dev` | no file ⇒ this project is not connected; tell the user to run `/delivery-init` |
| `.delivery/methodology.json` | `mode` (`scrum` \| `flow`) | key absent ⇒ `scrum`; **unrecognized ⇒ halt** — never infer the mode from the board |

`config.branchPair.dev` is **authoritative at run time** — read it, never `methodology.branches.dev`
and never a hardcoded `"dev"`. A project that renamed its pair must not be silently driven against
a branch that does not exist.

### 2. The plan guard — refuse a unit the engine owns

This is the boundary between the two drive modes, and it is the reason both can run over one
project at once.

Read `.delivery/plan.json` if it exists. If its `units[]` contains this `id`, the autonomous engine
has been given this unit: `atl work dispatch` admits from the plan and nothing else, so starting it
by hand means a worker may be spawned onto the same card and the same branch.

**Refuse by default**, and say exactly why:

```
/work-start: #<id> is in the active plan (.delivery/plan.json, sprint <slug>).
The dispatch engine admits from that plan, so driving this by hand risks a
worker on the same branch. Remove it from the plan, or confirm you want to
take it over.
```

Take it over only on an explicit confirmation from the human. No plan file, or the id absent from
it, means the unit is yours — proceed silently.

### 3. Read the unit, and refuse a finished one

Read the item by id over the adapter. Pull its spec field (Azure `System.Description`, GitHub the
issue body) and its comments.

The spec field's structure is fixed and identical on both backends — `## Problem`,
`## Business Value`, `## Scope`, `## Acceptance Criteria`, `## Out of Scope`. **Read back by
heading, never by position**; a section may be absent and the order is not a contract.

Refuse if the item is already closed or removed. Reopening is a decision, not a side effect of
starting work — route it through `/work-move` so the transition is explicit and leaves a record.

### 4. Resolve the sprint carrier — and admit the unit if it has none

The branch name needs a sprint slug, so the unit must belong to a sprint.

| `mode` | Carrier | Slug |
|---|---|---|
| `flow` | the `sprint:<n>` label | `<n>` |
| `scrum` | the Iteration field | derived from the resolved iteration name |

Under `flow`, compare ordinals **as integers** — `sprint:10` outranks `sprint:9`, and a lexical
"highest" hands back a stale ordinal. A unit carries **at most one** `sprint:` label; if you find
more than one, stop and surface it rather than picking, because a label accumulates where a field
replaces and two carriers mean "which sprint is this in?" has no answer.

**No carrier at all** means the unit is in the backlog, not in a sprint. That is the scope-creep
moment, so make it explicit: ask whether to pull it into the current sprint, and only on a yes
stamp the carrier (under `flow`, a single write that adds the current label — there is nothing to
remove). On a no, stop: there is no slug to build a branch from.

### 5. Derive the branch — the engine's own grammar, verbatim

```
delivery/<sprint-slug>/<id>
```

This is `dispatch.BranchName(slug, id)` in `cli/internal/dispatch/worktree.go`, and matching it is
load-bearing: it is what makes a hand-driven unit indistinguishable from an engine-driven one, so
the merge verification, the worktree reconcile, and the quarantine of unmerged branches keep
working on it. Do not improvise a friendlier name.

### 6. Git preflight, then cut the branch

In order, and stop on the first failure rather than working around it:

1. `git status --porcelain` — **non-empty ⇒ refuse.** Uncommitted changes might be someone's lost
   work, including a parallel session's. Surface them; do not stash on the user's behalf.
2. `git fetch origin <branchPair.dev>` — the base must be fresh, or the unit starts behind.
3. Branch already exists locally? **Ask** — resume it, or take a new name. Never silently reuse a
   branch: it may be an abandoned attempt, or another session's in-flight work.
4. `git switch -c <branch> origin/<branchPair.dev>`.

### 7. Claim the unit on the board

On GitHub, claiming is the Projects v2 **Status** field set to **In Progress** — the built-in
project workflows automate only the Done end (`item closed → Done`, `PR merged → Done`), so nothing
sets In Progress for you. Set it explicitly or the board shows no work in flight while the work is
in flight.

Self-assign as well (`gh issue edit <n> --add-assignee @me`) so the board says *who*. Note honestly
that this is one step beyond what the adapter documents today — the adapter binds claiming to the
Status field and calls self-assignment optional without giving a shape. If you add it, add it to
the adapter too, so the next reader finds a bound operation rather than a precedent.

### 8. Leave a claim comment

A plain, unsentineled comment — the same thing a developer worker leaves:

```
Branch `delivery/<slug>/<id>` cut off <dev>. Driving this by hand.
```

Deliberately **not** a sentinel. Sentinels (`**[Technical Analysis]**`, `**[Canonical Brief]**`,
`**[Request Decision]**`, `**[Promotion Approval]**`) are a read-back channel with a fixed
first-line contract and a named consumer. A human progress note has no consumer, so giving it a
sentinel would add a channel nothing reads and one more thing to keep in sync.

### 9. Brief the driver

Print a compact block, sourced from the board rather than paraphrased:

```
#<id>  <title>          <state> → In Progress
Branch:  delivery/<slug>/<id>   (off <dev>)
Sprint:  <carrier>
Area:    <area:… label, or "unset">

<the spec field, rendered by heading>

Brief:   <the **[Canonical Brief]** comment, if present>
TA:      <the **[Technical Analysis]** comment — see the fallback below>
```

Two read rules that are easy to get wrong:

- **Match the sentinel, not the newest comment.** Find the comment whose *first line* is the
  sentinel exactly; a later human comment must never shadow it.
- **Climb for the Technical Analysis.** A decomposed PBI or Task usually carries a
  `**[Canonical Brief]**` but **no** `**[Technical Analysis]**` of its own — that lives on an
  ancestor Feature. Walk parent links up to the nearest ancestor bearing one. On GitHub the parent
  is `Issue.parent` via GraphQL; the REST `sub_issues` endpoint lists an issue's *children*, so
  reaching for it here returns the wrong direction.

If the Canonical Brief names pages to load, load them before starting — that list is the unit's
context, and skipping it is how a unit gets built against the wrong conventions.

### 10. Report, then build

State what changed and what did not:

```
Started #<id> — <title>
  branch:  delivery/<slug>/<id> (off <dev>)
  board:   Status → In Progress[, assigned to <you>]
  sprint:  <carrier>[ — pulled into this sprint]
  comment: claim posted
```

The report is **retrospective** — what changed and what did not. What happens next is step 11's to
decide and to announce, because `--prep-only` and the stop-anyway exceptions all run after this
point, and a report promising a build that never starts is worse than one that says nothing.

### 11. Load the area's pack, then implement

**Proceed with implementation by default — do not stop and wait for a "go ahead."** The human
asked for this card by id; a pause here re-asks a question they already answered. Print
`Next: building it now. /work-finish when it is green.`, then:

1. **Load the area's stack pack** — `.claude/packs/<area>/`, keyed off the unit's `area:<name>`
   label. Its `production-unit.md` carries the blueprint, the test commands, and the registration
   step a scaffolded unit needs to exist at runtime. A unit built without its pack is built against
   the generic conventions instead of this stack's.

   **Check the pack matches the project's stack before you obey it.** All reference packs reflect
   into every project regardless of what it is written in, so `.claude/packs/api/` existing is not
   evidence that it describes *this* API. Read its opening lines: if it names a stack the repo does
   not use, the pack is a template that arrived by reflection and its commands, blueprint and
   registration step are all wrong here. Say so, work from the project's own `docs/conventions/`
   and `docs/architecture/` instead, and note the gap — a project whose stack has no pack is a real
   finding, not a thing to route around silently.
2. **Load whatever the Canonical Brief named** (step 9) — that list is the unit's bounded context.
3. **Plan, implement, and test**, against the acceptance criteria printed in the briefing. The test
   gate is not optional and it is not `/work-finish`'s job to write it for you:
   [`testing-surfaces.md`](../../knowledge/testing-surfaces.md) §7 — diff coverage ≥ 90% of the
   lines this unit adds or modifies, **and** at least one test that goes red when the change is
   reverted.
4. **Commit on the branch** once it is green. The branch is this loop's only state, and a clean
   tree is what both exits need: `/work-finish` refuses a dirty tree, and a resumed `/work-start`
   reads the working tree as this card's own progress. Leaving the work uncommitted is what makes
   a resume ambiguous.
5. **Hand control back**, so the human can review and invoke `/work-finish`.

**Area unset ⇒ ask which pack, do not guess.** Picking a pack by reading the title is how a unit
gets built against the wrong stack's conventions, and the mistake is invisible until review.

**Stop after the briefing instead — the exceptions:**

- `--prep-only` / `--no-implement`, or the same intent in the user's own words.
- The briefing exposed a card that **cannot be built as written** — no acceptance criterion, a
  scope step that contradicts the code, a dependency that has not landed. Say which, and stop.
  This is the case the briefing exists to catch; carrying on regardless is how a wrong card becomes
  a wrong PR.

Each of those prints its own reason in place of the `Next:` line above, so the report never
promises a build that is not happening.

## Idempotent re-run

Re-running on a unit you already started is a **resume**, and it must be safe:

- Branch exists **and is checked out** ⇒ re-print the briefing, then **carry on building**. Skip
  steps 4, 6, 7 and 8: the carrier write, the state change and the claim comment would change
  nothing and the comment would be noise, and **step 6 has nothing left to preflight** — its
  dirty-tree refusal guards a branch cut that is not happening, and uncommitted changes on the
  branch you are resuming are this card's own work in progress, not a parallel session's. Then
  proceed to step 11 exactly as a fresh start does — a resume should be safe *and* productive.
- Branch exists but is **not** checked out ⇒ this is **not** a resume. It is a stale branch from an
  earlier attempt while HEAD sits elsewhere, so steps 4–8 run normally and step 6 asks
  resume-or-rename as usual. Skipping ahead here would hand the driver a working tree on the wrong
  branch and get the work committed somewhere it does not belong.
- Already In Progress ⇒ leave the state alone; a no-op write still dirties the item's revision and
  makes the board's history lie about when work started.

Nothing about this skill accumulates: the carrier is one label, the branch name is a pure function
of `(slug, id)`, and the claim comment is written once.
