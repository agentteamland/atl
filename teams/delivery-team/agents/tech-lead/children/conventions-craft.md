---
knowledge-base-summary: "How I own the Conventions/ wiki namespace (adapter §8): project conventions layered ATOP the knowledge-pack's generic stack conventions — deciding what belongs here (project-specific overrides/additions) versus what belongs in a knowledge-pack (stack-generic craft), keeping the page lean and current-truth via upsert, and pointing each canonical brief at the relevant slice so a fresh worker inherits the project's rules."
---

# Conventions Craft

I own the `Conventions/` project-wiki namespace (adapter §8): the conventions that apply to
*this* project, layered **on top of** the generic conventions a knowledge-pack (`packs/<area>/`,
stone #5) already carries for a stack. Like `Architecture/`, this is **current-truth**, written
by upsert, and it is **project knowledge** (it lives in the Azure wiki, not in these `children/`).

The whole point of this page is to give a fresh, isolated `developer` worker — which has *no
carry-over context* between work-units — the project's house rules without a human re-stating
them every time. The worker's brief ([canonical-brief.md](canonical-brief.md)) points at the
relevant slice, and the worker reads it via `wiki_get_page_content`.

## The two-layer conventions model — what belongs where

There are two layers of "how we do things," and putting a convention in the wrong layer is the
main failure mode:

| Layer | Holds | Where it lives | Owner |
|---|---|---|---|
| **stack-generic craft** | conventions true for the *stack* on *any* project (idiomatic patterns, formatting, standard project layout for that framework) | the knowledge-pack `packs/<area>/` | stone #5 (the pack) — **not me** |
| **project conventions** | conventions specific to *this* project — overrides of a pack default, project-wide additions, cross-area agreements | `Conventions/` wiki page | **me** |

**The test for whether a convention belongs on my `Conventions/` page:** would it be true on a
*different* project using the *same* stack? If yes, it is stack-generic and belongs in the
knowledge-pack, not here — putting it here duplicates the pack and drifts. If it is only true
*because of a choice this project made*, it belongs on my page.

### What genuinely belongs on `Conventions/` (belongs here)

- **Overrides of a pack default** — "the pack says X by default; on this project we do Y, because
  <project reason>." The override + the reason.
- **Project-wide additions the pack can't know** — a naming scheme tied to this domain, an
  error-handling agreement across areas, a shared logging/observability expectation, a
  branch/commit convention beyond the two-branch flow.
- **Cross-area agreements** — a rule two areas must both honor so their code composes (this
  couples to `Architecture/` and sometimes to an ADR when the agreement is hard-to-reverse).

### What does NOT belong here (belongs in a knowledge-pack)

- Idiomatic patterns of the stack that hold everywhere (those are the pack's job).
- Anything stack-specific stated as if it were project-specific — that is stack expertise
  leaking into project knowledge. Stack expertise is a pack, per the team's stack-agnostic
  discipline; a role-agent (me) and a project page are both stack-independent.

## Keeping the page lean — current-truth via upsert

`wiki_create_or_update_page` is an idempotent upsert (adapter §8): when a convention changes I
**update the line**, I do not append a second one. A conventions page that has accreted
contradictory rules is worse than none — a worker reads it as present-tense law and can't tell
which line is live.

- Every convention states its **reason** in one clause. A rule with no reason gets deleted the
  first time someone doubts it; a rule with a reason survives. ("We do Y here because Z.")
- I prune ruthlessly. If a convention is no longer enforced, it comes off the page. Leaving dead
  rules trains workers to ignore the page.
- I resolve `wikiId` from `config.json` (cached at `/delivery-init`); I verify the namespace
  exists with `wiki_list_pages` before a first write; I **never** re-resolve the wiki id.

## How conventions reach a worker (the read path)

A worker never scans the whole wiki. My canonical brief for a unit embeds the specific
`Conventions/` page path (plus the `Architecture/` slice for the unit's area, per adapter §8's
read contract). The worker pulls it with `wiki_get_page_content`; `search_wiki` covers discovery
when a path isn't pre-named. So the conventions are *pushed* to the worker via the brief, not
left for it to find — which is what makes an isolated, contextless worker still obey the project's
rules. See [canonical-brief.md](canonical-brief.md) for how the brief bounds that context.

## Checklist

- [ ] Each convention passes the "would it be true on another project on the same stack?" test —
      if yes, it's a knowledge-pack concern, not mine.
- [ ] Every convention carries a one-clause reason.
- [ ] Page kept current via upsert; changed rules updated in place; dead rules pruned.
- [ ] No stack-specific rule stated as if project-specific (stack expertise stays in the pack).
- [ ] `wikiId` read from cache; namespace verified before first write; never re-resolved.
- [ ] The relevant slice is referenced by each unit's canonical brief so workers actually load it.
