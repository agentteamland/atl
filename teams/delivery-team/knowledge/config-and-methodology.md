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
engine). A second methodology (Kanban/SAFe) would ship as a second descriptor instance that
overrides these fields without touching any ceremony.

## 2. `config.json` — connection identity (no secret)

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

**The backend credential is referenced by name, never stored.** `pat.ref` names *where* the secret lives;
it is resolved at run time in priority: (1) the active backend's own configured auth,
(2) an env var named by `pat.ref` (default `AZURE_DEVOPS_PAT`), (3) the OS keychain. A committed
token is exactly the exfiltration surface `atl guard` + the `untrusted-input` rule exist to
protect — `/delivery-init` **refuses to write a literal token**, and no ceremony ever writes
one back.

> **Backend-specific schema — flagged for a follow-up (not this pass).** `pat.ref` (the "PAT"
> naming), `wikiId` (an Azure durable-knowledge-store id — the GitHub backend's store is in-repo
> `/docs`, which has none), and `transport` / `restFallbackEnabled` (the MCP-first transport
> policy) are still Azure-shaped fields. This pass neutralizes the *prose* and framing only;
> making the *schema* itself backend-neutral is a separate follow-up.

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
