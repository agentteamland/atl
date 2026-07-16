---
name: delivery-init
description: /delivery-init — connect a project to Azure DevOps for the delivery-team. LLM-driven Q&A that discovers org/project/repo, probes the live MCP surface, resolves the project wiki, and writes .delivery/config.json (secret-by-reference, never a literal PAT) + .delivery/methodology.json (the Scrum descriptor every ceremony reads). Run once per project before /kickoff.
---

# /delivery-init — connect a project to Azure DevOps

This is the delivery-team's **onboarding** step: the one-time, per-project setup that
records *where* the work lives (the Azure DevOps org/project/repo, the branch pair, the
project wiki) and *which methodology* the team runs — so every later ceremony (`/kickoff`,
`/refine`, `/sprint-plan`, `/sprint-start`, `/sprint-review`) reads a settled config instead
of re-asking. It writes two files into a committed `.delivery/` directory:

| File | What it holds |
|---|---|
| `.delivery/config.json` | non-secret connection **identity** — org/project/repo, branch pair, transport, the resolved `wikiId`, and a **by-name reference** to where the PAT lives (never the token itself) |
| `.delivery/methodology.json` | the flat **methodology descriptor** every ceremony reads — roles + dispatch, the artifact hierarchy, cadence, the velocity/capacity model, and branch names (v1 = one Scrum instance) |

Field semantics for both files live in the team's contract doc,
[`knowledge/config-and-methodology.md`](../../knowledge/config-and-methodology.md); the Azure
tool map + auth path live in [`backends/azure/adapter.md`](../../backends/azure/adapter.md).
This skill is the **conversational writer** of that config — discovery and connectivity are
judgment-heavy (which project? which repo? is the PAT reachable?), which is Skill territory
under the CLI/Skill boundary. All Azure reads here go through the `azureDevOps` MCP.

## When to run

- **Once per project**, before `/kickoff` — a greenfield project's cold-start ceremony
  requires `config.json` present and a live connectivity probe.
