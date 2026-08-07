#!/usr/bin/env bash
# needs: gh+token
# fixture: atl-e2e-delivery
# touches: teams/delivery-team/skills/sprint-plan, teams/delivery-team/skills/sprint-start, teams/delivery-team/agents/project-manager, teams/delivery-team/backends, teams/delivery-team/knowledge, cli/internal/dispatch
#
# github-sprint-carryover — MULTI-UNIT CARRYOVER: a sprint ends with two units
# incomplete, the second depending on the first, and the re-plan must admit BOTH
# together and carry the edge between them into plan.json.
#
# That path changed behaviour in the flow-admission fix (atl#333) and no blueprint
# exercised it. `main`'s carryover clause required all predecessors Done, so a
# carryover whose predecessor was ALSO carried over waited an entire sprint doing
# nothing; the fix admits them together, because admission cares about DAG
# CLOSURE (is every predecessor either complete or admitted alongside?) and not
# about the dispatch frontier (could this unit start today?). The two questions
# had been sharing one name.
#
# THE REGRESSION THIS CATCHES, precisely: admitting only the foundation and
# leaving the dependent behind. That outcome is not an error anywhere — the
# ceremony reports a clean sprint, the board looks right, and the dependent
# simply sits at the previous sprint's ordinal for another cycle. So the CORE
# assertion is that BOTH units advanced to the SAME new ordinal; either one alone
# is a red.
#
# WHAT THIS DOES NOT COVER, and why. atl#356 asks for both modes, since the rule
# is stated as mode-independent. Only flow is assertable on this backend: scrum's
# sprint carrier is a Projects v2 Iteration field, `gh` cannot create one
# (`field-create` supports TEXT/SINGLE_SELECT/DATE/NUMBER only), and the fixture
# board has none — so under scrum "was this unit admitted?" has no durable
# surface to read, and the only remaining evidence would be the ceremony's own
# prose, which is a claim rather than a state. Flow's carrier is a `sprint:<n>`
# label, which `gh` CAN write, so the same rule is observable there. The scrum
# half stays uncovered by an environmental constraint, not by an oversight; it is
# the same constraint github-delivery-full-chain records for flow-vs-scrum.
#
# ISOLATION: the ceremony chain that PRODUCES this board state (/kickoff ->
# /refine -> a sprint that ends incomplete -> /sprint-review tagging carryover) is
# already covered by github-delivery-loop and github-delivery-full-chain, and
# running it here would cost ~40 minutes to arrive at a state four `gh` calls can
# express. So the prior sprint is SEEDED — two open PBIs carrying `sprint:1` +
# `carryover`, the dependent's `## Depends On` naming the foundation, and the
# `docs/sprints/sprint-1-review.md` page that makes sprint:1 *reviewed* — and the
# blueprint runs only the two ceremonies under test.
#
# The seed is real board state written through the same surfaces the ceremonies
# use (`gh issue create` + labels + sentinel comments + the Contents API upsert
# §9), not an imitation of their output: what is under test is /sprint-plan's
# admission decision, and its input is the board.
#
# The prompts deliberately name NEITHER the units to admit NOR the sprint ordinal
# to open. Both are the contract under test — a prompt naming them would assert
# only that a turn can copy a string out of its own instructions.

source /e2e/lib.sh
note() { echo "  note - $1"; }

gh auth setup-git >/dev/null 2>&1 || true
LOGIN=$(gh_login)
[ -n "$LOGIN" ] || { bad "could not resolve the gh login: $(why)"; finish; exit 1; }
OWNER="${ATL_E2E_DELIVERY_OWNER:-agentteamland}"
REPO="$OWNER/$(delivery_fixture)"

reset_delivery_repo "$OWNER" \
  && ok "reset delivery fixture repo to baseline (main/dev/release)" \
  || bad "could not reset $REPO (does it exist? token 'repo' rights on $OWNER?)"
PROJNUM=$(reset_delivery_project "$OWNER")
[ -n "$PROJNUM" ] && ok "created a fresh GitHub Project #$PROJNUM" || bad "could not create the Project (token 'project' scope?)"

fresh
write_test_index_delivery
headless_claude_setup

rmdir "$PROJ" 2>/dev/null || true
git clone -q "https://github.com/$REPO.git" "$PROJ" || bad "clone of $REPO failed"
cd "$PROJ" || exit 2

