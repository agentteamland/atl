#!/usr/bin/env bash
# needs: gh+token
#
# github-delivery-autonomous-refine — the delivery-team's FULL ceremony chain on the GitHub
# backend, proving the ONE seam github-delivery-full-chain.sh (atl#271) deliberately leaves
# PRESCRIBED: /refine's AUTONOMOUS decomposition. That blueprint hands /refine an exact PBI
# list (count, files, dependency edges) and proves only the DAG->dispatch PLUMBING; here
# /refine is handed ONLY a vision-level Feature and must DECIDE the decomposition itself —
# how many PBIs, what each builds, which files it touches, and the dependency edges — then
# /sprint-start turns that into a MULTI-NODE plan.json and `atl work dispatch --cap 2` builds
# it in predecessor order with real workers. Chain: /kickoff -> /refine -> /sprint-plan ->
# /sprint-start -> dispatch, real `claude -p` throughout.
#
# What is NO LONGER prescribed (the delta over atl#271):
#   * /kickoff seeds ONE Feature at VISION level — a capability ("difference operations built
#     on a shared subtract primitive") with fixed spec-field H2s + a [Technical Analysis]
#     sentinel whose guidance is ARCHITECTURAL (foundation-first; modular one-file-per-helper
#     so units are conflict-free), NOT a named PBI list.
#   * /refine reads that Feature + analysis and AUTONOMOUSLY produces the PBIs, their files,
#     and their `## Depends On` edges. Count/files/edges are refine's call, not the harness's.
# A sound autonomous decomposition here is a subtract() foundation + >=1 derived difference
# helper in a DISJOINT file depending on it — but that is for /refine to ARRIVE at, not be told.
#
# HONEST SCOPE: this proves /refine can PRODUCE a sound multi-node decomposition (count/files/
# edges) from a vision Feature + architectural analysis and drive it through the real engine —
# the autonomous-sizing seam atl#271 left open. It does NOT test a vague/under-specified or a
# large Feature; the [Technical Analysis] supplies foundation-first + modular guidance (a real
# technical-analyst's job) and refine decides the breakdown from it. A vaguer/larger Feature is
# a follow-on.
#
# Two assertion tiers (the deterministic-plumbing / non-deterministic-LLM split):
#   * CORE (ok/bad) — the seam that MUST hold: install, kickoff's Epic+Feature+[Technical
#     Analysis], refine's >=2 area:web PBIs, a MULTI-NODE plan.json (>=2 units, >=1 edge),
#     the engine reporting >=2 units DONE (merge-verified, the engine's own summary — immune
#     to a squash/rebase that GitHub shows MERGED but MergedToBase saw as unmerged) AND
#     >=2 NEW merges to dev, with at least one dependent issue CLOSED no earlier than the
#     foundation (the scheduler enforced the edge).
#   * NOTE (ok/note) — LLM-variable fidelity: the full 3-way fan-out, worktree reclaim,
#     Status=Done, the exact code on dev, sprint-plan's Iteration write (the fixture Project
#     has no Iteration field — gh cannot create it — so sprint-plan stays NOTE and
#     sprint-start derives the DAG from the PBIs' `## Depends On` briefs, not the board).
#     The workers run against the no-per-PBI-TA steady state (they read the [Technical
#     Analysis] from the ancestor Feature, delivery-team >= 0.5.2); the ancestor traversal
#     itself is NOT independently asserted here — a trivial task completes from the brief.
#
# SETUP: same as github-delivery-full-chain — the fixture repo agentteamland/atl-e2e-delivery
# must exist and the token must carry repo + project scope (a fresh Project is created per
# run). needs: gh+token (a GH_TOKEN and a Claude token). The plumbing (multi-node DAG ->
# real-worker dispatch in dependency order) is already proven by github-delivery-full-chain;
# this run's delta is that /refine — not the harness — decides the decomposition.

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

# $PROJ = a clone of the fixture repo the ceremonies + engine + workers operate on.
rmdir "$PROJ" 2>/dev/null || true
git clone -q "https://github.com/$REPO.git" "$PROJ" || bad "clone of $REPO failed"
cd "$PROJ" || exit 2

