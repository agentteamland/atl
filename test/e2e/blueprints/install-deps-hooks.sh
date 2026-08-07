#!/usr/bin/env bash
# needs: none
# touches: cli/cmd/atl/commands/install.go, cli/internal/settings, cli/internal/manifest, cli/internal/teampkg, cli/internal/index, teams/personal-advisory-team, teams/profile-team
#
# install-deps-hooks — the three things `atl install` does BESIDES copying the
# team you asked for, none of which any blueprint exercised: it pulls the team's
# declared dependencies in transitively, it binds the automation hooks (decision
# doc D-3, "automation is mandatory"), and a RE-install converges instead of
# clobbering.
#
# The subject is personal-advisory-team, which is also the only first-party team
# no blueprint installs at all. That is not a coincidence — it is the one team
# whose team.json declares a `dependencies` entry ("profile-team": "^1.0.0"), so
# it is simultaneously the missing-coverage case and the ready-made transitive
# fixture. install.sh (atl-e2e-team) and profile-install.sh (profile-team) both
# install dependency-free teams, so `installWithDeps`' recursion had never run
# under test.
#
# Every assertion here is written against a BASELINE taken first, because each of
# the three properties has a null outcome that reads as a pass otherwise:
#   * transitive install — "profile-team is installed" is trivially true if
#     something else installed it, so the blueprint proves it was ABSENT before a
#     command that never named it.
#   * hook binding — "settings.json contains atl hooks" would pass on a leftover
#     file, so it proves ZERO atl-owned hooks existed before the install.
#   * re-install idempotency — "hooks are not duplicated" is satisfied by having
#     no hooks at all, and "the local edit survived" is satisfied by a file that
#     was never reflected. Both are asserted as exact counts against state this
#     blueprint put there, not as absences.
#
# Auth-free: both teams are monorepo subpaths fetched over public HTTPS, the same
# way profile-install.sh reaches profile-team.
source /e2e/lib.sh

fresh
write_test_index_advisory
cd "$PROJ" || exit 2

SETTINGS="$HOME/.claude/settings.json"

# hookcount <event> <command> — how many hook groups under <event> carry exactly
# <command>. The count, not the presence: duplication is the re-install failure
# mode, and a `grep -q` cannot tell one binding from three.
hookcount() {
  jq -r --arg e "$1" --arg c "$2" \
    '[.hooks[$e][]? | .hooks[]? | select(.command == $c)] | length' "$SETTINGS" 2>/dev/null || echo 0
}
# atlhooks — total atl-owned hook commands across every event.
atlhooks() {
  jq -r '[.hooks[]?[]? | .hooks[]? | .command | select(startswith("atl "))] | length' "$SETTINGS" 2>/dev/null || echo 0
}

# ---- BASELINE ---------------------------------------------------------------
#
# A settings.json that is NOT ours and NOT empty: a user-authored hook on an event
# atl also writes to (UserPromptSubmit — the collision case), plus a top-level key
# that is not `hooks` at all. Both must survive, and the second is the sharper of
# the two: InstallHooks rewrites the whole file, so a bug there costs the user
# their permissions/env/statusline, not just a hook.
mkdir -p "$HOME/.claude"
cat > "$SETTINGS" <<'EOF'
{
  "env": { "E2E_USER_KEY": "preserved" },
  "hooks": {
    "UserPromptSubmit": [
      { "hooks": [ { "type": "command", "command": "echo e2e-user-hook" } ] }
    ]
  }
}
EOF
[ "$(atlhooks)" = "0" ] && ok "baseline: settings.json carries no atl-owned hook" \
                        || bad "baseline is not clean — $(atlhooks) atl hook(s) already bound"
ls "$HOME/.atl/installed/"*profile-team*.json >/dev/null 2>&1 \
  && bad "baseline is not clean — profile-team is already installed" \
  || ok "baseline: profile-team is NOT installed"

# ---- 1. INSTALL — the consumer only. profile-team is never named. -------------
out="$(atl install agentteamland/personal-advisory-team 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && ok "install of personal-advisory-team returned rc=0" \
                || bad "install errored (rc=$rc) — $(echo "$out" | tail -3 | tr '\n' ' ')"