- **Re-run** to update the connection (a new repo, a corrected wiki, a rotated PAT
  reference). Re-running is idempotent — see [Idempotent re-run](#idempotent-re-run).

## Procedure

### 1. Confirm you are at the project root

`.delivery/` is a **committed** directory (like `.atl/`), so it belongs at the repository
root. If the working directory isn't a repo root, ask the user to run the skill from there.
Read any existing `.delivery/config.json` first (idempotent re-run — see the last section).

### 2. Probe the Azure DevOps MCP — fail fast, explain the PAT

Confirm the `azureDevOps` MCP is reachable **before** asking anything, by calling
`core_list_projects`. Interpret the result:

- **Projects returned** → the MCP is live and authenticated. Continue.
- **Auth error / no projects / tool unavailable** → the MCP isn't configured or its PAT is
  missing. **Stop and explain** (do not guess a token): the `azureDevOps` MCP authenticates
  with a Personal Access Token supplied to the MCP server, not by this skill. Point the user
  at their MCP configuration (the official `@azure-devops/mcp … --authentication pat` shape,
  PAT from the environment / OS keychain), and re-run once it's connected. **Never** ask the
  user to paste a PAT into the chat, and never write one anywhere.

### 3. Discover and confirm org / project / repo (Q&A)

Drive this as a short conversation, offering discovered options rather than asking the user
to type identifiers blind:

- **Project** — from `core_list_projects` (present the names; let the user pick). The **org**
  is derived from the chosen project's `url` authority (`https://dev.azure.com/<org>/…`) — do
  not ask for it separately.
- **Repo** — from `repo_list_repos_by_project` for the chosen project (present; let the user
  pick). Default to the repo matching the project name when there's an obvious one.
- **Branch pair** — the `dev` / `release` names (defaults `dev` / `release`; confirm, allow
  override). These are the two-branch delivery flow's integration + release lines.
- **Methodology** — ask, even though **Scrum is the only v1 instance** (the seam is real).
  Confirm Scrum.

### 4. Resolve the project wiki (`wikiId`)

Call `wiki_list_wikis` for the chosen project and cache the wiki id:

- Prefer the **project wiki** (`type: 0`); else fall back to the first wiki returned.
- **No wiki exists** → set `wikiId` to `null` and tell the user the project wiki isn't
  provisioned yet; `/kickoff` (or the first wiki write) needs it, so they should create a
  project wiki in Azure DevOps before knowledge-seeding. Do not fabricate an id.

The `wikiId` is resolved **once here** and cached so ceremonies never re-resolve it (per the
adapter contract §8).

### 5. Record the transport policy

Set `transport` to `"mcp"` and `restFallbackEnabled` to `true` — the team is MCP-first, with
exactly one REST carve-out (attachment upload, adapter §9). No probe beyond step 2 is needed;
these are fixed for v1.

### 6. Record the PAT reference — never the token

The config carries **only a pointer** to where the secret lives, in a `pat.ref` field naming
an environment variable (default `AZURE_DEVOPS_PAT`). The schema has **no token field at
all**. At run time the PAT is resolved in priority order: (1) the Azure DevOps MCP's own
configured auth, (2) an env var named by `pat.ref`, (3) the OS keychain. Confirm the env-var
name with the user; **refuse to write a literal token** even if the user offers one — a
committed secret is exactly the exfiltration surface `atl guard` + the `untrusted-input` rule
exist to prevent.

### 7. Write the two files

Create `.delivery/` if absent, then write both files.

**`.delivery/config.json`** — substitute the confirmed values; keep the shape exactly:

```json
{
  "org": "<org>",
  "project": "<project>",
  "repo": "<repo>",
  "branchPair": { "dev": "dev", "release": "release" },
  "methodology": "scrum",
  "transport": "mcp",
  "restFallbackEnabled": true,
  "wikiId": "<resolved-wiki-id-or-null>",
  "pat": { "ref": "AZURE_DEVOPS_PAT" }
}
```

**`.delivery/methodology.json`** — the canonical Scrum descriptor, written **verbatim** (do
not resolve the `workItemTypeMap` here — it stays `null`-seeded; ceremonies resolve the real
Azure type/state names at connect time via `wit_get_work_item_type`, never hardcoded):

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

If the user chose non-default `dev` / `release` names in step 3, mirror them into **both**
`config.branchPair` and `methodology.branches` so the two agree.

### 8. Confirm connectivity (never on a guessed type name)

Steps 2–4 already made live, **authenticated** MCP calls (`core_list_projects`,
`repo_list_repos_by_project`, `wiki_list_wikis`) — reaching this step means auth + Azure
reachability are already proven. Do **not** add a gate that guesses a concrete work-item type
name (`wit_get_work_item_type` *requires* a type name, and the real PBI type is
process-template-dependent — `Product Backlog Item`, `User Story`, or a custom/renamed value).
Guessing `Product Backlog Item` would make init spuriously "fail" on any project whose type is
named differently, even when connectivity is perfect. Concrete type + state names are resolved
by **ceremonies at run time** via `wit_get_work_item_type` (adapter §6), never at init — that
is what makes the team work on any process template.

If you still want a best-effort process-template sanity check, a `wit_get_work_item_type` call
that returns **not-found** means "reachable, type is named differently" — a **success** for
connectivity, not an init failure; only an auth/transport error is a real failure. Either way,
do not write resolved names into `methodology.json` (the type map stays `null`).

### 9. Report

Summarize what was written — the org/project/repo, the branch pair, the `wikiId` (or the
"provision a wiki" note), and the `pat.ref` **name** (never a value). Point the user to the
next step: `/kickoff` for a brand-new project, or `/refine` / `/sprint-plan` for a project
that already has a backlog.

## Security: the PAT is never stored

- `config.json` has **no token field** — only `pat.ref`, a name.
- This skill **refuses to write a literal token** into any file, and never asks the user to
  paste one into the chat.
- The MCP server, not this skill, holds the PAT; `atl work dispatch` passes it to workers via
  the environment (adapter §1). A committed secret would be caught by `atl guard`, but the
  discipline is upstream of the guard: don't create the secret-in-a-file in the first place.

## Idempotent re-run

If `.delivery/config.json` already exists, **do not blind-overwrite**. Read it, show the
current values, and ask what to change — then rewrite only the confirmed fields. Same for
`methodology.json`: if the user has edited it, offer to update rather than clobbering. A
re-run must converge on the intended config, never silently discard a prior edit.
