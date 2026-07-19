#!/usr/bin/env bash
# needs: gh+token
#
# github-delivery-engine — the delivery-team's Go orchestration engine (`atl work
# dispatch`) driving a REAL `claude -p` worker pipeline (developer -> tester -> tech-lead)
# on the GitHub backend. This is the follow-on the github-delivery-loop blueprint names:
# github-delivery-loop proves the ADAPTER via a hand-written gturn micro-loop; THIS proves
# the ENGINE spawns real workers whose (backend-neutral, post-neutralization) prompts reach
# GitHub via `gh` and land a real merge to `dev`. The github twin of the real-Azure engine
# run (atl#102-104); the deterministic engine orchestration itself is covered by
# work-dispatch.sh (fake worker).
#
# Isolation: the ceremony chain (kickoff/refine/sprint-plan/sprint-start) is already proven
# by github-delivery-loop.sh, so this blueprint SEEDS one buildable PBI + its plan.json
# deterministically and tests ONLY the new intersection — engine + real worker + GitHub.
# That keeps the proof focused and free of upstream ceremony LLM-variance.
#
# Two assertion tiers (the deterministic-plumbing / non-deterministic-LLM split):
#   * CORE (ok/bad) — the plumbing that MUST hold: install + adapter reflected, a valid
#     seeded plan, and — the heart of the test — the ENGINE's real worker opening a PR and
#     landing a REAL merge to `dev` (baseline-delta: a NEW merged PR this run) with the
#     seeded issue CLOSED (adapter §10 completion gate) and the merged worktree reclaimed.
#   * NOTE (ok/note) — LLM-variable worker fidelity: the exact code change, Status=Done,
#     the dispatch rc / "1 done" summary line (a blocked worker is surfaced by the CORE
#     merged-PR assertion, so these stay NOTE).

source /e2e/lib.sh
note() { echo "  note - $1"; }
ge()   { [ "${1:-0}" -ge 1 ] 2>/dev/null; }   # "got at least one"

command -v node >/dev/null 2>&1 && ok "node present (workers run node --test)" || bad "node missing in the image"

gh auth setup-git >/dev/null 2>&1 || true
LOGIN=$(gh_login)
[ -n "$LOGIN" ] || { bad "gh not authenticated"; finish; exit 1; }
OWNER="${ATL_E2E_DELIVERY_OWNER:-agentteamland}"
REPO="$OWNER/atl-e2e-delivery"

reset_delivery_repo "$OWNER" \
  && ok "reset delivery fixture repo to baseline (main/dev/release)" \
  || bad "could not reset $REPO (does it exist? token 'repo' rights on $OWNER?)"
PROJNUM=$(reset_delivery_project "$OWNER")
[ -n "$PROJNUM" ] && ok "created a fresh GitHub Project #$PROJNUM" || bad "could not create the Project (token 'project' scope?)"

fresh
write_test_index_delivery
headless_claude_setup

# $PROJ = a clone of the fixture repo the engine + workers operate on.
rmdir "$PROJ" 2>/dev/null || true
git clone -q "https://github.com/$REPO.git" "$PROJ" || bad "clone of $REPO failed"
cd "$PROJ" || exit 2

# delivery-team is project-scope -> install into the project (reflects agents + packs +
# knowledge + backends into $PROJ/.claude; the workers read them by absolute path).
atl install agentteamland/delivery-team >/dev/null 2>&1 || bad "install errored"
[ -f "$PROJ/.claude/agents/developer/agent.md" ]        && ok "developer agent reflected"  || bad "developer agent missing"
[ -f "$PROJ/.claude/backends/github/adapter.md" ]       && ok "github adapter reflected"    || bad "github adapter missing"
[ -d "$PROJ/.claude/packs/web" ]                        && ok "web pack reflected"          || note "web pack missing (worker may block on area load)"

# ---- .delivery/ config (GitHub shape) + methodology --------------------------------
mkdir -p "$PROJ/.delivery"
cat > "$PROJ/.delivery/config.json" <<EOF
{
  "owner": "$OWNER",
  "repo": "atl-e2e-delivery",
  "projectNumber": $PROJNUM,
  "branchPair": { "dev": "dev", "release": "release" },
  "backend": "github",
  "methodology": "scrum",
  "credential": { "ref": "GH_TOKEN" }
}
EOF
cat > "$PROJ/.delivery/methodology.json" <<'EOF'
{
  "id": "scrum",
  "displayName": "Scrum",
  "roles": [
    { "name": "intake",            "binding": "agent", "dispatch": "in-session" },
    { "name": "business-analyst",  "binding": "agent", "dispatch": "subagent" },
    { "name": "technical-analyst", "binding": "agent", "dispatch": "subagent" },
    { "name": "project-manager",   "binding": "agent", "dispatch": "subagent" },
    { "name": "tech-lead",         "binding": "agent", "dispatch": "subagent" },
    { "name": "tester",            "binding": "agent", "dispatch": "worker" },
    { "name": "developer",         "binding": "agent", "dispatch": "worker", "instances": "dynamic" },
    { "name": "product-owner",     "binding": "human" }
  ],
  "artifactHierarchy": ["Epic", "Feature", "Pbi", "Task"],
  "workItemTypeMap": { "Pbi": null, "Task": null, "Bug": null },
  "cadence": { "unit": "sprint", "planningCeremonies": ["sprint-plan", "sprint-start"], "reviewCeremony": "sprint-review" },
  "capacityModel": { "velocityWindowN": 3, "unit": "storyPoints", "coldStart": "po-seed", "seedVelocity": null, "availabilityFactorDefault": 1.0 },
  "branches": { "dev": "dev", "release": "release" }
}
EOF
ok "seeded .delivery/config.json (github) + methodology.json"