# The team asked for.
[ -f "$HOME/.claude/agents/advisor/agent.md" ]         && ok "advisor agent reflected"        || bad "advisor agent missing"
[ -f "$HOME/.claude/skills/advisor/SKILL.md" ]         && ok "/advisor skill reflected"       || bad "/advisor skill missing"
[ -f "$HOME/.claude/skills/advisor-home/SKILL.md" ]    && ok "/advisor-home skill reflected"  || bad "/advisor-home skill missing"
ls "$HOME/.atl/installed/"*personal-advisory-team*.json >/dev/null 2>&1 \
  && ok "personal-advisory-team manifest written" || bad "personal-advisory-team manifest missing"

# ---- 2. TRANSITIVE INSTALL — the dependency the command never mentioned -------
#
# Asserted three ways, because each catches a different half. The manifest proves
# the install RAN; the assets prove it actually reflected files rather than only
# recording a row; the report's "(dependency)" suffix proves the CLI knows the
# difference (it is the only surface a user has for "why is this here?").
ls "$HOME/.atl/installed/"*profile-team*.json >/dev/null 2>&1 \
  && ok "profile-team installed transitively (absent at baseline, never named on the command line)" \
  || bad "profile-team NOT installed — installWithDeps did not follow the dependencies edge"
[ -f "$HOME/.claude/agents/profile-curator/agent.md" ] && ok "the dependency's curator agent reflected"   || bad "profile-curator agent missing"
[ -f "$HOME/.claude/skills/profile-drain/SKILL.md" ]   && ok "the dependency's /profile-drain reflected"  || bad "profile-drain skill missing"
[ -f "$HOME/.claude/rules/profile-capture.md" ]        && ok "the dependency's profile-capture rule reflected" || bad "profile-capture rule missing"
echo "$out" | grep -qE 'installed agentteamland/profile-team@[^ ]+ at global scope \(dependency\)' \
  && ok "install reported profile-team as a (dependency)" \
  || bad "install report did not mark profile-team a dependency — [$(echo "$out" | tr '\n' ' ')]"

# ---- 3. HOOK BINDING (D-3: automation is mandatory) --------------------------
#
# Every hook in defaultHooks(), by event AND exact command. A blanket "some atl
# hook exists" would go green with four of the five silently dropped, and the
# ones that carry the whole learning + retrieval loop are the easiest to lose.
[ "$(hookcount SessionStart 'atl session-start')" = "1" ]        && ok "SessionStart -> atl session-start bound"            || bad "SessionStart hook not bound exactly once"
[ "$(hookcount UserPromptSubmit 'atl tick --throttle=10m')" = "1" ] && ok "UserPromptSubmit -> atl tick --throttle=10m bound" || bad "atl tick hook not bound exactly once"
[ "$(hookcount UserPromptSubmit 'atl retrieve')" = "1" ]         && ok "UserPromptSubmit -> atl retrieve bound"             || bad "atl retrieve hook not bound exactly once"
[ "$(hookcount PreToolUse 'atl guard')" = "1" ]                  && ok "PreToolUse -> atl guard bound"                      || bad "atl guard hook not bound exactly once"
[ "$(hookcount Stop 'atl retrieve turn-end')" = "1" ]            && ok "Stop -> atl retrieve turn-end bound"                || bad "atl retrieve turn-end hook not bound exactly once"

# The matcher is not decoration: without it the guard fires on EVERY tool call.
# It is also the field a Hook{} literal can lose silently — the guard still binds,
# still runs, and the settings file still looks right.
MATCHER=$(jq -r '[.hooks.PreToolUse[]? | select([.hooks[]?.command] | index("atl guard")) | .matcher // ""] | first // ""' "$SETTINGS" 2>/dev/null)
[ "$MATCHER" = "Bash|Edit|Write" ] && ok "the guard hook carries its Bash|Edit|Write matcher" \
                                   || bad "guard matcher is '$MATCHER', not Bash|Edit|Write"

# The user's own file is not ours to rewrite.
[ "$(hookcount UserPromptSubmit 'echo e2e-user-hook')" = "1" ] \
  && ok "the user's own UserPromptSubmit hook survived the install" \
  || bad "the user's own hook was dropped — isAtlGroup filtered a group it does not own"
