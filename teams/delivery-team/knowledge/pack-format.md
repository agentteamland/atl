# Knowledge-pack format — the delivery-team's stack-knowledge-as-data contract

This is the **single documented contract** every delivery-team role-agent, ceremony
skill, and `developer` worker follows for the **pack system**. It is the counterpart
to [`knowledge/azure-adapter.md`](azure-adapter.md): the adapter is the one contract
for *reaching Azure*; this is the one contract for *how a software team's stack
knowledge is packaged, bound to work, and loaded*. There is one pack format, one
binding rule, one read contract — written **once here** and inherited by every
caller. Do not improvise a pack shape that isn't described here.

Two callers read this doc: the **tech-lead** (owns area→pack binding at
decomposition) and the **`developer`** (loads its tagged area's pack before coding).

## 1. What a pack is, and why (the M1 seam)

A **pack** is a software team's stack knowledge shipped as **data**, not baked into a
prompt: a `packs/<area>/` directory of Markdown files describing how to build and test
one task-area in one stack. The generic `developer` agent (see
[`agents/developer/`](../agents/developer/agent.md)) carries **no** stack idioms — it
loads the tagged area's pack at run time and works *that* stack.

This is the **M1 knowledge-as-data seam**: one generic `developer` × N packs. Its
value is that a new stack-specific software team is **content, not code** — a team
authors `packs/<area>/` files and declares its areas; it writes no Go and forks no
agent. The `developer`'s role-craft (how to work a worktree, drive a pack, self-test,
open a PR) lives in its `children/` and travels unchanged; only the pack changes per
stack. Keeping stack knowledge *out* of the agent and *in* data is what lets one
delivery org serve web, mobile, and API work without a per-stack agent apiece.

## 2. Pack structure — `packs/<area>/`

A pack is a directory `packs/<area>/` containing a manifest plus topic files:

```
packs/<area>/
├── pack.md            ← the manifest (self-describing header) — required
└── <topic>.md ...     ← how-to / conventions / blueprints for this stack — one or more
```

- **`pack.md`** is the manifest and the **only** required file. The `developer` reads
  it first — it names the topics, the test commands, and the load-bearing conventions,
  so the worker knows what to load and how to verify before it writes a line.
- **topic `.md` files** carry the depth: component/widget/endpoint conventions, state
  and data patterns, dependency baselines, testing how-to. They are children-KB-style
  prose (a short intro + the WHY + worked patterns), but they are **pack content, not
  agent children** — so they need **no frontmatter** (`pack.md`'s `## Topics` list is
  the index, not an auto-rebuilt KB section).

### The `pack.md` manifest schema

```markdown
---
area: web
stack: "React + TypeScript + Vite"
---

# <Area> pack — <stack one-liner>

<one paragraph: what work this pack covers; when the tech-lead tags area:<this>.>

## Topics
- [<topic-file>.md](<topic-file>.md) — <one-line what-it-covers>
...

## Test commands
- unit / integration: `<command>`
- <web/mobile surface command as applicable>

## Key conventions
- <the 3–6 load-bearing rules a developer MUST honor for this stack>

## Dependency baseline
- <library ^version> — <why / role>
```

- **Frontmatter** — `area` (the tag suffix, matching the directory name) and `stack`
  (the concrete stack this pack fills its area with). These two make a pack
  self-describing: a reader knows what it binds to and what it teaches without opening
  a topic file.
