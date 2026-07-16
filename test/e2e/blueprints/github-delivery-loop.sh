#!/usr/bin/env bash
# needs: gh+token
#
# github-delivery-loop — the delivery-team's autonomous loop on the REAL GitHub
# backend (Issues / Projects v2 / Pull Requests). This is the GitHub Layer-B and the
# T-point signal (D6, #212): a real `claude -p` ceremony chain + a developer->tech-lead
# micro-loop against a real fixture repo (<login>/atl-e2e-delivery) + a fresh GitHub
# Project, driven by `gh` (GH_TOKEN). There is NO mock — this proves the github adapter
# (teams/delivery-team/backends/github/adapter.md) end-to-end against the live surface,
# the github twin of the real-Azure Layer-B (atl#102-104). It needs BOTH a GH_TOKEN
# (repo + project scope) and a Claude token, so it is the one `needs: gh+token` blueprint.
#
# Repeatable: reset_delivery_repo restores the repo baseline (main/dev/release, no stale
# issues/PRs/branches/tags) and reset_delivery_project recreates a clean board each run.
#
# Two assertion tiers (the same deterministic-plumbing / non-deterministic-LLM split as
# delivery-loop.sh):
#   * CORE (ok/bad) — the reliable e2e plumbing that must hold: install, kickoff's
#     Epic+Feature issues + the [Technical Analysis] sentinel comment, sprint-start's
#     valid plan.json, the developer->tech-lead PR merged to `dev` + its issue CLOSED
#     (the adapter §10 completion gate), and the sprint-review dev->release PR. A
#     regression here fails the test.
#   * NOTE (ok/note) — the less-deterministic ceremony field-writes (refine PBIs,
#     area/atl-key labels, the Project Iteration/Status/Story-Points writes, the docs/
#     seed). Across runs an LLM turn may skip one; a miss is NOTED, not failed — a
#     ceremony-fidelity concern, not an e2e-plumbing one.
#
# SETUP (one-time — see test/e2e/README.md): the fixture repo <login>/atl-e2e-delivery
# must exist, and the token must carry `repo` + `project` scope (the Project board is
# created per run). The full `atl work dispatch` engine run on GitHub (real worker
# spawn) is a follow-on — this blueprint proves the github ADAPTER's issue/PR/merge/
# close contract via a claude -p micro-loop (the engine itself is provider-agnostic and
# already proven for Azure by work-dispatch.sh + atl#102-104).

source /e2e/lib.sh
note() { echo "  note - $1"; }

gh auth setup-git >/dev/null 2>&1 || true
LOGIN=$(gh_login)
[ -n "$LOGIN" ] || { bad "gh not authenticated"; finish; exit 1; }
# The fixture repo + Project live in the agentteamland org (ATL's own infra), NOT the
# runner's personal namespace — overridable for a fork via ATL_E2E_DELIVERY_OWNER. The
# runner's token still authenticates; it just needs repo+project rights on the owner.
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

# $PROJ = a clone of the fixture repo the ceremonies + worker operate on (fresh emptied it).
rmdir "$PROJ" 2>/dev/null || true
git clone -q "https://github.com/$REPO.git" "$PROJ" || bad "clone of $REPO failed"
cd "$PROJ" || exit 2

# delivery-team is project-scope -> install into the project (reflects ceremonies +
# knowledge + scripts into $PROJ/.claude per the stone #3 reflection contract).
atl install agentteamland/delivery-team >/dev/null 2>&1 || bad "install errored"
[ -f "$PROJ/.claude/skills/kickoff/SKILL.md" ]      && ok "kickoff skill reflected"      || bad "kickoff skill missing"
[ -f "$PROJ/.claude/skills/sprint-review/SKILL.md" ] && ok "sprint-review skill reflected" || bad "sprint-review skill missing"

# ---- seed the project's .delivery/ config (GitHub shape) + methodology ------------
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

