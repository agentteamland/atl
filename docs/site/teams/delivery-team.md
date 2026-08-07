# delivery-team

**delivery-team** is a **work-item-driven, sprint-based autonomous software-delivery org on a
pluggable backend (Azure DevOps or GitHub)** — a team of role-agents that plan, decompose, build,
verify, review, and ship work items against a real tracker (Azure Boards + Repos, or GitHub Issues +
Projects + Pull Requests), with a human as the Product Owner. It is a **project-scope** team: it
installs into the repository it delivers.

```bash
atl install agentteamland/delivery-team
```

Installing lands the role-agents, the ceremony skills, the knowledge packs, and both backend adapter
packs (`backends/azure/`, `backends/github/`) into the project's `.claude/`; running `/delivery-init`
then writes the `.delivery/` config the ceremonies and the orchestration engine read.

## The org

delivery-team is a set of **role-agents**, each a specialist with its own reflexes:

| Role | What it does |
|---|---|
| `intake` | Triages a raw request into a shaped Epic/Feature backlog item. |
| `business-analyst` | Writes the business analysis — the Description's `## Problem / Business Value / Scope / Acceptance Criteria / Out of Scope`. |
| `technical-analyst` | Writes the `**[Technical Analysis]**` sentinel comment — approach, feasibility, NFRs, dependencies, suggested areas. |
| `project-manager` | Runs the sprint cadence — admits the sprint's units, stamps their sprint carrier, and writes the review report. Capacity and velocity only under the `scrum` mode. |
| `tech-lead` | Decomposes Features into work-units, writes each unit's `**[Canonical Brief]**`, owns the project wiki (`Architecture/`, `Conventions/`, ADRs), and is the **single review gate** — reviews each PR and, on green, completes it (= merge) and sets Done. |
| `tester` | Independent Level-2 verification — re-derives intent, runs the test-gates on the right surface, attaches evidence, emits a verdict. |
| `developer` | A dynamic, stack-agnostic worker spawned per work-unit; loads the tagged `area:<name>` knowledge-pack and implements the unit. |

A **software team** for a specific stack is just a set of area-keyed knowledge packs
(`packs/<area>/`) the generic `developer` loads — the M1 "knowledge-as-data" seam, so a React or a
.NET team plugs in without a new agent.

## The ceremonies

The sprint runs through skills you invoke, each acting as the right role:

```bash
/delivery-init      # select the backend (azure | github) + the sprint mode + wire the coordinates + methodology
/kickoff            # intake + business-analyst shape the Epic/Feature backlog
/refine             # technical-analyst + tech-lead decompose Features into briefed work-units
/sprint-plan        # project-manager selects the sprint's units — against capacity, or by priority with the set kept DAG-closed
/sprint-start       # materialize the work-unit DAG → hand it to the engine
/sprint-review      # the review outcome page, sprint close (+ velocity under the scrum mode)
/request            # (any time) mid-project request → triage → feasibility → honest PO gate → accept/defer/reject
```

Methodology is **config, not code**: `methodology.json` (Scrum in v1) declares the cadence the
ceremonies read — no workflow engine to maintain.

### Two sprint modes — `scrum` and `flow`

`methodology.json` carries a `mode`, and it changes exactly two things. **Absent means `scrum`**, so
an existing project's behaviour never shifts underneath it.

| | `mode: "scrum"` | `mode: "flow"` |
|---|---|---|
| What a sprint is | a **time-box** — a dated range on the backend's schedule, with a capacity ceiling | the same sprint, **with no dates and no capacity ceiling** |
| `/sprint-plan` admits | by priority, up to a velocity-derived point budget | by **priority**, with the admitted set kept **DAG-closed** — no admission ceiling of any kind |
| `/sprint-review` reports | actual velocity | **no velocity** — completed vs. carryover is the whole account |
| `capacityModel` | **required** | **absent** — the key is omitted |
| The sprint carrier | the backend's **iteration field** | a **`sprint:<n>` label** on each admitted unit |

**DAG-closed** means a unit is never admitted without the predecessors it depends on: they come in
with it, or they are already complete. A `flow` sprint therefore carries real dependency edges for
`/sprint-start` to order, rather than only the units that could start today — and the ~4–6
concurrency cap bounds how many units the engine runs **at once**, never how many the ceremony may
admit.

Everything else is identical: the same ceremonies, the same roles, the same DAG, the same promotion
gate. **The vocabulary does not change either — a sprint is a sprint in both modes.**

Pick `flow` when the project has **no stable capacity to plan against** — velocity over a fixed
period is a mean of something that has to be stable to mean anything, and a solo maintainer working
with an autonomous agent has no such number (one session ships eight units, the next ships zero). A
team that does have one keeps `scrum`, unchanged.

