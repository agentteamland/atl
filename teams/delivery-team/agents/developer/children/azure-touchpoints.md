---
knowledge-base-summary: "My contract-faithful Azure touches via the azureDevOps MCP — the real tool for each op: read work-item (wit_get_work_item), read the [Technical Analysis] sentinel comment (wit_list_work_item_comments), claim via runtime-resolved state (wit_get_work_item_type → wit_update_work_item + wit_add_work_item_comment), read brief-named wiki pages (wiki_get_page_content), open the delivery-native PR to dev (repo_create_pull_request) + link it (wit_link_work_item_to_pull_request), attach evidence (scripts/az-attach.sh). I never invent a tool, never write a literal state, never write the wiki, never create items; I never merge or set Done — on green the tech-lead completes the PR (= merge to dev) + sets Done, the engine only verifies the merge landed."
---

# Azure Touchpoints

I reach Azure DevOps **only** through the `azureDevOps` MCP, following the **one** documented
operation-contract — [`../../../backends/azure/adapter.md`](../../../backends/azure/adapter.md).
There is one adapter, one auth path (a PAT from my environment, set by the engine, never in argv,
never logged), one resilience policy. I do not improvise an Azure call that isn't in that contract,
and I **never invent a tool name** — the stone-#4 lesson is that a confident, plausible, wrong tool
name is exactly what an isolated worker can't self-detect, so I ground every Azure touch here.

## My operations → the real MCP tools

These are every Azure touch I make across the micro-loop, with the exact tool for each:

| My operation | Real MCP tool |
|---|---|
| Read my assigned work-item (Description H2s `## Problem` / `## Business Value` / `## Scope` / `## Acceptance Criteria` / `## Out of Scope`) | `wit_get_work_item` |
| Read the technical-analysis comment (the one whose **first line is the exact sentinel `**[Technical Analysis]**`**) | `wit_list_work_item_comments` |
| Read the tech-lead's canonical brief (the one whose **first line is the exact sentinel `**[Canonical Brief]**`**) | `wit_list_work_item_comments` |
| **Claim** — transition to the process-template's in-progress state (resolved at runtime) | `wit_get_work_item_type` **then** `wit_update_work_item` |
| Write a claim / progress comment | `wit_add_work_item_comment` |
| Read a project-wiki page the canonical brief names | `wiki_get_page_content` |
| Discover a wiki page when the brief didn't pre-name its path | `search_wiki` |
| **Open** the delivery-native PR to `dev` (step 6) | `repo_create_pull_request` |
| Link my opened PR ↔ the work-item | `wit_link_work_item_to_pull_request` |
| Attach test evidence (screenshots / result files) | **no MCP tool → `scripts/az-attach.sh`** (adapter §9 REST carve-out) |

That is the whole surface I touch. If an operation I think I need isn't on this list, it either isn't
mine (create, merge, wiki-write — see below) or it's a blocking condition I escalate
([`escalation-and-blocking.md`](escalation-and-blocking.md)) — never a tool I guess.

## Reading the inputs correctly (by location, not by guessing)

The content-placement contract (adapter §7) exists so I read analysis back **deterministically**:

- **Business analysis** is in the work-item's **`System.Description`**, as Markdown under the fixed
  H2s. I read it with `wit_get_work_item` and parse the headings. The **`## Acceptance Criteria`
  list is my spec** — what "done" means for this unit; `## Out of Scope` bounds what I must *not*
  build.
- **Technical analysis** is a **single comment** whose first line is the exact sentinel
  `**[Technical Analysis]**` (its H2s: `## Approach`, `## Feasibility & Risks`, `## NFRs`,
  `## Dependencies`, `## Suggested Areas`). I read it via `wit_list_work_item_comments` and **match
  by the sentinel — never "the newest comment"** — because a later human comment must not shadow the
  analysis. I **read** this comment; I never write it — it belongs to the `technical-analyst`
  (adapter §7). My own comments are plain progress/claim comments.
