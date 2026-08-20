#!/usr/bin/env bash
# needs: token
# touches: teams/delivery-team, cli/internal/dispatch
#
# delivery-loop — the delivery-team ceremony chain against the mock azureDevOps MCP
# (stone #9 parts ①+② — the Azure-free half). Installs delivery-team, seeds a
# project's .delivery/ config + methodology, points the ceremonies at the mock MCP
# server (test/e2e/mock-mcp) via --mcp-config, and drives a full sprint cycle:
#   /kickoff -> /refine -> /sprint-plan -> /sprint-start -> /sprint-review
# Each ceremony is a real `claude -p` turn (so this is `needs: token`, local-only,
# like profile-loop/learning-loop — NOT the auth-free backbone; the mock's OWN
# behavior is proven auth-free by `node --test test/e2e/mock-mcp`). Assertions key
# off the STATE the mock recorded (its store file) + the derived .delivery/plan.json,
# never exact tool-call sequences — the ceremony turn is a non-deterministic LLM turn.
#
# Two assertion tiers (the deterministic-mock / non-deterministic-driver split):
#   * CORE (ok/bad) — the reliable e2e PLUMBING: install + asset reflection (incl. the
#     backends/ per-backend adapters), kickoff's Epic/Feature + the [Technical Analysis]
#     comment, sprint-start's valid plan.json, and sprint-review's promotion gate
#     HOLDING (a state read, not an LLM judgement). A regression here fails the test.
#   * NOTE (ok/note) — the less-deterministic ceremonies (refine/sprint-plan) + the
#     secondary field-writes (atl-key/area tags, the wiki seed, IterationPath) +
#     sprint-review's review page and the OPENING of the dev->release PR (a full LLM
#     ceremony chain: compile -> upsert -> open PR, run-to-run variable). Across
#     3 runs these varied run-to-run (each run skipped a DIFFERENT one), so a miss is
#     NOTED, not failed — an LLM-fidelity / ceremony-quality concern, not an
#     e2e-plumbing one. The mock records everything faithfully when written (proven by
#     the auth-free unit tests); whether a given LLM turn writes it is ceremony-quality.
#     Tightening refine/sprint-plan fidelity is tracked separately (not stone #9's job).
#
# Out of scope here (real-Azure Layer-B ③, env-gated): the developer worker micro-loop,
# az-attach.sh evidence upload, the mobile emulator lane, and dispatch's per-unit
# review/merge/Done orchestration. /sprint-start is asserted up to plan.json; the
# engine run itself is the separate work-dispatch.sh blueprint (fake worker, auth-free).
source /e2e/lib.sh
note() { echo "  note - $1"; }

fresh
write_test_index_delivery
headless_claude_setup
cd "$PROJ" || exit 2

# delivery-team is project-scope -> install into the project (reflects ceremonies,
# knowledge, and scripts into $PROJ/.claude per the stone #3 reflection contract).
atl install agentteamland/delivery-team >/dev/null 2>&1 || bad "install errored"
[ -f "$PROJ/.claude/skills/kickoff/SKILL.md" ]        && ok "kickoff skill reflected"        || bad "kickoff skill missing"
[ -f "$PROJ/.claude/skills/sprint-start/SKILL.md" ]   && ok "sprint-start skill reflected"    || bad "sprint-start skill missing"
[ -f "$PROJ/.claude/skills/sprint-review/SKILL.md" ]  && ok "sprint-review skill reflected"   || bad "sprint-review skill missing"
[ -f "$PROJ/.claude/backends/azure/adapter.md" ]      && ok "azure adapter reflected"          || bad "azure adapter missing (backends/ reflection)"
[ -f "$PROJ/.claude/backends/github/adapter.md" ]     && ok "github adapter reflected"         || bad "github adapter missing (backends/ reflection)"

# ---- seed the project's .delivery/ config + point ceremonies at the mock MCP ----
mkdir -p "$PROJ/.delivery"
STORE="$PROJ/.delivery/mock-store.json"

# config.json — connection identity (generic; NO real org/project). project name
# matches the mock's seeded project so /kickoff's core_list_projects preflight passes.
cat > "$PROJ/.delivery/config.json" <<'EOF'
{
  "org": "delivery-test-org",
  "project": "DeliveryTest",
  "repo": "DeliveryTest",
  "branchPair": { "dev": "dev", "release": "release" },
  "methodology": "scrum",
  "transport": "mcp",
  "restFallbackEnabled": true,
  "wikiId": "wiki-1",
  "pat": { "ref": "AZURE_DEVOPS_PAT" }
}
EOF