# delivery-team is project-scope -> install into the project. This blueprint runs BOTH the
# ceremony skills AND the engine, so it needs the ceremony skills AND the worker agents/
# packs/backends reflected into $PROJ/.claude.
atl install agentteamland/delivery-team >/dev/null 2>&1 || bad "install errored"
[ -f "$PROJ/.claude/skills/kickoff/SKILL.md" ]      && ok "kickoff skill reflected"      || bad "kickoff skill missing"
[ -f "$PROJ/.claude/skills/refine/SKILL.md" ]       && ok "refine skill reflected"       || bad "refine skill missing"
[ -f "$PROJ/.claude/skills/sprint-start/SKILL.md" ] && ok "sprint-start skill reflected" || bad "sprint-start skill missing"
[ -f "$PROJ/.claude/agents/developer/agent.md" ]    && ok "developer agent reflected"    || bad "developer agent missing"
[ -f "$PROJ/.claude/backends/github/adapter.md" ]   && ok "github adapter reflected"      || bad "github adapter missing"
[ -d "$PROJ/.claude/packs/web" ]                    && ok "web pack reflected"            || note "web pack missing (worker may block on area load)"

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

# a ceremony turn: real claude -p; the delivery-team reaches GitHub through `gh` (which
# reads GH_TOKEN from the env) — NO --mcp-config (github is gh-native, not an MCP).
gturn() { ( cd "$PROJ" && claude -p "$1" --dangerously-skip-permissions --output-format json ) >>"$HOME/turns.log" 2>&1; }
ic()    { gh issue list --repo "$REPO" "$@" --json number -q 'length' 2>/dev/null || echo 0; }

# ---- 1. /kickoff — Epic + a DECOMPOSABLE Feature + [Technical Analysis] on the Feature
gturn "/kickoff. You are ALSO acting as the human product owner for this headless run — answer intake from these facts, do not wait for interactive input. Project 'Calc': a tiny arithmetic helper library (Node.js; the repo already has add(a,b) in app.js). Create ONE Epic and ONE Feature as GitHub issues (gh issue create); label them 'type:epic' / 'type:feature'; stamp each with an 'atl-key:<shorthash>' label. The Feature is a CAPABILITY stated at VISION level — do NOT pre-split it into named tasks, files, or a PBI count; that decomposition is /refine's job. Put the business framing in the FEATURE issue BODY under the fixed H2s: ## Problem (the library can add two numbers but cannot compute differences between them), ## Business Value (a complete difference toolkit for callers), ## Scope (the app should support computing the difference between two numbers AND difference-derived checks — e.g. the absolute difference and a zero-difference test — all built on ONE shared subtract primitive rather than duplicating subtraction), ## Acceptance Criteria (one independently-verifiable criterion per capability: a raw subtract, an absolute-difference helper, and a zero-difference-check helper — each with a node --test case), ## Out of Scope (mobile, UI, CLI). Add ONE comment ON THE FEATURE issue whose FIRST LINE is the exact sentinel '**[Technical Analysis]**', with ## Approach (subtract(a,b) is the shared foundation — establish it FIRST in app.js and export it; the difference-derived helpers each build on subtract's contract; favor a MODULAR layout — one helper per its own module/file — so the helpers are independent and can be built and merged in parallel), ## Feasibility & Risks (trivial; the only sequencing constraint is that the derived helpers require subtract first), ## NFRs (each helper covered by a node --test case), ## Dependencies (the difference-derived helpers each require the subtract foundation), ## Suggested Areas (web). Do NOT enumerate specific PBIs, file names, or a PBI count anywhere — leave the entire decomposition to /refine. Skip sprint-0." || bad "kickoff turn errored"

ge "$(ic --label type:epic --state all)" && ok "kickoff created a type:epic issue" || bad "no type:epic issue in $REPO"
FEAT=$(gh issue list --repo "$REPO" --label type:feature --state all --limit 5 --json number -q '.[0].number' 2>/dev/null)
[ -n "$FEAT" ] && ok "kickoff created a type:feature issue (#$FEAT)" || bad "no type:feature issue in $REPO"
# Fail-fast: with no Feature there is nothing to decompose — abort before the ~40-min
# refine/plan/start/dispatch budget is spent on a doomed run.
[ -n "$FEAT" ] || { echo "!! no Feature from kickoff — aborting before the expensive ceremonies/dispatch"; finish; exit 1; }
ta=0
if [ -n "$FEAT" ]; then gh issue view "$FEAT" --repo "$REPO" --comments 2>/dev/null | grep -q '\[Technical Analysis\]' && ta=1; fi
[ "$ta" = 1 ] && ok "a [Technical Analysis] sentinel comment landed on the Feature (§7)" || bad "no [Technical Analysis] comment on the Feature"