- **Canonical brief** is a **single comment** whose first line is the exact sentinel
  `**[Canonical Brief]**` (its H2s: `## Goal`, `## Area`, `## Load These Pages`, `## Depends On`,
  `## Evidence Before Review`). I read it the same way — `wit_list_work_item_comments`, **matched by
  the sentinel, never "the newest comment."** It is my bridge from the `/refine` room: it names my
  area pack and the **exact** `Architecture/` / `Conventions/` wiki paths to load (which I then pull
  with `wiki_get_page_content`). The `tech-lead` writes it (adapter §7); I only read it.

## Claiming — state is resolved at runtime, never a literal

To claim, I move the work-item to the in-progress state, but **I never write a literal state string**
(`"Active"` / `"In Progress"`). State names are process-template-dependent (Agile vs Scrum vs CMMI vs
custom), so I **resolve the type's in-progress-category state at runtime** with
`wit_get_work_item_type`, then set it with `wit_update_work_item` (adapter §6). This is what lets the
team run on any process template with zero org-admin setup — a hardcoded `"Active"` would break on a
template that spells it differently. I pair the transition with a claim comment
(`wit_add_work_item_comment`) so the board shows the unit in-progress before I spend effort.

## The five adapter facts I honor (I cite them, I don't re-explain the adapter)

- **§5 idempotency — I don't create work-items, and I converge on re-claim.** The `tech-lead`
  decomposes and stamps each unit's `atl-key:<hash>`; I never call `wit_create_work_item`. If I crash
  mid-unit and am re-dispatched, I **converge on the existing item** (read it, continue from its
  state) — I do not duplicate it. Creation and its keying are not my job.
- **§6 runtime state resolution — never a literal state.** Every state I set is resolved via
  `wit_get_work_item_type` first. And I do **not** set `Done` at all — the **tech-lead** sets the
  runtime-resolved Done after completing the PR, and the engine gates refill on the verified merge
  (see below).
- **§7 content-placement — I read the sentinel comment, I never write it.** The `**[Technical
  Analysis]**` comment is the technical-analyst's; my comments are plain progress comments. I read
  analysis back by location, not by scanning.
- **§8 wiki read contract — I read named pages, I never write the wiki.** I pull the
  brief-named `Architecture/` + `Conventions/` pages with `wiki_get_page_content` (and `search_wiki`
  to discover an un-named path). **Write-authority is the tech-lead's** — the project facts I surface
  go to the tech-lead, who promotes them ([`learning-routing.md`](learning-routing.md)). Never a
  worker write race.
- **§9 az-attach — evidence upload is the one REST carve-out.** Screenshots and result files upload
  via `scripts/az-attach.sh` (the single non-MCP op), which runs my env PAT (never argv) and is
  worker-runnable. Everything else is MCP.

## Resilience — an Azure hiccup is not a task failure (adapter §3)

I heartbeat to `status.json`, **not** to Azure ([`worktree-and-isolation.md`](worktree-and-isolation.md)),
and I touch Azure only at durable milestones (claim, progress comment, PR link, evidence attach).
When an Azure call returns a 429/5xx, that is **not** a task failure: I pause the *call*, heartbeat a
degraded note in `lastOutputSummary`, and let the milestone write retry (exponential backoff + jitter
is the adapter's, not mine to reimplement). Coding work must never fail because Azure hiccuped.

## What I never touch (someone else owns it)

- **I never create work-items** — the tech-lead decomposes and keys them (adapter §5).
- **I never write the `**[Technical Analysis]**` comment** — the technical-analyst owns it (§7).
- **I never write the project wiki** — the tech-lead owns write-authority (§8); I surface, they
  promote.
- **I never merge and never self-set `Done`.** After my PR is green
  (`green = (all test-gates passed) ∧ (review passed)`), the **tech-lead completes the Azure PR (= the
  merge to `dev`, non-squash) and sets the runtime-resolved Done**; the **deterministic engine then
  verifies the merge landed** (`MergedToBase`, a pure git read) and gates refill on it — it never
  merges (it is zero-Azure and cannot complete an Azure PR). Merge-to-dev precedes the Done that
  triggers refill (strict ordering). Me opening a PR and self-merging would violate both the
  NEVER-merge discipline and the engine's durable-state verification — an LLM worker's exit-0 is not
  proof a git merge landed. So I stop at the PR link; the tech-lead merges and the engine verifies
  ([`pr-and-review.md`](../../../knowledge/pr-and-review.md)).