# a ceremony/worker turn: real claude -p; the delivery-team reaches GitHub through `gh`
# (which reads GH_TOKEN from the env) — NO --mcp-config (github is gh-native, not an MCP).
gturn() { ( cd "$PROJ" && claude -p "$1" --dangerously-skip-permissions --output-format json ) >>"$HOME/turns.log" 2>&1; }
# issue-count with filters (>=1 -> the fact held); labels/comments queried inline below.
ic()    { gh issue list --repo "$REPO" "$@" --json number -q 'length' 2>/dev/null || echo 0; }
ge()    { [ "${1:-0}" -ge 1 ] 2>/dev/null; }   # "got at least one"

# ---- 1. /kickoff — greenfield cold-start: Epic + Feature issues + analysis + docs --
gturn "/kickoff. You are ALSO acting as the human product owner for this headless run — answer intake from these facts, do not wait for interactive input. Project 'Tasky': a simple task-tracking web app for small teams. Problem: teams lose track of who owns what. Goals: create tasks, assign an owner, mark complete, see a team board. Out of scope: billing, mobile. Create the first Epic and at least one Feature as GitHub issues (gh issue create), label them 'type:epic' / 'type:feature', put the business framing in each issue BODY under the fixed H2s, add one issue comment whose FIRST LINE is the exact sentinel '**[Technical Analysis]**', and seed a docs/domain/ and a docs/architecture/ page in the repo (via the local checkout + commit + push, or the Contents API). Stamp each created issue with an 'atl-key:<shorthash>' label. Skip sprint-0." || bad "kickoff turn errored"

ge "$(ic --label type:epic --state all)"    && ok "kickoff created a type:epic issue"    || bad "no type:epic issue in $REPO"
ge "$(ic --label type:feature --state all)" && ok "kickoff created a type:feature issue" || bad "no type:feature issue in $REPO"
# CORE: the [Technical Analysis] sentinel comment on some issue (content-placement §7)
ta=0
for n in $(gh issue list --repo "$REPO" --state all --limit 50 --json number -q '.[].number' 2>/dev/null); do
  gh issue view "$n" --repo "$REPO" --comments 2>/dev/null | grep -q '\[Technical Analysis\]' && { ta=1; break; }
done
[ "$ta" = 1 ] && ok "a [Technical Analysis] sentinel comment landed (§7)" || bad "no [Technical Analysis] comment"
# NOTE: docs/ seed + atl-key labels (ceremony-fidelity, LLM-variable)
if gh api "repos/$REPO/contents/docs/domain" >/dev/null 2>&1 || gh api "repos/$REPO/contents/docs/architecture" >/dev/null 2>&1; then ok "kickoff seeded a docs/domain or docs/architecture page (§9)"; else note "no docs/ namespace page this run (LLM-variable)"; fi
akc=$(gh issue list --repo "$REPO" --state all --json labels -q '[.[].labels[].name | select(startswith("atl-key:"))] | length' 2>/dev/null || echo 0)
ge "$akc" && ok "created issues carry an atl-key label (idempotency §5)" || note "no atl-key label this run (LLM-variable)"

# ---- 2. /refine — decompose the Feature into keyed, area-labelled PBI issues -------
gturn "/refine. Groom and decompose the analyzed Feature(s) into implementable work-units (PBIs) as GitHub issues under them (each nested via the sub_issues REST endpoint, adapter §1). Give each a Priority, record at least one dependency as a '## Depends On' line (#N) in the tech-lead's canonical-brief comment (adapter §8), and converge on existing items — do not duplicate the kickoff Epic/Feature. Label each work-unit 'area:web' (this is a web project) and stamp an 'atl-key:<shorthash>' label." || bad "refine turn errored"

