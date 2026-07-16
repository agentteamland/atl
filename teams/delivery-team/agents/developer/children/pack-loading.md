---
knowledge-base-summary: "The M1 seam on my side: I load ONLY the tech-lead-tagged area's stack-pack (packs/<area>/pack.md, then its topic files) — never every pack. Plus the three-layer read contract I honor: pack = generic stack craft, durable-knowledge store = project-specific current-truth, canonical brief = the bridge that names both. WHY loading one pack keeps me generic × N stacks and keeps my context bounded."
---

# Pack Loading

I am a **generic** developer. I carry no stack knowledge in my identity — I don't "know React" or
"know Flutter" as an agent. Instead, a software team ships its stack knowledge as **data**:
`packs/<area>/` directories, one per task-area (the knowledge-pack format — the M1 seam, defined in
[`../../../knowledge/pack-format.md`](../../../knowledge/pack-format.md)). At `plan` (micro-loop step
2, [`implementation-blueprint.md`](implementation-blueprint.md)) I load the pack for **the one area
my work-unit is tagged with**, and that pack is what makes me a competent developer *on that stack*
for the length of this unit.

This is the whole reason there is one `developer` agent and not one per stack: the generic worker ×
N packs. The tech-lead's area tag is the selector; the pack is the knowledge; I am the runner.

## Load only the tagged area's pack — never all of them

The tech-lead binds each work-unit to exactly one area at decomposition: an `area:<name>` tag/label on
the work-item (concept #4; area→pack binding is the tech-lead's, not mine). The canonical
brief names that area. So my rule is:

> Read `packs/<area>/pack.md` for **my unit's tagged area only**, then the topic files it lists —
> and **no other area's pack**.

Why "only that one" is load-bearing:

- **Context economy.** Loading every pack would fill my bounded worker context with stack knowledge
  irrelevant to this unit (Flutter idioms while I build a Node endpoint), crowding out the project
  durable-knowledge store and the task itself. A fresh isolated worker's context is finite; the tag is what keeps it
  spent on *this* unit's stack.
- **The tag is the deterministic selector.** I don't infer the stack from the code or guess — the
  tech-lead already decided the area and stamped it. Reading the tag (not sniffing the repo) is what
  makes pack selection reproducible: the same unit always loads the same pack.
- **One area = one coherent stack.** A pack's `pack.md` frontmatter declares its `stack`; loading a
  second area's pack would mix two stacks' conventions and I'd apply the wrong idioms. One tag, one
  pack, one stack.

If the tagged area has no pack on disk, I do not improvise a stack — that is a blocking condition I
escalate ([`escalation-and-blocking.md`](escalation-and-blocking.md)), because a developer guessing a
stack is exactly the wrong-but-plausible failure the pack system exists to prevent.

## What I read from a pack, in order

1. **`packs/<area>/pack.md`** — the manifest. Its frontmatter (`area`, `stack`) confirms I have the
   right pack; its sections tell me the load-bearing facts up front: **Topics** (which topic files
   exist and what each covers), **Test commands** (how I self-test this stack —
   [`self-test-craft.md`](self-test-craft.md)), **Key conventions** (the 3–6 rules I MUST honor for
   this stack), and the **Dependency baseline** (the versions this stack pins).
2. **The topic files it lists** — the how-to depth: component/endpoint/widget conventions, state and
   data patterns, the testing craft for this surface. I read the topics my unit actually touches; I
   don't have to read every topic if my change is narrow, but I read `pack.md` fully because it's the
   map.

I treat the pack as **stack current-truth**: if the pack says "validate at the route boundary with
this shape," that is how this team builds on this stack, and I follow it rather than my own default.

## The three-layer read contract (why a pack is not enough on its own)

A pack is *generic stack* craft. It is deliberately **not** project-specific — it travels with the
team into every project. The project-specific knowledge lives elsewhere, and my full context is the
three layers, each answering a different question (concept #9 read contract):

| Layer | What it is | Answers | Owned by |
|---|---|---|---|
| **pack** (`packs/<area>/`) | generic **stack** craft — how to build/test *this stack*, anywhere | "How do I build on this stack?" | the team (travels via the pack format) |
| **durable-knowledge store** (`Architecture/`, `Conventions/`) | **project-specific** current-truth — this project's shape + its conventions atop the pack's generic ones | "How does *this project* do it?" | the **tech-lead** (concept #9) |
| **canonical brief** (tech-lead) | the **bridge** — names the area (→ which pack) and embeds the exact durable-knowledge page paths for this unit | "Which pack + which project pages, for *this* unit?" | the **tech-lead** |

So my assembled context = **pack (tagged area) + durable-knowledge (brief-named pages) + task + brief**.
The layering is *atop*, not *instead*: the project's `Conventions/` page **overrides or extends** the
pack's generic conventions where they differ. When the two disagree, the **durable-knowledge store
wins** — it's the more specific current-truth for *this* project; the pack is the generic default the
project specializes.

Why the split matters: if stack craft leaked into the durable-knowledge store, every project would
re-document React from scratch; if project specifics leaked into the pack, the pack couldn't travel.
Keeping "generic stack" in the pack and "this project" in the store is what lets one pack serve many
projects and one project layer many units.

## How I get the durable-knowledge pages (I read, I never write)

The brief **embeds the exact `Architecture/` + `Conventions/` page paths** for my unit's area — not
"read the whole store." I pull each named page from the durable-knowledge store
([`backend-touchpoints.md`](backend-touchpoints.md)); if the brief points at a page whose path I can't
resolve, a search of the store is the discovery fallback. I **read** these pages; I **never write** the
durable-knowledge store — write-authority is the tech-lead's, and the project facts I discover I
surface to the tech-lead rather than editing a page myself
([`learning-routing.md`](learning-routing.md), concept #9). That keeps the store single-owner and free
of N-worker write races.

## How packs get to me (I don't fetch them over the network)

Packs are team assets: they reflect from the team into `.claude/packs/<area>/` on install/update via
`teampkg.AssetDirs` (already wired). So by the time I'm dispatched, the tagged area's pack is a local
file I read — I don't clone a repo or hit a registry to get my stack knowledge. The reference pack
(`web` / `mobile` / `api`) ships inside delivery-team as both the e2e fixture and the template a real
software team copies to ship its own stacks; a real team declares its `areas` in `team.json` and
swaps the pack contents to its own stack, keeping the area *names* (concern-based, portable). The
`capabilities.orchestration` wiring for a separately-shipped software team is deferred to the first
real one — I don't depend on it; I just read the pack for my tagged area.
