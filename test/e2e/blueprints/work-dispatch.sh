#!/usr/bin/env bash
# needs: none
#
# work-dispatch — the delivery-team's Go orchestration engine (the hidden
# `atl work dispatch`). Auth-free + Claude-free: a fake `claude` on PATH stands in
# for a worker, so the real supervisor loop — validate → admit up to cap → poll →
# advance stage → complete → teardown → refill — runs end to end against a real git
# repo with an origin/dev. The per-unit PIPELINE (developer → tester → tech-lead in
# one worktree; #8 back-half Phase 2) is exercised: only the tech-lead stage merges,
# and the engine merge-verifies only after it. Also covers the two deterministic
# refusals (no plan / cyclic plan), which need no worker at all.
source /e2e/lib.sh

fresh
mkdir -p "$HOME/fakebin"
export PATH="$HOME/fakebin:$PATH"

# The fake worker keys off the stage-role named in its prompt (`claude -p <prompt>`)
# — the same token the real stage prompts open with — and acts on the cwd (the unit's
# reused worktree). It appends its role to $HOME/stagelog so the test can prove all
# three stages ran per unit:
#   developer → implement + commit (opens a "PR"; does NOT merge)
#   tester    → attach evidence + commit (does NOT merge)
#   tech-lead → complete the PR = merge to dev (push HEAD:dev), then exit
# The supervisor advances stage on each exit-0 and only verifies the merge (via
# MergedToBase) after the tech-lead stage — so a non-tech-lead stage that never
# merged is never mistaken for a completed unit.
cat > "$HOME/fakebin/claude" <<'EOF'
#!/usr/bin/env bash
prompt="$*"
case "$prompt" in
  *"delivery-team developer"*)
    echo developer >> "$HOME/stagelog"
    printf '{"phase":"pr","heartbeatTs":"2026-07-04T12:00:00Z","blocker":"","lastOutputSummary":"PR opened"}\n' > status.json
    echo work > impl.txt; git add impl.txt; git commit -q -m "work-unit change (developer)" ;;
  *"delivery-team tester"*)
    echo tester >> "$HOME/stagelog"
    printf '{"phase":"verify","heartbeatTs":"2026-07-04T12:00:00Z","blocker":"","lastOutputSummary":"verified"}\n' > status.json
    echo evidence > evidence.txt; git add evidence.txt; git commit -q -m "test evidence (tester)" ;;
  *"delivery-team tech-lead"*)
    echo tech-lead >> "$HOME/stagelog"
    printf '{"phase":"done","heartbeatTs":"2026-07-04T12:00:00Z","blocker":"","lastOutputSummary":"reviewed + merged"}\n' > status.json
    git push -q origin HEAD:dev ;;   # completing the PR = the merge the supervisor verifies
  *)
    echo "fake claude: no known delivery stage in prompt" >&2; exit 3 ;;
esac
exit 0
EOF
chmod +x "$HOME/fakebin/claude"

# --- deterministic refusals (no worker) --------------------------------------

mkdir -p "$PROJ"
cd "$PROJ" || exit 2
git init -q
out="$(atl work dispatch 2>&1)"; rc=$?
{ [ "$rc" -ne 0 ] && echo "$out" | grep -q "sprint-start"; } \
  && ok "a missing plan refuses with a /sprint-start hint" || bad "missing-plan not handled -- rc=$rc [$out]"

mkdir -p "$PROJ/.delivery"
cat > "$PROJ/.delivery/plan.json" <<'EOF'
{"sprintSlug":"s1","granularity":"pbi","units":[
  {"id":1,"title":"A","predecessors":[2],"stackRank":1},
  {"id":2,"title":"B","predecessors":[1],"stackRank":2}]}
EOF
out="$(atl work dispatch 2>&1)"; rc=$?
{ [ "$rc" -ne 0 ] && echo "$out" | grep -qi "cycle"; } \
  && ok "a cyclic plan is refused, not silently broken" || bad "cycle not refused -- rc=$rc [$out]"

# --- happy path: real git repo (origin/dev) + a 2-unit dependency chain -------

ORIGIN="$HOME/origin.git"
WORK="$HOME/delivery"
rm -rf "$ORIGIN" "$WORK" "$HOME/seed"
git init -q --bare -b dev "$ORIGIN"
git clone -q "$ORIGIN" "$HOME/seed"
(
  cd "$HOME/seed" || exit 2
  git config user.email e2e@atl.local
  git config user.name atl
  echo seed > seed.txt
  git add -A
  git commit -q -m seed
  git push -q origin dev
)
rm -rf "$HOME/seed"

git clone -q "$ORIGIN" "$WORK"
cd "$WORK" || exit 2
git config user.email e2e@atl.local
git config user.name atl

mkdir -p .delivery
cat > .delivery/plan.json <<'EOF'
{"sprintSlug":"s1","granularity":"pbi","units":[
  {"id":1,"title":"foundation","predecessors":[],"stackRank":1},
  {"id":2,"title":"feature","predecessors":[1],"stackRank":2}]}
EOF

rm -f "$HOME/stagelog"
out="$(atl work dispatch --cap 2 2>&1)"; rc=$?
echo "$out"
[ "$rc" -eq 0 ] && ok "dispatch ran to completion" || bad "dispatch errored -- rc=$rc"
echo "$out" | grep -q "2 done" && ok "both work-units reported done" || bad "not all units done -- [$out]"
# Each of the 2 units ran the FULL 3-stage pipeline (developer→tester→tech-lead) —
# proves the engine spawned three sequential workers per unit, not one.
for role in developer tester tech-lead; do
  n="$(grep -c "^${role}\$" "$HOME/stagelog" 2>/dev/null || echo 0)"
  [ "$n" = "2" ] && ok "both units ran the $role stage" || bad "$role stage ran $n times (want 2) -- $(tr '\n' ',' < "$HOME/stagelog" 2>/dev/null)"
done
# Merged units → teardown; no leftover per-unit worktree.
leftover="$(ls "$WORK/.delivery/worktrees/s1" 2>/dev/null | wc -l | tr -d ' ')"
[ "$leftover" = "0" ] && ok "merged worktrees were reclaimed" || bad "worktrees left behind: $leftover"
[ -f "$WORK/.delivery/runstate.json" ] && ok "run-state was persisted (restart substrate)" || bad "no runstate.json"

finish
