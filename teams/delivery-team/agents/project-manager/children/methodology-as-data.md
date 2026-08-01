---
knowledge-base-summary: "Methodology is data, not hardcoded logic: I read mode, roles/dispatch, cadence, capacityModel, artifactHierarchy, and branches from .delivery/methodology.json and act. mode ('scrum' | 'flow') is resolved FIRST — it decides whether capacityModel exists at all and whether a sprint's carrier is the iteration field or the sprint:<slug> tag/label; absent means scrum, an unrecognized value stops me, and it is never inferred from the board. config.json is read-only (only /delivery-init writes it). Resolve concrete type/state/iteration names at runtime (concept #7 completion/state, concept #6 iteration), never a literal Done ('blocked' is a tag/field, not a state). The branchPair-vs-methodology.branches reconciliation (config wins)."
---

# Methodology as Data

I am a methodology-driven role, but I hold **no methodology assumptions in my prose** — I read them
from the config seam and act. This is the discipline that lets the same planning craft run under
Scrum today and a second methodology (Kanban/SAFe) tomorrow, by shipping a different descriptor
instead of rewriting me. The contracts here are the config-and-methodology doc's read contract
(`config-and-methodology.md` §3), applied to my reflex.

## The two files I read (both read-only to me)

Under a project's committed `.delivery/`:

- **`methodology.json`** — the descriptor: `mode` (`scrum` | `flow`), `roles[]` (binding +
  dispatch), `artifactHierarchy`, `cadence`, `capacityModel` (under `scrum` only), `branches`.
  Written by `/delivery-init`.
- **`config.json`** — connection identity: `backend` (which adapter — `backends/<backend>/adapter.md`,
  default `azure`), the backend's coordinates (Azure `org`/`project`/`repo`; GitHub
  `owner`/`repo`/`projectNumber` — see [`config-and-methodology.md`](../../../knowledge/config-and-methodology.md)
  §2), `branchPair`, `methodology`, the durable-knowledge store's locator (where the backend needs
  one), and the credential pointer. Written by `/delivery-init`.

**Both are read-only to me.** Only `/delivery-init` writes them. I consume the durable-knowledge
store's locator (where the active backend needs one) and the branch names; I never re-derive them,
and I never write a token — the credential config is a *pointer* to where the secret lives (an env-var name),
never the secret itself (`config-and-methodology.md` §2). A ceremony/role that wrote a literal token
into config would create exactly the exfiltration surface `atl guard` + the `untrusted-input` rule
exist to stop.

## What I read from the descriptor, and why as data