# ---- SEED one buildable, self-testing PBI + its analysis/brief ---------------------
# The fixture's app.js has add(); the unit asks for a sibling subtract() + a node --test
# case. Deliberately tiny so the proof is the delivery mechanics (engine -> real worker ->
# gh PR -> merge to dev -> close), not the app. The engine hands the worker ONLY the id;
# the worker fetches the issue + the sentinel comments + the area pack from GitHub.
gh label create "area:web" --repo "$REPO" -c "#1f883d" -d "web area" >/dev/null 2>&1 || true
gh label create "type:pbi" --repo "$REPO" -c "#0969da" -d "product backlog item" >/dev/null 2>&1 || true

cat > "$HOME/issue-body.md" <<'EOF'
## Problem
The fixture app exposes `add(a, b)` but no subtraction.

## Business Value
A second pure function lets the delivery loop exercise a real implement + test cycle.

## Scope
Add `subtract(a, b)` to `app.js` and export it; cover it in `app.test.js`.

## Acceptance Criteria
- `app.js` exports `subtract(a, b)` returning `a - b` (mirroring `add`).
- `app.test.js` has a `node --test` case asserting `subtract(5, 3) === 2`, and `node --test` passes.

## Out of Scope
Anything beyond `subtract` (no CLI, no formatting, no refactor of `add`).
EOF
ISSUE_URL=$(gh issue create --repo "$REPO" \
  --title "Add a subtract(a,b) function to app.js" \
  --label "area:web" --label "type:pbi" \
  --body-file "$HOME/issue-body.md" 2>/dev/null)
ISSUE=$(echo "$ISSUE_URL" | grep -oE '[0-9]+$')
[ -n "$ISSUE" ] && ok "seeded a buildable PBI (#$ISSUE, area:web)" || { bad "could not seed the PBI"; finish; exit 1; }

# The technical-analyst's sentinel comment (so the worker's "read [Technical Analysis]"
# step finds it and does not block on a missing analysis).
cat > "$HOME/ta.md" <<'EOF'
**[Technical Analysis]**

## Approach
Add a pure `subtract(a, b)` beside `add` in app.js; export it in module.exports. Add a `node:test` case in app.test.js mirroring the add test.

## Feasibility & Risks
Trivial, no dependencies, no risk.

## NFRs
None.

## Dependencies
None.

## Suggested Areas
web
EOF
gh issue comment "$ISSUE" --repo "$REPO" --body-file "$HOME/ta.md" >/dev/null 2>&1 \
  && ok "seeded the [Technical Analysis] sentinel comment" || note "TA comment not added"

# The tech-lead's Canonical Brief (the per-unit contract the developer worker reads).
cat > "$HOME/brief.md" <<'EOF'
**[Canonical Brief]**

## Goal
Add `subtract(a, b)` (returns `a - b`) to app.js and export it; add a `node --test` case in app.test.js asserting `subtract(5, 3) === 2`.

## Area
web

## Load These Pages
(none — a trivial pure function; no Architecture/Conventions page needed)

## Depends On
(none)

## Evidence Before Review
`node --test` passes with the new subtract case included.
EOF
gh issue comment "$ISSUE" --repo "$REPO" --body-file "$HOME/brief.md" >/dev/null 2>&1 \
  && ok "seeded the [Canonical Brief] sentinel comment" || note "brief comment not added"

gh project item-add "$PROJNUM" --owner "$OWNER" --url "$ISSUE_URL" >/dev/null 2>&1 \
  && ok "added the unit to Project #$PROJNUM" || note "unit not added to the board"

# plan.json referencing the seeded issue (one unit, no predecessors) — the exact
# dispatch.Plan schema the engine reads.
cat > "$PROJ/.delivery/plan.json" <<EOF
{"sprintSlug":"e2e-1","granularity":"pbi","units":[{"id":$ISSUE,"title":"Add subtract to app.js","predecessors":[],"stackRank":1}]}
EOF
jq -e '.units | length == 1 and (.[0].id | type == "number")' "$PROJ/.delivery/plan.json" >/dev/null 2>&1 \
  && ok "materialized a valid plan.json for the seeded unit" || bad "plan.json malformed"

