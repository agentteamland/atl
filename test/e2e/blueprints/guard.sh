#!/usr/bin/env bash
# needs: none
#
# guard — `atl guard` (the PreToolUse hook) reads a hook JSON on stdin and prints
# a deterministic decision on stdout. The catastrophe layer denies an irreversible
# Bash op; the quality layer injects a grep-before-edit nudge on a file's first
# edit and stays silent on the next; new-file creation and ordinary commands pass.
# Pure stdin -> stdout: no install, no network, no Claude — always-on backbone.
source /e2e/lib.sh

# catastrophe layer — an irreversible op is denied
out="$(echo '{"session_id":"s1","tool_name":"Bash","tool_input":{"command":"git push --force origin main"}}' | atl guard 2>&1)"
echo "$out" | grep -q '"permissionDecision":"deny"' && ok "force-push is denied" || bad "force-push not denied -- [$out]"

# the safe variant is allowed (no decision emitted)
out="$(echo '{"session_id":"s1","tool_name":"Bash","tool_input":{"command":"git push --force-with-lease"}}' | atl guard 2>&1)"
[ -z "$out" ] && ok "force-with-lease passes" || bad "force-with-lease wrongly handled -- [$out]"

# an ordinary command passes silently
out="$(echo '{"session_id":"s1","tool_name":"Bash","tool_input":{"command":"go test ./..."}}' | atl guard 2>&1)"
[ -z "$out" ] && ok "ordinary command passes" || bad "ordinary command wrongly handled -- [$out]"

# quality layer — first edit of an existing file injects a nudge, no permission decision
SEEN="$HOME/seen.txt"; echo x > "$SEEN"
out="$(echo '{"session_id":"sX","tool_name":"Edit","tool_input":{"file_path":"'"$SEEN"'"}}' | atl guard 2>&1)"
echo "$out" | grep -q '"additionalContext"' && ok "first edit injects a nudge" || bad "first edit had no nudge -- [$out]"
echo "$out" | grep -q '"permissionDecision"' && bad "nudge wrongly set a permission decision -- [$out]" || ok "nudge omits a permission decision"

# the second edit of the same file in the same session is silent
out="$(echo '{"session_id":"sX","tool_name":"Edit","tool_input":{"file_path":"'"$SEEN"'"}}' | atl guard 2>&1)"
[ -z "$out" ] && ok "second edit is silent" || bad "second edit not silent -- [$out]"

# new-file creation is exempt (nothing to grep)
out="$(echo '{"session_id":"sX","tool_name":"Write","tool_input":{"file_path":"'"$HOME/brand-new.txt"'"}}' | atl guard 2>&1)"
[ -z "$out" ] && ok "new-file write is exempt" || bad "new-file write wrongly nudged -- [$out]"

# malformed input never blocks (never fail a hook)
out="$(echo 'not json' | atl guard 2>&1)"; rc=$?
{ [ -z "$out" ] && [ "$rc" -eq 0 ]; } && ok "malformed input is a safe no-op" || bad "malformed input not safe -- rc=$rc [$out]"

finish
