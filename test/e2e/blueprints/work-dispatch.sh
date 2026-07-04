#!/usr/bin/env bash
# needs: none
#
# work-dispatch — the delivery-team's Go orchestration engine (the hidden
# `atl work dispatch`). Auth-free + Claude-free: a fake `claude` on PATH stands in
# for a worker (writes a terminal status.json, commits its change, exits 0), so
# the real supervisor loop — validate → admit up to cap → poll → complete →
# teardown → refill — runs end to end against a real git repo with an origin/dev.
# Also covers the two deterministic refusals (no plan / cyclic plan), which need
# no worker at all.
source /e2e/lib.sh

fresh
mkdir -p "$HOME/fakebin"
export PATH="$HOME/fakebin:$PATH"

# The fake worker: ignore the `claude -p …` argv, act on the cwd (the unit's
# worktree). Write a terminal status.json, commit the change, MERGE it to dev
# (push HEAD:dev — the supervisor verifies the merge via MergedToBase before it
# tears down), and exit 0. Committing keeps the tree clean; merging makes the
# branch contained in origin/dev so completion is confirmed, not just trusted.
cat > "$HOME/fakebin/claude" <<'EOF'
#!/usr/bin/env bash
printf '{"phase":"done","heartbeatTs":"2026-07-04T12:00:00Z","blocker":"","lastOutputSummary":"implemented"}\n' > status.json
echo "work" > impl.txt
git add impl.txt
git commit -q -m "work-unit change"
git push -q origin HEAD:dev   # integrate to dev (fast-forward) — the merge the supervisor confirms
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

out="$(atl work dispatch --cap 2 2>&1)"; rc=$?
echo "$out"
[ "$rc" -eq 0 ] && ok "dispatch ran to completion" || bad "dispatch errored -- rc=$rc"
echo "$out" | grep -q "2 done" && ok "both work-units reported done" || bad "not all units done -- [$out]"
# Merged units → teardown; no leftover per-unit worktree.
leftover="$(ls "$WORK/.delivery/worktrees/s1" 2>/dev/null | wc -l | tr -d ' ')"
[ "$leftover" = "0" ] && ok "merged worktrees were reclaimed" || bad "worktrees left behind: $leftover"
[ -f "$WORK/.delivery/runstate.json" ] && ok "run-state was persisted (restart substrate)" || bad "no runstate.json"

finish
