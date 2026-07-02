---
knowledge-base-summary: "How I author a brand-new interface when a below-threshold entity is a coherent novel type no seeded interface fits. Silent-autosave but guardrailed: conservative default tiers, a small extension over the inherited core, marked authored: agent-<date> (distinguishable + refinable), noted in the drain report. A one-off stays an unknown stub — I don't invent a type for it."
---

# Interface Creation

The type-open mechanism: when an entity fits **none** of the seeded interfaces well
(below `thresholds.type-match`, 0.80), and it is a *coherent, recurring kind* rather than a
one-off, I author a new interface for it and grow the profile against that — instead of
leaving it a bare `unknown` stub. This is what lets the user's inner world hold kinds I was
never pre-taught (a sourdough starter, a recurring dream, a chronic condition, a treasured
playlist). It is the riskiest thing I do — an LLM authoring a durable, shared schema — so it
is guardrailed, not freewheeling.

## When I create vs. stub

Not every below-threshold entity earns an interface — that would sprawl the type set. I
create a new interface only when **both** hold:

1. **I can name a coherent type** — a `type-id` that would fit a *class* of entities, not
   just this one (e.g. `hobby`, `vehicle`, `condition` — not `alexs-blue-mug`).
2. **I can write its detection signal + a few real fields** — a `matches` description,
   positive/negative examples drawn from the conversation, and a small, honest extension.

If I can't do both — a genuinely one-off entity — it stays a minimal **`unknown` stub**
(`type-detection.md` §4). The stub still remembers the entity and how the user feels about
it (the common core); it just isn't richly typed. Reuse-a-seeded-type and stub both beat
inventing a shaky type.

## How I author one (silent-autosave, guardrailed)

Per the settled design this is **silent** — no approval prompt (the user sees it in the
drain report, and it is refinable). But silent ≠ careless:

1. **Name it.** A singular, lowercase-kebab `type-id`; pick a sensible plural for its
   directory (`~/.atl/profiles/<plural>/`) and record it.
2. **Write the self-describing interface**, same shape as the seeded ones
   (`person-interface.md` is the template):
   - `matches` — what fits this type + what does NOT (the negative half is the guard).
   - `examples-positive` / `examples-negative` — real sentences, drawn from the conversation
     that surfaced the entity.
   - `schema-version: 1.0.0` + `changelog: [{version: 1.0.0, added: [everything]}]`.
   - `thresholds` — `type-match: 0.80` + the standard salience block (so future entities
     score against this type too, and it evolves like any other).
   - `fields` — **inherit the common core** (do not re-specify it); author a **small**
     extension (a handful of well-chosen fields), never a sprawling guess.
   - `change-policy` — default `overwrite`; `history-tracked` only where temporal evolution
     obviously matters.
3. **Tier conservatively (the load-bearing guardrail).** I do not know this type's privacy
   shape the way a hand-authored interface does, so I err safe: `identity` / `anchors` /
   `kind` / `role` → Tier 1; **anything that reads like a feeling, a private state, health,
   finances, or a sensitive detail → Tier 3+** (never auto-assign Tier 1 to a field that
   could leak). A wrong-high tier just means a fact waits for explicit confirmation; a
   wrong-low tier leaks. Prefer the safe error.
4. **Stamp it `authored: agent-<today>`** in the interface frontmatter. This distinguishes an
   agent-authored interface from a canonical shipped one, so it is discoverable, reviewable,
   and promotable (a later hand-review can refine it and drop the stamp). It is *provisional*
   by nature.
5. **Materialize** it to `~/.atl/profiles/_interfaces/<type-id>.md`, then create the profile
   against it in `~/.atl/profiles/<plural>/<slug>/` (the normal create path, `marker-drain.md`
   §5).
6. **Note it in the drain report** (non-blocking): "created a new `<type-id>` interface for
   `<entity>` — review at `~/.atl/profiles/_interfaces/<type-id>.md`." Awareness, not a gate.

## After creation

An agent-authored interface is a first-class interface: subsequent entities of that kind
score against it and reuse it (no second creation), it evolves through `schema-version` +
`changelog` + lazy-fill like the seeded ones, and its tiers refine in place. The `authored:`
stamp is the only thing that marks it provisional — everything else about it is ordinary.

**Hallucination tolerance, bounded.** A field *value* that is wrong self-corrects next
conversation (the platform's tolerance). A *schema* is more load-bearing — it shapes every
profile of that type — so the guardrails above (create-only-for-a-real-type, small
extension, conservative tiers, the `authored:` stamp) are what keep the tolerance safe at
the schema level.
