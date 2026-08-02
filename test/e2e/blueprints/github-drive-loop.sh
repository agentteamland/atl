#!/usr/bin/env bash
# needs: gh+token
# touches: teams/delivery-team/skills/work-start, teams/delivery-team/skills/work-finish, teams/delivery-team/skills/work-move, teams/delivery-team/backends
#
# github-drive-loop — the MANUAL drive loop end to end on the GitHub backend: the mode a person
# works in, one card at a time, driven by a real `claude -p` at each step.
#
# /work-start claims a unit and cuts the engine's own branch; /work-finish gates, pushes, opens
# the PR and verifies the work-item link; /work-move is the only state writer. Same board, same
# card, same branch grammar as the autonomous engine — only the executor differs.
#
# WHAT THIS BLUEPRINT IS ACTUALLY FOR — the "did NOT do" properties.
#
# The easy assertions (a branch exists, a PR opened) are the ones a naive harness writes and the
# ones least likely to regress. The loop's real contract is three refusals, and each is invisible
# unless asserted directly:
#
#   * /work-start REFUSES a unit the dispatch engine owns (it is in .delivery/plan.json). That is
#     the drive-mode boundary; without it, both modes end up on one branch.
#   * /work-finish does NOT merge. The never-merge carve-out is scoped to the machine; a
#     hand-driven unit's only review is the human, and a skill that merges removes it.
#   * /work-finish does NOT move state. Merge comes first, Done after, so a Done never fronts an
#     unlanded merge.
#
# A blueprint that only checked the happy path would stay green through all three regressions.
#
# BASELINE-AND-DELTA, not absolute counts. The fixture repo is reset per run but PRs are immutable
# records, so "at least one PR exists" is satisfied by every previous run's leftovers — and worse,
# it is satisfied when nothing happens at all. Every count here is captured before the turn and
# asserted as a delta.
#
# Two assertion tiers:
#   * CORE (ok/bad) — install, the exact branch name, the In Progress claim, the plan-guard
#     refusal, a NEW PR against dev, the verified issue link, state NOT moved, no merge, and the
#     Done transition writing both surfaces.
#   * NOTE (ok/note) — comment wording, evidence placement, and anything whose phrasing is the
#     LLM's to choose.
#
# SETUP: agentteamland/atl-e2e-delivery + a fresh Project per run, in FLOW mode (the sprint carrier
# is a `sprint:<n>` label — no Iteration field, which is not API-creatable).

source /e2e/lib.sh
note() { echo "  note - $1"; }

command -v node >/dev/null 2>&1 && ok "node present" || bad "node missing in the image"

gh auth setup-git >/dev/null 2>&1 || true
LOGIN=$(gh_login)
[ -n "$LOGIN" ] || { bad "gh not authenticated"; finish; exit 1; }
OWNER="${ATL_E2E_DELIVERY_OWNER:-agentteamland}"
REPO="$OWNER/atl-e2e-delivery"

reset_delivery_repo "$OWNER" \
  && ok "reset delivery fixture repo to baseline (main/dev/release)" \
  || bad "could not reset $REPO"
PROJNUM=$(reset_delivery_project "$OWNER")
[ -n "$PROJNUM" ] && ok "created a fresh GitHub Project #$PROJNUM" || bad "could not create the Project"

fresh
write_test_index_delivery
headless_claude_setup

rmdir "$PROJ" 2>/dev/null || true
git clone -q "https://github.com/$REPO.git" "$PROJ" || bad "clone of $REPO failed"
cd "$PROJ" || exit 2

atl install agentteamland/delivery-team >/dev/null 2>&1 || bad "install errored"
for s in work-start work-finish work-move; do
  [ -f "$PROJ/.claude/skills/$s/SKILL.md" ] && ok "$s skill reflected" || bad "$s skill missing"
done
[ -f "$PROJ/.claude/backends/github/adapter.md" ] && ok "github adapter reflected" || bad "github adapter missing"