# ---- 2. /refine — decompose the Feature into 3 dependency-linked PBIs ----------------
gturn "/refine. Groom and decompose the Feature #$FEAT into PBIs — this is YOUR decomposition to make; NO PBI list is given. Read the Feature's spec-field H2s (## Scope, ## Acceptance Criteria) AND its '**[Technical Analysis]**' comment, and DECIDE the breakdown yourself: how many PBIs, what each one implements, which file(s) it touches, and the dependency edges between them — derive all of it from the analysis. Honor the Technical Analysis's architecture: subtract() is the shared foundation, established FIRST in app.js and exported; the difference-derived helpers each build on it; keep the layout MODULAR (one helper per its own file) so independent units touch DISJOINT files and can run concurrently under dispatch (two units that touched the same file would conflict on the second merge). Create each PBI with gh issue create, nested under #$FEAT via the sub_issues REST endpoint (adapter §1), each labelled with the area from the Feature's '## Suggested Areas' (web) AND 'type:pbi' AND stamped an 'atl-key:<shorthash>' label. Converge — do NOT duplicate the kickoff Epic/Feature, and do NOT re-create a PBI that already exists. For EACH PBI add a comment whose FIRST LINE is the exact sentinel '**[Canonical Brief]**' with H2s: ## Goal (the precise implementation YOU decided — INCLUDING the exact file name(s), the 'modify ONLY these files' constraint, and the node --test assertion that proves it), ## Area (web), ## Load These Pages (none), ## Depends On, ## Evidence Before Review ('node --test' passes with the new case). Under '## Depends On' list the '#<n>' issue number(s) of the PBI(s) this one depends on: a foundation PBI others build on gets 'none'; a dependent lists its foundation's '#<n>'. The '## Depends On #<n>' lines ARE the dependency edges /sprint-start reads to build the DAG — wire them to match the real code dependencies you chose; they are load-bearing." || bad "refine turn errored"

# CORE: refine produced >=2 area:web PBIs (the decomposition held)
PBI_CT=$(gh issue list --repo "$REPO" --state all --json labels -q '[.[] | select((.labels // []) | map(.name) | any(. == "area:web"))] | length' 2>/dev/null || echo 0)
{ [ "${PBI_CT:-0}" -ge 2 ]; } 2>/dev/null \
  && ok "refine produced $PBI_CT area:web PBIs (>=2 — a multi-node decomposition)" \
  || bad "refine produced <2 area:web PBIs ($PBI_CT) — no multi-node decomposition to dispatch"

# No per-PBI [Technical Analysis] backstop (removed with delivery-team >= 0.5.2). /refine writes
# the [Technical Analysis] at the FEATURE level; a decomposed PBI carries only its [Canonical
# Brief], and the developer/tester workers read the analysis from the nearest ancestor Feature
# via the parent link (backend-interface concept #3 fallback). Dropping the old shim (which
# seeded a TA on every PBI) makes this run exercise the real no-per-PBI-TA steady state instead
# of masking it. Honest caveat: the ancestor traversal is NOT independently asserted — this
# trivial task (subtract/absDiff/isZeroDiff) completes from the Canonical Brief alone, so a green
# run does not by itself prove the fallback fired; the fix's correctness is prose-verified (the
# worker prompts + the concept #3 rule + the adapters). This gate confirms only no-regression:
# the workers still complete with no per-PBI TA present.
own_ta=0
for n in $(gh issue list --repo "$REPO" --label area:web --state open --json number -q '.[].number' 2>/dev/null); do
  gh issue view "$n" --repo "$REPO" --comments 2>/dev/null | grep -q '\[Technical Analysis\]' && own_ta=$((own_ta + 1))
