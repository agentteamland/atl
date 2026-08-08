#!/usr/bin/env bash
# needs: none
# touches: cli/internal/retrieve, cli/cmd/atl/commands/retrieve.go, cli/cmd/atl/commands/retrieve_translate.go
#
# retrieve — `atl retrieve` (the UserPromptSubmit hook) ranks the project's
# knowledge pages (BM25 + a local embedder, RRF-fused) and prints the top matches
# as context. `--lexical` builds a BM25-only index (no model, no network), so this
# exercises the whole CLI plumbing deterministically: walk -> index -> save ->
# load -> query -> format. Fail-open: malformed input and a missing index print
# nothing and never error. Pure stdin -> stdout, always-on backbone.
source /e2e/lib.sh

PROJ="$HOME/retrieve-proj"
mkdir -p "$PROJ/.atl/wiki" "$PROJ/docs" "$PROJ/.delivery"
# a .delivery marker makes this a delivery project, so docs/ joins the corpus
echo '{"backend":"github"}' > "$PROJ/.delivery/config.json"
cat > "$PROJ/.atl/wiki/merge-verify.md" <<'MD'
# Verify durable state not worker exit-code
A deterministic supervisor confirms a git merge by reading the durable branch state, never trusting an LLM worker exit code.
MD
cat > "$PROJ/.atl/wiki/pr-merge.md" <<'MD'
# PR merge discipline
Never merge pull requests from Claude; surface the URL and stop.
MD
# a delivery-style in-repo docs/ page is part of the corpus too
cat > "$PROJ/docs/architecture.md" <<'MD'
# Architecture store
The GitHub backend keeps its durable knowledge in the in-repo docs tree.
MD
cd "$PROJ"

# build a lexical-only index (deterministic, offline) — covers wiki + docs/
out="$(atl retrieve index --lexical 2>&1)"
echo "$out" | grep -q "indexed 3 pages" && ok "index built 3 pages (wiki + docs)" || bad "index build -- [$out]"

# a second build is idempotent (incremental reuse path stays correct)
out="$(atl retrieve index --lexical 2>&1)"
echo "$out" | grep -q "indexed 3 pages" && ok "re-index is idempotent" || bad "re-index -- [$out]"

# the docs/ page is retrievable
out="$(echo '{"prompt":"where does the github backend keep its durable knowledge","cwd":"'"$PROJ"'"}' | atl retrieve 2>&1)"
echo "$out" | grep -q "docs/architecture.md" && ok "docs/ page surfaced" || bad "docs page missing -- [$out]"

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

# --- the translator credential ------------------------------------------------
# session-start's translation notice has no coverage at all today: the notice
# itself is unit-tested, its CALL SITE is not. And the file source exists exactly
# because the environment cannot always be reached, so an env-only test would
# assert the arm that was never the problem.
#
# `needs: none` is right for this: the image HAS the claude binary but no token,
# which is precisely the arm required.
unset CLAUDE_CODE_OAUTH_TOKEN ANTHROPIC_API_KEY

out="$(atl session-start 2>&1)"
echo "$out" | grep -q "claude-token" && ok "no credential: session-start names the file to create" || bad "notice missing or does not name the file -- [$out]"

# the two places that do NOT work must be named, or a user who used one of them
# reads the same notice next session and concludes the notice is wrong
echo "$out" | grep -q "zshrc" && ok "notice warns about ~/.zshrc" || bad "notice does not warn about .zshrc -- [$out]"

mkdir -p "$HOME/.atl"
printf 'sk-ant-oat01-e2e-not-a-real-token\n' > "$HOME/.atl/claude-token"
out="$(atl session-start 2>&1)"
echo "$out" | grep -q "claude-token" && bad "notice still shown after the file was created -- [$out]" || ok "credential file silences the notice"

# the doctor tightens a hand-created file: atl never writes it, so it arrives
# with the user's umask (0644 on a default macOS shell) and nothing else notices
chmod 0644 "$HOME/.atl/claude-token"
out="$(atl doctor 2>&1)"
mode="$(stat -c '%a' "$HOME/.atl/claude-token" 2>/dev/null || stat -f '%Lp' "$HOME/.atl/claude-token")"
[ "$mode" = "600" ] && ok "doctor tightened the credential file to 0600" || bad "credential file left at $mode -- [$out]"

# and having healed once it goes quiet, or it becomes the always-fires channel
out="$(atl doctor 2>&1)"
echo "$out" | grep -q "tightened" && bad "doctor still reports a heal it already applied -- [$out]" || ok "doctor is silent once the mode is right"

rm -f "$HOME/.atl/claude-token"

finish