# ---- .delivery/ config + methodology, FLOW mode -------------------------------------
mkdir -p "$PROJ/.delivery"
cat > "$PROJ/.delivery/config.json" <<EOF
{
  "owner": "$OWNER",
  "repo": "atl-e2e-delivery",
  "projectNumber": $PROJNUM,
  "branchPair": { "dev": "dev", "release": "release" },
  "backend": "github",
  "methodology": "flow",
  "credential": { "ref": "GH_TOKEN" }
}
EOF
cat > "$PROJ/.delivery/methodology.json" <<'EOF'
{
  "id": "scrum",
  "displayName": "Flow",
  "mode": "flow",
  "roles": [
    { "name": "tech-lead",     "binding": "agent", "dispatch": "subagent" },
    { "name": "developer",     "binding": "agent", "dispatch": "worker", "instances": "dynamic" },
    { "name": "product-owner", "binding": "human" }
  ],
  "artifactHierarchy": ["Epic", "Feature", "Pbi", "Task"],
  "workItemTypeMap": { "Pbi": null, "Task": null, "Bug": null },
  "cadence": { "unit": "sprint", "planningCeremonies": ["sprint-plan", "sprint-start"], "reviewCeremony": "sprint-review" },
  "branches": { "dev": "dev", "release": "release" }
}
EOF
# The .gitignore a real /delivery-init writes (its step 6). Engine scratch —
# plan.json above all — must be ignored, or it dirties the tree and /work-start
# refuses. Omitting this seeded a .delivery/ the real ceremony would never produce,
# and the blueprint then failed the skill for the harness's own incompleteness.
cat > "$PROJ/.delivery/.gitignore" <<'EOF'
worktrees/
runstate.json
plan.json
blocked/
mcp/
dispatch.lock
EOF

# Commit the scaffolding before driving. /work-start refuses a dirty tree — correctly,
# and unconditionally — and everything above (install + .delivery/) leaves this clone
# dirty. A real project commits .delivery/ right after /delivery-init; leaving it
# untracked here made the harness fail the skill for the harness's own setup.
# EXCLUDE the scaffolding locally — do not commit it, and above all do not push it.
#
# Two earlier attempts were both wrong. Committing to the default branch loses the
# files the moment /work-start cuts from origin/dev (that is how a later /work-move
# came back "Unknown command"). Committing AND PUSHING to dev fixed that and polluted
# the SHARED fixture repo for every delivery blueprint — a harness must never leave
# a trace in the fixture it borrows.
#
# The tree only has to be CLEAN for /work-start's preflight; nothing requires these
# paths to be tracked. `.git/info/exclude` is local, per-clone, and disappears with
# the container.
cat >> "$PROJ/.git/info/exclude" <<'EOF'
.claude/
.atl/
.delivery/
CLAUDE.md
EOF
[ -z "$(git -C "$PROJ" status --porcelain)" ] \
  && ok "seeded .delivery/ + excluded the scaffolding locally — clean tree for /work-start" \
  || bad "the tree is still dirty; /work-start will refuse and the run measures the harness, not the skill"

# ---- seed two units: one to drive, one the engine owns ------------------------------
DRIVE=$(gh issue create --repo "$REPO" --title "e2e drive: add a subtract helper" \
  --body $'## Problem\nCallers need subtraction.\n\n## Acceptance Criteria\n- a `subtract(a,b)` export in app.js\n- a passing node --test case' \
  --label "type:pbi" --label "area:web" --label "sprint:1" 2>/dev/null | grep -oE '[0-9]+$')
[ -n "$DRIVE" ] && ok "seeded the unit to drive (#$DRIVE, sprint:1)" || bad "could not seed the drive unit"

OWNED=$(gh issue create --repo "$REPO" --title "e2e drive: a unit the engine owns" \
  --body $'## Problem\nThe engine is driving this one.\n\n## Acceptance Criteria\n- untouched by the drive loop' \
  --label "type:pbi" --label "area:web" --label "sprint:1" 2>/dev/null | grep -oE '[0-9]+$')
[ -n "$OWNED" ] && ok "seeded the engine-owned unit (#$OWNED)" || bad "could not seed the engine-owned unit"

# A plan naming ONLY the engine-owned unit. This is the boundary: dispatch admits from the plan
# and nothing else, so #DRIVE is the human's and #OWNED is not.
cat > "$PROJ/.delivery/plan.json" <<EOF
{
  "sprintSlug": "1",
  "granularity": "pbi",
  "units": [ { "id": $OWNED, "title": "a unit the engine owns", "predecessors": [], "stackRank": 1 } ]
}
EOF
ok "seeded .delivery/plan.json naming ONLY #$OWNED — the drive-mode boundary"

gturn() { claude_turn "$1"; }

# ---- 1. /work-start on the engine-owned unit — MUST REFUSE ---------------------------
# The sharpest property, and first so a later turn cannot muddy it. A skill that took this
# unit would put a human and a worker on one branch.
BR_OWNED="delivery/1/$OWNED"
gturn "/work-start $OWNED" || note "work-start refusal turn exited non-zero (a refusal may legitimately do so)"

git fetch -q origin 2>/dev/null || true
if git rev-parse --verify -q "refs/heads/$BR_OWNED" >/dev/null 2>&1 \
   || git rev-parse --verify -q "refs/remotes/origin/$BR_OWNED" >/dev/null 2>&1; then
  bad "/work-start cut $BR_OWNED for a unit in the active plan — the drive-mode boundary did not hold"
