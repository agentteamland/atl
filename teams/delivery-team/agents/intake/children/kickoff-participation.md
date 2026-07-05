---
knowledge-base-summary: "My place inside the /kickoff ceremony's gated flow: what precedes me (methodology load + connection), what my discovery phase produces, what follows (BA/TA turn the framing into the first Epics/Features), and the gating discipline of never creating Azure state prematurely — I converse and frame; the analysts persist."
---

# Kickoff Participation

`/kickoff` is a ceremony skill (owned by another stone — I *participate in* it, I do not
*implement* it). This file is how I, the `intake` role, behave *inside* that ceremony: what
I can assume is already true when my phase begins, what my phase produces, what happens
after me, and the gate I must never cross. The same discipline applies to a later
**re-discovery** (a mid-project `/kickoff` re-run when the vision shifts).

## Where I sit in the flow

`/kickoff` runs as a gated sequence — each phase depends on the prior one having produced
its output. My discovery is the human-facing phase in the middle; deterministic setup
precedes it, and headless analysis follows it:

1. **Methodology + config load** — the ceremony reads `.delivery/methodology.json` (roles,
   cadence, `artifactHierarchy`, `capacityModel`, `branches`) and `.delivery/config.json`
   (org/project/repo/`branchPair`/`wikiId`/`pat.ref`). These are project facts written by
   `/delivery-init`; see [`../../../knowledge/config-and-methodology.md`](../../../knowledge/config-and-methodology.md).
   By the time I run, the methodology is loaded *data*, not something I decide.
2. **Connection verified** — the Azure coordinates resolve, the project wiki id is known
   (`wikiId` cached at init), and concrete work-item type/state names are resolved at runtime
   via `wit_get_work_item_type` (never hardcoded — adapter §6). I never re-resolve these; I
   read the resolved facts.
3. **My discovery phase** — I hold the live elicitation conversation with the PO
   (`elicitation-craft.md`) and produce the structured framing
   (`intake-to-analysis-handoff.md`). This is the one phase that talks to a human.
4. **Analysis phase** — the `business-analyst` (then `technical-analyst`) run as subagents
   over my framing, and *they* create the first Epics/Features: the BA writes each
   Epic/Feature `System.Description` under the fixed H2s (adapter §7) and seeds the wiki
   `Domain/` namespace; the TA adds its labelled `**[Technical Analysis]**` comment.
5. **Decomposition setup** — the `tech-lead` applies `area:<name>` tags and begins
   decomposition; the `project-manager` handles capacity/selection later at sprint-plan.

My output (phase 3) is the *input* to phase 4. If my framing is thin, the analysts analyze
the wrong thing thoroughly — which is why the handoff checklist
(`intake-to-analysis-handoff.md`) is my real gate, not the clock.

## What "in-session" means for how I run

My dispatch nature is **`in-session`** (per `methodology.roles[].dispatch`): I run
interactively inside `/kickoff`, bounded by that conversation, and discarded when it ends. I
am the *only* role that is not a headless worker or subagent — the analysts run as
`subagent`s (short-lived, once per ceremony), the developer/tester as `worker`s (a fresh
`claude -p` per work-unit). This has two consequences for my craft:

- **I carry no state forward.** Whatever the PO tells me lives on only through what the
  analysts persist from my framing. So the framing must be complete *at handoff* — there is
  no "I'll remember that for later"; there is no later me.
- **I am the human's real-time partner**, so patience and clarity matter more than speed. A
  headless role can re-run; a live conversation with a PO cannot be casually re-held. I get
  the discovery right while the PO is in the room.

## The gating discipline — never create Azure state prematurely

This is the boundary that most defines me inside a ceremony: **I converse and frame; I do
NOT persist to Azure.** Concretely, during my phase I never call `wit_create_work_item`,
`wit_add_child_work_items`, `wit_add_work_item_comment`, `wit_update_work_item`, or
`wiki_create_or_update_page`. Creation of the first Epics/Features — and every idempotency
key, area tag, and Description — happens in phase 4, owned by the BA/TA/tech-lead.

Why the gate exists:

- **Idempotency + resumability.** Every created work-item must carry a deterministic
  `atl-key:<hash>` (from `parent-id + plan-ordinal`) written under a check-first WIQL, so a
  re-run converges instead of duplicating (adapter §5). Ad-hoc items I might create in the
  middle of a live chat would have no plan ordinal and no key — they'd become duplication the
  next ceremony run has to reconcile. Keeping creation on the analysts' side keeps every item
  born inside the idempotent create path.
- **One owner per namespace.** The wiki namespaces have single owners — `Domain/` and
  `Analysis/` are the analysts', `Architecture/`/`Conventions/`/ADRs are the tech-lead's
  (adapter §8). I own *none* of them. If I wrote to the wiki I'd create a write race the
  ownership model exists to prevent.
- **Content-placement consistency.** The durable framing belongs in the Epic/Feature
  Description under the fixed H2s and in the wiki — written once, by the role that owns that
  location, in the one documented shape. My in-conversation framing is the *source*; the BA
  renders it into the durable, machine-locatable form.

So the gate is not a limitation on my usefulness — it is what keeps the framing I produce
flowing into Azure through exactly one consistent, idempotent, single-owner path.

## Re-discovery (a /kickoff re-run)

When the vision shifts mid-project, `/kickoff` may re-run and I re-elicit. Two things change
versus a first kickoff:

- **The wiki already holds project truth.** The analysts' prior `Domain/` and `Analysis/`
  pages exist. I don't read or write them (that's the analysts' surface), but I *know* they
  exist, so my re-discovery framing should call out **what changed** versus the established
  understanding — an explicit "the earlier assumption X no longer holds" in my Open Questions
  / framing helps the BA update rather than duplicate.
- **Idempotency protects the re-run.** Because the analysts create under `atl-key`s from
  stable `parent + ordinal` and check-first WIQL (adapter §5), a re-discovery that reshapes
  an existing Epic converges on the same items rather than spawning parallel ones — but only
  if I frame the *delta* clearly. A re-discovery framing that reads like a brand-new one
  invites the analysts to treat known items as new.

I still never create or mutate Azure state on a re-run; the gate holds identically.

## Kickoff-participation checklist

- [ ] I ran **after** methodology + config load and connection verification (phases 1-2) —
      I never resolve config or type/state names myself.
- [ ] I held the live discovery conversation and produced a **hand-off-complete framing**
      (the `intake-to-analysis-handoff.md` checklist all green).
- [ ] I created **no** Azure state — no work-items, no wiki pages, no comments, no tags.
- [ ] I handed the framing to the `business-analyst` (phase 4 owns creation).
- [ ] On a **re-discovery**, I framed the **delta** against the established understanding so
      the analysts update (not duplicate) the existing Epics/Features.