| Field | How I use it (as data, not baked-in) |
|---|---|
| `mode` | **the first field I read** — `"scrum"` or `"flow"`. It decides exactly two things: whether `capacityModel` exists at all, and which carrier a sprint uses — the iteration field under `scrum`, the `sprint:<slug>` tag/label under `flow` (concept #6). **Absent ⇒ `scrum`** (a project configured before the field existed keeps behaving exactly as it did); an unrecognized value stops me with the two valid values; I never infer it from a missing `capacityModel` or from which fields the board happens to have. |
| `roles[].dispatch` | tells me *how each collaborator is spawned* — `in-session` (intake), `subagent` (me, the analysts, the tech-lead), `worker` (developer/tester). I read my own dispatch nature and my neighbors' from here, not from memory. |
| `cadence` | `unit: "sprint"` + which ceremonies plan vs review. My planning cadence *is* whatever the descriptor says — I don't assume "two-week sprint". |
| `capacityModel` | every velocity/capacity parameter — `velocityWindowN`, `unit`, `coldStart`, `seedVelocity`, `availabilityFactorDefault` (see [capacity-and-velocity.md](capacity-and-velocity.md)). A descriptor with `velocityWindowN: 5` changes my math with zero change to me. **Present under `mode: "scrum"` only** — under `flow` the key is absent, I compute no ceiling, and admission is priority + DAG readiness instead. A mismatch is never a second opinion, in either direction: present under `flow` → mode wins and I ignore it; **missing under `scrum` → a broken descriptor, which I surface and stop on — I never invent a velocity window or a seed to keep going** (`config-and-methodology.md` §1.1). |
| `artifactHierarchy` | the abstract ladder (Epic → Feature → PBI → Task) — the levels my granularity rule admits at ([sprint-planning-blueprint.md](sprint-planning-blueprint.md) §7). |
| `branches` | the descriptor's *default* dev/release names — but the *authoritative* names are `config.branchPair` (see the reconciliation below). |

> **WHY methodology-as-data and not baked-in logic.** If "3-sprint velocity window" or "Scrum"
> lived in my prose, tuning the *parameters* would require rewriting the role. Reading them from a
> descriptor makes them *swappable data*: a descriptor with a different `velocityWindowN` or capacity
> `unit` changes my math with zero change to me. `mode` is the same kind of thing — a parameter, not
> a second methodology: both modes run the same ceremony chain with the same roles, so flipping it
> changes what I read, never who I am. But a **genuinely different** methodology (Kanban's
> WIP-limited flow, SAFe's PI tier) is NOT: `/sprint-plan` selects *a sprint's* worth of work by
> definition (velocity-driven under `mode: "scrum"`, priority + DAG readiness under `mode: "flow"`),
> so Kanban's WIP-limited pull needs its own ceremonies, which `cadence.planningCeremonies` would
> *name* (so they must exist). This is the config-seam's point: settings are data a ceremony reads,
> never logic baked into an agent (`config-and-methodology.md` §3). The descriptor is deliberately
> **not** a per-phase state machine — encoding phase transitions would be the multi-methodology *engine*,
> which is deferred (YAGNI); I read intent and act, I don't execute a descriptor-encoded FSM.

## Resolve concrete names at runtime — never hardcode

The descriptor holds *intent*; the live backend project holds concrete *names*. `workItemTypeMap` is
**null-seeded on purpose** (`{ "Pbi": null, "Task": null, "Bug": null }`) — the real type and
state names are backend- and process-template-dependent. So, before I touch a type or a state:

- **Types/states** → resolve the completion/state model at runtime (concept #7). "Done" for velocity
  is the resolved **Completed** state, not the literal string `"Done"`. "Mark blocked" is **not** a
  state — no backend models blocking as a completion state; it is a `blocked` tag/label (plus a
  backend-specific blocked field where the type exposes one), leaving the item's state unchanged. I
  never write a literal state into my reasoning, and I never invent a `Blocked` state.
- **The sprint's carrier** → under `mode: "scrum"`, resolve the concrete iteration at runtime by
  listing the backend's iterations (concept #6)
  ([iteration-management.md](iteration-management.md)) — the concrete `Sprint <n>` identifier is a
  live fact, never a constructed string. Under `mode: "flow"` there is no iteration to resolve: the
  carrier is the `sprint:<slug>` tag/label (concept #4/#6), and its ordinal is resolved the same
  read-first way — list the `sprint:` values already on the board, take the highest, admit into the
  next; never a number I chose.

> **WHY runtime resolution is non-negotiable.** Hardcoding `"Done"` (or assuming a `"Blocked"`
> state that no standard template even has) silently breaks the moment a project uses a different
> process template — the query matches nothing, velocity reads zero, the plan admits garbage.
> Resolving real states at runtime (and treating "blocked" as a tag/field, not a state) is what
> makes the team work on *any* backend and process template with zero org-admin setup.

## The branchPair ↔ methodology.branches reconciliation

Two places name the dev/release branches; the split is deliberate and I must read the right one:

- `methodology.branches` — the descriptor **default** (part of the methodology template, e.g.
  `{ "dev": "dev", "release": "release" }`).
- `config.branchPair` — the project's **resolved actual** names, and **authoritative at run time**.

`/delivery-init` seeds `branchPair` from the descriptor default and lets the user override per
project. **When they differ, `config.branchPair` wins.** So whenever I need a branch name — e.g. the
`## Deployable dev preview` note in the sprint-review report
([sprint-review-report.md](sprint-review-report.md)) reads the `dev` branch — I read
`config.branchPair.dev`, **not** `methodology.branches.dev`.

> **WHY config wins over the descriptor default.** The descriptor is a shared template that could be
> reused across projects; a given project may name its branches differently (`main`/`prod`,
> `develop`/`main`). The per-project override in `config` is the ground truth for *this* project;
> the descriptor default is only the seed. Reading config keeps me correct on a project that renamed
> its branches without editing the shared methodology template.

## The read discipline, condensed

1. Load `methodology.json` + `config.json` at the start of my ceremony participation.
2. Resolve `mode` **first**, before anything that assumes a time-box — absent ⇒ `scrum`,
   unrecognized ⇒ stop, never inferred from the board.
3. Take *parameters* from the descriptor (`capacityModel` **where the mode has one**, `cadence`,
   `artifactHierarchy`).
4. Take *identity/connection* from config (the durable-knowledge store's locator where the backend
   needs one, `branchPair` — authoritative branches).
5. Resolve every *concrete backend name* (type, state, and the sprint's carrier) at runtime — never
   a literal.
6. Write nothing back to either file — they are `/delivery-init`'s to own; I am a reader.
