#!/usr/bin/env bash
# needs: none
#
# retrieve — `atl retrieve` (the UserPromptSubmit hook) ranks the project's
# knowledge pages (BM25 + a local embedder, RRF-fused) and prints the top matches
# as context. `--lexical` builds a BM25-only index (no model, no network), so this
# exercises the whole CLI plumbing deterministically: walk -> index -> save ->
# load -> query -> format. Fail-open: malformed input and a missing index print
# nothing and never error. Pure stdin -> stdout, always-on backbone.
source /e2e/lib.sh

PROJ="$HOME/retrieve-proj"
mkdir -p "$PROJ/.atl/wiki"
cat > "$PROJ/.atl/wiki/merge-verify.md" <<'MD'
# Verify durable state not worker exit-code
A deterministic supervisor confirms a git merge by reading the durable branch state, never trusting an LLM worker exit code.
MD
cat > "$PROJ/.atl/wiki/pr-merge.md" <<'MD'
# PR merge discipline
Never merge pull requests from Claude; surface the URL and stop.
MD
cd "$PROJ"

# build a lexical-only index (deterministic, offline)
out="$(atl retrieve index --lexical 2>&1)"
echo "$out" | grep -q "indexed 2 pages" && ok "index built 2 pages" || bad "index build -- [$out]"

# the hook surfaces the relevant page with the context header
out="$(echo '{"prompt":"how does the supervisor confirm a merge landed on the branch","cwd":"'"$PROJ"'"}' | atl retrieve 2>&1)"
echo "$out" | grep -q "Verify durable state" && ok "relevant page surfaced" || bad "relevant page missing -- [$out]"
echo "$out" | grep -q "atl#140" && ok "context header present" || bad "no header -- [$out]"

# fail-open: malformed input prints nothing and never errors
out="$(echo 'not json' | atl retrieve 2>&1)"; rc=$?
{ [ -z "$out" ] && [ "$rc" -eq 0 ]; } && ok "malformed input is a safe no-op" || bad "malformed not safe -- rc=$rc [$out]"

# fail-open: a cwd with no index prints nothing
mkdir -p "$HOME/empty"
out="$(echo '{"prompt":"anything","cwd":"'"$HOME/empty"'"}' | atl retrieve 2>&1)"
[ -z "$out" ] && ok "no-index cwd is a safe no-op" || bad "no-index not safe -- [$out]"

finish
