---
knowledge-base-summary: "How I map technical dependencies as real Azure Dependency links (wit_work_items_link) — not just prose — so the project-manager's scheduling DAG is machine-readable, and how I identify and frame risks with a mitigation for each. The distinction between a dependency (a hard ordering constraint) and a risk (a probability of trouble), and how both feed downstream scheduling and review."
---

# Dependency & Risk

My `## Dependencies` and the risk half of `## Feasibility & Risks` produce the two inputs that
let the rest of the team *sequence and de-risk* a sprint: a machine-readable ordering graph and
an honest catalogue of what could go wrong. A dependency the `project-manager` can't see becomes
a mid-sprint surprise; a risk without a mitigation is just anxiety. Both are my job to make
concrete.

## Dependencies must be links, not just prose

A dependency I only mention in the comment text is invisible to the `project-manager`'s
scheduling DAG. The PM builds its ordering from the work-items' **Azure Dependency links**, not
from parsing English. So for every real technical dependency I record it **twice**:

1. **In prose** under `## Dependencies` — human-readable, with the *why* ("B needs A's paging
   capability before B can be estimated").
2. **As an Azure link** via `wit_work_items_link` — the machine-readable edge the PM's DAG
   consumes.

Prose without the link is a dependency the scheduler doesn't know about. The link without prose
is an edge with no rationale — a future reader can't tell why it exists. Both, always.

### The link mechanics (adapter §2)

- I create the dependency edge with `wit_work_items_link` (the adapter's tool for
  Dependency/Parent links). I use the **Dependency** relation, not the Parent/Child hierarchy —
  Parent/Child is the artifact ladder (`artifactHierarchy`, config §1), Dependency is the
  ordering constraint. Conflating them corrupts both the tree and the DAG.
- To remove a stale edge (a dependency that turned out not to hold), I use `wit_work_item_unlink`
  — never leave a false edge that would over-constrain the schedule.
- Linking is idempotent-friendly: re-asserting an existing link is a safe no-op, consistent with
  the team's re-run convergence discipline (adapter §5). I don't need an `atl-key` here — the
  link's endpoints *are* its identity.
- If a dependency is on an item that doesn't exist yet (a spike the tech-lead hasn't created),
  I state it in prose and flag the sequencing; I create the link once both endpoints exist. I
  never invent a work-item to link to — creating work-items is the tech-lead's decomposition.

### Dependency vs risk — a distinction I keep sharp

| | **Dependency** | **Risk** |
|---|---|---|
| What it is | A *certain* ordering constraint: B cannot proceed until A. | A *probability* of trouble: this might go wrong. |
| Where it goes | `## Dependencies` + an Azure Dependency link | `## Feasibility & Risks` with a mitigation |
| Who consumes it | the `project-manager` (scheduling DAG) | the `tech-lead` (decomposition), the `developer` (build), the PO (expectations) |
| The test | "Is A *required before* B?" → yes = dependency | "Could X *cause* B to fail/slip?" → yes = risk |

A spike is often *both*: the unknown is a **risk** (it might reveal the approach doesn't hold)
*and* a **dependency** (the main work can't be estimated until it resolves). When it's both, I
record it in both places — the risk framing for the tech-lead/PO, the dependency link for the
PM's sequencing.

## Kinds of technical dependency I look for

- **Inter-work-item** — this item needs another item's output first (the paging spike before the
  export; a shared library change before the feature that uses it). These become Dependency
  links.
- **External-system** — a third-party API, an upstream service, an infrastructure capability that
  must exist. I can't link to a non-work-item, so I state it in prose and, if it needs a
  provisioning task, flag that the tech-lead may need to create one.
- **Sequencing / data** — a migration that must run first, a feature-flag that must be wired
  before the feature can ship dark. Ordering constraints even when there's no single blocking
  item.
- **Capacity / skill** — rare, but if the work needs a capability the team demonstrably lacks, I
  name it as a risk (not a dependency), because the PM/tech-lead handle staffing, not me.

## Risk identification + mitigation framing

Every risk I list carries a **mitigation** — the response that reduces its probability or blast
radius. A risk without a mitigation is incomplete; it tells the team to worry without telling
them what to do.

The shape I use for each risk:

> **<risk>** *(likelihood × impact)* — <why it could happen>. **Mitigation:** <the concrete
> response>.

Worked (generic):

> - **Large-account memory pressure** *(medium × high)* — naive in-memory file assembly under
>   concurrent exports could exhaust memory. **Mitigation:** stream to the output; assemble
>   nothing whole in memory. (This is a *hard* problem, not an unknown — the mitigation is known;
>   it's a sizing input, [feasibility-and-approach.md](feasibility-and-approach.md).)
> - **Paging feasibility unknown** *(unknown × high)* — the record-access layer may not page a
>   heavy account within the latency ceiling. **Mitigation:** a ~0.5-day spike resolves it before
>   we commit a size; also recorded as a dependency (the main work waits on the spike's outcome).

The `likelihood × impact` note lets the PO and tech-lead triage — a `low × low` risk is
noted-and-moved-on; a `high × high` may reshape the approach or the sprint. I don't compute a
false-precision score; the two-axis tag is enough to sort.

## How this feeds the rest of the team

- **project-manager** — reads my Dependency links to build the scheduling DAG; a missing link
  means an item gets scheduled before its blocker, and the sprint stalls. My discipline of
  link-plus-prose is what keeps the PM's plan sound.
- **tech-lead** — reads my risks to shape the decomposition (a high-impact risk may become its
  own de-risking work-item with a stable plan-ordinal, adapter §5) and to weigh the approach at
  `/refine`.
- **developer** — reads the risks and mitigations in the sentinel-located analysis comment
  (adapter §7) as build guidance ("stream, don't buffer") before touching code.
- **product-owner** — sees the honest risk catalogue so sprint commitments are made with eyes
  open, not on a rosy read.

The through-line: I don't *decide* the schedule, the decomposition, or the architecture — I
produce the **inputs** (links + framed risks) those decisions rest on, and I make them concrete
enough (a link, a number, a mitigation) that the decision is sound rather than guessed.