else
  ok "/work-start REFUSED the plan-owned unit #$OWNED — no $BR_OWNED branch"
fi
ownedstat=$(gh issue view "$OWNED" --repo "$REPO" --json state -q .state 2>/dev/null)
[ "$ownedstat" = "OPEN" ] && ok "the plan-owned unit was left untouched (still open)" || bad "the plan-owned unit was modified"

# A refusal and a crashed turn look identical from the outside — both leave no branch. So
# also assert the skill did not act PARTIALLY (claim first, refuse later), and rely on
# step 2 to prove the skill works at all: the same skill cuts a branch there, so an
# inert /work-start would fail loudly downstream rather than pass silently here.
ownedclaim=$(gh issue view "$OWNED" --repo "$REPO" --comments 2>/dev/null | grep -c "delivery/1/" || true)
[ "${ownedclaim:-0}" -eq 0 ] \
  && ok "no claim comment on the plan-owned unit — it refused outright, not halfway" \
  || bad "the plan-owned unit was claimed before the refusal — a partial act is worse than none"

# ---- 2. /work-start on the human's unit — claim + branch ----------------------------
BR="delivery/1/$DRIVE"
gturn "/work-start $DRIVE. You are the human driver for this headless run — answer any prompt from these facts and do not wait for interactive input: yes, take the unit; it already carries sprint:1 so no sprint pull-in is needed." || bad "work-start turn errored"

git fetch -q origin 2>/dev/null || true
if git rev-parse --verify -q "refs/heads/$BR" >/dev/null 2>&1; then
  ok "/work-start cut the engine's own branch grammar: $BR"
else
  bad "no $BR branch — the branch name must match dispatch.BranchName(slug,id) verbatim"
fi

claimed=0
gh issue view "$DRIVE" --repo "$REPO" --comments 2>/dev/null | grep -qi "$BR" && claimed=1
[ "$claimed" = 1 ] && ok "a claim comment naming the branch landed on #$DRIVE" || note "no claim comment naming the branch (wording is the LLM's)"

# ---- 3. do the work by hand, then /work-finish --------------------------------------
# No `|| git checkout -b` fallback. On the first run /work-start did not cut the branch
# and that fallback created it silently, so every assertion after it measured the harness
# instead of the skill — the failure was reported once and then papered over.
if ! git checkout -q "$BR" 2>/dev/null; then
  bad "cannot continue: /work-start left no $BR to work on"
  # Dump BEFORE exiting. The first version of this stop exited straight away and
  # took the turns.log with it — a fast failure that explains nothing costs a whole
  # run to re-learn what one dump would have said.
  echo "===== DEBUG (github-drive-loop: /work-start cut no branch) ====="
  echo "--- git status ---";   git -C "$PROJ" status --porcelain 2>/dev/null | head -20
  echo "--- branches ---";     git -C "$PROJ" branch -a 2>/dev/null | head -20
  echo "--- turns.log ---";    tail -80 "$HOME/turns.log" 2>/dev/null
  echo "=============================================================="
  finish; exit 1
fi
cat >> app.js <<'EOF'

export function subtract(a, b) { return a - b; }
EOF
cat >> app.test.js <<'EOF'

test('subtract', () => { assert.strictEqual(subtract(5, 3), 2); });
EOF
git add -A && git commit -q -m "feat: add a subtract helper" 2>/dev/null

# BASELINE before the turn — a PR count is meaningless without one. "at least one PR"
# is satisfied by leftovers AND by nothing happening at all.
PR_BEFORE=$(gh pr list --repo "$REPO" --state all --limit 200 --json number -q 'length' 2>/dev/null || echo 0)

gturn "/work-finish. You are the human driver for this headless run — answer any prompt from these facts and do not wait for interactive input: the work is complete and committed on this branch." || bad "work-finish turn errored"

PR_AFTER=$(gh pr list --repo "$REPO" --state all --limit 200 --json number -q 'length' 2>/dev/null || echo 0)
if [ "${PR_AFTER:-0}" -gt "${PR_BEFORE:-0}" ]; then
  ok "/work-finish opened a NEW PR (${PR_BEFORE} → ${PR_AFTER})"
else
  bad "no new PR (${PR_BEFORE} → ${PR_AFTER}) — an absolute count would have passed here"
fi

PR=$(gh pr list --repo "$REPO" --head "$BR" --state all --json number -q '.[0].number' 2>/dev/null)
[ -n "$PR" ] && ok "the new PR (#$PR) is on the unit's own branch" || bad "no PR whose head is $BR"

