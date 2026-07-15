#!/usr/bin/env bash
# needs: none
#
# init — `atl init` scaffolds a starter CLAUDE.md per tier, only if one doesn't
# already exist (never clobbers a user file), and the flags are mutually
# exclusive. Pure filesystem, no network — always-on backbone.
source /e2e/lib.sh

fresh
cd "$PROJ" || exit 2

# default = project tier: creates CLAUDE.md with the owned fact sections
atl init >/dev/null || bad "init errored"
[ -f "$PROJ/CLAUDE.md" ] && ok "init created a project CLAUDE.md" || bad "no CLAUDE.md created"
grep -q "## Stack" "$PROJ/CLAUDE.md" && ok "project skeleton has the owned sections" || bad "project skeleton wrong"

# project tier also scaffolds the .atl/ decision-state files (backlog + tasks)
[ -f "$PROJ/.atl/backlog.md" ] && ok "init created .atl/backlog.md" || bad "no .atl/backlog.md created"
[ -f "$PROJ/.atl/tasks.md" ] && ok "init created .atl/tasks.md" || bad "no .atl/tasks.md created"
grep -q "# Backlog" "$PROJ/.atl/backlog.md" && ok "backlog skeleton has content" || bad "backlog skeleton wrong"

# idempotent: a second init must not overwrite the user's files
echo "USER OWNED LINE" >> "$PROJ/CLAUDE.md"
echo "USER BACKLOG LINE" >> "$PROJ/.atl/backlog.md"
atl init >/dev/null
grep -q "USER OWNED LINE" "$PROJ/CLAUDE.md" && ok "re-run left the existing CLAUDE.md untouched" || bad "init clobbered an existing CLAUDE.md"
grep -q "USER BACKLOG LINE" "$PROJ/.atl/backlog.md" && ok "re-run left .atl/backlog.md untouched" || bad "init clobbered an existing backlog.md"

# global tier writes the persona into ~/.claude (no managed marker blocks)
atl init --global >/dev/null || bad "init --global errored"
[ -f "$HOME/.claude/CLAUDE.md" ] && ok "init --global created ~/.claude/CLAUDE.md" || bad "no global CLAUDE.md"
grep -q ":start -->" "$HOME/.claude/CLAUDE.md" && bad "global persona must carry no managed blocks" || ok "global persona has no managed blocks"

# mutually exclusive flags fail cleanly
atl init --global --project >/dev/null 2>&1 && bad "mutually exclusive flags were accepted" || ok "mutually exclusive flags rejected"

finish