Because a label carries the sprint, a `flow` project needs **neither the `Iteration` nor the
`Story Points` board field** — which on GitHub removes the one setup step `gh` cannot automate (a
Projects v2 Iteration field has to be added by hand in the settings UI). `Status` and `Priority` are
still needed, and `/delivery-init` creates or verifies both for you.

## The engine — `atl work dispatch`

`/sprint-start` materializes the selected units into a `.delivery/plan.json` dependency DAG, then the
**deterministic Go engine** `atl work dispatch` takes over. It holds **zero LLM context and makes
zero Azure calls**: it admits ready units up to a concurrency cap, and for each spawns a **pipeline of
three isolated `claude -p` workers in one git worktree** —

```
developer  →  tester  →  tech-lead
(implement    (Level-2     (review → vote →
 + open PR)    verify)      complete PR = merge to dev → Done)
```

The engine advances a stage on a worker's clean exit, verifies the tech-lead's merge landed on `dev`
by a pure git read (never trusting a worker's exit code), reclaims the worktree, and refills the DAG.
A stalled or crashed worker is reclaimed and retried once, then mark-blocked — a durable report that
`/sprint-review` reflects back to the backend (the `blocked` tag or label + a diagnostic comment) and
clears. Each worker reaches the tracker only through what the engine wires it — the project-scoped
`azureDevOps` MCP on the Azure backend, or the `gh` CLI with an engine-injected `GH_TOKEN` (resolved
from `config.credential.ref`) on the GitHub backend — never the operator's ambient MCP config or
credentials.

## The backend is the single source of truth

There is no local database. **Work-items are the transient execution state** and the **durable-knowledge
store holds the durable knowledge** (the ATL wiki/journal split, living in the backend: the project wiki
on Azure, an in-repo `docs/` tree on GitHub). Every role reaches the backend through one documented
**provider-neutral operation-contract** (`knowledge/backend-interface.md`), bound per provider by an
adapter pack — `backends/azure/adapter.md` (the `azureDevOps` MCP: `wit_*` for work-items, `repo_*` for
PRs, `wiki_*` for knowledge, with a thin REST carve-out for the one operation the MCP lacks, evidence
attachment) or `backends/github/adapter.md` (the `gh` CLI: Issues, Projects v2, Pull Requests, and the
in-repo `docs/` store). Content is placed by **machine-locatable sentinels**: the business analysis
in the Description, the `**[Technical Analysis]**` and `**[Canonical Brief]**` comments each matched by
their exact first line (never "the newest comment"), and area binding by a `System.Tags: area:<name>`.

## Shipping the work — the two-branch flow

Work integrates to **`dev`** (the tech-lead completes each unit's PR on green — the scoped exception to
the platform's never-merge rule), and the Product Owner promotes an approved sprint from `dev` to
**`release`** — never on a conversational approval, but on a commit-bound approval record that
`atl work promote` reads back from the backend and merges against (the gate below; v1 binds it on
**GitHub**, and **holds** the promotion on Azure until the missing read is bound there). Review is
**delivery-native**: the tech-lead runs the
adversarial review pattern (evidence gate + refute-to-keep) directly on the backend's PR — `repo_*`
threads and vote on Azure, `gh pr comment` / `gh pr review` on GitHub — not `/create-pr`.

## The promotion gate — a commit-bound approval

`dev` → `release` is the one irreversible step, so it is not granted in conversation. `/sprint-review`
compiles the report, opens (or finds) the `dev` → `release` **promotion PR**, and then hands the decision
to **`atl work promote`** — a deterministic command that reads a durable **approval record** back from the
backend, compares it to the PR's current head, and merges only on an exact match. Opening that PR is no
longer the promotion — **merging it is**. In v1 the gate is bound on the **GitHub** backend only — see
**GitHub-only in v1** at the end of this section for what is still missing on Azure.

**The check is code, not ceremony instructions, and that is the point.** It first shipped as a procedure
written into the `/sprint-review` skill, and a real run had the same ceremony follow it on one turn and
silently skip it on the next — falling back to asking "Approve or Reject?" in chat, the exact gate the
design removes. A step that depends on being remembered eventually isn't, and the failure is quiet. So the
command **verifies and merges in one call**: there is no separate merge for a ceremony to reach without the
check, and the ceremony's job is reduced to running it and relaying the verdict.

To approve, the Product Owner adds a comment on the promotion PR whose **first line is exactly**
`**[Promotion Approval]**` and which names the commit being approved:

```markdown
**[Promotion Approval]**

## Approved Commit
<40-character lowercase hex commit id>

## Sprint
Sprint <n> · <iteration-name>

## Decision
APPROVE
```

(Under `mode: "flow"` there is no iteration to name, so the `## Sprint` line is just `Sprint <n>`.)

Only `## Approved Commit` is load-bearing — the rest is audit context. Paste it into the PR's comment box,
or post it from the CLI:

```bash
gh pr comment <PR#> --repo <owner>/<repo> --body-file approval.md
```

That is the whole of the PO's job. `atl work promote` then reads the PR's head and every record on it in
one call, and merges only when a record names that exact head — SHA-pinned
(`gh pr merge --merge --match-head-commit <approved-commit>`), so the provider itself refuses the merge if
the head moved in between. The outcome is recorded on the sprint's review page under
`## Promotion decision`: approved commit, approver, timestamp, PR link. Every other outcome is a **HOLD** —
the command exits non-zero, nothing merges, no work-item changes state, and the message tells you exactly
what to set:

| What the gate finds | What it does (`reason`) |
|---|---|
| No `**[Promotion Approval]**` record on the PR | Holds, and prints the PR link + the head commit to approve (`no-record`). |
| A record with no 40-hex id under `## Approved Commit` | Holds, and asks for a re-post naming the current head (`unparseable-record`). |
| A record naming a commit that is not the PR's head | Holds — the approved state is not the state that would ship (`superseded`). |
| The record could not be read | Holds. Unverified is not approved (`read-failed`). |
| The record matched, but GitHub refused the pinned merge | Holds (`merge-refused`) — `dev` moved between the check and the merge, so the provider rejected it. Nothing was promoted; approve the new head. |
| No open `dev` → `release` PR to act on | Holds — open it first (`no-open-pr`). This is also how an already-promoted sprint converges: a re-run merges nothing. |
| The command cannot reach the backend | Holds (`backend-unbound`) — see **GitHub-only in v1** below. |

A hold is not a rejection: nothing is closed, nothing is tagged carryover, and the record is left where it
is. An explicit **reject** stays conversational and runs the existing carryover path — the gate protects
only the irreversible direction, and declining a promotion cannot over-ship.

**When `dev` advances after an approval, the approval expires with it.** The gate reports the approved
commit and the current head, and it does **not** carry the approval forward. To promote the current state,
re-read the refreshed report and add a new record for the new head; to promote exactly what you approved,
reset `dev` back to that commit first. The stale record stays in place as audit history — the channel is
append-and-supersede, and the next record supersedes it by naming a newer commit.

Two limits, stated plainly:

**Checkable, not unforgeable.** In an interactive session the ceremony holds the same credential as the PO,
so an author check cannot tell a ceremony-written record from a PO-written one. What the gate buys is that a
promotion is commit-scoped (it can never silently ship a state later than the one that was reviewed) and
attributable (a durable record naming a commit, an author, and a timestamp) — a correctness gate, not an
authentication one.

**GitHub-only in v1 — and the reason is the transport, not the data.** On Azure both of the gate's reads
are bound in the adapter: the approval record via PR threads, and the head commit via the branch read
(`repo_branch`, `action: "get"`), whose response carries it in `objectId` — resolved against a live server,
so it is written down. But those are MCP tools, callable only from an LLM turn, and `atl work promote` is a
Go binary with no MCP client. It can issue neither read, so **the commit-bound gate does not operate on the
Azure backend yet.**

That is a **HOLD, not a fallback** — a read the command cannot issue is a read failure, so on Azure
`/sprint-review` compiles the report, opens (or finds) the promotion PR, and then holds: `atl work promote`
reports `backend-unbound`, calls no Azure surface at all, and merges nothing. It does **not** revert to
promoting on a conversational approval, and the ceremony does not read the head itself and promote on it
either — a head the caller supplies is a head the caller can get wrong, which is the prose gate this
replaced. Until a transport exists, an Azure project on this version promotes by completing that promotion
PR itself, in Azure DevOps. Choosing how a deterministic Go gate should reach Azure — a team-owned REST
helper like the attachment carve-out, or an Azure client inside `atl` — is the named next step, and it is
an open design decision rather than a lookup.

## What ships

The full role-agent org, the six ceremony skills, the `atl work dispatch` engine, the provider-neutral
backend interface with Azure DevOps and GitHub adapter packs, a Scrum `methodology.json`, and a
four-area reference pack (web / mobile / api / go-cli).
The autonomous developer→tester→tech-lead loop is proven end-to-end against a live Azure DevOps project.

Deferred (design captured, trigger-gated): **multi-methodology** support beyond Scrum, **stack-specific
override** of the generic developer, **dynamic-capacity** concurrency, a **hotfix flow**, and
**device-farm** emulators. The **mobile-emulator** test lane is built but its live validation is gated on
a desktop (GUI) session.

## See also

- [`atl install`](/cli/install) — how a team resolves and installs
- [Teams](/teams/) — the catalog and the first-party rebuild
- [Concepts: scope](/guide/concepts#scope-global-and-project) — project vs. global teams