done
[ "$own_ta" = 0 ] \
  && note "no PBI carries its own [Technical Analysis] — workers ran the ancestor-Feature fallback steady state (traversal not independently asserted; trivial task completes from the brief)" \
  || note "$own_ta PBI(s) carry an own TA this run (refine-variable); the ancestor fallback still applies to the rest"

# ---- 3. /sprint-plan — cold-start; NOTE (the fixture Project has no Iteration field) -
gturn "/sprint-plan. You are ALSO acting as the human product owner. This is a cold-start project (no closed sprints) — use the po-seed velocity path with a seed velocity of 8 story points. Set a Story Points estimate (1-2 points each, they are small) on every open area:web PBI and admit them to the sprint. NOTE: the fixture GitHub Project has no Iteration field (it cannot be created via the CLI); if you cannot set Iteration, say so and proceed — /sprint-start will derive the plan from the open area:web PBIs. Report the seed velocity used." || bad "sprint-plan turn errored"
items=$(gh project item-list "$PROJNUM" --owner "$OWNER" --format json -q '.items | length' 2>/dev/null || echo 0)
ge "$items" && ok "sprint-plan added units to the Project board" || note "no board items this run (LLM-variable; sprint-start derives the plan from open PBIs)"

# ---- 4. /sprint-start — build the MULTI-NODE DAG + materialize plan.json (harness dispatches)
gturn "/sprint-start. Read the sprint's admitted work-units (the open area:web PBIs; if none are on the board, use ALL open area:web PBIs). Read each PBI's '**[Canonical Brief]**' comment '## Depends On' lines to build the dependency DAG (a '#<n>' line under ## Depends On means this unit depends on unit n). Validate the DAG is acyclic. Materialize .delivery/plan.json in the EXACT dispatch.Plan schema: {\"sprintSlug\":\"<fs-safe-slug>\",\"granularity\":\"pbi\",\"units\":[{\"id\":<issue#>,\"title\":\"<title>\",\"predecessors\":[<issue#>...],\"stackRank\":<n>}]}. Use the JSON key 'stackRank' (the engine accepts 'priority' too, but this blueprint's assertions check 'stackRank'). There are no mobile-tagged units, so skip the emulator preflight. STOP after writing plan.json — do NOT run 'atl work dispatch'; the harness drives the engine." || bad "sprint-start turn errored"

# CORE checkpoint: plan.json exists, valid, and a MULTI-NODE DAG (>=2 units, >=1 real edge, stackRank keys)
if [ -f "$PROJ/.delivery/plan.json" ] && jq -e '.' "$PROJ/.delivery/plan.json" >/dev/null 2>&1; then
  ok "sprint-start materialized a valid .delivery/plan.json"
  jq -e 'has("sprintSlug") and has("granularity") and (.units | type == "array")' "$PROJ/.delivery/plan.json" >/dev/null 2>&1 \
    && ok "plan.json matches the dispatch.Plan skeleton" || bad "plan.json skeleton malformed"
  if jq -e '(.units | length >= 2) and (.units | any(.predecessors | length >= 1)) and (.units | all(has("stackRank")))' "$PROJ/.delivery/plan.json" >/dev/null 2>&1; then
    ok "plan.json is a MULTI-NODE DAG (>=2 units, >=1 predecessor edge, stackRank keys) — the seam"
  else
    bad "plan.json is NOT a multi-node DAG (need >=2 units, >=1 predecessor edge, stackRank keys) — units: $(jq -c '.units' "$PROJ/.delivery/plan.json" 2>/dev/null)"
  fi
else
  bad "no valid .delivery/plan.json materialized"
fi

# ---- 5. DRIVE THE ENGINE: atl work dispatch --cap 2 (real claude -p workers) --------
export GH_TOKEN="${GH_TOKEN:-$(gh auth token 2>/dev/null)}"
[ -n "$GH_TOKEN" ] && ok "GH_TOKEN exported for worker injection" || bad "no GH_TOKEN for the workers"

SLUG=$(jq -r '.sprintSlug' "$PROJ/.delivery/plan.json" 2>/dev/null)
UNIT_CT=$(jq '.units | length' "$PROJ/.delivery/plan.json" 2>/dev/null || echo 0)
A_ID=$(jq -r '[.units[] | select((.predecessors|length)==0) | .id] | first // empty' "$PROJ/.delivery/plan.json" 2>/dev/null)
DEP_IDS=$(jq -r '.units[] | select((.predecessors|length)>=1) | .id' "$PROJ/.delivery/plan.json" 2>/dev/null)

