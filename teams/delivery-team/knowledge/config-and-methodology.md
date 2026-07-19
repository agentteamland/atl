# Config & methodology — the delivery-team's per-project settings contract

Two files under a project's **committed** `.delivery/` directory hold everything a ceremony
needs to know about *this* project. They are written once by [`/delivery-init`](../skills/delivery-init/SKILL.md)
(and refined on re-run), and read by every ceremony (`/kickoff`, `/refine`, `/sprint-plan`,
`/sprint-start`, `/sprint-review`) and every role-agent. This is the **config-seam**: settings
are *data* a ceremony reads, never logic baked into an agent.

| File | Purpose | Written by |
|---|---|---|
| `.delivery/config.json` | connection **identity** — where the work lives + how to reach it | `/delivery-init` |
| `.delivery/methodology.json` | the **methodology descriptor** — how the team works | `/delivery-init` (v1: one Scrum instance) |

Both are committed (like `.atl/`): the connection identity and the methodology are project
facts the whole team shares, not per-machine state. Neither file ever contains a secret.

## 1. `methodology.json` — the descriptor every ceremony reads

A **flat, ceremony-read descriptor**. v1 ships exactly one instance — Scrum. The descriptor
holds *intent*; the active backend holds concrete *names* (resolved at runtime, §3).

```json
{
  "id": "scrum",
  "displayName": "Scrum",
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
| `id` / `displayName` | the methodology key + human label. |
| `roles[]` | each role's `binding` (**`agent`** vs **`human`**) *and* its `dispatch` nature — **`in-session`** (interactive, e.g. `intake`), **`subagent`** (a short-lived ceremony subagent: analysts, PM, tech-lead), or **`worker`** (a fresh isolated `claude -p` per work-unit: `developer`, `tester`). A ceremony reads *both* facts from one place — "is this a human?" and "how do I spawn it?". `developer` carries `"instances": "dynamic"` (the dispatcher decides how many). |
| `artifactHierarchy` | the abstract, template-independent work-item ladder (Epic → Feature → PBI → Task). |
| `workItemTypeMap` | **null-seeded on purpose.** Concrete type + state names are backend- and process-template-dependent — they differ across backends and templates. Ceremonies fill this at connect time by resolving the type/state model per the active backend's adapter (completion/state, concept #7) — **never** hardcode a literal (§3). |
| `cadence` | the time unit + which ceremonies plan vs review a cycle. |
| `capacityModel` | the velocity/capacity formula co-located with the methodology (not baked into the PM agent): `velocityWindowN` (mean over the last N closed sprints), `unit` (story points), `coldStart` (`po-seed` when `< N` sprints exist), `seedVelocity` (PO-set at kickoff), `availabilityFactorDefault` (a 0–1 dial for short-staffed sprints). |
| `branches` | the descriptor's **default** dev/release branch names. The project's *actual* names live in `config.branchPair` (§2) — see the reconciliation note there. |

The descriptor is deliberately **not** a per-phase state-machine — phase flow lives in the
ceremony-skills, not in descriptor-encoded transitions. Encoding transitions/guards here would
be the multi-methodology *engine*, which is deferred (YAGNI: build the config-seam, not the
engine). A second methodology **that stays Scrum-shaped** (e.g. a different velocity window or capacity
unit) ships as a second descriptor instance overriding these fields with no ceremony edit. But a
**genuinely different** methodology (Kanban's WIP-limited continuous flow, SAFe's program-increment
tier) is **not** a descriptor swap: `/sprint-plan` is velocity-driven sprint selection *by
definition*, so Kanban needs its own planning/dispatch/review ceremonies, and the descriptor's
`cadence.planningCeremonies` *names* them — so those ceremonies must exist. The config-seam is
ceremony-agnostic (the deterministic engine reads `plan.json`, never the descriptor), which is why
the seam is done; but "multi-methodology" means writing a second ceremony chain, not one more JSON
file.

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
| `methodology` | the `id` of the active `methodology.json` (v1: `"scrum"`). |
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
| `methodology` | the `id` of the active `methodology.json` (v1: `"scrum"`) — backend-independent. |
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
