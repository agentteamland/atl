#!/usr/bin/env bash
# needs: gh+token
# fixture: atl-e2e-delivery-3
# touches: teams/delivery-team, cli/internal/dispatch
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
#     (the adapter §10 completion gate), and sprint-review's commit-bound promotion
#     gate (#16) — `atl work promote` HOLDs without a matching approval record and
#     merges dev->release only for the exact commit that was approved. A regression
#     here fails the test.
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
REPO="$OWNER/$(delivery_fixture)"

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
gturn() { claude_turn "$1"; }
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
# The labelling instruction is deliberately UNAMBIGUOUS. An earlier version said
# "label each work-unit 'area:web' (this is a web project)", which contradicts the
# architecture doc /kickoff seeds one step earlier (a React frontend AND a Node
# backend) — so the ceremony correctly stopped to ask which reading was meant, and
# a headless run has nobody to answer. See atl#336.
gturn "/refine. Groom and decompose the analyzed Feature(s) into implementable work-units (PBIs) as GitHub issues under them (each nested via the sub_issues REST endpoint, adapter §1). Give each a Priority, record at least one dependency as a '## Depends On' line (#N) in the tech-lead's canonical-brief comment (adapter §8), and converge on existing items — do not duplicate the kickoff Epic/Feature. Labelling, decided: split the work by real system shape and label each unit with the area that matches it — 'area:web' for user-facing/UI units, 'area:api' for backend units. Those two are the only areas in play here; never 'area:mobile' or 'area:go-cli'. At least one unit must be 'area:web'. This is a settled instruction from the product owner, so do not pause to confirm it. Stamp each unit with an 'atl-key:<shorthash>' label." || bad "refine turn errored"

# refine is the less-deterministic ceremony -> NOTE, not fail:
pbi=$(gh issue list --repo "$REPO" --state all --json labels -q '[.[] | select((.labels // []) | map(.name) | any(. == "area:web"))] | length' 2>/dev/null || echo 0)
ge "$pbi" && ok "refine produced area:web work-units (PBIs)" || note "no area:web PBI this run (LLM-variable: refine is less-deterministic)"

# ---- 3. /sprint-plan — velocity (cold-start), admit units onto the board ----------
# The seed velocity is SUPPLIED here, as the PO. methodology.json carries
# seedVelocity: null on purpose — this exercises the po-seed path, where the
# ceremony must refuse to invent the number and wait for the human. A headless run
# has to play that human, or the ceremony correctly pauses forever. See atl#336.
gturn "/sprint-plan. You ARE the human product owner for this run. This is a cold-start project (no closed sprints), so use the po-seed velocity path — and here is the seed, decided by you: **8 story points** for Sprint 1. That is your number; take it and proceed, do not ask for it and do not invent a different one. The candidate backlog is the open PBI issues (labelled area:web or area:api) not yet on the board. Select the top units within that capacity and admit them to the current sprint by setting their Projects v2 Iteration field (add each issue to Project #$PROJNUM and set Iteration). Report the seed velocity used and which units you admitted." || bad "sprint-plan turn errored"

items=$(gh project item-list "$PROJNUM" --owner "$OWNER" --format json -q '.items | length' 2>/dev/null || echo 0)
ge "$items" && ok "sprint-plan added units to the Project board" || note "no board items this run (LLM-variable; sprint-start still derives the plan below)"

# ---- 4. /sprint-start — build the DAG + materialize plan.json (NO dispatch) --------
gturn "/sprint-start. Read the sprint's admitted work-units (the area:web and area:api PBIs; if none are on the board yet, use the open ones), read each unit's '## Depends On' lines to build the dependency DAG, validate it is acyclic, and materialize .delivery/plan.json in the exact dispatch.Plan schema (sprintSlug, granularity, units[] with id/title/predecessors/stackRank). This is a ceremony test: STOP after writing plan.json — do NOT run 'atl work dispatch'. There are no mobile-tagged units, so skip the emulator preflight." || bad "sprint-start turn errored"

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
prev_dev=$(gh pr list --repo "$REPO" --base dev --state merged --limit 400 --json number -q 'length' 2>/dev/null || echo 0)
gturn "Act as the developer, then the tech-lead, for ONE open area:web PBI issue (pick the lowest-numbered open one; if none, pick any open type:feature issue). DEVELOPER: from the local checkout create a branch off 'dev', add a small real change to app.js (e.g. extend the add function or add a sub function) plus a matching case in app.test.js, run 'node --test' to confirm it passes, commit, push the branch, and open a PR into 'dev' with 'Fixes #<n>' in the body (gh pr create --base dev). TECH-LEAD: review the PR, then merge it with 'gh pr merge --merge' ONLY (never --squash/--rebase), then 'gh issue close #<n>' and set the issue's Project #$PROJNUM Status field to Done. Report the PR number and the merged issue number." || bad "developer/tech-lead micro-loop turn errored"

