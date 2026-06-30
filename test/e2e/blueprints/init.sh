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

# idempotent: a second init must not overwrite the user's file
echo "USER OWNED LINE" >> "$PROJ/CLAUDE.md"
atl init >/dev/null
grep -q "USER OWNED LINE" "$PROJ/CLAUDE.md" && ok "re-run left the existing file untouched" || bad "init clobbered an existing CLAUDE.md"

# global tier writes the persona into ~/.claude (no managed marker blocks)
atl init --global >/dev/null || bad "init --global errored"
[ -f "$HOME/.claude/CLAUDE.md" ] && ok "init --global created ~/.claude/CLAUDE.md" || bad "no global CLAUDE.md"
grep -q ":start -->" "$HOME/.claude/CLAUDE.md" && bad "global persona must carry no managed blocks" || ok "global persona has no managed blocks"

# mutually exclusive flags fail cleanly
atl init --global --project >/dev/null 2>&1 && bad "mutually exclusive flags were accepted" || ok "mutually exclusive flags rejected"

finish