# Baseline the merged-into-dev PR count BEFORE dispatch: merged PRs are immutable GitHub
# records the repo reset CANNOT remove, so assert an INCREASE this run, never an all-time count.
prev_dev=$(gh pr list --repo "$REPO" --base dev --state merged --limit 400 --json number -q 'length' 2>/dev/null || echo 0)

# PRECONDITION: agentteamland/atl-e2e-delivery must allow MERGE COMMITS — every unit lands
# via `gh pr merge --merge`, and the engine's MergedToBase (worktree.go) false-blocks a
# squash/rebase (rewritten SHAs). The passing loop/engine blueprints rely on the same setting.
echo ">> running: atl work dispatch --cap 2 (real claude -p workers over a $UNIT_CT-unit DAG; slug=$SLUG) ..."
out="$(cd "$PROJ" && atl work dispatch --cap 2 2>&1)"; rc=$?
echo "$out" | tail -30

# NOTE: engine exit code (a blocked worker is a real signal — the CORE gates below are authoritative)
[ "$rc" -eq 0 ] && ok "engine dispatch returned rc=0" || note "dispatch rc=$rc (a blocked worker is a real signal — the CORE gates below are authoritative)"

# CORE: the engine's OWN summary reports >=2 units DONE. The count is anchored to the
# engine's terminal summary line "dispatch complete: N done, ..." (work.go) — NOT any
# "[0-9]+ done" in the stream, which would also match the per-unit progress line
# "unit <id> done — merged to dev" and capture a WORK-ITEM ID instead of the done count.
# The summary counts ONLY merge-verified (MergedToBase) units — a --squash/--rebase-blocked
# unit is "blocked", NOT "done" — so this gate is immune to the "GitHub shows MERGED but the
# engine saw the branch as unmerged" false-pass.
done_count=$(echo "$out" | grep -oE 'complete: [0-9]+ done' | tail -1 | grep -oE '[0-9]+' | head -1)
{ [ "${done_count:-0}" -ge 2 ]; } 2>/dev/null \
  && ok "the engine reported >=2 units DONE (merge-verified completion: '$done_count done')" \
  || bad "the engine reported <2 units done (${done_count:-none}) — the multi-node dispatch did not complete (see DEBUG for blocked units)"

# CORE: GitHub-side cross-check — >=2 NEW merges to dev this run (real merge commits, §10).
mrg=$(gh pr list --repo "$REPO" --base dev --state merged --limit 400 --json number -q 'length' 2>/dev/null || echo 0)
{ [ "$mrg" -ge "$((prev_dev + 2))" ]; } 2>/dev/null \
  && ok "the ENGINE landed >=2 NEW merges into dev this run ($prev_dev -> $mrg; real merge commits)" \
  || bad "engine landed <2 new merges into dev ($prev_dev -> $mrg) — the multi-node dispatch did not complete"

# CORE: at least one DEPENDENT closed no earlier than the foundation. The scheduler cannot
# admit a dependent until its predecessor reaches stateDone (merged + MergedToBase-verified),
# so a dependent completing after the foundation proves the edge was enforced. Only CLOSED
# issues carry a real closedAt (jq `// ""` maps an open issue's null to empty, not the string
# "null"), so we compare ONLY closed dependents and require >=1 valid comparison — an unclosed
# fan-out straggler is NOTE-tolerable (asserted separately below), never an ordering violation.
# closedAt is ISO-8601 UTC → bash lexical compare is sound; `! A > D` means A closed <= D.
if [ -n "$A_ID" ] && [ -n "$DEP_IDS" ]; then
  A_JSON=$(gh issue view "$A_ID" --repo "$REPO" --json state,closedAt -q '.state + "|" + (.closedAt // "")' 2>/dev/null)
  A_ST="${A_JSON%%|*}"; A_CLOSED="${A_JSON#*|}"
  compared=0; violated=0; detail="A(#$A_ID) state=$A_ST closed=$A_CLOSED"
  for D in $DEP_IDS; do
    D_JSON=$(gh issue view "$D" --repo "$REPO" --json state,closedAt -q '.state + "|" + (.closedAt // "")' 2>/dev/null)
    D_ST="${D_JSON%%|*}"; D_CLOSED="${D_JSON#*|}"
    detail="$detail | dep(#$D) state=$D_ST closed=$D_CLOSED"
    { [ "$D_ST" = "CLOSED" ] && [ -n "$D_CLOSED" ]; } || continue   # unclosed straggler -> NOTE, not the proof
    compared=$((compared + 1))
    { [ "$A_ST" = "CLOSED" ] && [ -n "$A_CLOSED" ] && [[ ! "$A_CLOSED" > "$D_CLOSED" ]]; } || violated=1
  done
  if [ "$compared" -ge 1 ] && [ "$violated" -eq 0 ]; then
    ok "foundation #$A_ID closed no later than $compared closed dependent(s) — scheduler enforced the dependency edge"
  else
    bad "dependency ORDER not proven (compared=$compared violated=$violated) — $detail"
  fi