# CORE: a NEW PR to dev was MERGED this run (real merge commit -> MergedToBase valid)
mrg=$(gh pr list --repo "$REPO" --base dev --state merged --limit 400 --json number -q 'length' 2>/dev/null || echo 0)
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

# ---- 6. /sprint-review — the COMMIT-BOUND PO approval gate (concept #16) ----------
# The gate reads state, not prose — and it no longer lives in the ceremony at all. The
# PO's approval is a durable record on the promotion PR whose first line is exactly
# '**[Promotion Approval]**' and which names ONE commit under '## Approved Commit'; the
# comparison against the PR's CURRENT head, and the merge, are BOTH performed by
# `atl work promote`. /sprint-review opens-or-finds the PR (step 6a) and runs that
# command.
#
# So these phases assert on the COMMAND — its exit code and its --json verdict — not on
# the ceremony's wording. That is the whole reason the decision moved into code: the
# prose version of this check was honoured on one turn and silently skipped on the very
# next, and no assertion on ceremony prose can tell those two apart.
#
# The PO-identity framing this turn used to carry ("you ARE the product owner … you
# APPROVE") stays DELETED: its only job was to talk the gate into proceeding, and
# against a state-read gate it would make it impossible to tell whether the STATE or
# the PROSE crossed the gate. The harness sets the signal out of band with plain `gh`,
# exactly as the human PO would.
#
# Three phases, one ceremony turn each, each followed by a direct gate assertion:
#   1. no record naming the head           -> HOLD (nothing merges)
#   2. a record naming a SUPERSEDED commit -> HOLD, reason 'superseded' (the binding)
#   3. a record naming the CURRENT head    -> verify + merge; `release` then CONTAINS it
# Phase 2 runs BEFORE the merge deliberately: a merged PR cannot be reused, so a stale
# record posted after phase 3 would land on a freshly-opened PR carrying no record at
# all, and the assertion would pass for the wrong reason (no-record, not stale-SHA).
#
# Baseline MERGED dev->release PRs — upgraded from `--state all`, which also counted
# OPEN ones: opening the promotion PR is now a PRE-gate step (6a), so only a MERGED PR
# is a promotion. Its head is `dev` (never branch-deleted by the reset), so the merged
# record persists across runs — assert an INCREASE this run, never an all-time count.
relm()   { gh pr list --repo "$REPO" --base release --state merged --limit 400 --json number -q 'length' 2>/dev/null || echo 0; }
prhead() { gh pr view "$1" --repo "$REPO" --json headRefOid -q .headRefOid 2>/dev/null; }
# rel_contains <sha> — does `release` actually CONTAIN that commit? This is the MERGED
# state. Comparing the promotion PR's head to the approved sha is NOT: those two are
# trivially equal when nothing merged at all, which is how that check passed in a run
# where the promotion never happened. compare/BASE...HEAD reports 'identical' when the
# two are the same commit and 'behind' when HEAD is an ancestor of BASE — both mean
# release has it; 'ahead'/'diverged' mean it does not.
rel_contains() {
  local st; st=$(gh api "repos/$REPO/compare/release...$1" -q .status 2>/dev/null)
  [ "$st" = "identical" ] || [ "$st" = "behind" ]
}
# gate — run the deterministic promotion gate DIRECTLY and capture its machine verdict.
# The --json verdict goes to stdout; cobra's one-line error summary goes to stderr and
# would corrupt it for jq, so stderr is dropped. Exit 0 = promoted, non-zero = hold
# (which merges nothing, so calling this in the HOLD phases is side-effect-free).
GATE_RC=0; GATE_VERDICT=""; GATE_REASON=""
gate() {
  local out; out=$(cd "$PROJ" && atl work promote --json 2>/dev/null); GATE_RC=$?
  GATE_VERDICT=$(printf '%s' "$out" | jq -r '.verdict // empty' 2>/dev/null)
  GATE_REASON=$(printf '%s' "$out" | jq -r '.reason // empty' 2>/dev/null)
}
# post_approval <pr#> <sha> — set the #16 record out of band, as the PO would.
# --body-file, not --body: `atl guard` scans the whole Bash command string.
post_approval() {
  printf '**[Promotion Approval]**\n\n## Approved Commit\n%s\n\n## Sprint\nSprint 1\n\n## Decision\nAPPROVE\n' "$2" > "$HOME/approval.md"
  gh pr comment "$1" --repo "$REPO" --body-file "$HOME/approval.md" >/dev/null 2>&1
}
prev_rel=$(relm)

