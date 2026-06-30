#!/usr/bin/env bash
# needs: none
#
# install — a clean user installs the fixture team at both scopes; assets +
# manifests land, and the embedded core (rules + skills) reflects to global.
source /e2e/lib.sh

fresh
write_test_index
cd "$PROJ" || exit 2

atl install --global  agentteamland/atl-e2e-team || bad "global install errored"
atl install --project agentteamland/atl-e2e-team || bad "project install errored"

[ -f "$HOME/.claude/agents/e2e-agent/agent.md" ] && ok "global agent reflected"  || bad "global agent missing"
[ -f "$PROJ/.claude/agents/e2e-agent/agent.md" ] && ok "project agent reflected" || bad "project agent missing"
[ -f "$HOME/.claude/skills/e2e-skill/SKILL.md" ] && ok "global skill reflected"  || bad "global skill missing"
ls "$HOME/.atl/installed/"*.json >/dev/null 2>&1 && ok "global manifest written"  || bad "global manifest missing"
ls "$PROJ/.atl/installed/"*.json >/dev/null 2>&1 && ok "project manifest written" || bad "project manifest missing"
[ -f "$HOME/.claude/rules/learning-capture.md" ] && ok "core rules reflected"      || bad "core rules missing"
[ -f "$HOME/.claude/skills/drain/SKILL.md" ]     && ok "core drain skill reflected" || bad "core drain missing"

# install drops a project CLAUDE.md starter when the project has none (only-if-absent)
[ -f "$PROJ/CLAUDE.md" ] && ok "install scaffolded a project CLAUDE.md" || bad "project CLAUDE.md not scaffolded"

finish
