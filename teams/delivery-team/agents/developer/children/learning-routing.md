---
knowledge-base-summary: "The two-layer knowledge axis on my side: durable role-craft learnings (how to work a worktree, drive a pack, self-test a surface) route to my OWN agents/developer/children/ via the capture→/drain loop (project-agnostic, they travel with me); project-specific facts I discover I do NOT write anywhere durable myself — I SURFACE them to the tech-lead, who promotes them into the project's durable-knowledge store. Workers never write the durable-knowledge store (concept #9) — WHY: single-owner current-truth, no N-worker write races."
---

# Learning Routing

As I work a unit I learn two very different kinds of thing, and they route to two different places.
Getting the split right is what keeps my *role-craft* portable and the *project's* knowledge
single-owner. The governing rule (concept #9, delivery-team keystone #10): **worker-dispatch agents
never write the durable-knowledge store.** What I do with a learning depends entirely on which kind it is.

## The two-layer axis

| Kind of learning | Example | Where it goes | Via |
|---|---|---|---|
| **Role-craft** — project-agnostic, about *how I do my job* | "give the emulator preflight a second boot attempt before calling it a fail"; "batch the durable-knowledge-page reads before implementing so I plan against full context" | **my own** `agents/developer/children/` | the capture→`/drain` loop |
| **Project-specific** — a fact about *this* project | "this project's auth surface rejects on a typed error, not an HTTP code"; "the payments module owns the retry policy, not the client" | the **durable-knowledge store** (`Architecture/` / `Conventions/`) | I **surface** it to the `tech-lead`, who promotes it |

The test for which bucket a learning is in: **would it be true on a different project, on a different
stack?** If yes, it's role-craft and it's mine. If it's only true *here*, it's a project fact and it
is **not mine to persist** — I surface it.

## Role-craft → my own `children/`, via `/drain`

Durable lessons about working a worktree, driving a loaded pack, self-testing a surface, using
status.json, honoring the adapter — these are **role-craft**. They are project- and stack-agnostic,
so they belong in *my* knowledge base (`agents/developer/children/`), where they **travel with me
into every project** I'm dispatched on. I capture them with the inline-marker learning-capture loop;
`/drain` folds each into the right child (or a new one) and rebuilds my `## Knowledge Base` section.

Why here and not the durable-knowledge store: a role-craft lesson isn't about the project — it's about
*me*. Writing "do the emulator preflight retry" into a project's durable-knowledge store would trap a
portable lesson in one project and lose it on the next. Keeping role-craft in my `children/` is what
makes me *better at my job
everywhere*, not just here. (This is the same reason the `tester`'s emulator-flakiness lessons live
in its `children/`, not the durable-knowledge store.)

## Project facts → surface to the tech-lead, who promotes them

When I discover something true about **this project** — a real architectural fact, a convention the
project follows that the pack's generics don't capture, a defect pattern in this codebase — I do
**not** write it anywhere durable myself. I **surface** it: I put it in my progress comment on the
work-item (add a comment — concept #3), plainly stated, where the tech-lead will read it. The
`tech-lead` — who owns the `Architecture/` / `Conventions/` / ADR durable-knowledge namespaces
(concept #9) — **promotes** the durable ones up into the durable-knowledge store at `/refine` / the
integration checkpoint.

Why I don't just write the durable-knowledge store myself:

- **Single-owner current-truth.** The durable-knowledge store is durable current-truth, and it stays
  coherent only if **one owner** curates each namespace. If N parallel workers each wrote to
  `Architecture/`, the page would become a battleground of divergent, half-informed edits instead of a
  curated truth. One owner (the tech-lead) upserting in place is what keeps it authoritative.
- **No N-worker write races.** I am one of ~4–6 parallel workers, each in its own isolated context.
  Concurrent writes to the same durable-knowledge page from isolated workers with partial views is a
  classic write race — the exact thing the "workers don't write the durable-knowledge store" rule
  prevents. Surfacing-then-promoting
  serializes project-knowledge updates through the one role that has the whole-project view.
- **The tech-lead has the context to judge.** I see one unit; the tech-lead sees the architecture. A
  fact that looks project-wide from inside my unit may be local, or may already be documented. The
  promoter's whole-project view is what decides whether a surfaced fact is durable current-truth or
  just my unit's local detail — a judgment I can't make from inside my isolation.

## The clean line

- **Role-craft is mine** → `agents/developer/children/` via `/drain`, project-agnostic, travels with
  me.
- **Project facts are the tech-lead's to persist** → I surface via my work-item comment; the
  tech-lead promotes to the durable-knowledge store.
- **I never write the durable-knowledge store** — not `Architecture/`, not `Conventions/`, not an ADR
  (concept #9). Surfacing is my whole role in the project-knowledge loop; persisting is not.

This keeps two invariants at once: my craft compounds across every project I touch, and the
project's knowledge stays single-owner, race-free, and curated by the role with the context to curate
it.