- **`## Test commands`** — the exact commands the `developer`'s Level-1 self-test runs
  (see [§7](#7-the-developers-level-1-self-test-uses-the-pack)). They are in the
  manifest, not buried in a topic, because self-test is not optional and the worker
  must find the commands deterministically.
- **`## Key conventions`** — the few rules a developer **must** honor for this stack.
  Short list by design: a manifest that restates the whole topic set stops being a
  routing header.
- **`## Dependency baseline`** — the library versions a team **pins**, framed as a
  baseline the project may move, not gospel — because a stale hardcoded version rots,
  and the project wiki (not the pack) owns the project's actual lockfile truth.

## 3. Area → pack binding

Binding is a **tech-lead** responsibility, exercised at decomposition. The chain:

1. The `technical-analyst` only *suggests* areas (under `## Suggested Areas` in its
   sentinel comment — adapter §7). It does not decide them.
2. The **tech-lead** decides and applies each work-unit's area, writing
   `area:<name>` to `System.Tags` (adapter §7) — see the tech-lead's
   [`decomposition-blueprint.md`](../agents/tech-lead/children/decomposition-blueprint.md).
   The tech-lead owns this because the area tag *is* the pack binding.
3. The **`developer`** worker, spawned for that unit, loads **only** the tagged area's
   pack — `.claude/packs/<area>/` (see [§5](#5-how-packs-reflect)). One primary area
   per unit, so one pack; a unit that genuinely spans two areas is usually two units
   (a decomposition smell the tech-lead resolves, not the developer).

**Area vocabulary is project-shaped, and the tech-lead owns it.** Per the shipped
decomposition blueprint, areas are functional slices of *this* system (e.g.
`area:auth`, `area:reporting`), kept stable across the project on the tech-lead's
`Architecture/` page, so the same `packs/<area>/` binds consistently sprint over
sprint. The reference pack's `web`/`mobile`/`api` (see [§6](#6-the-v1-reference-pack))
are **concern-based, portable template names** — a real team keeps the area name it
needs and swaps the pack contents to its own stack; it does not have to adopt those
three. The binding rule is identical either way: **the tag names the directory, and
the developer loads exactly that directory.**

## 4. The three-layer read contract (load-bearing)

A pack is only one of three knowledge layers a worker reads. Keeping them separate is
what lets one pack serve every project on its stack while each project keeps its own
truth:

| Layer | What it is | Owner / source | Scope |
|---|---|---|---|
| **pack** (`packs/<area>/`) | generic **stack** craft — how to build/test *this stack*, anywhere | the software team (ships with it) | travels with the team, project-agnostic |
| **project wiki** (`Architecture/`, `Conventions/`) | **project-specific** current-truth — this project's shape + its conventions layered ATOP the pack's generics (adapter §8) | tech-lead | this project only |
| **canonical brief** (tech-lead) | the **bridge** — names the area (→ which pack) and embeds the exact wiki page paths for the unit | tech-lead, per work-unit | this work-unit only |

A **developer's context** = **pack (tagged area) + project-wiki (brief-named pages) +
task + brief**. The brief is what makes a fresh, isolated worker load the *right*
project knowledge: it names the area so the pack loads, and it embeds the
`Architecture/`/`Conventions/` page paths so the worker pulls them via
`wiki_get_page_content` (`search_wiki` for discovery when a path isn't pre-named —
adapter §8). The pack does **not** restate project specifics, and the project wiki
does **not** restate the stack's generic idioms — the split avoids duplication and its
inevitable drift. See the tech-lead's
[`canonical-brief.md`](../agents/tech-lead/children/canonical-brief.md) for how the
bridge is written.

## 5. How packs reflect

`packs` is one of `teampkg.AssetDirs` — the single source of truth for a team's
installable assets, alongside `agents`/`skills`/`rules`/`knowledge`/`scripts`. On
install / update / session-start, the reflection copies `teams/delivery-team/packs/`
into **`.claude/packs/<area>/`** in the target project, the same mechanism that
reflects agents and skills. So a `developer` worker — whose cwd is its own git
worktree, with the project's `.claude/` on its path — reads its pack from
`.claude/packs/<area>/` with no extra wiring. Adding a new area is adding a
`packs/<area>/` directory to the team; no code change is needed, because `AssetDirs`
already carries `packs`.

## 6. The v1 reference pack

For v1 the reference pack ships **inside delivery-team** (`teams/delivery-team/packs/`)
and serves two purposes at once: it is the **e2e fixture** (real pack content for the
dispatch loop to load) **and** the **template** a real software team copies to author
its own. This mirrors how `capabilities.profile` enforcement waited for its first
consumer — we ship a working reference now and grow the separate-team wiring when a
real consumer needs it.

The reference pack fills three portable, concern-based areas, each with one concrete
stack — chosen so the pack exercises all three test surfaces (see the tester's
`mobile-and-web-surfaces.md`):

| area (`packs/<name>/`) | reference stack | testing surface it exercises |
|---|---|---|
| `web` | React + TypeScript + Vite | web (preview / chrome-devtools MCP) + code (CI) |
| `mobile` | Flutter + Dart | mobile (booted emulator/simulator) + code (CI) — the design's emphasized, riskiest surface |
| `api` | Node.js + Express + TypeScript | code (CI) |

Each is `pack.md` + topic files, release-grade and real (not stubs) but minimal and
focused. The `mobile` pack carries the *knowledge* of how to boot and drive an
emulator for its `integration_test` runs (single-slot lease, preflight bootability,
block-never-silent-pass, screenshot evidence via `scripts/az-attach.sh`) — the emulator
runtime/lease wiring itself lives in [`testing-surfaces.md`](testing-surfaces.md) §3 + the
`scripts/emulator-lease.sh` / `emulator-preflight.sh` helpers; the pack describes how to *use*
that surface, driving those.

### How a real software team ships packs

When the first stack-specific software team is built, it ships packs the same way,
plus two declarations:

1. **Declare its areas in `team.json`** — an `areas: [{name, description}]` array,
   making the area vocabulary team-data (so the tech-lead's binding and the
   `developer`'s load agree on a declared set).
2. **Consume `capabilities.orchestration`** — the seam by which `atl work dispatch`
   drives a software team's role-agents + packs. This is **deferred** to the first real
   software team (the same pattern as `capabilities.profile`); delivery-team does not
   declare it in v1, because there is no separate consumer yet and building the
   enforcement ahead of a consumer would be speculative.

Until then the reference pack **is** the template: a new team copies `packs/<area>/`,
swaps the stack contents behind the concern-based area name, and adds its two
declarations. No new reflection subsystem is involved — `AssetDirs` already reflects
`packs`.

## 7. The `developer`'s Level-1 self-test uses the pack

The pack is not just build knowledge — it is the source of the `developer`'s
**Level-1 self-test** (the tester's `mobile-and-web-surfaces.md` establishes the two
levels; do not contradict it). At the `self-test` phase of the worker micro-loop, the
developer runs the pack's `## Test commands` on the surface(s) its area exercises,
attaches evidence to the Azure work-item via `scripts/az-attach.sh` (adapter §9), and
treats an un-run surface (an emulator that won't boot, a lease timeout) as
**unverified — never a silent pass**. Level-1 is fast and self-gating; the thorough
**Level-2** pass is a separate `tester` worker's job, and the `green` that authorizes
auto-merge is the ordered conjunction `green = test-gates ∧ review` — neither of which
the developer decides. The pack teaches *how* to self-test; it does not grant the
authority to declare done.

## 8. Worker phase vocabulary

A `developer` worker reports progress to its supervisor through the four
`status.json` fields (adapter §3 / the dispatch contract), one of which is `phase` —
the current micro-loop stage. The **canonical phase values** a developer writes, in
order, are:

```
claim → plan → implement → self-test → comment → pr
```

- `claim` — read the work-item + the `**[Technical Analysis]**` comment, transition
  the item to the runtime-resolved in-progress state (`wit_get_work_item_type`, never
  a literal — adapter §6), post a claim comment.
- `plan` — decide the approach against the loaded pack + the brief-named wiki pages.
- `implement` — write the change in the worktree.
- `self-test` — run the pack's `## Test commands` + attach evidence (§7).
- `comment` — post the progress/PR comment (a plain comment; the developer never
  writes the `**[Technical Analysis]**` sentinel or the wiki — adapter §7/§8).
- `pr` — open the PR and link it to the work-item
  (`wit_link_work_item_to_pull_request`). **This is where the worker's job ends.**

`review` and `merge` are **not** developer phases: the tech-lead reviews, and the
**deterministic engine** merges to `dev` and verifies the durable git state before the
Azure→Done transition (strict ordering — merge precedes the Done that triggers refill).
A worker that self-merged would violate both NEVER-merge and the engine's
durable-state verification (the dispatch worker contract; adapter §6 keeps the Done
transition runtime-resolved). The developer's contract ends at handoff
to review — that boundary is what keeps the merge gate deterministic and the loop
resumable.
