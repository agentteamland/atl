---
name: work-finish
description: /work-finish — close out the unit on the current delivery branch: derive its id from the branch name, gate on a scoped build and test run, push, open (or update) the PR to the integration branch with the work item linked, and verify the link actually landed by reading it back. Deliberately does NOT merge and does NOT move the item's state — merge is a human decision and Done comes after it. Mutating and interactive; run it explicitly. Hand-driven half of the drive loop; `/work-start` opens the unit.
---

# /work-finish — gate, push, PR, verify the link

The drive loop's exit. It takes the branch you have been working and turns it into a reviewable
PR that is genuinely, verifiably attached to its work item — then stops, because the two things
that come next are decisions: a human merges, and only after the merge does the item become Done.

The one property this skill exists to guarantee is that **the PR is really linked**. A PR that
looks linked and is not produces a card nobody can trace to code, and it is silent by
construction — everything reports success. So the link is written *and read back*, and a missing
link stops the skill loudly rather than being reported as done.

Reads its contracts from [`config-and-methodology.md`](../../knowledge/config-and-methodology.md),
[the backend interface](../../knowledge/backend-interface.md), and the active
`backends/<backend>/adapter.md`. **Never invent a tool name.**

## When to run

- **When the unit's work is built and you believe it is done** — after `/work-start` cut the branch
  and you did the work.
- **Not** to merge. It opens the PR; a human merges.
- **Not** to mark the item Done. That is `/work-move`, *after* the merge lands.

## Backend support

**GitHub is bound; Azure halts** — same reason and same message shape as `/work-start`. The Azure
adapter self-disclaims its operation map as unverified against the live MCP, and it additionally
binds **no read at all** that verifies a work-item↔PR link landed. Since verifying that link is
this skill's core guarantee, there is nothing honest to run on Azure yet.

## Procedure

### 1. Derive the unit from the branch — the branch IS the state

```
delivery/<sprint-slug>/<id>
```

Parse the id out of the current branch. **A branch that does not match this shape is refused**:

```
/work-finish: `<branch>` is not a delivery branch.
Expected delivery/<sprint-slug>/<id>. Run /work-start <id> to cut one.
```

This is deliberate and it is the loop's central simplification: there is **no per-unit state file**
on disk. The branch name is the whole of it, derived by a pure function and re-parsed by every
consumer. A second store would be a second thing to drift.

Resolve `.delivery/config.json` for `backend` and `branchPair.dev`, then read the item by id to
confirm it exists and to get its title. An id that no longer resolves stops the skill — never open
a PR carrying a broken reference.

### 2. Preflight — a clean tree, then a scoped gate

**`git status --porcelain` non-empty ⇒ refuse.** Commit or stash first; a PR opened over a dirty
tree ships a diff that does not match the branch.

Then run the checks the change can actually reach, and **stop on any failure**. Compute the touched
surfaces from `git diff --name-only origin/<dev>...HEAD` and run only what those paths imply — a
docs-only change should not sit through an unrelated test suite, and a run that was skipped must be
**named as skipped in the report** rather than passing silently as green.

This gate is the loop's definition of done, not polish. In a loop with no gate, a defect does not
disappear — it re-enters later as a bug, as carryover, or underneath a second unit built on the
broken first, and the wall-clock is spent twice on work already reported complete.

If the diff adds code paths with no matching test, **say so and offer to add them**. Do not block
on it — that is the driver's call — but do not let it pass unmentioned either.

**A check that could not run is UNVERIFIED, and unverified is never a pass.** Report it as blocked
with the evidence; never fake a green.

### 3. Push

```
git push -u origin HEAD
```

Never force. A non-fast-forward rejection means the base moved: explain the rebase, and let the
human run it. `atl guard` blocks force variants from the assistant by design — that is the
guardrail working, not an obstacle to route around.

### 4. Open the PR — or update the one that exists

**Check first.** If a PR already exists for this branch, update its description and return its
URL. Never open a second: two PRs for one unit split the review and break the one-branch-one-PR
property the branch grammar exists to give.

Otherwise open it against **`config.branchPair.dev`** — read from config, never hardcoded:

```
gh pr create --base <dev> --title "<type>(<scope>): <subject>" --body-file <file>
```

Use `--body-file`, not `--body`: `atl guard` scans the entire Bash command string, so a body that
quotes a blocked command in prose gets the whole call denied.

The body must contain **`Fixes #<id>`** — that is the link mechanism on this backend.

### 5. Verify the link landed — the step that must not be skipped

Read it back:

```
gh api graphql -f query='{ repository(owner:"<o>", name:"<r>") {
  pullRequest(number: <pr>) { closingIssuesReferences(first:10) { nodes { number } } } } }'
```

The unit's id must appear. **If it does not, stop and say so loudly** — do not report the PR as
done. A PR that is not linked is a card that cannot be traced to its code, and nothing downstream
will notice.

One trap worth naming: **`Fixes #N` auto-closes an issue only on a merge to the repository's
default branch.** This flow merges to the integration branch, so the auto-close never fires here.
The reference still creates the link, which is what is being verified — but do not read
"the issue did not close" as "the link failed", and do not read "the link exists" as "the issue
will close itself".

### 6. Attach the evidence

On this backend evidence rides **in the PR**, committed under `docs/sprints/evidence/<unit>/…` —
durable and review-visible without a separate upload API. (On Azure evidence attaches to the work
item instead; the two are not the same shape, which is another reason this skill is backend-bound
rather than pretending to be neutral.)

Evidence is not decoration. A gate with evidence is a verification the rest of the loop can stand
on; a gate without it is a claim.

### 7. Report — and name what you did NOT do

```
PR #<n> opened → <url>
  unit:     #<id> — <title>
  base:     <dev>
  link:     verified (closingIssuesReferences contains #<id>)
  build:    <result | skipped — docs-only>
  tests:    <result | none run — no test-bearing paths touched>
  evidence: <path | none>

State stays <current>. Merge is yours; after it lands, /work-move <id> done.
```

### 8. Do not merge, and do not move the state

Both omissions are deliberate.

**Merge.** The delivery-team's carve-out from the never-merge rule is scoped to the *machine* — the
autonomous tech-lead worker merges a green PR to the integration branch. A hand-driven unit has no
such worker; the human at the keyboard is the reviewer, and reviewing your own PR by merging it
from a skill removes the only review the unit gets. Surface the URL and stop.

**State.** Merge happens first, Done after — so a Done never fronts an unlanded merge. Moving the
item here would assert a merge that has not happened. `/work-move` does it afterwards, and
`/work-sync` sweeps the ones that get forgotten.

**Promotion.** `dev`→`release` is not this skill's business at all: it is `atl work promote`, which
verifies a durable approval record against the PR's current head commit and merges in one call.
Re-deriving that read-compare-merge in prose here would re-create exactly the skippable path that
command exists to close.

## Idempotent re-run

Safe by construction, and the safety is worth being explicit about:

- The preflight re-runs — cheap, and it re-proves the gate on whatever changed since.
- Step 4 **finds the existing PR and updates it** rather than opening a second.
- Step 5 re-verifies the link; a link that was fine stays fine, and one that silently never landed
  gets caught on this pass instead of never.
- Nothing about the item's state is touched, so a re-run cannot corrupt the board.
