---
name: delivery-init
description: /delivery-init — connect a project to the delivery-team's tracker. LLM-driven Q&A that first selects the backend (azure | github), then discovers the coordinates, probes live connectivity, resolves the board + durable-knowledge store, and writes .delivery/config.json (secret-by-reference, never a literal token) + .delivery/methodology.json (the Scrum descriptor every ceremony reads). Run once per project before /kickoff.
---

# /delivery-init — connect a project to its delivery backend

This is the delivery-team's **onboarding** step: the one-time, per-project setup that records
*which backend* the project runs on, *where* the work lives (the tracker coordinates, the
branch pair, the board, the durable-knowledge store), and *which methodology* the team runs —
so every later ceremony (`/kickoff`, `/refine`, `/sprint-plan`, `/sprint-start`,
`/sprint-review`) reads a settled config instead of re-asking. It writes two config files — plus a
`.gitignore` guard — into a committed `.delivery/` directory:

| File | What it holds |
|---|---|
| `.delivery/config.json` | non-secret connection **identity** — the `backend`, the backend's coordinates, the branch pair, and a **by-name reference** to where the credential lives (never the token itself). The exact fields are **backend-specific** (see step 4). |
| `.delivery/methodology.json` | the flat **methodology descriptor** every ceremony reads — roles + dispatch, the artifact hierarchy, cadence, the velocity/capacity model, and branch names. Backend-independent (v1 = one Scrum instance). |

Field semantics for both files live in the team's contract doc,
[`knowledge/config-and-methodology.md`](../../knowledge/config-and-methodology.md); the
per-backend tool map + auth path live in the active backend's adapter —
[`backends/azure/adapter.md`](../../backends/azure/adapter.md) or
[`backends/github/adapter.md`](../../backends/github/adapter.md). This skill is the
**conversational writer** of that config — backend selection, discovery, and connectivity are
judgment-heavy (which project? which repo? is the credential reachable?), which is Skill
territory under the CLI/Skill boundary.

## When to run

- **Once per project**, before `/kickoff` — a greenfield project's cold-start ceremony
  requires `config.json` present and a live connectivity probe.
