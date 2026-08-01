# Config & methodology — the delivery-team's per-project settings contract

Two files under a project's **committed** `.delivery/` directory hold everything a ceremony
needs to know about *this* project. They are written once by [`/delivery-init`](../skills/delivery-init/SKILL.md)
(and refined on re-run), and read by every ceremony (`/kickoff`, `/refine`, `/sprint-plan`,
`/sprint-start`, `/sprint-review`) and every role-agent. This is the **config-seam**: settings
are *data* a ceremony reads, never logic baked into an agent.

| File | Purpose | Written by |
|---|---|---|
| `.delivery/config.json` | connection **identity** — where the work lives + how to reach it | `/delivery-init` |
| `.delivery/methodology.json` | the **methodology descriptor** — how the team works | `/delivery-init` (v1: one Scrum instance, in `scrum` or `flow` **mode** — §1.1) |

Both are committed (like `.atl/`): the connection identity and the methodology are project
facts the whole team shares, not per-machine state. Neither file ever contains a secret.

## 1. `methodology.json` — the descriptor every ceremony reads

A **flat, ceremony-read descriptor**. v1 ships exactly one instance — Scrum, run in one of two
**modes** (`scrum` or `flow`, §1.1). The descriptor holds *intent*; the active backend holds
concrete *names* (resolved at runtime, §3).

```json
{
  "id": "scrum",
  "displayName": "Scrum",
  "mode": "scrum",
  "roles": [
    { "name": "intake",            "binding": "agent", "dispatch": "in-session" },
    { "name": "business-analyst",  "binding": "agent", "dispatch": "subagent" },
    { "name": "technical-analyst", "binding": "agent", "dispatch": "subagent" },
    { "name": "project-manager",   "binding": "agent", "dispatch": "subagent" },
    { "name": "tech-lead",         "binding": "agent", "dispatch": "subagent" },
    { "name": "tester",            "binding": "agent", "dispatch": "worker" },
    { "name": "developer",         "binding": "agent", "dispatch": "worker", "instances": "dynamic" },
    { "name": "product-owner",     "binding": "human" }
  ],
  "artifactHierarchy": ["Epic", "Feature", "Pbi", "Task"],
  "workItemTypeMap": { "Pbi": null, "Task": null, "Bug": null },
  "cadence": { "unit": "sprint", "planningCeremonies": ["sprint-plan", "sprint-start"], "reviewCeremony": "sprint-review" },
  "capacityModel": { "velocityWindowN": 3, "unit": "storyPoints", "coldStart": "po-seed", "seedVelocity": null, "availabilityFactorDefault": 1.0 },
  "branches": { "dev": "dev", "release": "release" }
}
```

| Field | Meaning |
|---|---|
| `id` / `displayName` | the methodology key + human label. They name the *methodology* — the ceremony chain + role set — so they are **mode-independent** (a flow-mode project still carries `"scrum"`/`"Scrum"`). |
| `mode` | **`"scrum"`** (the sprint is a time-box with a capacity ceiling) or **`"flow"`** (the sprint has no dates and no capacity ceiling). Selects two things and nothing else: whether `capacityModel` is present, and which carrier the sprint uses (§1.2). **Absent ⇒ `scrum`** — see the resolution rule in §1.1. |
| `roles[]` | each role's `binding` (**`agent`** vs **`human`**) *and* its `dispatch` nature — **`in-session`** (interactive, e.g. `intake`), **`subagent`** (a short-lived ceremony subagent: analysts, PM, tech-lead), or **`worker`** (a fresh isolated `claude -p` per work-unit: `developer`, `tester`). A ceremony reads *both* facts from one place — "is this a human?" and "how do I spawn it?". `developer` carries `"instances": "dynamic"` (the dispatcher decides how many). |
| `artifactHierarchy` | the abstract, template-independent work-item ladder (Epic → Feature → PBI → Task). |
| `workItemTypeMap` | **null-seeded on purpose.** Concrete type + state names are backend- and process-template-dependent — they differ across backends and templates. Ceremonies fill this at connect time by resolving the type/state model per the active backend's adapter (completion/state, concept #7) — **never** hardcode a literal (§3). |
| `cadence` | the time unit + which ceremonies plan vs review a cycle. **Mode-independent** — the unit is `sprint` and the same ceremonies plan and review it in both modes. |
| `capacityModel` | the velocity/capacity formula co-located with the methodology (not baked into the PM agent): `velocityWindowN` (mean over the last N closed sprints), `unit` (story points), `coldStart` (`po-seed` when `< N` sprints exist), `seedVelocity` (PO-set at kickoff), `availabilityFactorDefault` (a 0–1 dial for short-staffed sprints). **REQUIRED under `mode: "scrum"`, ABSENT under `mode: "flow"`** (the key is omitted entirely — §1.1). |
| `branches` | the descriptor's **default** dev/release branch names. The project's *actual* names live in `config.branchPair` (§2) — see the reconciliation note there. |

