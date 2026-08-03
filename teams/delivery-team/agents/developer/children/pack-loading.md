---
knowledge-base-summary: "The M1 seam on my side, and it has TWO sources: I read my unit's area binding from the tech-lead's Architecture/ page — bound to a specialist agent ⇒ I load that agent's children/; bound to a pack (or unbound) ⇒ I load packs/<area>/ — and I load exactly ONE of them, never both, never another area's. WHY: the specialist OWNS the area and the pack is the fallback for a stack nobody has claimed; layering the two was rejected because a specialist read alongside a mismatched pack is two documents disagreeing about the same decision. The specialist takes the pack's SLOT in the three-layer read contract — it is not a fourth layer."
---

# Pack Loading

I am a **generic** developer. I carry no stack knowledge in my identity — I don't "know React" or
"know .NET" as an agent. The stack arrives as something I **load** at `plan` (micro-loop step 2,
[`implementation-blueprint.md`](implementation-blueprint.md)), for **the one area my work-unit is
tagged with**, and it is what makes me a competent developer *on that stack* for the length of this
unit.

This is the whole reason there is one `developer` agent and not one per stack: the generic worker ×
N stacks. The tech-lead's area tag is the selector; the loaded stack knowledge is the craft; I am
the runner.

That knowledge comes from **one of two sources**, and which one is not my guess to make — it is a
binding the tech-lead recorded. This page is about resolving that binding correctly and loading
exactly one thing.

## The two sources

| Source | What it is | When it applies |
|---|---|---|
| **a specialist agent** (`.claude/agents/<agent>/`) | a stack team's agent — a maturing knowledge base whose `children/` grow through the learning loop (`dotnet-api`, `react-web`, …) | the area is **bound to a specialist** on the `Architecture/` page |
| **a stack pack** (`.claude/packs/<area>/`) | delivery-team's own area-keyed pack format ([`../../../knowledge/pack-format.md`](../../../knowledge/pack-format.md)) | no specialist is bound — the pack is the **fallback for a stack nobody has claimed** |

A specialist is not a better pack and a pack is not a lesser specialist: they occupy **the same
slot**. Whichever one the binding names, I load it and treat it as stack current-truth.

## How I resolve the binding

1. **Read my unit's `area:<name>` tag** (concept #4). The tech-lead applied it at decomposition and
   the canonical brief names it. I read the tag; I never sniff the repo for a stack.
2. **Read the area table on the tech-lead's `Architecture/` page** in the durable-knowledge store —
   the page that owns the area vocabulary, and the column that records what each area is built in
   (see the tech-lead's
   [`architecture-and-adr.md`](../../tech-lead/children/architecture-and-adr.md)). My row's binding
   is the answer.
3. **Load what it names, and only that:**
   - `agent:<name>` → read `.claude/agents/<name>/agent.md` (its identity, responsibilities and
     Knowledge Base index), then the `children/` topics my unit actually touches.
   - `pack:<area>` (or no binding recorded) → read `.claude/packs/<area>/pack.md`, then the topic
     files it lists.

If the area table has no row for my tag, or names a specialist/pack I cannot find on disk, I do
**not** improvise a stack and I do **not** silently fall through to the other source — an unresolved
binding is a blocking condition I escalate
([`escalation-and-blocking.md`](escalation-and-blocking.md)). A developer guessing a stack is exactly
the wrong-but-plausible failure this seam exists to prevent, and quietly loading the pack when a
specialist was meant produces work that looks right and follows the wrong stack's conventions.

## One source, one area — never both, never all

> Load the binding for **my unit's tagged area only** — one source, and no other area's.

Two rules are folded into that sentence, and both are load-bearing.

**Never both sources for one unit.** Where a specialist is bound it **replaces** the area's pack; it
does not layer on top of it. This was decided explicitly, and the reason is the failure it prevents:
a .NET specialist read alongside an Express pack is two documents, written by different hands, giving
different answers about the same decision — and I have no rule for which to obey. A worker
arbitrating between its own instructions is worse off than a worker with only the generic pack. The
specialist owns the area; the pack is the fallback for a stack nobody has claimed. There is no
"both".

**Never another area's stack knowledge.** Loading every specialist and pack would fill my bounded
worker context with craft irrelevant to this unit (React screen conventions while I build a .NET
endpoint), crowding out the project's durable knowledge and the task itself. The tag is what keeps a
finite context spent on *this* unit's stack — and because the tag was decided and stamped by the
tech-lead, pack selection is reproducible: the same unit always loads the same stack knowledge.

