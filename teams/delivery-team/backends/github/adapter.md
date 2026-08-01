# GitHub adapter — the delivery-team's GitHub backend

The GitHub implementation of the [backend interface](../../knowledge/backend-interface.md).
It binds each interface concept to a concrete GitHub mechanism, and states the
GitHub-specific procedures the neutral cross-cutting policies compile to. It is the peer of
`backends/azure/adapter.md`; a project on the `github` backend follows **this** contract for
every work-tracking / knowledge / PR operation. Do not improvise a GitHub call that isn't
described here.

## 1. The `gh`-first principle (why there is almost no adapter *code*)

- **GitHub is reached through the `gh` CLI** (+ `gh api` / `gh api graphql` for surfaces
  without a first-class subcommand: Projects v2, sub-issues, issue advanced search). `gh` is
  mature, ubiquitous, and already authenticated in the worker's environment. So the "adapter"
  is this **documented contract**, not a code layer — a worker gets the full GitHub surface
  with zero new code, exactly as the Azure backend gets it through the MCP.
- **NOT the GitHub MCP, NOT `/create-pr`.** The GitHub MCP needs an interactive OAuth flow that
  a headless `claude -p` worker cannot run; `gh` with a token env is auth-agnostic. And
  `/create-pr` carries its own review-chain + branch logic that collides with the delivery
  loop — the team is **delivery-native** on `gh`, same stance as the Azure backend is
  delivery-native on the MCP (not `/create-pr`).
- **The Go orchestrator (`atl work dispatch`) never calls GitHub.** Same as the Azure backend:
  it is a deterministic, zero-LLM, zero-provider supervisor that spawns workers and observes
  `status.json`. Every GitHub call originates from an LLM caller (a ceremony or a worker).
- **Auth = a token from the environment, never a file, never the argv.** `gh` reads `GH_TOKEN`
  (or `GITHUB_TOKEN`) from the environment; `atl work dispatch` sets it via `WorkerSpec.ExtraEnv`
  (the GitHub twin of the Azure PAT injection), so the worker never sees a literal secret and
  never logs one. The token needs `repo` + `project` scope (issues/PRs/labels + Projects v2).

## 2. Operation → `gh` map