# methodology.json — the flat Scrum descriptor (config-and-methodology.md §1),
# workItemTypeMap null-seeded so ceremonies resolve real type/state names at runtime.
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

# .mcp.json — bind the `azureDevOps` MCP name to the mock stdio server, with the
# store path in its env so state persists across the fresh process each turn spawns.
cat > "$PROJ/mock-mcp.json" <<EOF
{ "mcpServers": { "azureDevOps": { "command": "node", "args": ["/e2e/mock-mcp/server.js"], "env": { "MOCK_STORE": "$STORE" } } } }
EOF

# a ceremony turn: real claude -p, ceremonies see the MOCK as their azureDevOps MCP.
dturn() { claude_turn "$1" --mcp-config "$PROJ/mock-mcp.json"; }
# a jq query over the mock store (empty if the store doesn't exist yet)
q() { jq -r "$1" "$STORE" 2>/dev/null; }
has() { q "$1" | grep -qE '^[1-9][0-9]*$'; }   # true if the count query returns >= 1

# ---- 1. /kickoff — greenfield cold-start (PO facts pre-baked; headless) ----------
dturn "/kickoff. You are ALSO acting as the human product owner for this headless run — answer intake from these facts, do not wait for interactive input. Project 'Tasky': a simple task-tracking web app for small teams. Problem: teams lose track of who owns what. Goals: create tasks, assign an owner, mark complete, see a team board. Out of scope: billing, mobile. Create the first Epic and at least one Feature, put the business framing in each item's System.Description under the fixed H2s, add one [Technical Analysis] sentinel comment, and seed a Domain/ and an Architecture/ wiki page. IMPORTANT: stamp each created item's tags by including a System.Tags field in the wit_create_work_item / wit_add_child_work_items call, with a value like 'atl-run:kickoff:1; atl-key:<hash>'. Skip sprint-0." || bad "kickoff turn errored"

has '[.workItems[] | select(.fields."System.WorkItemType"=="Epic")] | length'   && ok "kickoff created an Epic"                     || bad "no Epic in the mock store"
has '[.workItems[] | select(.fields."System.WorkItemType"=="Feature")] | length' && ok "kickoff created a Feature"                   || bad "no Feature in the mock store"
has '[.workItems[] | select((.comments // []) | map(.text) | any(test("\\[Technical Analysis\\]")))] | length' && ok "a [Technical Analysis] comment landed (content-placement §7)" || bad "no technical-analysis comment"
# field-write / less-deterministic (ceremony fidelity, not plumbing):
if has '[.wikiPages | keys[] | select(test("^(Domain|Architecture)"))] | length'; then ok "kickoff seeded a Domain/ or Architecture/ wiki page (§8)"; else note "no Domain/Architecture wiki page this run (LLM-variable)"; fi
if has '[.workItems[] | select((.fields."System.Tags" // "") | test("atl-key:"))] | length'; then ok "created items stamped with an atl-key tag (idempotency §5)"; else note "no atl-key tag written this run (LLM-variable; the mock persists tags when written — see unit tests)"; fi

# ---- 2. /refine — decompose Features into keyed, area-tagged, linked work-units ---
dturn "/refine. Groom and decompose the analyzed Feature(s) into implementable work-units (PBIs) under them. Give each a StackRank, add at least one Dependency link between two work-units where one must precede another (wit_work_items_link), and converge on existing items — do not duplicate the kickoff Epic/Feature. IMPORTANT: tag each work-unit's area by including 'area:web' in its System.Tags field (this is a web project), and stamp an atl-key tag the same way." || bad "refine turn errored"

# refine/sprint-plan are the less-deterministic ceremonies (varied run-to-run across 3 runs) -> NOTE, not fail:
if has '[.workItems[] | select(.fields."System.WorkItemType"=="Product Backlog Item" and (.fields."System.Tags"//""|test("historic")|not))] | length'; then ok "refine decomposed the Feature into work-units (PBIs)"; else note "no new PBIs this run (LLM-variable: refine is a less-deterministic ceremony)"; fi
if has '[.workItems[] | select((.relations // []) | map(.rel) | any(test("Dependency")))] | length'; then ok "a Dependency link was recorded"; else note "no Dependency link this run (LLM-variable)"; fi
if has '[.workItems[] | select((.fields."System.Tags"//"")|test("area:"))] | length'; then ok "work-units carry area tags (pack binding)"; else note "no area tag written this run (LLM-variable; pack binding is exercised at Layer-B)"; fi
# The ESTIMATE, observed where the gap happens rather than only as an absence three ceremonies
# downstream. sprint-plan's admission turns on this field — "an item with no estimate is a planning
# gap; never admit an unestimated unit" — so a unit without one is unplannable by contract, and the
# whole chain below correctly refuses. Until this line existed the field was checked NOWHERE in this
# blueprint, and its absence surfaced only as a red assertion naming sprint-start.
if has '[.workItems[] | select(.fields."System.WorkItemType"=="Product Backlog Item" and (.fields."System.Tags"//""|test("historic")|not) and (.fields."Microsoft.VSTS.Scheduling.StoryPoints" // null) != null)] | length'; then ok "refine estimated at least one work-unit (StoryPoints set — the admission precondition)"; else note "no work-unit carries StoryPoints this run (LLM-variable; sprint-plan will correctly refuse to admit unestimated units, and the branch below expects that)"; fi