The descriptor is deliberately **not** a per-phase state-machine — phase flow lives in the
ceremony-skills, not in descriptor-encoded transitions. Encoding transitions/guards here would
be the multi-methodology *engine*, which is deferred (YAGNI: build the config-seam, not the
engine). A second methodology **that stays Scrum-shaped** (e.g. a different velocity window or capacity
unit) ships as a second descriptor instance overriding these fields with no ceremony edit. But a
**genuinely different** methodology (Kanban's WIP-limited continuous flow, SAFe's program-increment
tier) is **not** a descriptor swap: `/sprint-plan` selects *a sprint's* worth of work by definition
(velocity-driven under `mode: "scrum"`, priority + DAG closure under `mode: "flow"` — §1.1.1), so
Kanban's WIP-limited pull needs its own planning/dispatch/review ceremonies, and the descriptor's
`cadence.planningCeremonies` *names* them — so those ceremonies must exist. The config-seam is
ceremony-agnostic (the deterministic engine reads `plan.json`, never the descriptor), which is why
the seam is done; but "multi-methodology" means writing a second ceremony chain, not one more JSON
file.

### 1.1 `mode` — `scrum` or `flow`

**Mode is not a second methodology.** The paragraph above draws the line at the ceremony chain: a
methodology that needs *different ceremonies* is a different methodology. Mode stays on the near
side of that line — same chain, same `roles`, same `artifactHierarchy`, same `cadence` unit, the
same ceremonies planning and reviewing it. Exactly **two** things differ between the modes, and
everything else in the descriptor is identical. (Flow **mode** is not Kanban: Kanban's WIP-limited
continuous flow is still the far side of that line and would need its own ceremonies.)