atl install agentteamland/delivery-team >/dev/null 2>&1 || bad "install errored"
[ -f "$PROJ/.claude/skills/sprint-plan/SKILL.md" ]  && ok "sprint-plan skill reflected"  || bad "sprint-plan skill missing"
[ -f "$PROJ/.claude/skills/sprint-start/SKILL.md" ] && ok "sprint-start skill reflected" || bad "sprint-start skill missing"
[ -f "$PROJ/.claude/backends/github/adapter.md" ]   && ok "github adapter reflected"     || bad "github adapter missing"

# ---- .delivery/ config + FLOW methodology ------------------------------------
mkdir -p "$PROJ/.delivery"
cat > "$PROJ/.delivery/config.json" <<EOF
{
  "owner": "$OWNER",
  "repo": "$(delivery_fixture)",
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
  "mode": "flow",
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
  "branches": { "dev": "dev", "release": "release" }
}
EOF
# capacityModel omitted, never null — the flow contract's shape (a `"capacityModel": null`
# is the half-written scrum descriptor the ceremonies are told to surface and stop on).
jq -e '.mode == "flow" and (has("capacityModel") | not)' "$PROJ/.delivery/methodology.json" >/dev/null 2>&1 \
  && ok "seeded config.json (github) + methodology.json (mode: flow)" \
  || bad "flow-mode methodology.json seed is malformed"

# ---- SEED the prior sprint: two incomplete units, one depending on the other --
for L in "area:web" "type:pbi" "sprint:1" "carryover"; do
  gh label create "$L" --repo "$REPO" -c "#0969da" -d "e2e seed" >/dev/null 2>&1 || true
done
# `sprint:2` is deliberately NOT pre-created. Resolving the next ordinal and
# putting the label on the board is the ceremony's job; creating it here would
# hand it the answer.

cat > "$HOME/a-body.md" <<'EOF'
## Problem
The fixture app exposes `add(a, b)` but no subtraction primitive.

## Business Value
`subtract` is the shared foundation the difference helpers build on.

## Scope
Add `subtract(a, b)` to `app.js` and export it; cover it in `app.test.js`.

## Acceptance Criteria
- `app.js` exports `subtract(a, b)` returning `a - b`.
- `app.test.js` asserts `subtract(5, 3) === 2` and `node --test` passes.

## Out of Scope
Anything beyond `subtract`.
EOF
A_ID=$(gh_seed_issue "$REPO" "Add subtract(a,b) to app.js" "$HOME/a-body.md" \
        "area:web" "type:pbi" "sprint:1" "carryover")
[ -n "$A_ID" ] && ok "seeded the FOUNDATION unit #$A_ID (sprint:1, carryover, open)" \
  || { bad "could not seed the foundation unit: $(why)"; finish; exit 1; }

cat > "$HOME/b-body.md" <<'EOF'
## Problem
There is no absolute-difference helper.

## Business Value
`absDiff` is the first consumer of the shared `subtract` primitive.

## Scope
A new `absdiff.js` requiring `./app`, exporting `absDiff(a, b) = Math.abs(subtract(a, b))`.

## Acceptance Criteria
- `absdiff.js` exports `absDiff(a, b)`.
- `absdiff.test.js` asserts `absDiff(3, 5) === 2` and `node --test` passes.

## Out of Scope
Editing `app.js` or `app.test.js`.
EOF
B_ID=$(gh_seed_issue "$REPO" "Add absDiff helper (absdiff.js)" "$HOME/b-body.md" \
        "area:web" "type:pbi" "sprint:1" "carryover")
[ -n "$B_ID" ] && ok "seeded the DEPENDENT unit #$B_ID (sprint:1, carryover, open)" \
  || { bad "could not seed the dependent unit: $(why)"; finish; exit 1; }

# The `## Depends On` line in the Canonical Brief IS the dependency edge — both
# ceremonies read it from here and nowhere else (adapter §8: GitHub's native
# "blocked by" relations are not uniformly queryable, so the brief line is the
# single authoritative form).
cat > "$HOME/a-brief.md" <<EOF
**[Canonical Brief]**