# --- phase 1: compile + open-or-find the promotion PR + HOLD (no approval on record) -
gturn "/sprint-review. Compile the Sprint Review Report and upsert it to docs/sprints/sprint-1-review.md. Open or find the dev->release promotion PR, then run the gate. Report the promotion PR number and its head commit." || bad "sprint-review turn 1 errored"

# CORE: the gate HELD — nothing merged without an approval record. "matching" is the
# honest predicate: the reset leaves a record-free baseline, but a promotion PR left open
# by a FAILED prior run survives it (its head is `dev`, never deleted) carrying that run's
# records — all superseded, since the reset force-pushes a fresh `dev` head. Either way
# the property asserted is the same: no merge without a record naming the current head.
rel_h=$(relm)
{ [ "$rel_h" -eq "$prev_rel" ]; } 2>/dev/null && ok "gate HELD with no matching approval record — no dev->release merge (#16 fail-closed)" || bad "promotion merged with no matching approval record on the PR"
# CORE liveness (so the negative above is not vacuous): the ceremony did reach the gate.
PRNUM=$(gh pr list --repo "$REPO" --base release --state open --limit 400 --json number -q '.[0].number // empty' 2>/dev/null)
[ -n "$PRNUM" ] && ok "promotion PR opened before the gate (step 6a)" || bad "no promotion PR — the ceremony never reached the gate"
# CORE: the gate's OWN verdict, on the same PR. Both 'no-record' and 'superseded' are
# correct here — the reset leaves a record-free baseline, but a promotion PR left OPEN
# by a failed prior run survives it (its head is `dev`, never deleted) carrying that
# run's records, all superseded by the fresh force-pushed head. What must hold either
# way is a HOLD verdict and a non-zero exit: no record names the current head.
gate
{ [ "$GATE_RC" -ne 0 ] && [ "$GATE_VERDICT" = "hold" ]; } && ok "atl work promote HELD with no record naming the head (reason=$GATE_REASON, exit $GATE_RC)" || bad "atl work promote did not hold: verdict='$GATE_VERDICT' reason='$GATE_REASON' exit $GATE_RC"
# NOTE: the refusal wording is LLM-variable
grep -Eiq 'promotion approval|no approval on record|HOLD' "$HOME/turns.log" 2>/dev/null && ok "the hold was surfaced to the PO" || note "hold message not detected (LLM-variable wording)"

# --- phase 2: approve a commit, then let `dev` advance past it ----------------------
# Read every SHA through the SERVER, never from $PROJ: the merges happen server-side
# (gh pr merge), so the local clone never holds the promoted state.
STALE_SHA=$(prhead "$PRNUM")
post_approval "$PRNUM" "$STALE_SHA" || bad "could not set the approval signal"
git -C "$PROJ" fetch -q origin
git -C "$PROJ" checkout -q -f dev
git -C "$PROJ" reset -q --hard origin/dev
git -C "$PROJ" -c user.email=e2e@atl.local -c user.name=atl-e2e commit -q --allow-empty -m "e2e: advance dev after the approval"
git -C "$PROJ" push -q origin HEAD:dev || bad "could not advance dev — the stale control is not armed"
# CORE: the control is ARMED. Without this the HOLD below passes vacuously — a `dev`
# that never moved leaves the on-record approval CURRENT, not stale.
# POLL, don't read once: `git push` returns before GitHub has updated the PR's
# headRefOid, so a single read races the server — it reported the OLD sha and failed
# this assertion in a run whose end-of-run debug showed the head HAD advanced.
moved=$(prhead "$PRNUM")
tries=0
while [ "$tries" -lt 30 ] && { [ -z "$moved" ] || [ "$moved" = "$STALE_SHA" ]; }; do
  sleep 2
  tries=$((tries + 1))
  moved=$(prhead "$PRNUM")
done
{ [ -n "$moved" ] && [ "$moved" != "$STALE_SHA" ]; } && ok "stale control armed — the promotion PR head advanced past the approved commit (after $tries poll(s))" || bad "the promotion PR head did not advance past $STALE_SHA within 60s"

