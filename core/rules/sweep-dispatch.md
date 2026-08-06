# Sweep dispatch — run a due sweep, don't just read that it is due

`atl session-start` reports when a periodic sweep is due. **This rule is what to do about it:
spawn one background subagent and let it run alongside your turn**, exactly as the learning and
profile-fact queues are drained.

## The signals

Four, all printed by `atl session-start`, each naming its skill and citing this rule:

```
atl: a proactive observer sweep is due — run /observe now in a background subagent …
atl docs: a full audit is due — run /docs-audit now in a background subagent …
atl skills: a stocktake is due — run /skill-stocktake now in a background subagent …
atl rules: a distill is due — run /rules-distill now in a background subagent …
```

## What to do

**Spawn ONE background subagent per signal**, using the Agent tool with `run_in_background`.
Single-in-flight, per skill: if a subagent for that sweep is still running, do not spawn another —
the signal keeps appearing until the sweep records its cursor, and the running one will do it.

Several signals can appear on the same turn. They are different skills over different corpora, so
each gets its own subagent; a stretch that is due for two sweeps genuinely needs both.

**Do not ask first, do not run the sweep inline, and do not wait for it.** Your turn belongs to the
user's request; the sweep happens beside it. There is no completion message to relay — the sweep
routes its own output (below).

**Nothing is lost if you miss one.** A sweep's cursor only advances when the sweep records it, so an
unrun sweep is still due next session. The signal is idempotent by construction, which is why it is
safe to skip a turn and wrong to suppress one.

## Where the output goes

Split by whether a finding needs a decision:

- **Actionable and already decided** — a ripe deferral trigger, a deterministic drift — goes to the
  **board** as a work item, checked-first by a stable key so a daily sweep converges instead of
  re-carding the same thing.
- **Needing judgement** — a latent gap, a proposed rule, a redundancy between two skills — reaches
  the **user as a digest**. Never auto-carded: a finding that needs a decision needs it *before* it
  needs a ticket, and `/observe` states that constraint for itself.

The split is the point. The board already carries actionable work through to closure, so a ripe item
needs no new surface; and because the actionable half leaves, the digest shrinks to only what a human
must actually decide — which is the only version short enough to get read.

## What this rule does NOT cover

**A signal whose remedy needs a person cannot be dispatched.** Some conditions are reported by
`session-start` and are deliberately *not* sweeps: a durable store with no remote needs a URL only
the user can supply, and a background subagent has nobody to ask. Those signals say so in their own
text and their owning rule says to act with the user present. Raising one in the session is the whole
remedy; spawning a subagent to do it is theatre.

**Work that needs the live turn's situation cannot be dispatched either.** `/consult` reads what you
are about to claim or decide, which does not exist as durable state. It stays in your turn, always.

The test, in order: *can this be finished from durable state alone?* — no, because it needs the turn →
in-turn; no, because it needs a human answer → raise it, don't dispatch; yes → is there enough work to
be worth a subagent? A one-line notice already did its whole job.
