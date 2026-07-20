# Idempotent writes

Any operation that writes to a durable store must be **idempotent**: running it again converges on the same intended end-state rather than duplicating work or silently clobbering a prior edit. Check-before-create by a stable key; overwrite current-truth in place; never blind-write.

## Why this rule exists

The same discipline is re-derived independently across the platform — a sign it belongs in one place, not scattered:

- The delivery loop makes it a named cross-backend contract (**concept #10**, `check-first-by-key before every create → found→reuse+update, not-found→create-then-stamp`), consumed by `/kickoff` ("must converge, never blind re-create"), `/refine` ("a re-run must never duplicate"), and `/delivery-init` ("if config already exists, do not blind-overwrite").
- `/brainstorm` implements it with its own provenance key ("`done` on the same brainstorm must converge, never duplicate").
- `/create-code-diagram` states the overwrite-in-place flavor ("running again overwrites the previous diagram — always fresh").
- The profile layer's `/profile-restore` is a guarded overlay that "never silently clobbers global data that is newer".

Every autonomous worker and every ceremony re-runs — on restart, on retry, on a resumed sprint. A non-idempotent write turns a re-run into a duplicate work-item, a doubled PR, or a lost edit. The stakes are highest for the delivery engine (a resumed run must converge on the durable state, not re-create it) and for any long-lived knowledge store.

## What the agent must do

1. **Check-before-create by a stable key.** Before creating a durable item, search for it by a key derived from stable inputs (parent + ordinal, a content hash, a slug) — never a per-run GUID. Found → reuse and update it toward the intended state; not-found → create, then stamp the key. A create that collides with an existing item is *resolved to it*, not surfaced as an error.
2. **Overwrite current-truth in place.** For a store that holds "what is true now" (a config file, a wiki page, a generated artifact, a review page), re-running replaces the prior value rather than appending a second copy.
3. **Never blind-clobber a newer edit.** When overwriting could destroy data written since you last read it, guard the write (compare timestamps/versions; diff and confirm) instead of assuming your copy is authoritative. Losing a newer edit is worse than refusing the write.
4. **Design the re-run as the normal case.** Assume every write path will run again — after a crash, a retry, or a resume — and make convergence the default, not a special recovery mode.

## Failure modes to watch

- **Per-run key** — keying idempotency off a GUID/timestamp generated this run, so the "check" never matches on the next run and every re-run creates a duplicate.
- **Blind overwrite** — writing current-truth without checking whether the target changed since you read it, silently dropping a concurrent/newer edit.
- **Partial-write non-convergence** — a multi-step write that, interrupted and re-run, leaves duplicates because only the final step stamped the key (stamp/commit the key at creation, not at the end).

## When this rule does NOT apply

- **Append-only stores by design** — a journal, an audit log, an event stream. There, appending a new dated entry on each run is the intended behavior, not a duplicate; idempotency lives in the entry's content, not in refusing the append.
- **A genuinely new, distinct entity** — creating a second, different work-item/page is not a duplicate. The rule targets re-creating *the same* logical item, identified by its stable key.

## How this rule interacts with other rules

- **`knowledge-system`** — its "wiki = replace / journal = append" split is this rule applied to the two knowledge layers: current-truth pages overwrite (idempotent), the history layer appends (the exception above).
- **`learning-capture`** — the durable queue's exactly-once transfer + content-hash dedup is this rule at the pipeline level (a marker enqueues once, a drained item can never re-enter).
- **`branch-hygiene`** — "verify state, don't assume" is the read half that makes a safe overwrite possible: check the target before you write it.