[ "$(jq -r '.env.E2E_USER_KEY // ""' "$SETTINGS" 2>/dev/null)" = "preserved" ] \
  && ok "a non-hook top-level key survived the install" \
  || bad "a non-hook settings key was lost — InstallHooks overwrote the file"

# ---- 4. RE-INSTALL — converges, never clobbers -------------------------------
#
# The documented guarantee is "pull, never clobber": a re-install reflects over
# the existing baseline so files the user or the learning loop modified survive.
# That is exactly what a naive "copy the assets again" would break, and nothing
# about the second install's output would say so.
EDIT_MARK="e2e-local-edit-must-survive-reinstall"
printf '\n<!-- %s -->\n' "$EDIT_MARK" >> "$HOME/.claude/agents/advisor/agent.md"
grep -q "$EDIT_MARK" "$HOME/.claude/agents/advisor/agent.md" \
  && ok "seeded a local edit in the reflected advisor agent" \
  || bad "could not seed the local edit — the rest of section 4 would be vacuous"

before_manifests=$(ls "$HOME/.atl/installed/"*.json 2>/dev/null | wc -l | tr -d ' ')
before_atl=$(atlhooks)

out2="$(atl install agentteamland/personal-advisory-team 2>&1)"; rc2=$?
[ "$rc2" -eq 0 ] && ok "re-install returned rc=0" \
                 || bad "re-install errored (rc=$rc2) — $(echo "$out2" | tail -3 | tr '\n' ' ')"

grep -q "$EDIT_MARK" "$HOME/.claude/agents/advisor/agent.md" \
  && ok "the local edit SURVIVED the re-install (pull, never clobber)" \
  || bad "the re-install clobbered a locally modified reflected file"

# Hook count is unchanged, not merely non-zero: InstallHooks filters this event's
# existing atl groups ONCE and then appends every wanted group. Getting that wrong
# duplicates on the events carrying two hooks, which is precisely the event a user
# hook also lives on.
after_atl=$(atlhooks)
[ "$after_atl" = "$before_atl" ] && [ "$after_atl" = "5" ] \
  && ok "re-install left exactly 5 atl hooks bound (no duplication)" \
  || bad "atl hook count went $before_atl -> $after_atl (expected a stable 5)"
[ "$(hookcount UserPromptSubmit 'atl retrieve')" = "1" ] \
  && ok "the twice-written UserPromptSubmit event still has one atl retrieve" \
  || bad "atl retrieve duplicated on re-install"
[ "$(hookcount UserPromptSubmit 'echo e2e-user-hook')" = "1" ] \
  && ok "the user's own hook survived the SECOND install too" \
  || bad "the user's own hook was lost on re-install"

after_manifests=$(ls "$HOME/.atl/installed/"*.json 2>/dev/null | wc -l | tr -d ' ')
[ "$after_manifests" = "$before_manifests" ] && [ "$after_manifests" = "2" ] \
  && ok "still exactly 2 install manifests after the re-install (converged, not duplicated)" \
  || bad "manifest count went $before_manifests -> $after_manifests (expected a stable 2)"

lst="$(atl list 2>&1)"
echo "$lst" | grep -q "personal-advisory-team" && ok "list shows personal-advisory-team" || bad "list missing personal-advisory-team -- [$lst]"
echo "$lst" | grep -q "profile-team"           && ok "list shows the transitive profile-team" || bad "list missing profile-team -- [$lst]"

# ---- on failure, surface what the torn-down container would otherwise lose ----
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (install-deps-hooks failed) ====="
  echo "--- first install output ---";  echo "${out:-<none>}"
  echo "--- re-install output ---";     echo "${out2:-<none>}"
  echo "--- settings.json ---";         cat "$SETTINGS" 2>/dev/null
  echo "--- ~/.atl/installed ---";      ls -la "$HOME/.atl/installed/" 2>/dev/null
  echo "--- ~/.claude tree ---";        find "$HOME/.claude" -maxdepth 3 2>/dev/null | head -60
  echo "============================================"
fi

finish