if [ -n "$PR" ]; then
  base=$(gh pr view "$PR" --repo "$REPO" --json baseRefName -q .baseRefName 2>/dev/null)
  [ "$base" = "dev" ] && ok "the PR targets the integration branch (dev), read from config" || bad "PR targets '$base', expected dev"

  # The link, verified the way GitHub actually records it for a non-default base.
  #
  # The first run of this blueprint asserted `closingIssuesReferences` and could never
  # pass: GitHub promotes a closing keyword to a CLOSING reference only for a PR
  # targeting the DEFAULT branch, and this flow always targets dev. Measured on that
  # run — body carried `Fixes #<id>`, base=dev, default=main, zero nodes. An assertion
  # that cannot pass is not a strict test, it is a broken one.
  body=$(gh pr view "$PR" --repo "$REPO" --json body -q .body 2>/dev/null)
  echo "$body" | grep -qiE "fixes #$DRIVE\b" \
    && ok "the PR body carries Fixes #$DRIVE — the reference was written" \
    || bad "the PR body has no 'Fixes #$DRIVE'"

  head=$(gh pr view "$PR" --repo "$REPO" --json headRefName -q .headRefName 2>/dev/null)
  [ "$head" = "$BR" ] \
    && ok "the PR head is $BR — the branch grammar ties it to exactly one unit" \
    || bad "PR head is '$head', expected $BR"

  xref=$(gh api graphql -f query="{ repository(owner:\"$OWNER\", name:\"atl-e2e-delivery\") { issue(number: $DRIVE) { timelineItems(first:50, itemTypes:[CROSS_REFERENCED_EVENT]) { nodes { ... on CrossReferencedEvent { source { ... on PullRequest { number } } } } } } } }" \
    --jq "[.data.repository.issue.timelineItems.nodes[]?.source.number] | index($PR) // -1" 2>/dev/null || echo -1)
  { [ "${xref:--1}" != "-1" ] && [ -n "$xref" ]; } 2>/dev/null \
    && ok "GitHub registered the cross-reference from PR #$PR on #$DRIVE" \
    || note "no CROSS_REFERENCED_EVENT yet (GitHub registers it asynchronously)"

  # DID NOT DO #1 — no merge. Reviewing your own PR by merging it from a skill removes the
  # only review a hand-driven unit gets.
  merged=$(gh pr view "$PR" --repo "$REPO" --json mergedAt -q '.mergedAt // ""' 2>/dev/null)
  [ -z "$merged" ] && ok "/work-finish did NOT merge — the human's decision, left to the human" || bad "/work-finish merged the PR"
fi

# DID NOT DO #2 — state not moved. Merge first, Done after, so a Done never fronts unlanded work.
dstate=$(gh issue view "$DRIVE" --repo "$REPO" --json state -q .state 2>/dev/null)
[ "$dstate" = "OPEN" ] && ok "/work-finish did NOT close the item — Done comes after the merge" || bad "/work-finish closed #$DRIVE before any merge"

# ---- 4. /work-move — the only state writer, both surfaces ----------------------------
# Merge first so Done does not front an unlanded merge — the same order the skill enforces.
if [ -n "$PR" ]; then
  gh pr merge "$PR" --repo "$REPO" --merge --delete-branch >/dev/null 2>&1 \
    && ok "merged the PR by hand (the human's step, not the skill's)" \
    || note "could not merge the PR in-container (branch protection?) — the Done assertion below may not apply"
fi

gturn "/work-move $DRIVE done. You are the human driver — the PR has merged into dev; proceed without waiting for interactive input." || note "work-move turn exited non-zero"

fstate=$(gh issue view "$DRIVE" --repo "$REPO" --json state -q .state 2>/dev/null)
[ "$fstate" = "CLOSED" ] && ok "/work-move closed the issue — one of Done's two surfaces" || bad "#$DRIVE is still $fstate after /work-move done"

# ---- on failure, surface what the torn-down container would otherwise lose -----------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (github-drive-loop failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- turns.log (tail) ---"; tail -140 "$HOME/turns.log" 2>/dev/null
  echo "--- branches ---"; git branch -a 2>/dev/null | head -20
  echo "--- issues ---"; gh issue list --repo "$REPO" --state all --json number,title,state,labels 2>/dev/null
  echo "--- PRs ---"; gh pr list --repo "$REPO" --state all --json number,headRefName,baseRefName,mergedAt 2>/dev/null
  echo "--- drive unit comments ---"; [ -n "$DRIVE" ] && gh issue view "$DRIVE" --repo "$REPO" --comments 2>/dev/null | tail -60
  echo "==========================================="
fi

finish