# ---- RUN THE ENGINE: real claude -p workers (NO fake on PATH) ----------------------
# The real `claude` (in the image) is used — the engine spawns developer -> tester ->
# tech-lead workers with the neutralized prompts. GH_TOKEN reaches the workers via the
# engine's deliveryWorkerEnv injection (workerenv.go), so they authenticate `gh`.
export GH_TOKEN="${GH_TOKEN:-$(gh auth token 2>/dev/null)}"
[ -n "$GH_TOKEN" ] && ok "GH_TOKEN exported for worker injection" || bad "no GH_TOKEN for the workers"

# Baseline the merged-into-dev PR count BEFORE dispatch: a merged PR is an immutable
# GitHub record the repo reset CANNOT remove, so assert an INCREASE this run, never an
# all-time count (which would false-pass on every run after the first).
prev_dev=$(gh pr list --repo "$REPO" --base dev --state merged --limit 400 --json number -q 'length' 2>/dev/null || echo 0)

echo ">> running: atl work dispatch --cap 1 (real claude -p workers) ..."
out="$(cd "$PROJ" && atl work dispatch --cap 1 2>&1)"; rc=$?
echo "$out" | tail -25

# NOTE: the engine's own summary (LLM worker may block → surfaced by the CORE PR assertion)
[ "$rc" -eq 0 ] && ok "engine dispatch returned rc=0" || note "dispatch rc=$rc (a blocked worker is a real signal — the merged-PR gate below is the CORE truth)"
echo "$out" | grep -q "complete: 1 done" && ok "the engine reported the unit done" || note "engine did not print 'complete: 1 done' (LLM worker fidelity; see the merged-PR gate)"

# CORE: the ENGINE's real worker opened + landed a merge to dev THIS run.
mrg=$(gh pr list --repo "$REPO" --base dev --state merged --limit 400 --json number -q 'length' 2>/dev/null || echo 0)
{ [ "$mrg" -gt "$prev_dev" ]; } 2>/dev/null \
  && ok "the ENGINE's real worker merged a PR into dev (neutralized prompt reached gh; adapter §10 real merge commit)" \
  || bad "no NEW merged PR into dev — the engine's real worker did not land a merge"

# CORE: the seeded issue was CLOSED (the §10 completion gate; auto-close doesn't fire on
# the dev base, so the tech-lead worker must have explicitly closed it).
st=$(gh issue view "$ISSUE" --repo "$REPO" --json state -q '.state' 2>/dev/null)
[ "$st" = "CLOSED" ] && ok "the worked issue #$ISSUE was closed on merge-verify (§10)" || bad "issue #$ISSUE not closed (state=$st)"

# CORE: the merged unit's worktree was reclaimed (engine teardown after merge-verify).
leftover="$(ls "$PROJ/.delivery/worktrees/e2e-1" 2>/dev/null | wc -l | tr -d ' ')"
[ "$leftover" = "0" ] && ok "the merged unit's worktree was reclaimed" || note "worktree leftover: $leftover (engine teardown)"
[ -f "$PROJ/.delivery/runstate.json" ] && ok "run-state persisted (restart substrate)" || note "no runstate.json"

# NOTE: Status=Done on the board (built-in automation + the ceremony's explicit set)
done_ct=$(gh project item-list "$PROJNUM" --owner "$OWNER" --format json -q '[.items[] | select((.status // "") == "Done")] | length' 2>/dev/null || echo 0)
ge "$done_ct" && ok "a board item reached Status=Done" || note "no board item at Status=Done (LLM-variable; the close still happened)"

# NOTE: the actual code change landed on dev
subtract_on_dev=$(gh api "repos/$REPO/contents/app.js?ref=dev" -q '.content' 2>/dev/null | base64 -d 2>/dev/null | grep -c 'subtract')
ge "$subtract_on_dev" && ok "subtract() is present in app.js on dev (the real implementation landed)" || note "subtract() not found on dev (worker's code choice; the merge still landed)"

# ---- on failure, surface what the torn-down container would otherwise lose ---------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (github-delivery-engine failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- dispatch output (full) ---"; echo "$out"
  echo "--- blocked reports ---"; cat "$PROJ"/.delivery/blocked/*.json 2>/dev/null
  echo "--- issues ---"; gh issue list --repo "$REPO" --state all --json number,title,state 2>/dev/null
  echo "--- PRs ---"; gh pr list --repo "$REPO" --state all --json number,baseRefName,state 2>/dev/null
  echo "--- worktrees ---"; ls -la "$PROJ"/.delivery/worktrees/ 2>/dev/null
  echo "--- runstate.json ---"; cat "$PROJ"/.delivery/runstate.json 2>/dev/null
  echo "==============================================="
fi

finish