| Operation (interface concept) | `gh` binding |
|---|---|
| Create a work-item (#1) | `gh issue create --title … --body … [--label type:<t>]` (+ `gh` REST `POST …/issues/{parent}/sub_issues` to nest) |
| Create child / nest under a parent (#1) | REST `sub_issues` endpoint: `gh api --method POST repos/{o}/{r}/issues/{parent}/sub_issues -F sub_issue_id=<child REST id>` (note: `gh issue create --parent` is NOT available in current `gh`) |
| Read one / a batch of work-items (#1) | `gh issue view <n> --json …` / `gh api graphql` for a batch |
| Read a sprint's items (#6) | scrum: `gh project item-list <n> --owner <o> --format json` filtered by the Iteration field client-side · flow: `gh search issues 'label:sprint:<n>' --repo <o>/<r>` — server-side, no client-side filter needed (§5) |
| Update fields (state, iteration, labels…) (#4/#5/#6/#7) | `gh issue edit` (labels/body/state) + `gh api graphql updateProjectV2ItemFieldValue` (project fields: Status/Priority always; Iteration + `Story Points` only under `mode: "scrum"`) |
| The ready-to-pull / idempotency / velocity query (#10) | `gh search issues` / `gh api graphql search(type:ISSUE, query:…)` — **server-side** filtering by label/state/type/assignee |
| Add / read the analysis / brief comment (#3) | `gh issue comment <n> --body …` / `gh api …/issues/{n}/comments` (sentinel-matched) |
| Link a work-item ↔ a PR (#11) | native: `Fixes #N` in the PR body + `PullRequest.closingIssuesReferences` (GraphQL) |
| Record a dependency edge (#8) | the **`## Depends On` convention** (§8) — GitHub has no native typed dependency link |
| Iteration/sprint membership (#6) | scrum: the Projects v2 **Iteration** field on the item (idempotent field set) · flow: the `sprint:<n>` **label** for the resolved ordinal `<n>` — `gh issue edit <issue#> --add-label sprint:<n> [--remove-label sprint:<prior>]`, the add and the carryover swap in one call (§5) |
| Open / review / merge a PR (#11) | `gh pr create` / `gh pr review` + `gh pr comment` / **`gh pr merge --merge`** (§10) |
| Read a branch's head commit (#16) | `gh api repos/{o}/{r}/commits/<branch> --jq .sha` — the plain branch read. **The gate does not use this one:** it compares against the promotion PR's own `headRefOid` (row below), which is the commit `--match-head-commit` pins at merge (§10) |
| Promotion approval record (#16) | write (the PO): `gh pr comment <n> --body-file <file>` · read: `gh pr view <n> --json headRefOid,comments` (one call returns the record **and** the head to compare it against) |
| Verify the approval + merge `dev`→`release` (#16) | **`atl work promote`** — the deterministic gate. It issues the read above and, on an exact match, `gh pr merge … --match-head-commit <SHA>` (§10). A ceremony runs the command; it does **not** issue these `gh` calls itself |
| Durable-knowledge read/upsert (#9) | in-repo `/docs` via the Contents API: `gh api --method PUT repos/{o}/{r}/contents/{path}` / read via `gh api …/contents/{path}` (§9) |
| Discovery search | `gh search issues` / `gh search code` |
| Resolve repo / default branch / identity | `gh repo view --json …` / `gh api user` |
| Test-evidence attachment (#12) | comment image upload or a repo-committed artifact (§11) |

## 3. Resilience — secondary rate limits / backoff

Same principle as the interface's resilience policy: every write (and every read under
N-parallel load) wraps in exponential backoff + jitter, honours GitHub's `Retry-After` /
`X-RateLimit-*` headers, caps at ~5 attempts. GitHub's constraints:

- **Primary:** GraphQL = 5,000 points/hr; REST = 5,000 req/hr. **Secondary:** ~2,000 GraphQL
  points/min + a mutation burst cap — expected under `atl work dispatch`'s ~4–6 parallel workers.
- **First-line defense = write GitHub only at durable milestones** (the worker heartbeats to
  `status.json`, not to GitHub) — the tracker-agnostic engine already means no high-frequency
  tracker polling; backoff is the second line.
- A rate-limit that exhausts retries is **not a task failure**: pause the call, heartbeat
  `provider-degraded`, let the supervisor hold the durable-milestone write and retry.

## 4. Pagination — "list means all"

The interface's "never silently truncate" principle, per GitHub surface:

- **Projects v2 GraphQL** (`ProjectV2.items`) — cursor-paginated, `first: 100` max/page: **loop
  the connection to exhaustion** following `pageInfo.endCursor` until `hasNextPage` is false.
  There is **no server-side field filter** — page all items, filter client-side by Status/Iteration.
- **Issue advanced search** (`gh search issues` / GraphQL `search`) — server-side filtering
  (preferred for the ready-to-pull query, §10); page its cursor to exhaustion.
  **No `--state all` here.** Unlike `gh issue list`, `gh search issues` accepts only
  `{open|closed}` and **errors out** on `all` (`invalid argument "all" for "--state" flag`,
  verified on gh 2.86.0) — it returns both states by default, so the correct form simply omits
  the flag. Every `gh search issues` call in this file is written that way on purpose; do not
  "restore" a `--state all` for symmetry with `gh issue list`.
- Treat a partial page read as complete = a bug, identical to the Azure `wiql`-cap rule.

## 5. Idempotency — the load-bearing part

Same mechanism as the interface: **stamp + check-before-create**, GitHub-side as source of
truth (no local ledger).

- **Every created issue carries the deterministic key as a LABEL:** `atl-key:<hash>` where
  `hash = hash(parent-id + plan-ordinal)`, plus `atl-run:<ceremony>:<sprint>` provenance.
  Labels are free-form, zero-setup, and **queryable via issue advanced search** — the GitHub
  analogue of `System.Tags`.
- **Before any create, run a check-first search** for that `atl-key` label
  (`gh search issues 'label:atl-key:<hash>' --repo <o>/<r> --json number`): **found
  → reuse + update** (converge), **not-found → create-then-stamp**. A create colliding with an
  existing item is resolved to it, not surfaced as an error.
- Keys derive from stable `parent + ordinal` (not a per-run GUID) → convergent resume, not
  merely dedup-attempted. Label length limit (50 chars) bounds the hash — use a short digest.
- **Brainstorm-sourced items carry `atl-brainstorm:<slug>`, not `atl-key`.** `/brainstorm done`'s
  board-sync creates deferred backlog issues with an `atl-brainstorm:<brainstorm-slug>` label (its own
  provenance key — a brainstorm item has no parent/plan-ordinal, so no `atl-key`), and dedups its own
  re-runs by a check-first `gh search issues 'label:atl-brainstorm:<slug>'` plus the item title. When
  `/refine` later plans a unit that IS such an issue, its `atl-key` search misses it; before creating,
  it searches `label:atl-brainstorm:<slug>` for the in-scope item and, on a title match, **adopts** it
  — `gh issue edit` in place + stamp the computed `atl-key:<hash>` label — instead of opening a
  duplicate. After adoption it converges via the normal `atl-key` search. (50-char label limit — use a
  short slug/digest, as with `atl-key`.)
- **`/request` candidate issues carry `atl-request:<slug>:<initiator>`, not `atl-key`** (concept #14).
  `/request`'s capture step opens a candidate issue with an `atl-request:<request-slug>:<initiator>`
  label + `candidate` + `triage:<tier>` labels + Projects v2 **Status = `candidate`**, and dedups its
  own re-runs by a check-first `gh search issues 'label:atl-request:<slug>' --repo <o>/<r>`
  plus the item title. A `candidate`-Status issue is **excluded from the ready-to-pull / selection
  query** (concept #13) until the PO accepts it; on accept, `/request` flips the Status off `candidate`
  (the issue enters the frontier) and `/refine` materializes PBIs with their own `atl-key`. (50-char
  label limit — short slug/digest, as with `atl-key`.)
- **Under `mode: "flow"` the sprint carrier is a `sprint:<n>` LABEL, not the Iteration field**
  (concept #6). `<n>` is the sprint's ordinal (`sprint:[0-9]+`, unpadded), **resolved not invented**:
  read the labels the **items actually carry** (`gh issue list --repo <o>/<r> --state all
  --limit 500 --json number,labels`, raised until the result is complete per §4 — `gh issue list`
  does accept `--state all`, unlike `gh search issues`), keep the ones matching `sprint:[0-9]+`,
  and take the highest ordinal
  `k` **compared as an integer** (`sprint:10` outranks `sprint:9`; a lexical "highest" hands back a
  stale ordinal and the next sprint reuses a number already in use). `sprint:<k>` is the **current**
  sprint and stays current until it is *reviewed*: admit into **`sprint:<k>`** while its
  `docs/sprints/sprint-<k>-review.md` page (§9) is absent, and open **`sprint:<k+1>`** only once that
  page exists — advancing on the highest ordinal *alone* would open a fresh sprint on every re-plan,
  defeating the convergence claimed immediately below. No `sprint:` label at all means `sprint:1`,
  **except on a project migrating from scrum**: it keeps its existing numbering rather than
  restarting, so take the highest ordinal `m` among the existing `docs/sprints/sprint-<m>-review.md`
  pages (§9) and open `sprint:<m+1>` — otherwise the first flow sprint upserts `sprint-1-review.md`
  straight over the scrum sprint-1 page. `/delivery-init` documents the scrum→flow switch and
  migrates nothing, so this board state is reachable, not hypothetical.
  > **Read the items, not the label registry.** `gh label list` returns the repo's label
  > *definitions*, which outlive the issues that carried them (an issue deleted or transferred, a
  > hand-created label, a template label set) — a strict superset. So a carrier-less `sprint:7`
  > definition sitting next to a genuinely in-flight `sprint:6` would resolve `k=7`, find no
  > sprint-7 review page, admit into 7, and then review `label:sprint:7` — silently excluding every
  > sprint-6 unit, while Azure resolved 6 and carried on. The Azure adapter closes the same
  > ambiguity from the other side ("a WIQL returns work-items, not a tag list"); this is that rule
  > stated for GitHub.
  Admission is `gh issue edit <issue#> --add-label sprint:<n>` for the resolved `<n>`, **idempotent**
  (adding a label the issue already carries is a no-op, so a re-planned or crash-resumed sprint
  converges — never model it as a create-membership). **At most ONE `sprint:` label per issue:**
  re-admitting a carryover passes `--remove-label sprint:<prior> --add-label sprint:<n>` — `<prior>`
  being the one `sprint:` label it already carries — in the **same** `gh issue edit`, since a label
  accumulates where a field replaces. The label is never removed to mean "done" — that is `gh issue
  close` (§6). A sprint's items are then a server-side read,
  `gh search issues 'label:sprint:<n>' --repo <o>/<r>` (§2) — cheaper and more exact than
  the Projects v2 page-then-filter the Iteration field forces. Under `mode: "scrum"` none of this
  applies: the Iteration field remains the carrier and no `sprint:` label is written.

## 6. State & completion — one fixed model (no runtime template resolution)

Unlike Azure (where the Completed category is resolved per process-template at runtime), GitHub
has **one model**, so "resolution" collapses to it:

- **"Done" = the issue is CLOSED** (+ the Projects v2 **Status** field set to its **Done**
  category). The tech-lead closes the issue on merge-verify (§10). Under `mode: "scrum"`, velocity
  sums the `Story Points` field over closed issues in the last N sprints; a `flow` project has no
  `Story Points` field and `/sprint-review` reports no velocity — completion itself is unchanged.
- **Claim = set Status to In Progress** (+ optionally self-assign).
- **"Blocked" is a FLAG, never a state:** add a `blocked` **label** + a diagnostic comment,
  leaving the issue open. Same discipline as the Azure `blocked` tag — never invent a blocked
  status to transition to.
- **"Candidate" is a Projects v2 Status option (board-setup, NOT auto-created)** (concept #13). A
  `/request` candidate sets **Status = `candidate`** on the issue's project item + a `candidate` label
  + a `triage:<tier>` label. Projects v2 Status **options are not cleanly API-settable** (see
  `/delivery-init` + the Iteration field), so the `candidate` option is added to the Status field **via
  the Projects settings UI** (Projects v2 → ⚙ → Status → new option), exactly like Iteration. Until
  accepted, a `candidate`-Status issue is **excluded from the ready-to-pull query** (§4 / concept #10);
  accept flips the Status off `candidate` (e.g. to Todo). (Azure needs no such setup — its `candidate`
  tag is zero-setup.)

## 7. Content-placement contract (deterministic read-back)

Identical discipline to the interface (#2/#3), GitHub binding:

- **Business analysis** (`business-analyst`) → the **issue body** (Markdown) under fixed H2s:
  `## Problem`, `## Business Value`, `## Scope`, `## Acceptance Criteria`, `## Out of Scope`.
- **Technical analysis** (`technical-analyst`) → a **single issue comment** whose first line is
  the exact sentinel `**[Technical Analysis]**`, then the fixed H2s.
- **Canonical brief** (`tech-lead`) → a **single issue comment**, first line `**[Canonical
  Brief]**`, then `## Goal`, `## Area`, `## Load These Pages`, `## Depends On`, `## Evidence
  Before Review` — it names the area pack and embeds the exact `/docs` paths for the task.
- **Request decision** (`/request` gate) → a **single issue comment** on the candidate, first line
  `**[Request Decision]**` (concept #15), then `## Recommendation` (the `YES`/`NO`/`DEFER`/`NEEDS-INFO`
  verdict), `## Deliberation` (thesis / the team's anti-thesis (refute-to-keep) / the surviving
  position), `## PO Decision` (accept/reject/defer + the convergence mechanism: concession /
  judgment-call standoff / human-authority lock), `## Dissent On Record` (the team's preserved dissent
  under standoff/lock; empty on a merit-win).
- **Promotion approval** (the human PO, at `/sprint-review`'s gate) → a **PR comment on the
  `dev`→`release` promotion PR**, first line the exact sentinel `**[Promotion Approval]**`
  (concept #16), then `## Approved Commit` (a 40-character lowercase hex commit id — **the only
  load-bearing section**), `## Sprint` (`Sprint <n> · <iteration-name>` under `mode: "scrum"`; just
  `Sprint <n>` under `flow`, which has no iteration to name), `## Decision`
  (`APPROVE`). Everything else in the body is audit context. Posted with
  `gh pr comment <n> --body-file <file>`, never `--body`: `atl guard` scans the whole Bash
  command string, so a multi-line record belongs in a file, not in the argv.
- **Area** → an `area:<name>` **label**, applied by the tech-lead at decomposition.
- **Read-back** = `gh issue view` (parse body headings) + list comments filtered to the one
  starting with its sentinel — a **sentinel match, not "the newest comment"**, so a later human
  comment never shadows it. For a decomposed child unit (a sub-issue) with no
  `**[Technical Analysis]**` of its own (only the tech-lead's `**[Canonical Brief]**`), read the
  analysis from its **nearest ancestor** Feature — resolve the parent via **GraphQL `Issue.parent`**
  (`gh api graphql` — `{ repository(owner,name){ issue(number:N){ parent { number } } } }`; the REST
  `sub_issues` endpoint lists an issue's *children*, not its parent), climbing parent links until you
  reach the Feature that bears a `**[Technical Analysis]**` (concept #1).
- **The approval record is the one exception to that read-back rule** — its channel is
  **append-and-supersede**, so converging on a single comment is wrong here. **`atl work promote`
  implements this read; no ceremony performs it** (§10). It collects **every** comment whose first
  line is exactly `**[Promotion Approval]**`; in each, it takes the first `[0-9a-f]{40}` token on a
  line after `## Approved Commit`. Selection is then **by commit id, never by recency**: a record
  counts as an approval only if the commit it names equals the promotion PR's current `headRefOid`.
  A newer record supersedes an older one *by naming a newer commit*, not by being newer, and a
  record naming any other commit is not an approval and is never carried forward. Records are never
  edited or deleted — stale ones stay as audit history. The format is documented here because the
  **PO** writes it by hand; reading it back is the command's job.

## 8. Dependency links — the `## Depends On` convention

GitHub has **no native typed work-item dependency** (interface #8 gap). The scheduling DAG is
carried by a **convention the ceremony reads**, not a platform primitive:

- The tech-lead records each real prerequisite in the work-unit's **Canonical Brief comment**
  under a `## Depends On` section, one predecessor per line as `#<N>` (the issue number of the
  prerequisite unit). This is the durable, sentinel-located source of truth for the edge.
- `/sprint-start` builds `plan.json` by reading each admitted issue's `## Depends On` list
  (restricted to this sprint's units; an edge to an already-closed unit is dropped, an edge to
  an out-of-sprint open unit makes the dependent un-startable and is surfaced) — exactly the
  Azure flow, only the edge's storage differs (a brief line vs a typed link). The Go engine
  topo-sorts the resulting flat DAG unchanged.
- (GitHub's native "blocked by" task-list relations are weaker and not uniformly queryable; the
  `## Depends On` brief line is the single authoritative form.)

## 9. Durable-knowledge store — in-repo `/docs` (namespaced, single-owner)

The interface's durable-knowledge concept (#9), bound to **in-repo `/docs` Markdown** (the
repo Wiki has no list/create API — avoid it). Diffable, PR-reviewable, and API-writable via the
Contents API — it behaves as an API-first project wiki that rides the same PR flow.

| Namespace (path under `/docs`) | Content | Owner |
|---|---|---|
| `docs/domain/` | glossary, entities, business rules | business-analyst |
| `docs/analysis/` | per-Epic/Feature deep analysis | business-analyst + technical-analyst |
| `docs/architecture/` | system shape, module boundaries, stack decisions | tech-lead |
| `docs/architecture/adr/adr-<n>-<slug>.md` | one file per architecture decision | tech-lead |
| `docs/conventions/` | project conventions atop the pack's generic ones | tech-lead |
| `docs/sprints/sprint-<n>-review.md` | sprint-review outcome pages | `/sprint-review` |

- **Workers do NOT write `/docs`** — their role-craft learnings route to their agent
  `children/` via `/drain`; project-specific facts are promoted to `/docs` by the tech-lead.
  Single-owner-per-namespace, no N-worker write races.
- **Read:** the canonical brief embeds the relevant `/docs` paths; the worker reads them via
  `gh api …/contents/{path}` (or plain file read in its worktree — the repo is checked out).
- **Write mechanics:** `PUT …/contents/{path}` is an idempotent upsert and — unlike the Azure
  wiki — **creates ancestors implicitly** (a nested file path just works; no
  `WikiAncestorPageNotFoundException`, no parent-first dance). A `/docs` write can also ride a
  normal PR (diffable + reviewable), which the Azure wiki cannot.

## 10. PR + review + merge — the completion gate

The interface's PR concept (#11), bound to `gh`, honouring the D3 decision:

- **Open:** the developer opens a PR to `dev` with `gh pr create --base dev`, references the
  unit as `Fixes #N` in the body (traceability; note the auto-close caveat below), and its job
  ends at the open PR.
- **Review:** the tech-lead reviews **on the PR** — `gh pr diff`, findings as `gh pr comment` /
  `gh pr review`, never `/create-pr`.
- **Merge = `gh pr merge --merge` ONLY** (a real merge commit). **Never `--squash`/`--rebase`** —
  they rewrite the SHA and false-block the engine's `MergedToBase` (`rev-list origin/dev..branch
  == 0`), the exact GitHub twin of Azure's NoFastForward requirement. **Repo prerequisite:**
  "Allow merge commits" enabled on the repo — `/delivery-init` preflights this (§3B.2:
  `gh api repos/<owner>/<repo> --jq '.allow_merge_commit'`) and warns if it is disabled, but a
  repo switched to squash-only *after* init still needs it re-enabled.
- **Issue completion:** `Fixes #N` auto-closes an issue **only on merge to the DEFAULT branch**,
  but the flow merges to `dev` → so on merge-verify the tech-lead explicitly `gh issue close #N`
  + sets the Projects Status to Done. Do not rely on auto-close for the `dev` merge.
- **NEVER-merge carve-out (D3):** the autonomous tech-lead **worker** performs this green
  feature→`dev` merge — that is the machine's job and the loop breaks without it. The human PO
  reviews only at sprint review (`dev`→`release` promotion). The carve-out is scoped to the
  machine, never the interactive session.
- **The `dev`→`release` promotion is SHA-pinned (#16), and it is `atl work promote` that does it —
  not a ceremony issuing `gh` by hand.** **Opening** the promotion PR is not the promotion —
  **merging** it is, and the merge the command issues is
  `gh pr merge <PR#> --repo <o>/<r> --merge --match-head-commit <APPROVED_SHA>`, where
  `<APPROVED_SHA>` is the commit named in the verified `**[Promotion Approval]**` record (§7).
  `--match-head-commit` is what makes the binding real: **GitHub itself** refuses the merge unless
  the PR head is still that exact commit, so the verify→merge window is closed by the provider
  rather than by how fast the caller acts — if `dev` advances between the comparison and the
  merge, the merge fails instead of promoting a commit nobody approved. It is a second,
  provider-enforced line of defence behind the command's own comparison, not a replacement for
  it. `--merge` stays mandatory here too — never `--squash`/`--rebase`.
  **Why a command and not an instruction:** the comparison shipped as `/sprint-review` prose first,
  in a spec that contradicted itself — a blocking Approve/Reject question ordered *before* the check
  by one paragraph, and conversational approval forbidden outright by another. The command
  verifies **and** merges in one call, so there is no separate merge step a caller can reach without
  the check — which is also why this row is the whole binding: reproducing the read+compare+merge
  sequence in a ceremony re-creates the skippable path.

## 11. Test-evidence attachment

The interface's evidence concept (#12): the tester attaches verifiable proof
(screenshots/results) the tech-lead reads back before weighing the diff; an un-runnable surface
is UNVERIFIED → block, never fake-green.

- **Bind:** commit the evidence artifact into the PR under `docs/sprints/evidence/<unit>/…` (it
  rides the PR, is diffable, and the tech-lead reads it in the diff), **or** upload an image to
  the review comment. Prefer the committed artifact — it is durable and review-visible without a
  separate upload API.
- Missing required evidence (e.g. the mobile-emulator screenshot when the area demands it) =
  **NOT green**, full stop.

## 12. Mockability & testing

- **Layer-B (real, the load-bearing proof):** a real `claude -p` worker doing a full single-task
  micro-lifecycle against a **real GitHub test repo** — claim → comment → open PR → merge to
  `dev` → close issue — driven by `GH_TOKEN` in its env. This is the GitHub twin of the Azure
  Layer-B and the **T-point signal** (D6): the first autonomous GitHub sprint on ATL's real
  backlog closing green. It runs locally / in CI where a token is available.
- **Layer-A:** `gh` against a fixture repo (or a recorded-response shim) for fast, always-on
  ceremony tests — the same "mock the surface, not fork the contract" discipline as the Azure
  Layer-A mock.