# ---- 3. /sprint-plan — velocity from the seeded closed sprints, then admit --------
dturn "/sprint-plan. Compute capacity from the velocity of the last 3 closed sprints (read them via work_list_iterations + wit_get_work_items_for_iteration). The candidate backlog work-units are the New-state PBIs at the project-root IterationPath (read them via wit_list_backlog_work_items or a WIQL query for state='New' PBIs) — the current sprint is empty. Select the top backlog units by StackRank at a single granularity, and MOVE each admitted unit into the current sprint by setting its System.IterationPath (wit_update_work_item) to the sprint's path. Report the computed velocity." || bad "sprint-plan turn errored"

# field-write (the sprint admission signal — LLM-variable IterationPath write):
if has '[.workItems[] | select(.fields."System.WorkItemType"=="Product Backlog Item" and ((.fields."System.IterationPath"//"")|test("Sprint [0-9]")) and ((.fields."System.State"//"")!="Done"))] | length'; then ok "sprint-plan admitted units to a sprint (IterationPath set)"; else note "no non-historic unit carries a sprint IterationPath this run (LLM-variable; sprint-start still derives the plan below)"; fi

# ---- 4. /sprint-start — build the DAG + materialize plan.json (NO dispatch) -------
dturn "/sprint-start. Read the sprint's admitted work-units (the New-state PBIs you planned; if none carry the sprint IterationPath yet, use the New-state backlog PBIs), build their dependency DAG, validate it is acyclic, and materialize .delivery/plan.json in the exact dispatch.Plan schema (sprintSlug, granularity, units[] with id/title/predecessors/stackRank) — OR, if the sprint is degenerate (no workable units), refuse per the skill's fail-fast rather than writing an empty plan. This is a Layer-A ceremony test: STOP after writing plan.json (or after the refusal) — do NOT run 'atl work dispatch' (the engine run is covered by a separate test). There are no mobile-tagged units, so skip the emulator preflight." || bad "sprint-start turn errored"

if [ -f "$PROJ/.delivery/plan.json" ] && jq -e '.' "$PROJ/.delivery/plan.json" >/dev/null 2>&1; then
  ok "sprint-start materialized a valid .delivery/plan.json"
  # CORE: the dispatch.Plan skeleton — a materialized plan is always non-degenerate (sprint-start
  # refuses rather than writing an empty plan, per sprint-start-edge-cases)
  jq -e 'has("sprintSlug") and has("granularity") and (.units | type == "array")' "$PROJ/.delivery/plan.json" >/dev/null 2>&1 && ok "plan.json matches the dispatch.Plan skeleton (sprintSlug/granularity/units[])" || bad "plan.json skeleton malformed"
  # CORE: a materialized plan must carry >=1 populated unit — an empty units[] is the REJECTED
  # behavior now (sprint-start fail-fast refuses a degenerate sprint, never writes a silent empty plan)
  if jq -e '.units | length >= 1 and (.[0] | has("id") and has("predecessors") and has("stackRank"))' "$PROJ/.delivery/plan.json" >/dev/null 2>&1; then ok "plan.json carries populated units (id/predecessors/stackRank)"; else bad "plan.json materialized with empty/malformed units — sprint-start must refuse a degenerate sprint, not write an empty plan"; fi