# refine is the less-deterministic ceremony -> NOTE, not fail:
pbi=$(gh issue list --repo "$REPO" --state all --json labels -q '[.[] | select((.labels // []) | map(.name) | any(. == "area:web"))] | length' 2>/dev/null || echo 0)
ge "$pbi" && ok "refine produced area:web work-units (PBIs)" || note "no area:web PBI this run (LLM-variable: refine is less-deterministic)"

# ---- 3. /sprint-plan — velocity (cold-start), admit units onto the board ----------
gturn "/sprint-plan. This is a cold-start project (no closed sprints), so use the po-seed velocity path. The candidate backlog is the open PBI issues (label area:web) not yet on the board. Select the top units and admit them to the current sprint by setting their Projects v2 Iteration field (add the issue to Project #$PROJNUM and set Iteration). Report the seed velocity used." || bad "sprint-plan turn errored"

items=$(gh project item-list "$PROJNUM" --owner "$OWNER" --format json -q '.items | length' 2>/dev/null || echo 0)
ge "$items" && ok "sprint-plan added units to the Project board" || note "no board items this run (LLM-variable; sprint-start still derives the plan below)"

# ---- 4. /sprint-start — build the DAG + materialize plan.json (NO dispatch) --------
gturn "/sprint-start. Read the sprint's admitted work-units (the area:web PBIs; if none are on the board yet, use the open area:web PBIs), read each unit's '## Depends On' lines to build the dependency DAG, validate it is acyclic, and materialize .delivery/plan.json in the exact dispatch.Plan schema (sprintSlug, granularity, units[] with id/title/predecessors/stackRank). This is a ceremony test: STOP after writing plan.json — do NOT run 'atl work dispatch'. There are no mobile-tagged units, so skip the emulator preflight." || bad "sprint-start turn errored"

if [ -f "$PROJ/.delivery/plan.json" ] && jq -e '.' "$PROJ/.delivery/plan.json" >/dev/null 2>&1; then
  ok "sprint-start materialized a valid .delivery/plan.json"
  jq -e 'has("sprintSlug") and has("granularity") and (.units | type == "array")' "$PROJ/.delivery/plan.json" >/dev/null 2>&1 && ok "plan.json matches the dispatch.Plan skeleton" || bad "plan.json skeleton malformed"
  if jq -e '.units | length >= 1 and (.[0] | has("id") and has("predecessors") and has("stackRank"))' "$PROJ/.delivery/plan.json" >/dev/null 2>&1; then ok "plan.json carries populated units"; else note "plan.json units empty this run (refine produced no PBIs upstream; LLM-variable chain)"; fi
else
  bad "no valid .delivery/plan.json materialized"
fi

# ---- 5. developer -> tech-lead micro-loop: PR to dev, merge --merge, close (§10) ---
# The github-adapter completion gate (§10): a developer opens a PR to dev referencing
# the issue, the tech-lead merges it with a REAL merge commit (gh pr merge --merge, so
# the engine's MergedToBase stays valid), then explicitly closes the issue (Fixes #N
# only auto-closes on the DEFAULT branch; the flow merges to dev) and sets Status=Done.
# Baseline the merged-into-dev PR count BEFORE the micro-loop: a merged PR is an
# immutable GitHub record the repo reset CANNOT remove, so assert an INCREASE this run,
# never an all-time count (which would false-pass on every run after the first).
prev_dev=$(gh pr list --repo "$REPO" --base dev --state merged --json number -q 'length' 2>/dev/null || echo 0)
gturn "Act as the developer, then the tech-lead, for ONE open area:web PBI issue (pick the lowest-numbered open one; if none, pick any open type:feature issue). DEVELOPER: from the local checkout create a branch off 'dev', add a small real change to app.js (e.g. extend the add function or add a sub function) plus a matching case in app.test.js, run 'node --test' to confirm it passes, commit, push the branch, and open a PR into 'dev' with 'Fixes #<n>' in the body (gh pr create --base dev). TECH-LEAD: review the PR, then merge it with 'gh pr merge --merge' ONLY (never --squash/--rebase), then 'gh issue close #<n>' and set the issue's Project #$PROJNUM Status field to Done. Report the PR number and the merged issue number." || bad "developer/tech-lead micro-loop turn errored"