else
  bad "could not resolve foundation/dependent ids from plan.json (A_ID='$A_ID' DEP_IDS='$DEP_IDS')"
fi

# CORE: merged units' worktrees were reclaimed (engine teardown after merge-verify).
if [ -n "$SLUG" ]; then
  leftover=$(ls "$PROJ/.delivery/worktrees/$SLUG" 2>/dev/null | wc -l | tr -d ' ')
  [ "${leftover:-0}" = "0" ] && ok "merged units' worktrees were reclaimed (worktrees/$SLUG empty)" || note "worktree leftover under $SLUG: $leftover (a unit may have blocked)"
fi
[ -f "$PROJ/.delivery/runstate.json" ] && ok "run-state persisted (restart substrate)" || note "no runstate.json"

# NOTE: the FULL fan-out — every plan unit closed (the 3-way concurrency, LLM-variable).
closed_all=1; closed_n=0
for U in $(jq -r '.units[].id' "$PROJ/.delivery/plan.json" 2>/dev/null); do
  s=$(gh issue view "$U" --repo "$REPO" --json state -q '.state' 2>/dev/null)
  [ "$s" = "CLOSED" ] && closed_n=$((closed_n + 1)) || closed_all=0
done
[ "$closed_all" = 1 ] && ok "ALL $UNIT_CT plan units closed (full fan-out completed)" || note "$closed_n/$UNIT_CT plan units closed (the seam held with >=2; a straggler is worker-fidelity)"

# NOTE: Status=Done on the board + the real code on dev.
done_ct=$(gh project item-list "$PROJNUM" --owner "$OWNER" --format json -q '[.items[] | select((.status // "") == "Done")] | length' 2>/dev/null || echo 0)
ge "$done_ct" && ok "board items reached Status=Done ($done_ct)" || note "no board item at Status=Done (LLM-variable; the closes still happened)"
sub_on_dev=$(gh api "repos/$REPO/contents/app.js?ref=dev" -q '.content' 2>/dev/null | base64 -d 2>/dev/null | grep -c 'subtract')
ge "$sub_on_dev" && ok "subtract() present in app.js on dev (the foundation implementation landed)" || note "subtract() not found on dev (worker code choice; the merge still landed)"

# ---- on failure, surface what the torn-down container would otherwise lose -----------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (github-delivery-full-chain failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- turns.log (tail) ---"; tail -120 "$HOME/turns.log" 2>/dev/null
  echo "--- dispatch output (full) ---"; echo "$out"
  echo "--- plan.json ---"; cat "$PROJ/.delivery/plan.json" 2>/dev/null
  echo "--- blocked reports ---"; cat "$PROJ"/.delivery/blocked/*.json 2>/dev/null
  echo "--- issues ---"; gh issue list --repo "$REPO" --state all --json number,title,state,labels 2>/dev/null
  echo "--- PRs ---"; gh pr list --repo "$REPO" --state all --json number,baseRefName,state,title 2>/dev/null
  echo "--- worktrees ---"; ls -la "$PROJ"/.delivery/worktrees/ 2>/dev/null
  echo "--- runstate.json ---"; cat "$PROJ"/.delivery/runstate.json 2>/dev/null
  echo "===================================================="
fi

finish