gturn "/sprint-review. Re-run the gate on the existing promotion PR." || bad "sprint-review turn 2 errored"
# CORE: an approval naming a superseded commit is not an approval
rel_s=$(relm)
{ [ "$rel_s" -eq "$prev_rel" ]; } 2>/dev/null && ok "a stale approval (dev advanced) did NOT promote (#16 commit-binding)" || bad "stale approval promoted an unapproved commit"
# CORE: the binding itself, on the gate's own verdict. 'superseded' is exact here
# (unlike phase 1): a record for a commit `dev` has since moved past is on the PR, so
# no other hold reason is correct.
gate
{ [ "$GATE_RC" -ne 0 ] && [ "$GATE_REASON" = "superseded" ]; } && ok "atl work promote reported reason=superseded for the stale record (#16 commit-binding)" || bad "expected a superseded hold: got verdict='$GATE_VERDICT' reason='$GATE_REASON' exit $GATE_RC"

# --- phase 3: approve the CURRENT head -> verify + merge ----------------------------
APPROVED=$(prhead "$PRNUM")
post_approval "$PRNUM" "$APPROVED" || bad "could not set the approval signal"
gturn "/sprint-review. Re-run the gate on the existing promotion PR." || bad "sprint-review turn 3 errored"

# CORE: a NEW dev->release PR was MERGED this run
rel=$(relm)
{ [ "$rel" -gt "$prev_rel" ]; } 2>/dev/null && ok "commit-bound PO approval promoted dev->release (#16)" || bad "no NEW MERGED dev->release PR after a valid approval"
# CORE: `release` actually CONTAINS the approved commit — the MERGED state, not the PR
# head. The old form compared the PR head to $APPROVED, which is trivially equal when
# nothing merged at all: it passed in a run where the promotion never happened.
rel_contains "$APPROVED" && ok "release contains the approved commit $APPROVED (#16 binding)" || bad "release does NOT contain the approved commit $APPROVED — nothing was promoted"
# CORE: a re-run converges. With the promotion merged there is no open dev->release PR
# left, so the gate holds with 'no-open-pr' and nothing merges twice. (If the ceremony
# above failed to promote, this call promotes instead and reports 'promoted' — an extra
# honest FAIL here, never a false pass: the two assertions above have already fired.)
gate
{ [ "$GATE_RC" -ne 0 ] && [ "$GATE_REASON" = "no-open-pr" ]; } && ok "a re-run converges — no open promotion PR left, nothing merged twice" || bad "expected a no-open-pr hold on re-run: got verdict='$GATE_VERDICT' reason='$GATE_REASON' exit $GATE_RC"
# NOTE: the sprint-review page written to docs/sprints/.
# ?ref=dev is load-bearing: the ceremony works on the dev branch, but `contents`
# without a ref resolves against the DEFAULT branch, so this could never find the
# page and reported "LLM-variable" on every run — a deterministic false negative
# wearing the label we use for genuine model variance. Its tier stays NOTE only
# until we have real data: before this fix it was never actually observed. (atl#336)
if gh api "repos/$REPO/contents/docs/sprints/sprint-1-review.md?ref=dev" >/dev/null 2>&1; then ok "sprint-review upserted a docs/sprints review page (§9)"; else note "no docs/sprints review page this run"; fi

# ---- on failure, surface what the torn-down container would otherwise lose ---------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (github-delivery-loop failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- gh --version ---";     gh --version 2>&1 | head -1
  echo "--- turns.log (tail) ---"; tail -100 "$HOME/turns.log" 2>/dev/null
  echo "--- issues ---";           gh issue list --repo "$REPO" --state all --json number,title,state,labels 2>/dev/null
  echo "--- PRs ---";              gh pr list --repo "$REPO" --state all --json number,baseRefName,state 2>/dev/null
  if [ -n "${PRNUM:-}" ]; then
    echo "--- promotion PR head + approval records ---"
    gh pr view "$PRNUM" --repo "$REPO" --json headRefOid,comments -q '{head: .headRefOid, approvals: [.comments[] | select(.body | startswith("**[Promotion Approval]**")) | {author: .author.login, at: .createdAt, body: .body}]}' 2>/dev/null
  fi
  # The last gate verdict as captured, NOT a fresh `atl work promote` — re-running it
  # here could merge a promotion the failing run had not made.
  echo "--- last gate verdict ---"; echo "verdict='${GATE_VERDICT:-}' reason='${GATE_REASON:-}' exit=${GATE_RC:-}"
  echo "--- release head ---";      gh api "repos/$REPO/commits/release" -q .sha 2>/dev/null
  echo "--- board items ---";      gh project item-list "$PROJNUM" --owner "$OWNER" --format json 2>/dev/null | jq '[.items[] | {title, status}]' 2>/dev/null
  echo "--- plan.json ---";        cat "$PROJ/.delivery/plan.json" 2>/dev/null
  echo "==============================================="
fi

finish
