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

**A signal without those words is not a dispatch signal.** Setting `ATL_NO_SWEEP_DISPATCH`
makes every one of them print its passive form instead — `… is due — run /observe to …`,
naming no subagent and citing no rule. That is a report addressed to a person, and this rule
does not apply to it: do not spawn anything, do not offer to, and do not treat the shorter
wording as an abbreviation of the longer one. Mention the sweep if it is relevant to what the
user is doing, and otherwise carry on.

The brake exists because the addressee *is* the behaviour. These signals ship to every user, and
spawning background subagents unasked in someone else's project is a different decision from
making it in the project that authored this rule. The opt-out drops the dispatch and keeps the
report, on purpose — someone who does not want automatic work has not asked to stop being told
the work is due.

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

### The digest, concretely

A sweep records each judgement-call finding with:

```bash
printf '<the evidence and why it matters>' | atl digest add --sweep observe --title '<one line>'
```

Idempotent by `(sweep, title)`: re-reporting refreshes the body and **leaves the read state alone**,
so a sweep that fires whenever its paths move converges instead of re-interrupting about the same
thing.

**When a session starts with findings waiting**, `atl session-start` prints how many — and only how
many. On that signal:

- **Show them to the user, in your own words**: run `atl digest`, read what comes back, and present
  it. This is a foreground, in-turn action, not a background subagent — the whole category is
  *things a human must decide*, so there is nobody for a subagent to hand them to. (Same boundary as
  a store with no remote, above.)
- **Do not card them.** `/observe`'s own rule that latent-gap findings are never auto-carded is not
  overridden here. Offer to open a brainstorm or a board item once the user has decided; that is
  their call to make, and making it for them is what the split exists to prevent.
- **Drop what gets settled** — `atl digest drop <id>` — so the count means what it says.

The count is the entire payload of the signal on purpose. A signal that restated its findings every
session would be the constant channel this design exists to avoid, arriving through the mechanism
built to avoid it.

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