| `mode` | What a sprint is | Admission | `capacityModel` | Sprint carrier (§1.2) |
|---|---|---|---|---|
| `"scrum"` | a **time-box** — a dated range on the backend's schedule, with a capacity ceiling | by priority, up to a velocity-derived point budget | **REQUIRED** | the **iteration field** (concept #6) |
| `"flow"` | **the same sprint, with no dates and no capacity ceiling** — a named set of admitted units, nothing more | by **priority**, with the admitted set kept **DAG-closed** (§1.1.1); no point budget, and no admission ceiling of any kind | **ABSENT** — the key is omitted, not `null` | the **`sprint:<slug>` label** (concept #4) |

Under `flow`, the ceremonies that read capacity stop reading it: `/sprint-plan` admits by priority
without a ceiling — closing the admitted set over its dependencies rather than filtering it down to
what could start today (§1.1.1) — and `/sprint-review` reports **no velocity** (with no
time-box to divide by, a points-per-sprint number is arithmetic without a meaning). Nothing else
in the chain changes — decomposition, the dependency DAG, the promotion gate, and mid-flight
intake are all mode-independent.

> **WHY the time-box is optional.** Velocity over a fixed period assumes a *stable capacity* to
> measure. A solo maintainer working with an autonomous agent has none — one session produces
> eight shippable units, the next produces zero — so a mean over the last three sprints predicts
> nothing, and the ceiling derived from it silently caps real work behind a fiction. A team with a
> stable capacity still wants that ceiling, which is why `scrum` is unchanged and stays the
> default.

**The vocabulary does not change.** A sprint is a **sprint** in both modes; flow mode means
exactly and only *a sprint with no dates and no capacity*. There is no second noun for it, and
`id`/`displayName` keep naming the methodology (`"scrum"`/`"Scrum"`), so `config.methodology` (§2)
still reads `"scrum"` on a flow-mode project.

**Resolution — how a ceremony gets the mode:**

- **Read `mode` from `.delivery/methodology.json`**, in the descriptor load every ceremony already
  does before its first step. It is a *methodology* fact, never a connection fact — it is not in
  `config.json`.
- **Absent ⇒ `scrum`.** Every project configured before this field existed ran a time-boxed,
  capacity-driven sprint, so `scrum` is the only default that leaves an existing project's
  behaviour exactly as it was: the field is additive, and a project changes mode when someone
  writes the field, never by upgrading. (The same backward-compatibility posture as
  `config.backend` defaulting to `azure`.)
- **Never infer it.** Not from a missing `capacityModel`, not from which board fields exist, not
  from whether the backend has iterations. A derived signal lies — a leftover block after a mode
  switch, a hand-edit — while the declared field is the only authority, and its absence *is* an
  answer (`scrum`).
- **An unrecognized value stops the ceremony.** `mode` is `"scrum"` or `"flow"`; anything else (a
  typo, a methodology name) is surfaced with the two valid values and the ceremony halts. Falling
  back to `scrum` on a typo would impose a capacity ceiling on a project that asked not to have
  one and present the resulting plan as correct.
- **A mismatched `capacityModel` is not a second opinion.** Present under `mode: "flow"` → ignored
  (mode wins; `/delivery-init` removes it on a re-run). Missing under `mode: "scrum"` → a broken
  descriptor: surface it and stop, never invent a velocity window or a seed.

A complete **flow-mode** `methodology.json` — the scrum example above with `mode` flipped and
`capacityModel` gone; every other field byte-identical:

```json
{
  "id": "scrum",
  "displayName": "Scrum",
  "mode": "flow",
  "roles": [
    { "name": "intake",            "binding": "agent", "dispatch": "in-session" },
    { "name": "business-analyst",  "binding": "agent", "dispatch": "subagent" },
    { "name": "technical-analyst", "binding": "agent", "dispatch": "subagent" },
    { "name": "project-manager",   "binding": "agent", "dispatch": "subagent" },
    { "name": "tech-lead",         "binding": "agent", "dispatch": "subagent" },
    { "name": "tester",            "binding": "agent", "dispatch": "worker" },
    { "name": "developer",         "binding": "agent", "dispatch": "worker", "instances": "dynamic" },
    { "name": "product-owner",     "binding": "human" }
  ],
  "artifactHierarchy": ["Epic", "Feature", "Pbi", "Task"],
  "workItemTypeMap": { "Pbi": null, "Task": null, "Bug": null },
  "cadence": { "unit": "sprint", "planningCeremonies": ["sprint-plan", "sprint-start"], "reviewCeremony": "sprint-review" },
  "branches": { "dev": "dev", "release": "release" }
}
```

#### 1.1.1 Admission vs dispatch readiness — and what `DAG-ready` means

Two different questions get asked about a work-unit's dependencies, by two different consumers, at
two different times. They have **different answers**, and answering the first with the second is
the one mistake that quietly disables the delivery engine.

| Question | Asked by | Answer |
|---|---|---|
| **Admission** — *does this unit belong to this sprint?* | `/sprint-plan`, once, at planning time | **priority**, and the admitted set must be **DAG-closed** |
| **Dispatch readiness** — *can this unit start right now?* | `atl work dispatch`, continuously, at run time | **all of its predecessors are Done** — the **ready frontier**, and the only thing `DAG-ready` means |

**The canonical admission rule — quote this verbatim, do not paraphrase it:**

> **Under `mode: "flow"`, `/sprint-plan` admits by PRIORITY, and the admitted set must be
> DAG-CLOSED: whenever a unit is admitted, every predecessor it depends on is admitted with it, or
> is already complete. Never admit a unit whose predecessor stays *outside* the sprint and
> incomplete — that unit could never start, and that is the only "readiness" admission cares
> about. A unit blocked by another unit *in the same sprint* is entirely normal: that is precisely
> the edge `/sprint-start` puts in the DAG and the engine orders.**

**There is no admission ceiling under `flow`.** The `project-manager`'s ~4–6 concurrency cap bounds
**execution** — how many units the engine keeps in flight at once — not **membership**. Nothing
caps how many units a flow sprint may hold.

**`DAG-ready` keeps its single, pre-existing meaning: the dispatch frontier** (predecessors Done),
which is what the `project-manager`'s
[`sprint-planning-blueprint.md`](../agents/project-manager/children/sprint-planning-blueprint.md)
means by it. It is **not** an admission criterion. Admission never asks whether a unit can start
*today*; it asks only whether the unit could **ever** start inside this sprint — which is exactly
the DAG-closure condition above.

**A unit whose predecessor stays outside the sprint and incomplete is BLOCKED, not admitted.** That
is the same case
[`reject-and-carryover.md`](../agents/project-manager/children/reject-and-carryover.md) carries and
surfaces rather than pulling forward; DAG closure is that rule stated from the admission side —
either the predecessor comes in too, or the unit stays out, and either way nothing is silently
dropped.

**None of this is a third thing `mode` selects.** Mode still selects exactly the two things §1.1
names — whether `capacityModel` is present, and the sprint carrier. Admission is **by priority** in
*both* modes; all that differs is what bounds it: the capacity ceiling under `scrum`, nothing at all
under `flow`. Closure is not a bound — it is the shape the admitted set has to have for the sprint
to be schedulable, and it is written down here because **removing the budget is what made the wrong
reading available**. Under `scrum` the ceiling bounds a by-priority selection that already admits an
admitted unit's dependents as ordinary backlog units; scrum was never narrowed to the ready
frontier, and this subsection changes nothing about it.

> **WHY the obvious reading is wrong.** Reading flow admission as *"admit the units that are
> DAG-ready"* is the natural mistake, and it silently disables the engine's headline capability.
> If every admitted unit already has all its predecessors Done, then **no admitted unit depends on
> another admitted unit — the sprint contains no EDGE.** `/sprint-start` then materializes a
> `plan.json` of isolated nodes, and `atl work dispatch`'s dependency ordering and parallel
> `--cap N` scheduling — the entire reason the DAG is built — can never run. This was observed for
> real: `/refine` decomposed a Feature into A, B→A and C→A; a ready-frontier admission admitted
> **only A**, and a three-unit sprint shipped as a single-node plan. Under `scrum` the same three
> units are all admitted (they fit the point budget) and the engine orders them correctly — which
> is the tell that the defect was in the flow *admission rule*, never in the engine.

### 1.2 The sprint carrier — `sprint:<slug>` under flow

A sprint needs a **carrier**: the durable mark that says *this unit belongs to this sprint*, and
which the "read a sprint's items" query filters on. The carrier is mode-selected — the second and
last thing mode changes.

| `mode` | Carrier | Membership is |
|---|---|---|
| `"scrum"` | the **iteration field** (concept #6) — unchanged | an idempotent iteration **field** set; the sprint is a dated node on the backend's schedule |
| `"flow"` | the **`sprint:<slug>` label** (concept #4) | an idempotent **tag/label add**; there is no schedule node at all |

**The literal shape is `sprint:<slug>`** — the lowercase word `sprint`, one colon, then the slug,
and nothing else in the label: no dates, no title, no spaces.

**`<slug>` is the sprint's ordinal** — a positive decimal integer, unpadded, no leading zeros:
`sprint:1`, `sprint:2`, … `sprint:14`. The whole label therefore matches `sprint:[0-9]+`, and sits
far inside the tighter of the two backends' limits (GitHub's 50-character label cap — stated in
`backends/github/adapter.md` §5, the same cap that bounds `atl-key` and `atl-request`).

- **Form it by resolving, never by inventing.** List the `sprint:*` labels/tags already on the
  board (concept #10 — "list means all"; a capped read is a truncation to surface, not a complete
  one) and take the highest ordinal `k`, **compared as an integer** (`sprint:10` outranks
  `sprint:9`; a lexical "highest" hands back a stale ordinal and the next sprint reuses a number
  already in use). `sprint:<k>` is the **current** sprint and stays current until it is *reviewed*,
  so a run plans **into `sprint:<k>`** while its `Sprints/Sprint-<k>-Review` page (concept #9) is
  absent, and opens **`sprint:<k+1>`** only once that page exists — advancing on the highest
  ordinal *alone* would open a fresh sprint on every re-plan, defeating the convergence the
  idempotency bullet below promises. A board with none starts at `sprint:1`. A project **migrating
  from scrum** continues its existing numbering rather than restarting at 1: take the highest
  ordinal `m` among the existing `Sprints/Sprint-<m>-Review` pages (concept #9) and open
  `sprint:<m+1>` — those pages are the reliable read, since iteration *names* are arbitrary — so
  review pages never collide.
- **`<n>` is whichever ordinal that resolved to.** The `Sprints/Sprint-<n>-Review` durable-knowledge
  page keeps its name and its meaning in both modes; under flow, `<n>` comes from the label instead
  of from the resolved iteration name.
- **Written by the admission step, read by everyone else.** The label is stamped by the same
  ceremony step that sets the iteration field under scrum. Everything downstream treats it as
  read-only. Leaving a unit in a sprint — a `/sprint-review` carryover keeps its sprint — means
  leaving the label in place; a label is **never** removed to mean "done", because completion is a
  state (concept #7) and always was.
- **At most ONE `sprint:` label per unit — re-admission SWAPS it.** A field replaces its value
  where a label would accumulate, so the contract closes that gap by hand: admitting a carryover
  unit into the sprint being planned removes **the `sprint:` label that unit actually carries** —
  read that ordinal off the unit, never assume it is the immediately-preceding one, since a unit
  that stayed blocked across sprints was never re-admitted and still carries the ordinal of the last
  sprint it *was* in — and adds `sprint:<n>` in the same step. Two
  sprint labels on one unit is a corrupt state — "which sprint is this in?" stops having an answer
  and the sprint's item read returns units that moved on. History is not lost by the swap any more
  than it is under scrum (where the field is likewise overwritten): the sprint's membership record
  is its `Sprints/Sprint-<n>-Review` page.
- **Idempotent by nature, exactly like the field set.** Adding a label that is already there is a
  no-op, so a re-planned or crash-resumed sprint converges instead of duplicating. Never model
  membership as a "create membership" operation that could double.
- **A flow sprint has no object on the backend.** It exists only as the set of units carrying its
  label — nothing creates, dates, or closes a sprint entity. The **current** sprint is the highest
  ordinal present; it stays current until `/sprint-review` reviews it, and reviewed (the flow
  analogue of a *closed* iteration) means its `Sprints/Sprint-<n>-Review` page exists.

> **WHY a label rather than the iteration field.** It is the only carrier that needs **zero
> board-admin setup on either backend** — Azure tags and GitHub issue labels are free-form and
> queryable out of the box, while a Projects v2 **Iteration** field cannot be created through
> `gh`/GraphQL at all (`field-create --data-type` offers no iteration type) and has to be added by
> hand in the Projects settings UI. It also matches the machine-contract convention already carrying every
> other cross-cutting fact on an item — `type:<t>`, `area:<name>`, `atl-key:<hash>`,
> `atl-run:<…>`, `atl-brainstorm:<slug>`, `atl-request:<slug>:<initiator>` (concept #4) — so it
> introduces a value, not a mechanism.

**Consequence — a flow-mode project needs neither the `Iteration` field nor the `Story Points`
field.** The label carries membership, and with no capacity ceiling nothing reads an estimate. On
GitHub that removes the one setup step that could not be automated (the manual Iteration field);
on both backends it removes a per-sprint schedule chore. `Priority` is still needed — flow admits
by priority (§1.1.1) — and so is `Status`.

## 2. `config.json` — connection identity (no secret)

The shape is **backend-specific** — it carries the selected `backend`, that backend's
coordinates, the branch pair, the methodology id, and a **by-name** credential reference (never
a token). Written once by [`/delivery-init`](../skills/delivery-init/SKILL.md) (§4 of its
procedure).

### Azure shape (`backend: "azure"`)

```json
{
  "org": "<org>",
  "project": "<project>",
  "repo": "<repo>",
  "branchPair": { "dev": "dev", "release": "release" },
  "backend": "azure",
  "methodology": "scrum",
  "transport": "mcp",
  "restFallbackEnabled": true,
  "wikiId": "<resolved-at-init-or-null>",
  "pat": { "ref": "AZURE_DEVOPS_PAT" }
}
```

| Field | Meaning |
|---|---|
| `org` / `project` / `repo` | the active backend's project coordinates. `org` is derived from the project's `url` authority at init, not asked separately. |
| `branchPair` | the project's **actual** `dev` / `release` branch names (the two-branch delivery flow's integration + release lines). |
| `backend` | the active backend the project runs on — `"azure"` or `"github"`, chosen once at `/delivery-init` and cached here. Selects which `backends/<backend>/adapter.md` every ceremony and worker loads to bind each interface concept to a concrete tool. Default `azure`. |
| `methodology` | the `id` of the active `methodology.json` (v1: `"scrum"`). The **mode** lives in the descriptor, not here (§1.1). |
| `transport` | the transport the active adapter uses — `"mcp"` for the Azure backend (see `backends/<backend>/adapter.md`). |
| `restFallbackEnabled` | `true` — enables the Azure backend's one REST carve-out for evidence attachment (concept #12; see `backends/azure/adapter.md`). |
| `wikiId` | the Azure backend's durable-knowledge store id, resolved **once** at init and cached so ceremonies never re-resolve it (durable-knowledge store, concept #9; see `backends/azure/adapter.md`). `null` when none is provisioned yet — the store must exist before `/kickoff` seeds knowledge. |
| `pat` | **`{ "ref": "<env-var-name>" }` — a pointer, never the token.** There is no token field in the schema. |

### GitHub shape (`backend: "github"`)

```json
{
  "owner": "<owner>",
  "repo": "<repo>",
  "projectNumber": <n>,
  "branchPair": { "dev": "dev", "release": "release" },
  "backend": "github",
  "methodology": "scrum",
  "credential": { "ref": "GH_TOKEN" }
}
```

| Field | Meaning |
|---|---|
| `owner` / `repo` | the GitHub coordinates the `gh` CLI needs (`--repo <owner>/<repo>`). `owner` is the repo's org or user login; it also owns the board unless overridden. |
| `projectNumber` | the **owner-scoped GitHub Projects v2 board number** (`gh project … --owner <owner> <projectNumber>`). Distinct from `repo` — a board is owner-level, not nested under the repo. Resolved (or created) once at init. |
| `branchPair` | the project's actual `dev` / `release` branch names — same two-branch delivery flow as Azure. |
| `backend` | `"github"`. |
| `methodology` | the `id` of the active `methodology.json` (v1: `"scrum"`) — backend-independent. The **mode** lives in the descriptor, not here (§1.1). |
| `credential` | **`{ "ref": "GH_TOKEN" }` — a **by-name** pointer to the env var the GitHub token lives in, never the token itself.** The engine reads the value from that env var (`os.Getenv(config.credential.ref)`, defaulting to `GH_TOKEN`) and injects it into workers **as** `GH_TOKEN` so `gh` finds it (`workerenv.go`) — so `credential.ref` names the SOURCE env var the engine reads from (parity with Azure's `pat.ref`), and re-pointing it re-points the read. There is no token field. |

GitHub carries **no** `wikiId` (its durable-knowledge store is in-repo `/docs`, which has no id —
see `backends/github/adapter.md` §9) and **no** `transport` / `restFallbackEnabled` (those are
the Azure MCP-first transport policy; GitHub is `gh`-native, no MCP, no REST carve-out).

**The backend credential is referenced by name, never stored.** The credential field names
*where* the secret lives, and is resolved at run time:

- **Azure** — `pat.ref` in priority: (1) the Azure MCP's own configured auth, (2) an env var
  named by `pat.ref` (default `AZURE_DEVOPS_PAT`), (3) the OS keychain.
- **GitHub** — `credential.ref` (default `GH_TOKEN`): the engine reads the value from the env var it
  names (`os.Getenv(config.credential.ref)`) and injects it into workers **as** `GH_TOKEN` so `gh`
  finds it (`workerenv.go`) — the same configurable indirection as Azure's `pat.ref` (the worker's
  token is always exposed as `GH_TOKEN`; `credential.ref` names where the engine reads it from).

A committed token is exactly the exfiltration surface `atl guard` + the `untrusted-input` rule
exist to protect — `/delivery-init` **refuses to write a literal token**, and no ceremony ever
writes one back.

> **Backend-specific schema — each backend has its own concrete shape; unifying them is a
> deferred follow-up.** The two shapes above diverge on purpose: Azure carries `org`/`project`/
> `wikiId`/`transport`/`pat.ref`; GitHub carries `owner`/`projectNumber`/`credential.ref` and
> omits the Azure-only fields. Each is documented and consumed as-is. Collapsing them into one
> *neutral* schema (e.g. a shared core + a nested per-backend `backendConfig` block, so
> ceremonies read one common shape) is a separate refactor follow-up — the two documented shapes
> are the current contract.
>
> **Ceremony consumers still name Azure fields.** The ceremonies (`/kickoff`, `/refine`,
> `/sprint-review`) currently read `org`/`project`/`repo`/`wikiId` by name; teaching them to read
> the GitHub coordinates (`owner`/`projectNumber`) is the consumer-side companion to this writer,
> tracked with the GitHub-backend e2e (#212). `/delivery-init` is the **writer** and defines the
> shape here; the consumer neutralization lands before the loop is proven end-to-end.

> **Reconciled — `config.branchPair` vs `methodology.branches`.** Both name the dev/release
> branches; the split is deliberate. `methodology.branches` is the descriptor **default**
> (part of the methodology template); `config.branchPair` is the project's **resolved actual**
> and is **authoritative at run time**. `/delivery-init` seeds `branchPair` from the descriptor
> default and lets the user override per project; a ceremony that needs the branch names reads
> `config.branchPair`. When they differ (a per-project override), config wins.

## 3. Read contract — how ceremonies consume this

- **Methodology is data.** A ceremony loads `methodology.json`, reads the roles/cadence/
  capacity it needs, and acts — it does not encode methodology assumptions in its own prose.
- **Resolve the mode before anything that assumes a time-box.** `mode` (§1.1) decides whether a
  ceremony reads `capacityModel` at all, which sprint carrier it reads and writes (§1.2), and what
  bounds admission (§1.1.1 — the capacity ceiling under `scrum`, DAG closure under `flow`), so it
  is resolved in the same descriptor load, before the first step that touches any of them. **Absent ⇒
  `scrum`** (an existing project's behaviour never changes underneath it); an unrecognized value
  halts the ceremony; the declared field is the only authority — never infer the mode from the
  board.
- **Resolve concrete names at runtime, never hardcode.** `workItemTypeMap` is null in the
  descriptor by design. Before touching a work-item's type or state, resolve the real name per
  the active backend's adapter (completion/state, concept #7): "Done" for velocity is the
  resolved completion state, not the literal string `"Done"`. "Mark blocked" is **not** a
  state — it is a `blocked` tag/label (tags, concept #4) plus a diagnostic comment, never a
  state transition. This is what makes the team work on any backend and process template with
  zero org-admin setup.
- **Connection identity is read-only to ceremonies.** Ceremonies read `config.json`; only
  `/delivery-init` writes it. The `wikiId` cache and the `pat.ref` name are consumed, never
  re-derived, per ceremony run.

For the full operation contract — the operation map, resilience, idempotency, and
content-placement rules — see the [backend interface](backend-interface.md) and the active
backend's adapter under `backends/`.