else
  # No plan.json. THREE upstream states produce that, and collapsing them into two is what made
  # this assertion blame the wrong ceremony:
  #
  #   (a) refine produced no PBIs at all            -> degenerate sprint; the refusal is correct
  #   (b) PBIs exist but none carry an ESTIMATE     -> sprint-plan must refuse admission, by its
  #                                                    own contract ("an item with no estimate is a
  #                                                    planning gap — never admit an unestimated
  #                                                    unit"), so the refusal is correct here too
  #   (c) ESTIMATED PBIs exist and none were planned -> the genuine failure
  #
  # The old predicate was (a) versus "any PBI exists", which folded (b) into (c): it demanded a
  # plan the planner's own spec forbids producing, and reported a correct DOUBLE refusal as a
  # product defect named after sprint-start — the last ceremony to speak rather than the one the
  # gap was in. Measured on the v2.32.0 gate: refine produced six dependency-linked, area-tagged
  # PBIs with no StoryPoints; sprint-plan quoted its own rule and stopped without moving anything;
  # sprint-start refused a sprint with zero admitted units; the run went red.
  #
  # And it passed VACUOUSLY whenever nothing ran. On the gate immediately before that one the
  # Claude quota was exhausted, refine produced nothing, "any PBI exists" was false, and the
  # assertion never fired. An assertion that passes both when everything works and when nothing
  # runs carries almost no information about either.
  if has '[.workItems[] | select(.fields."System.WorkItemType"=="Product Backlog Item" and (.fields."System.Tags"//""|test("historic")|not) and (.fields."Microsoft.VSTS.Scheduling.StoryPoints" // null) != null)] | length'; then
    bad "estimated PBIs exist and no plan.json was written — the planning chain admitted nothing it was able to admit"
  elif has '[.workItems[] | select(.fields."System.WorkItemType"=="Product Backlog Item" and (.fields."System.Tags"//""|test("historic")|not))] | length'; then
    # (b) The units exist and are unplannable. Both ceremonies refusing is the CORRECT outcome, so
    # what is checked here is that the reason was SURFACED — a silent stop and a reasoned refusal
    # are identical in the tally, and only one of them is the behaviour the spec asks for.
    if grep -Eiq 'no estimate|planning gap|unestimated|story ?points|zero admitted|nothing .{0,20}admitted|no admitted work-units' "$HOME/turns.log" 2>/dev/null; then
      ok "refine left the units unestimated and the planning chain refused them, naming the reason (estimate = admission precondition)"
    else
      note "PBIs exist but none carry an estimate, and no refusal reason was detected in turns.log (LLM-variable wording; the refusal itself is correct)"
    fi
  else
    # (a) no workable PBIs upstream (refine produced none) -> sprint-start SHOULD fail-fast refuse.
    # Positive check: confirm it SURFACED the refusal, else a silent no-op (the original bug) passes identically.
    if grep -Eiq 'sprint is (empty|complete)|no admitted work-units|nothing to dispatch|degenerate sprint' "$HOME/turns.log" 2>/dev/null; then
      ok "sprint-start fail-fast refused the degenerate sprint + surfaced the reason (no workable PBIs upstream)"
    else
      note "no plan.json + no workable PBIs upstream this run; sprint-start's refusal message not detected in turns.log (LLM-variable wording)"
    fi
  fi
fi

# ---- 5. /sprint-review — the COMMIT-BOUND approval gate (concept #16) -------------
# The gate no longer reads prose, so the PO-identity framing this turn used to carry
# ("you are ALSO acting as the human product owner … APPROVE the sprint") is DELETED:
# its only job was to talk the gate into proceeding, and against a state-read gate it
# makes it impossible to tell whether the STATE or the PROSE crossed the gate.
#
# v1 enables the commit-bound gate on GITHUB ONLY. On Azure the RECORD leg is bound (a PO
# can post the record today, and the mock persists PR threads), but the HEAD-COMMIT read
# is UNRESOLVED: the response field carrying the commit id has never been confirmed
# against a live server, so it is not named anywhere. The ceremony therefore cannot
# compare the approved commit against the promotion PR's head here and it fails CLOSED —
# compile the report, open-or-find the promotion PR, HOLD. Two phases:
#   1. no record on the PR                          -> HOLD
#   2. a record IS on the PR, naming another commit -> still HOLD
# Phase 2 is the load-bearing half: it proves the gate does not promote on the mere
# EXISTENCE of an approval record. Its record deliberately names a commit that is NOT
# the mock's branch head, so the assertion stays correct if/when the Azure head-commit
# read is bound (it is then a SHA mismatch — still a HOLD, never a promotion). The
# stale-SHA control proper is GitHub-only in v1: the mock's branch head never advances.
dturn "/sprint-review. Compile the Sprint Review Report and upsert it to the Sprints/Sprint-<n>-Review wiki page. Open or find the dev->release promotion PR (repo_create_pull_request, from the dev branch into the release branch), then run the gate. Report the promotion PR id and the gate's outcome." || bad "sprint-review turn 1 errored"

