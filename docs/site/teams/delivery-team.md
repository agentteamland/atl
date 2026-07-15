# delivery-team

**delivery-team** is an **Azure DevOps work-item-driven, sprint-based autonomous software-delivery
org** — a team of role-agents that plan, decompose, build, verify, review, and ship work items
against a real Azure DevOps project, with a human as the Product Owner. It is a **project-scope**
team: it installs into the repository it delivers.

```bash
atl install agentteamland/delivery-team
```

Installing lands the role-agents, the ceremony skills, the knowledge packs, and the Azure
operation-contract into the project's `.claude/`, plus a `.delivery/` config the orchestration
engine reads.

## The org

delivery-team is a set of **role-agents**, each a specialist with its own reflexes:

| Role | What it does |
|---|---|
| `intake` | Triages a raw request into a shaped Epic/Feature backlog item. |
| `business-analyst` | Writes the business analysis — the Description's `## Problem / Business Value / Scope / Acceptance Criteria / Out of Scope`. |
| `technical-analyst` | Writes the `**[Technical Analysis]**` sentinel comment — approach, feasibility, NFRs, dependencies, suggested areas. |
| `project-manager` | Runs the sprint cadence — capacity, iteration assignment, velocity. |
| `tech-lead` | Decomposes Features into work-units, writes each unit's `**[Canonical Brief]**`, owns the project wiki (`Architecture/`, `Conventions/`, ADRs), and is the **single review gate** — reviews each PR and, on green, completes it (= merge) and sets Done. |
| `tester` | Independent Level-2 verification — re-derives intent, runs the test-gates on the right surface, attaches evidence, emits a verdict. |
| `developer` | A dynamic, stack-agnostic worker spawned per work-unit; loads the tagged `area:<name>` knowledge-pack and implements the unit. |

A **software team** for a specific stack is just a set of area-keyed knowledge packs
(`packs/<area>/`) the generic `developer` loads — the M1 "knowledge-as-data" seam, so a React or a
.NET team plugs in without a new agent.

## The ceremonies

The sprint runs through skills you invoke, each acting as the right role:

```bash
/delivery-init      # wire the project to its Azure org/project/repo + methodology
/kickoff            # intake + business-analyst shape the Epic/Feature backlog
/refine             # technical-analyst + tech-lead decompose Features into briefed work-units
/sprint-plan        # project-manager selects the sprint's units against capacity
/sprint-start       # materialize the work-unit DAG → hand it to the engine
/sprint-review      # velocity, the review outcome wiki page, sprint close
```

Methodology is **config, not code**: `methodology.json` (Scrum in v1) declares the cadence the
ceremonies read — no workflow engine to maintain.

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
`/sprint-review` reflects back to Azure (the `blocked` tag + a diagnostic comment) and clears. Each
worker reaches Azure only through the project-scoped `azureDevOps`
MCP the engine wires it, so a worker can never touch anything but the configured test/target project.

## Azure is the single source of truth

There is no local database. **Work-items are the transient execution state** and the **project wiki is
the durable knowledge** (the ATL wiki/journal split, living in Azure). Every role reaches Azure through
one documented **MCP-first operation-contract** (`knowledge/azure-adapter.md`) — `wit_*` for work-items,
`repo_*` for PRs, `wiki_*` for knowledge — with a thin REST carve-out for the one operation the MCP
lacks (evidence attachment). Content is placed by **machine-locatable sentinels**: the business analysis
in the Description, the `**[Technical Analysis]**` and `**[Canonical Brief]**` comments each matched by
their exact first line (never "the newest comment"), and area binding by a `System.Tags: area:<name>`.

## Shipping the work — the two-branch flow

Work integrates to **`dev`** (the tech-lead completes each unit's PR on green — the scoped exception to
the platform's never-merge rule), and the Product Owner promotes an approved sprint from `dev` to
**`release`**. Review is **delivery-native**: the tech-lead runs the adversarial review pattern
(evidence gate + refute-to-keep) directly on the Azure PR via `repo_*` threads and vote — not
`/create-pr`.

## What ships

The full role-agent org, the five ceremony skills, the `atl work dispatch` engine, the Azure
operation-contract, a Scrum `methodology.json`, and a three-area reference pack (web / mobile / api).
The autonomous developer→tester→tech-lead loop is proven end-to-end against a live Azure DevOps project.

Deferred (design captured, trigger-gated): **multi-methodology** support beyond Scrum, **stack-specific
override** of the generic developer, **dynamic-capacity** concurrency, a **hotfix flow**, and
**device-farm** emulators. The **mobile-emulator** test lane is built but its live validation is gated on
a desktop (GUI) session.

## See also

- [`atl install`](/cli/install) — how a team resolves and installs
- [Teams](/teams/) — the catalog and the first-party rebuild
- [Concepts: scope](/guide/concepts#scope-global-and-project) — project vs. global teams
