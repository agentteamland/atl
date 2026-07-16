---
knowledge-base-summary: "Methodology is data, not hardcoded logic: I read roles/dispatch, cadence, capacityModel, artifactHierarchy, and branches from .delivery/methodology.json and act. config.json is read-only (only /delivery-init writes it). Resolve concrete type/state/iteration names at runtime (concept #7 completion/state, concept #6 iteration), never a literal Done ('blocked' is a tag/field, not a state). The branchPair-vs-methodology.branches reconciliation (config wins)."
---

# Methodology as Data

I am a methodology-driven role, but I hold **no methodology assumptions in my prose** — I read them
from the config seam and act. This is the discipline that lets the same planning craft run under
Scrum today and a second methodology (Kanban/SAFe) tomorrow, by shipping a different descriptor
instead of rewriting me. The contracts here are the config-and-methodology doc's read contract
(`config-and-methodology.md` §3), applied to my reflex.

## The two files I read (both read-only to me)

Under a project's committed `.delivery/`:

- **`methodology.json`** — the descriptor: `roles[]` (binding + dispatch), `artifactHierarchy`,
  `cadence`, `capacityModel`, `branches`. Written by `/delivery-init`.
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
| `roles[].dispatch` | tells me *how each collaborator is spawned* — `in-session` (intake), `subagent` (me, the analysts, the tech-lead), `worker` (developer/tester). I read my own dispatch nature and my neighbors' from here, not from memory. |
| `cadence` | `unit: "sprint"` + which ceremonies plan vs review. My planning cadence *is* whatever the descriptor says — I don't assume "two-week sprint". |
| `capacityModel` | every velocity/capacity parameter — `velocityWindowN`, `unit`, `coldStart`, `seedVelocity`, `availabilityFactorDefault` (see [capacity-and-velocity.md](capacity-and-velocity.md)). A descriptor with `velocityWindowN: 5` changes my math with zero change to me. |
| `artifactHierarchy` | the abstract ladder (Epic → Feature → PBI → Task) — the levels my granularity rule admits at ([sprint-planning-blueprint.md](sprint-planning-blueprint.md) §7). |
| `branches` | the descriptor's *default* dev/release names — but the *authoritative* names are `config.branchPair` (see the reconciliation below). |

> **WHY methodology-as-data and not baked-in logic.** If "3-sprint velocity window" or "Scrum"
> lived in my prose, a second methodology would require rewriting the role. Reading it from a
> descriptor makes the methodology a *swappable instance*: Kanban ships as a second
> `methodology.json` that overrides these fields, and every ceremony/role — including me — works
> unchanged. This is the config-seam's whole point: settings are data a ceremony reads, never logic
> baked into an agent (`config-and-methodology.md` §3). The descriptor is deliberately **not** a
> per-phase state machine — encoding phase transitions would be the multi-methodology *engine*,
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
- **Iteration names** → resolve at runtime by listing the backend's iterations (concept #6)
  ([iteration-management.md](iteration-management.md)) — the concrete `Sprint <n>` identifier is a
  live fact, never a constructed string.

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
2. Take *parameters* from the descriptor (`capacityModel`, `cadence`, `artifactHierarchy`).
3. Take *identity/connection* from config (the durable-knowledge store's locator where the backend
   needs one, `branchPair` — authoritative branches).
4. Resolve every *concrete backend name* (type, state, iteration) at runtime — never a literal.
5. Write nothing back to either file — they are `/delivery-init`'s to own; I am a reader.