has '[.wikiPages | keys[] | select(test("Sprint"))] | length'  && ok "sprint-review upserted a Sprints/ review wiki page" || note "sprint-review review page not upserted this run (LLM-variable ceremony fidelity)"

# CORE: the gate HELD — no dev->release PR was COMPLETED (completion is the promotion
# on Azure: repo_update_pull_request with autoComplete).
COMPLETED_PROMOTION='[.pullRequests[] | select(((.targetRefName // "") | test("release")) and (.status == "completed"))] | length'
if has "$COMPLETED_PROMOTION"; then bad "a dev->release PR was COMPLETED with no approval record (#16 fail-closed)"; else ok "gate HELD with no approval record — no dev->release completion (#16 fail-closed)"; fi
# liveness (NOTE — opening the PR is the LLM-variable half on this blueprint): did the
# ceremony reach the gate at all? Deliberately NOT promoted to CORE, which means the hold
# assertion above CAN pass vacuously on a run that never opened a promotion PR. Accepted
# here: the non-vacuous CORE liveness lives on the github blueprint, where opening the PR
# is deterministic; this blueprint's PR-open step has documented run-to-run variance.
PRID=$(q '[.pullRequests[] | select((.targetRefName // "") | test("release"))] | .[0].pullRequestId // empty')
[ -n "$PRID" ] && ok "promotion PR opened before the gate (step 6a)" || note "no dev->release promotion PR this run (LLM-variable ceremony fidelity)"

if [ -n "$PRID" ]; then
  # --- the PO sets the record OUT OF BAND, straight into the mock store -------------
  # The human's own write (on a live backend: the PR comment box, or the adapter's
  # thread-write tool) with no LLM in the loop: first line EXACTLY the sentinel, a 40-hex
  # commit id under the fixed '## Approved Commit' H2 — the #16 record shape, identical
  # on both backends. Written straight into the store rather than through a mock tool:
  # the mock still carries the PRE-consolidation thread tool names the curated map lists
  # (repo_create_pull_request_thread / repo_list_pull_request_threads) while the shipped
  # MCP consolidates them (backends/azure/adapter.md §2 records the resolved names) —
  # re-syncing the mock's surface is a separate pass, and a store write is name-agnostic.
  RECORD_SHA=2a7d4e9b1c6f8035d2e4a6b8c0d1e3f5a7b9c1d3
  APPROVAL=$(printf '**[Promotion Approval]**\n\n## Approved Commit\n%s\n\n## Sprint\nSprint 4\n\n## Decision\nAPPROVE\n' "$RECORD_SHA")
  if jq --arg pr "$PRID" --arg body "$APPROVAL" '.pullRequests[$pr].threads |= (. + [{id: (length + 1), content: $body, comments: [{content: $body}]}])' "$STORE" > "$STORE.tmp" 2>/dev/null && mv "$STORE.tmp" "$STORE"; then
    ok "seeded a **[Promotion Approval]** record onto the promotion PR (out of band)"
  else
    rm -f "$STORE.tmp"
    bad "could not seed the approval record into the mock store"
  fi

  dturn "/sprint-review. Re-run the gate on the existing dev->release promotion PR." || bad "sprint-review turn 2 errored"
  # CORE: a record the ceremony cannot match against the PR's head is NOT an approval.
  if has "$COMPLETED_PROMOTION"; then bad "an unmatched approval record COMPLETED the promotion (#16 must fail closed)"; else ok "an approval record the gate could not verify did NOT promote (#16 fail-closed)"; fi
else
  note "approval-record phase skipped — no promotion PR to attach the record to"
fi

# ---- on failure, surface what the torn-down container would otherwise lose --------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (delivery-loop failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- turns.log (tail) ---"; tail -80 "$HOME/turns.log" 2>/dev/null
  echo "--- .delivery/ tree ---"; find "$PROJ/.delivery" 2>/dev/null
  echo "--- mock store (summary) ---"
  jq '{workItems: [.workItems[] | {id, type: .fields."System.WorkItemType", state: .fields."System.State", tags: .fields."System.Tags", iter: .fields."System.IterationPath"}], wiki: (.wikiPages|keys), prs: [.pullRequests[] | {id: .pullRequestId, src: .sourceRefName, tgt: .targetRefName, status, threads: ((.threads // []) | map(.content))}]}' "$STORE" 2>/dev/null
  echo "--- plan.json ---"; cat "$PROJ/.delivery/plan.json" 2>/dev/null
  echo "========================================"
fi

finish