## Goal
Add \`subtract(a, b)\` (returns \`a - b\`) to app.js and export it; add a \`node --test\` case in app.test.js asserting \`subtract(5, 3) === 2\`.

## Area
web

## Load These Pages
(none)

## Depends On
none

## Evidence Before Review
\`node --test\` passes with the new subtract case.
EOF
gh issue comment "$A_ID" --repo "$REPO" --body-file "$HOME/a-brief.md" >/dev/null 2>&1 \
  && ok "seeded the foundation's [Canonical Brief] (## Depends On: none)" || bad "could not comment the foundation brief"

cat > "$HOME/b-brief.md" <<EOF
**[Canonical Brief]**

## Goal
Create \`absdiff.js\` requiring \`./app\` and exporting \`absDiff(a, b) = Math.abs(subtract(a, b))\`; add \`absdiff.test.js\` asserting \`absDiff(3, 5) === 2\`. Do NOT edit app.js or app.test.js.

## Area
web

## Load These Pages
(none)

## Depends On
#$A_ID

## Evidence Before Review
\`node --test\` passes with the new absDiff case.
EOF
gh issue comment "$B_ID" --repo "$REPO" --body-file "$HOME/b-brief.md" >/dev/null 2>&1 \
  && ok "seeded the dependent's [Canonical Brief] (## Depends On: #$A_ID)" || bad "could not comment the dependent brief"

gh project item-add "$PROJNUM" --owner "$OWNER" --url "https://github.com/$REPO/issues/$A_ID" >/dev/null 2>&1 || true
gh project item-add "$PROJNUM" --owner "$OWNER" --url "https://github.com/$REPO/issues/$B_ID" >/dev/null 2>&1 || true

# The review page is what makes sprint:1 *reviewed* — the flow analogue of a
# closed iteration (adapter §5). Without it the contract says this run RE-PLANS
# into sprint:1 rather than opening sprint:2, and the whole carryover question
# never arises. Written through the Contents API upsert the adapter itself binds
# concept #9 to, ON THE INTEGRATION BRANCH — the branch that adapter now reads and
# writes (`config.branchPair.dev`), resolved from the same config rather than
# hardcoded, exactly as the adapter instructs.
#
# This used to seed the DEFAULT branch, and said so, because that is where the
# ceremony read before atl#486 moved the binding. When the read moved, this seed
# did not, so the page became invisible: /sprint-plan saw sprint:1 as unreviewed,
# admitted no carryover, and the label swap never happened. The harness is a THIRD
# writer to the same store — the adapter is the single binding for the TEAM, not
# for the system — so a change to that binding has to move the harness with it.
cat > "$HOME/sprint-1-review.md" <<EOF
# Sprint 1 Review

## Completed
(none)

## Carryover
- #$A_ID — Add subtract(a,b) to app.js — not completed; carried forward.
- #$B_ID — Add absDiff helper (absdiff.js) — not completed; carried forward. Depends on #$A_ID.

## Notes
Seeded by the e2e harness to close sprint 1 with two incomplete units.
EOF
REVIEW_B64=$(base64 < "$HOME/sprint-1-review.md" | tr -d '\n')
DEV_BRANCH="$(jq -r '.branchPair.dev' "$PROJ/.delivery/config.json")"
gh api --method PUT "repos/$REPO/contents/docs/sprints/sprint-1-review.md" \
  -f message="e2e: seed the sprint-1 review page" -f content="$REVIEW_B64" \
  -f branch="$DEV_BRANCH" >/dev/null 2>&1 || true
gh_try gh api "repos/$REPO/contents/docs/sprints/sprint-1-review.md?ref=$DEV_BRANCH" -q .name >/dev/null \
  && ok "sprint-1 review page readable at docs/sprints/sprint-1-review.md on $DEV_BRANCH (sprint:1 is reviewed)" \
  || { bad "could not seed/read the sprint-1 review page: $(why)"; finish; exit 1; }

# ---- BASELINE ---------------------------------------------------------------
#
# Recorded, not assumed: every assertion below is "these two units MOVED", and a
# move is only observable against where they started. It also proves the null
# outcome cannot pass — if /sprint-plan does nothing at all, both units still
# carry sprint:1 and the ordinal assertions go red.
labels_of() {   # labels_of <issue#> -> space-separated label names
  gh issue view "$1" --repo "$REPO" --json labels -q '[.labels[].name] | join(" ")' 2>/dev/null
}
sprint_ords() { # sprint_ords <issue#> -> the sprint: ordinals it carries, space-separated
  gh issue view "$1" --repo "$REPO" --json labels \
    -q '[.labels[].name | select(startswith("sprint:")) | sub("^sprint:";"")] | join(" ")' 2>/dev/null
}
A_BEFORE=$(sprint_ords "$A_ID"); B_BEFORE=$(sprint_ords "$B_ID")
{ [ "$A_BEFORE" = "1" ] && [ "$B_BEFORE" = "1" ]; } \
  && ok "baseline: both units carry exactly sprint:1" \
  || bad "baseline is wrong (foundation='$A_BEFORE' dependent='$B_BEFORE') — the delta below would be unreadable"
case " $(labels_of "$A_ID") $(labels_of "$B_ID") " in
  *" carryover "*) ok "baseline: the carryover tag is on the seeded units" ;;
  *) bad "baseline: no carryover tag — /sprint-plan reads carryover by that tag" ;;
esac

gturn() { claude_turn "$1"; }

# ---- 1. /sprint-plan — the re-plan ------------------------------------------
#
# Neutral by construction: it does not say which units to admit, does not mention
# carryover or dependencies, and does not name an ordinal. Everything the
# assertions check has to come from the board and the contract.
gturn "/sprint-plan. You are ALSO acting as the human product owner for this headless run — answer any product-owner input from what the board and the repository already record, and do not wait for interactive input. This project runs in FLOW mode (see .delivery/methodology.json), so do not ask for a seed velocity and do not set story-point estimates. Run the ceremony as your mode's contract specifies: resolve which sprint this run plans into, select from the board, and stamp the sprint carrier your mode specifies on each admitted unit. Report which sprint you opened, which units you admitted, and your reasoning for each." || bad "sprint-plan turn errored"

A_AFTER=$(sprint_ords "$A_ID"); B_AFTER=$(sprint_ords "$B_ID")

# CORE: at most one sprint label each. A label ACCUMULATES where a field REPLACES,
# so the swap has to be written by hand; two of them and "which sprint is this in?"
# stops having an answer.
one_each=1
for v in "$A_AFTER" "$B_AFTER"; do
  [ "$(echo "$v" | wc -w | tr -d ' ')" -le 1 ] || one_each=0
done
[ "$one_each" = 1 ] && ok "each unit carries at most one sprint: label (foundation='$A_AFTER' dependent='$B_AFTER')" \
                    || bad "a unit carries multiple sprint labels (foundation='$A_AFTER' dependent='$B_AFTER') — the corrupt two-label state"

# CORE: BOTH advanced, and to the SAME sprint. This is the whole card. The
# regression it names — admitting the foundation and leaving the dependent behind
# because its predecessor is not Done — lands here as a mismatch, and admitting
# nothing lands here as an unchanged '1'.
if [ -n "$A_AFTER" ] && [ "$A_AFTER" = "$B_AFTER" ]; then
  ok "both carryover units carry the SAME sprint ordinal (sprint:$A_AFTER)"
  { [ "$A_AFTER" -gt 1 ]; } 2>/dev/null \
    && ok "both were re-admitted to a LATER sprint than the one they carried over from (1 -> $A_AFTER) — multi-unit carryover with a live edge" \
    || bad "both still sit at sprint:$A_AFTER — the re-plan admitted no carryover at all"
else
  bad "the two carryover units did NOT land in the same sprint (foundation='$A_AFTER' dependent='$B_AFTER') — a dependent left behind by its own predecessor is exactly the regression atl#356 tracks"
fi

# The exact ordinal is the resolution contract (highest ordinal 1 + its review page
# exists -> open 2), one step removed from what this blueprint is about, and it is
# LLM-resolved. NOTE, mirroring github-delivery-full-chain's treatment of the
# first-sprint-is-1 resolution.
[ "$A_AFTER" = "2" ] && ok "the reviewed sprint:1 advanced to sprint:2 (the ordinal resolution)" \
                     || note "opened sprint:$A_AFTER rather than 2 (ordinal resolution is LLM-variable; the carryover gate above is the CORE claim)"

# The swap: the old ordinal is GONE, not merely joined by a new one. Only checkable
# when the ordinal actually moved, so it is skipped rather than false-passed when
# the assertion above already went red.
if [ -n "$A_AFTER" ] && [ "$A_AFTER" != "1" ]; then
  case " $(labels_of "$A_ID") $(labels_of "$B_ID") " in
    *" sprint:1 "*) bad "a re-admitted unit still carries its old sprint:1 label — the carrier was added, not swapped" ;;
    *) ok "the old sprint:1 label was removed from both units (swap, not accumulate)" ;;
  esac
fi

# ---- 2. /sprint-start — does the edge survive into plan.json? ----------------
#
# The card's second half. A sprint that admitted both units but dropped the edge
# produces a plan the engine cannot order: it would start the dependent against a
# predecessor that has not landed.
gturn "/sprint-start. Read the sprint's admitted work-units the way your mode's contract says a sprint's membership is read. Read each unit's '**[Canonical Brief]**' comment '## Depends On' lines to build the dependency DAG (a '#<n>' line under ## Depends On means this unit depends on unit n). Validate the DAG is acyclic. Materialize .delivery/plan.json in the EXACT dispatch.Plan schema: {\"sprintSlug\":\"<fs-safe-slug>\",\"granularity\":\"pbi\",\"units\":[{\"id\":<issue#>,\"title\":\"<title>\",\"predecessors\":[<issue#>...],\"stackRank\":<n>}]}. Use the JSON key 'stackRank'. There are no mobile-tagged units, so skip the emulator preflight. STOP after writing plan.json — do NOT run 'atl work dispatch'." || bad "sprint-start turn errored"

PLAN="$PROJ/.delivery/plan.json"
if [ -f "$PLAN" ] && jq -e '.' "$PLAN" >/dev/null 2>&1; then
  ok "sprint-start materialized a valid .delivery/plan.json"

  # sprintSlug must stay a JSON STRING: dispatch.Plan.SprintSlug is a Go string, and
  # a bare `"sprintSlug": 2` fails json.Unmarshal and kills the whole plan load. A
  # flow-specific footgun, because the flow slug reads like an integer.
  jq -e '.sprintSlug | type == "string"' "$PLAN" >/dev/null 2>&1 \
    && ok "plan.json sprintSlug is a JSON string (the engine's Go type)" \
    || bad "plan.json sprintSlug is $(jq -r '.sprintSlug | type' "$PLAN" 2>/dev/null), not a string — json.Unmarshal will reject the plan"

  # CORE: both carryover units are IN the plan.
  jq -e --argjson a "$A_ID" --argjson b "$B_ID" \
     '([.units[].id] | index($a)) != null and ([.units[].id] | index($b)) != null' "$PLAN" >/dev/null 2>&1 \
    && ok "plan.json carries BOTH carryover units (#$A_ID and #$B_ID)" \
    || bad "plan.json is missing a carryover unit — units: $(jq -c '[.units[].id]' "$PLAN" 2>/dev/null)"

  # CORE: the EDGE between them survived. Asserted on the specific pair, not on
  # "some unit has some predecessor" — a plan where the dependent depends on
  # nothing, or on the wrong unit, would satisfy the loose form.
  jq -e --argjson a "$A_ID" --argjson b "$B_ID" \
     '[.units[] | select(.id == $b) | .predecessors[]?] | index($a) != null' "$PLAN" >/dev/null 2>&1 \
    && ok "plan.json records the edge #$B_ID -> #$A_ID (the dependent's predecessor is the carried-over foundation)" \
    || bad "plan.json does NOT record #$B_ID depending on #$A_ID — the carried-over edge was lost: $(jq -c '.units' "$PLAN" 2>/dev/null)"

  # The foundation carries no predecessor of its own — the shape that makes the
  # plan dispatchable at all (a plan with no root has nothing the engine can start).
  jq -e --argjson a "$A_ID" '[.units[] | select(.id == $a) | (.predecessors | length)] | first == 0' "$PLAN" >/dev/null 2>&1 \
    && ok "the foundation #$A_ID has no predecessors (the plan has a dispatchable root)" \
    || note "the foundation carries predecessors: $(jq -c --argjson a "$A_ID" '[.units[] | select(.id == $a) | .predecessors]' "$PLAN" 2>/dev/null)"
else
  bad "no valid .delivery/plan.json materialized — the edge assertion cannot run"
fi

# ---- on failure, surface what the torn-down container would otherwise lose ----
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (github-sprint-carryover failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- seeded ids ---"; echo "foundation=#$A_ID dependent=#$B_ID project=#$PROJNUM"
  echo "--- labels now ---"; gh issue list --repo "$REPO" --state all --json number,title,state,labels 2>/dev/null
  echo "--- plan.json ---"; cat "$PLAN" 2>/dev/null
  echo "--- docs/sprints ---"; gh api "repos/$REPO/contents/docs/sprints" -q '.[].name' 2>/dev/null
  echo "--- turns.log (tail) ---"; tail -160 "$HOME/turns.log" 2>/dev/null
  echo "================================================="
fi

finish