# CORE: a NEW PR to dev was MERGED this run (real merge commit -> MergedToBase valid)
mrg=$(gh pr list --repo "$REPO" --base dev --state merged --json number -q 'length' 2>/dev/null || echo 0)
{ [ "$mrg" -gt "$prev_dev" ]; } 2>/dev/null && ok "a developer PR was merged into dev this run (adapter §10, real merge commit)" || bad "no NEW merged PR into dev"
# CORE: an issue is CLOSED (the completion gate; auto-close doesn't fire on the dev base).
# Sound because reset_delivery_repo verified a zero-issue baseline, so any closed issue is
# this run's — kickoff/refine only create OPEN issues; only the micro-loop closes one.
clc=$(ic --state closed)
ge "$clc" && ok "the worked issue was closed on merge-verify (§10)" || bad "no closed issue after the micro-loop"
# NOTE: Project Status=Done (the built-in automation only sets Done on close/merge; the
# ceremony may or may not have set it explicitly -> ceremony-fidelity)
done_ct=$(gh project item-list "$PROJNUM" --owner "$OWNER" --format json -q '[.items[] | select((.status // "") == "Done")] | length' 2>/dev/null || echo 0)
ge "$done_ct" && ok "a board item reached Status=Done" || note "no board item at Status=Done this run (LLM-variable; the close still happened)"

# ---- 6. /sprint-review — report to docs/sprints + PO approve -> dev->release PR ----
# Baseline the dev->release PR count FIRST: its head is `dev` (never branch-deleted by
# the reset), so the PR record persists across runs — assert an INCREASE this run.
prev_rel=$(gh pr list --repo "$REPO" --base release --state all --json number -q 'length' 2>/dev/null || echo 0)
gturn "/sprint-review. You are ALSO acting as the human product owner for this headless run. Compile the Sprint Review Report and upsert it to docs/sprints/sprint-1-review.md in the repo (the in-repo durable-knowledge store). Then, at the Approve/Reject gate, APPROVE the sprint — open the dev->release promotion PR (gh pr create --base release --head dev). Do not wait for interactive input; approve based on this instruction." || bad "sprint-review turn errored"

# CORE: a NEW dev->release promotion PR opened this run
rel=$(gh pr list --repo "$REPO" --base release --state all --json number -q 'length' 2>/dev/null || echo 0)
{ [ "$rel" -gt "$prev_rel" ]; } 2>/dev/null && ok "PO-approved dev->release promotion PR opened this run (§10)" || bad "no NEW dev->release PR opened"
# NOTE: the sprint-review page written to docs/sprints/
if gh api "repos/$REPO/contents/docs/sprints/sprint-1-review.md" >/dev/null 2>&1; then ok "sprint-review upserted a docs/sprints review page (§9)"; else note "no docs/sprints review page this run (LLM-variable)"; fi

# ---- on failure, surface what the torn-down container would otherwise lose ---------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (github-delivery-loop failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- gh --version ---";     gh --version 2>&1 | head -1
  echo "--- turns.log (tail) ---"; tail -100 "$HOME/turns.log" 2>/dev/null
  echo "--- issues ---";           gh issue list --repo "$REPO" --state all --json number,title,state,labels 2>/dev/null
  echo "--- PRs ---";              gh pr list --repo "$REPO" --state all --json number,baseRefName,state 2>/dev/null
  echo "--- board items ---";      gh project item-list "$PROJNUM" --owner "$OWNER" --format json 2>/dev/null | jq '[.items[] | {title, status}]' 2>/dev/null
  echo "--- plan.json ---";        cat "$PROJ/.delivery/plan.json" 2>/dev/null
  echo "==============================================="
fi

finish