- **Re-run** to update the connection (a new repo, a corrected board, a rotated credential
  reference). Re-running is idempotent — see [Idempotent re-run](#idempotent-re-run). Switching
  the *backend* on a re-run is an explicit re-scope, not a silent field edit — see that section.

## Procedure

### 1. Confirm you are at the project root

`.delivery/` is a **committed** directory (like `.atl/`), so it belongs at the repository root.
If the working directory isn't a repo root, ask the user to run the skill from there. Read any
existing `.delivery/config.json` first (idempotent re-run — see the last section); if one
exists, its `backend` is the current selection.

### 2. Select the backend

Ask which backend this project runs on — **`azure`** (Azure DevOps: Boards + Repos + Wiki via
the `azureDevOps` MCP) or **`github`** (GitHub Issues + Projects v2 + Pull Requests via the `gh`
CLI). **Default `azure`** for backward-compatibility. On a re-run, present the existing
`config.backend` as the default and confirm.

The choice selects which `backends/<backend>/adapter.md` every ceremony and worker loads, and
which discovery path this skill follows next: **§3A (Azure)** or **§3B (GitHub)**. Do only the
selected backend's discovery.

---

### 3A. Azure backend — discover, probe, resolve the wiki

*(Follow this branch only when `backend = azure`.)*

#### 3A.1 Probe the Azure DevOps MCP — fail fast, explain the PAT

Confirm the `azureDevOps` MCP is reachable **before** asking anything, by calling
`core_list_projects`. Interpret the result:

- **Projects returned** → the MCP is live and authenticated. Continue.
- **Auth error / no projects / tool unavailable** → the MCP isn't configured or its PAT is
  missing. **Stop and explain** (do not guess a token): the `azureDevOps` MCP authenticates
  with a Personal Access Token supplied to the MCP server, not by this skill. Point the user
  at their MCP configuration (the official `@azure-devops/mcp … --authentication pat` shape,
  PAT from the environment / OS keychain), and re-run once it's connected. **Never** ask the
  user to paste a PAT into the chat, and never write one anywhere.

#### 3A.2 Discover and confirm org / project / repo (Q&A)

Drive this as a short conversation, offering discovered options rather than asking the user to
type identifiers blind:

- **Project** — from `core_list_projects` (present the names; let the user pick). The **org**
  is derived from the chosen project's `url` authority (`https://dev.azure.com/<org>/…`) — do
  not ask for it separately.
- **Repo** — from `repo_list_repos_by_project` for the chosen project (present; let the user
  pick). Default to the repo matching the project name when there's an obvious one.
- **Branch pair** — the `dev` / `release` names (defaults `dev` / `release`; confirm, allow
  override). These are the two-branch delivery flow's integration + release lines.
- **Methodology** — ask, even though **Scrum is the only v1 instance** (the seam is real).
  Confirm Scrum.

#### 3A.3 Resolve the project wiki (`wikiId`)

Call `wiki_list_wikis` for the chosen project and cache the wiki id:

- Prefer the **project wiki** (`type: 0`); else fall back to the first wiki returned.
- **No wiki exists** → set `wikiId` to `null` and tell the user the project wiki isn't
  provisioned yet; `/kickoff` (or the first wiki write) needs it, so they should create a
  project wiki in Azure DevOps before knowledge-seeding. Do not fabricate an id.

The `wikiId` is resolved **once here** and cached so ceremonies never re-resolve it (per the
adapter contract §8).

#### 3A.4 Record the transport policy + the PAT reference — never the token

- Set `transport` to `"mcp"` and `restFallbackEnabled` to `true` — the team is MCP-first, with
  exactly one REST carve-out (attachment upload, `backends/azure/adapter.md` §9). These are
  fixed for v1; no probe beyond §3A.1 is needed.
- The config carries **only a pointer** to where the secret lives, in a `pat.ref` field naming
  an environment variable (default `AZURE_DEVOPS_PAT`). The schema has **no token field at
  all**. At run time the PAT is resolved in priority order: (1) the Azure DevOps MCP's own
  configured auth, (2) an env var named by `pat.ref`, (3) the OS keychain. Confirm the env-var
  name with the user; **refuse to write a literal token** even if the user offers one.

Then go to **step 4** (write the files) with the Azure config shape.

---

### 3B. GitHub backend — discover, probe, resolve the board

*(Follow this branch only when `backend = github`.)*

The GitHub backend is reached through the **`gh` CLI**, not an MCP. All reads here go through
`gh` (`gh auth status`, `gh repo view`, `gh project …`), per
[`backends/github/adapter.md`](../../backends/github/adapter.md) §1. There is **no wiki** — the
durable-knowledge store is in-repo `/docs` (§9), so nothing to resolve.

#### 3B.1 Probe `gh` connectivity — fail fast, explain the token scopes

Confirm `gh` is installed and authenticated **before** asking anything, by calling
`gh auth status`. Interpret the result:

- **Authenticated, with `repo` + `project` scopes** → live. Continue.
- **Not installed / not logged in** → **stop and explain**: this skill drives GitHub through
  `gh`, which needs an authenticated session (`gh auth login`) or a `GH_TOKEN` in the
  environment. Point the user there and re-run once connected.
- **Authenticated but missing `project` scope** → `gh project …` calls will fail. Tell the user
  to refresh the token with `gh auth refresh -s project` (Projects v2 read/write) and re-run.
  The board setup below needs `project`; issue/PR work needs `repo`.

> **Interactive init vs. the autonomous worker.** This skill runs interactively under the
> user's own `gh` auth. At run time the autonomous worker gets its token as `GH_TOKEN` injected
> by the engine (`workerenv.go`, per `backends/github/adapter.md` §1) — **never** from this
> config. Both need `repo` + `project` scope. The worker always receives its token **as `GH_TOKEN`**
> (what `gh` reads), but the SOURCE is `config.credential.ref` (step 4): the engine reads
> `os.Getenv(credential.ref)` (default `GH_TOKEN`) and injects it as `GH_TOKEN` — parity with Azure's
> `pat.ref`. It never stores the token.

#### 3B.2 Discover and confirm the repo (`owner/repo`)

Offer discovered values rather than asking the user to type identifiers blind:

- **Repo** — from `gh repo view --json nameWithOwner,defaultBranchRef` in the current directory
  (the repo is checked out at init). Present `owner/repo`; let the user confirm or override
  (e.g. if the delivery target is a different repo). If the directory has no default GitHub
  remote, ask for `owner/repo` explicitly.
- **Branch pair** — the `dev` / `release` names (defaults `dev` / `release`; confirm, allow
  override). Same two-branch delivery flow as Azure.
- **Methodology** — confirm Scrum (the only v1 instance; the seam is real).
- **Merge method (preflight — warn, don't block)** — the autonomous loop completes a PBI by
  merging its PR with a **real merge commit** (`gh pr merge --merge`), which the engine's
  `MergedToBase` check then verifies; a squash- or rebase-merge rewrites the SHA, so a
  genuinely-merged unit would look unmerged and false-block (see
  [`backends/github/adapter.md`](../../backends/github/adapter.md) §10). Read the repo's setting
  with `gh api repos/<owner>/<repo> --jq '.allow_merge_commit'`. If it returns `false`, **warn
  and continue** (this is a fixable repo setting, not a config value — same warn-and-continue
  posture as a missing Iteration field below): tell the user that *Allow merge commits* is
  disabled on this repo, so the autonomous merge gate will error or false-block completions, and
  to enable it under **Settings → General → Pull Requests** before running a sprint. `true`
  (GitHub's default) → nothing to do.

#### 3B.3 Board (GitHub Project) — connect an existing one or create a new one

A GitHub Projects v2 board is **owner-scoped** (org or user), identified by an owner + a
**number**, not nested under the repo. Default the **owner** to the repo's owner; allow an
override (e.g. a personal repo tracked on an org-level board). Ask **existing or new**:

**Existing board** → prompt for its **name / URL / number**:

- Accept a number directly, or extract it from a URL
  (`https://github.com/orgs/<owner>/projects/<n>` or
  `https://github.com/users/<owner>/projects/<n>`).
- Validate with `gh project view <n> --owner <owner>` — a not-found / access error means the
  number or owner is wrong, or the token lacks `project` scope; surface it and re-ask.
- Then run the **field check** below (create only what's missing — an existing board may already
  have the fields).

**New board** → create it and set up the fields:

- `gh project create --owner <owner> --title "<repo> delivery"` — capture the returned project
  **number** (`--format json` → `.number`).
- Then run the **field check** below (a fresh board has only the default fields).

**Field check (both paths — this is the field-setup half of #213).** List the board's fields
with `gh project field-list <n> --owner <owner> --format json` and reconcile against the four
the autonomous loop needs. **Create only what is missing** (idempotent):

| Field | Type | How to ensure it |
|---|---|---|
| **Status** | single-select | **Already exists by default** (`Todo` / `In Progress` / `Done`). Do **not** try to recreate it — the built-in Status field's *options* are not cleanly API-settable (see wiki `gh-projects-wip-manual`). Verify it carries `In Progress` + `Done` (a new board does); the loop's three states are exactly these. Note for the team: Projects v2's built-in automation only sets **Done** (on close/merge) — the *start* → `In Progress` transition is the ceremony's job, not the platform's. **If the project will use `/request` (mid-project intake):** it needs an extra **`candidate`** option on this Status field (a request captured pre-accept sits at `candidate`, excluded from the ready frontier). Options are not API-settable, so **tell the user to add a `candidate` option via the Projects settings UI** (Projects v2 → ⚙ → Status → new option), like the Iteration field — do not attempt it via `gh`. Optional: skip it if the project won't run `/request`. |
| **Iteration** | iteration | **`gh` / GraphQL cannot create an iteration field** (`field-create --data-type` supports only TEXT / SINGLE_SELECT / DATE / NUMBER). If it is missing, **tell the user to add an "Iteration" field via the Project's settings UI** (Projects v2 → ⚙ → new field → Iteration) and re-run — do not fake it as a text/number field. |
| **Story Points** | number | `gh project field-create <n> --owner <owner> --name "Story Points" --data-type NUMBER` |
| **Priority** | single-select | `gh project field-create <n> --owner <owner> --name "Priority" --data-type SINGLE_SELECT --single-select-options "P0,P1,P2,P3"` |

> **Scope boundary — native Issue Types stay in #213.** This field setup does **not** adopt
> native org **Issue Types** (Epic / Feature / PBI / Task / Bug) — that needs an `admin:org`
> scope refresh and is the other half of #213. The artifact ladder is carried by `type:<t>`
> **labels** for now (the Layer-1 approach); the ceremonies stamp them, and the native-type
> upgrade is tracked separately.

Then go to **step 4** with the GitHub config shape.

---

### 4. Write `.delivery/config.json` — the backend-specific shape

Create `.delivery/` if absent, then write `config.json`. **The shape is backend-specific**;
write the one matching the selected backend, substituting the confirmed values and keeping the
shape exactly. Both carry `backend`, `branchPair`, `methodology`, and a **by-name** credential
pointer (never a token). Field semantics: [`knowledge/config-and-methodology.md`](../../knowledge/config-and-methodology.md) §2.

**Azure** (`backend: "azure"`):

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

**GitHub** (`backend: "github"`) — note there is **no** `wikiId` (durable knowledge is in-repo
`/docs`), no `transport` / `restFallbackEnabled` (MCP-only policy), and the coordinates are
GitHub-native (`owner` / `repo` / `projectNumber`, not `org` / `project`):

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

For GitHub the worker receives its token **as `GH_TOKEN`** (what `gh` reads), but `credential.ref`
names the SOURCE env var the engine reads the value FROM: the engine reads
`os.Getenv(config.credential.ref)` (default `GH_TOKEN`) and injects it as `GH_TOKEN` — the same
configurable indirection Azure's `pat.ref` is. `"GH_TOKEN"` is the sensible default; re-point it
only if your token lives in a differently-named var.

### 5. Write `.delivery/methodology.json` — the shared Scrum descriptor

Backend-independent — write it **verbatim** for both backends (do not resolve the
`workItemTypeMap` here — it stays `null`-seeded; ceremonies resolve the real per-backend
type/state names at connect time via the active adapter, never hardcoded):

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

If the user chose non-default `dev` / `release` names, mirror them into **both**
`config.branchPair` and `methodology.branches` so the two agree.

### 6. Write `.delivery/.gitignore` — keep engine scratch out of git

`.delivery/` is committed, but only its **two source files** belong in git — `config.json`
and `methodology.json`. Everything else the engine and ceremonies drop here is regenerated
every run and must never be committed (the `mcp/` config can even embed a credential). Write
`.delivery/.gitignore` with exactly these entries (every `#` is a full-line comment — git does
not honor a trailing inline `#`, so keep each "why" on its own line above the pattern):

```gitignore
# .delivery/ is committed, but ONLY these two source files belong in git:
#   config.json, methodology.json  (written by /delivery-init).
# Everything below is engine-derived scratch — regenerated each run; never commit it.

# per-unit git worktrees, incl. the nested .quarantine/ (atl work dispatch)
worktrees/
# supervisor run state (atl work dispatch)
runstate.json
# materialized sprint DAG, derived from the backend (/sprint-start)
plan.json
# blocked-unit reports, drained back to the board by /sprint-review
blocked/
# generated worker MCP config — may embed a credential (atl work dispatch)
mcp/
# single-dispatch mutex
dispatch.lock
```

Idempotent: on a re-run, if `.delivery/.gitignore` already exists and matches, leave it;
otherwise (re)write it. `config.json` / `methodology.json` match no pattern, so they stay
tracked.

### 7. Report

Summarize what was written — the backend, its coordinates (Azure: org/project/repo + the
`wikiId` or the "provision a wiki" note; GitHub: `owner/repo` + the resolved `projectNumber` +
any field the user must still add in the UI), the branch pair, and the credential **reference
name** (never a value). Point the user to the next step: `/kickoff` for a brand-new project, or
`/refine` / `/sprint-plan` for a project that already has a backlog.

## Security: the credential is never stored

- `config.json` has **no token field** — only a by-name reference (`pat.ref` for Azure,
  `credential.ref` for GitHub, e.g. `GH_TOKEN`).
- This skill **refuses to write a literal token** into any file, and never asks the user to
  paste one into the chat.
- The backend, not this skill, holds the credential: the Azure MCP holds the PAT; the GitHub
  worker gets `GH_TOKEN` from the engine's env (`workerenv.go`). `atl work dispatch` passes it to
  workers via the environment (adapter §1). A committed secret would be caught by `atl guard`,
  but the discipline is upstream of the guard: don't create the secret-in-a-file in the first
  place.

## Idempotent re-run

If `.delivery/config.json` already exists, **do not blind-overwrite**. Read it, show the current
values, and ask what to change — then rewrite only the confirmed fields. Same for
`methodology.json`: if the user has edited it, offer to update rather than clobbering.

**Switching the backend** on a re-run is an explicit re-scope, not a field edit: the coordinate
fields differ between backends (Azure `org`/`project`/`wikiId` vs GitHub `owner`/`projectNumber`),
so a backend switch re-runs the whole selected branch (§3A or §3B) from scratch and writes the
new shape — confirm the switch with the user before discarding the old backend's coordinates. A
re-run must converge on the intended config, never silently discard a prior edit.