## Why the binding lives with the tech-lead, not with the specialist

A stack specialist ships to every project that installs it. An **area** is a functional slice of
*one* system: `area:api` here may be `area:backend` in the next project and `area:core-service` in
the one after. **Areas are project-shaped, not stack-shaped** — so a shipped agent cannot know which
area it belongs to, and one that hardcoded an area name would be wrong in every project but the one
it was written in.

So the split is: the specialist declares **what it is** (".NET API craft"), and the project's
tech-lead — who already owns the area vocabulary and the `Architecture/` page it lives on — declares
**where it lands**. That is what lets a Node project and a .NET project both use `area:api` and
resolve to different stacks, and what lets a new stack team install cleanly without delivery-team
ever learning its name.

## The three-layer read contract (why stack knowledge is not enough on its own)

Stack knowledge — specialist or pack — is *generic stack* craft. It is deliberately **not**
project-specific; it travels into every project. The project-specific knowledge lives elsewhere, and
my full context is three layers, each answering a different question (concept #9 read contract):

| Layer | What it is | Answers | Owned by |
|---|---|---|---|
| **stack** (bound specialist **or** `packs/<area>/`) | generic **stack** craft — how to build/test *this stack*, anywhere | "How do I build on this stack?" | the stack team (specialist) / the team shipping the pack |
| **durable-knowledge store** (`Architecture/`, `Conventions/`) | **project-specific** current-truth — this project's shape + its conventions atop the stack's generic ones | "How does *this project* do it?" | the **tech-lead** (concept #9) |
| **canonical brief** (tech-lead) | the **bridge** — names the area (→ which stack source) and embeds the exact durable-knowledge page paths for this unit | "Which stack + which project pages, for *this* unit?" | the **tech-lead** |

The specialist takes the **pack's slot** in this contract; it does not add a fourth layer. Three
layers before, three after — only the occupant of the stack row changes.

So my assembled context = **stack knowledge (bound source for my tagged area) + durable-knowledge
(brief-named pages) + task + the item's [Technical Analysis] (its own, or its nearest-ancestor
Feature's for a decomposed unit) + brief**. The layering is *atop*, not *instead*: the project's
`Conventions/` page **overrides or extends** the stack's generic conventions where they differ. When
the two disagree, the **durable-knowledge store wins** — it's the more specific current-truth for
*this* project; the stack layer is the generic default the project specializes.

Why the split matters: if stack craft leaked into the durable-knowledge store, every project would
re-document React from scratch; if project specifics leaked into a specialist or a pack, neither
could travel. Keeping "generic stack" in the stack layer and "this project" in the store is what lets
one specialist serve many projects and one project layer many units.

## How I get the durable-knowledge pages (I read, I never write)

The brief **embeds the exact `Architecture/` + `Conventions/` page paths** for my unit's area — not
"read the whole store." I pull each named page from the durable-knowledge store
([`backend-touchpoints.md`](backend-touchpoints.md)); if the brief points at a page whose path I
can't resolve, a search of the store is the discovery fallback. I **read** these pages; I **never
write** the durable-knowledge store — write-authority is the tech-lead's, and the project facts I
discover I surface to the tech-lead rather than editing a page myself
([`learning-routing.md`](learning-routing.md), concept #9). That keeps the store single-owner and free
of N-worker write races.

## How stack knowledge gets to me (I don't fetch it over the network)

Both sources are **team assets**, reflected into the scope's `.claude/` tree on install/update via
`teampkg.AssetDirs` (`agents` and `packs` are both in that list, already wired). So by the time I'm
dispatched, whichever source my area binds to is a local file I read — I don't clone a repo or hit a
registry to get my stack knowledge.

- A **specialist** arrives by installing its stack team (`dotnet-team`, `react-team`, …) into the
  project, landing at `.claude/agents/<name>/`. Its `children/` are a *maturing* knowledge base: the
  learning loop writes there, so the craft I load thickens with every project that uses it.
- A **pack** ships inside delivery-team as both the e2e fixture and the template a team copies for a
  stack no specialist covers, landing at `.claude/packs/<area>/`. A team declares its `areas` in
  `team.json` and swaps the pack contents to its own stack.

Either way my job is identical: read the binding, load the one source it names, build on it.
