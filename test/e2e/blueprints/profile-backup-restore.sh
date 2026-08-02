#!/usr/bin/env bash
# needs: gh+token
# touches: teams/profile-team, cli/internal/teampkg
#
# profile-backup-restore — the two skills that touch the user's most sensitive data, and the
# only two whose failure mode is IRREVERSIBLE. /profile-backup copies ~/.atl/profiles into
# whatever repo you are standing in and `git add -f`s it: its visibility guard is the only
# thing between a mis-invocation and permanently publishing perception-flagged records about
# other people, state.emotional, and a consent-gated Tier-4 state.financial (git history keeps
# it after a delete). /profile-restore writes INTO the global store: its newer-guard is the
# only thing that stops it trading memory accumulated since a snapshot for older data.
#
# Both guards ship as EXECUTABLE shell inside a SKILL.md, hand-verified, previously with no
# regression test at all — so the next edit to either file could silently remove them. This
# blueprint is that regression test.
#
# SHAPE — deterministic-first, two LLM turns.
#   The guards are deterministic shell, so most of this blueprint extracts the skill's own
#   fenced bash block FROM THE INSTALLED SKILL.md and runs it directly, under each condition.
#   That is a genuine regression test of the shipped artifact (delete the guard and the
#   extracted script stops refusing) at zero LLM cost. What extraction does NOT prove is that
#   a real session runs the body rather than improvising, so the two paths where improvising
#   is unrecoverable get one real `claude -p` turn each: the public-repo refusal, and a bare
#   /profile-restore that must never reach --apply on its own.
#
# WHY NO REPO IS CREATED OR DELETED. /profile-backup never pushes — every write it makes is
# container-local. The only thing this blueprint needs from GitHub is the isPrivate bit, so it
# clones two PERMANENT fixtures read-only and lets the container carry the writes to the grave.
# Nothing to clean up, nothing to leak, nothing to reset. (Creating a throwaway private repo
# per run was considered and rejected: `gh repo delete` needs the `delete_repo` scope, which
# the harness token does not carry — a failed delete would leak a PRIVATE repo per run, the
# one outcome that is not acceptable here.)
#
# FIXTURES (both permanent, both asserted as preconditions before anything is trusted):
#   * public  — agentteamland/atl-e2e-team      (the publish-propose upstream; isPrivate=false)
#   * private — agentteamland/atl-e2e-delivery  (the delivery fixture; isPrivate=true)
#   Override either with ATL_E2E_PUBLIC_REPO / ATL_E2E_PRIVATE_REPO. The profile store under
#   test is SYNTHETIC, built in-container under a throwaway HOME; the real ~/.atl/profiles is
#   never touched.
#
# ASSERTION TIERS — CORE (ok/bad) is everything deterministic: the guards, the marker each
# path prints, the snapshot's content fidelity, the commit's scope, the overlay's preserve
# property. NOTE (ok/note) is only genuinely LLM-variable wording. Every ABSENCE assertion
# ("nothing was written") is paired with a positive one proving the run happened AND refused
# for the stated reason — an absence assertion alone passes when the thing under test never
# ran at all, which is the vacuous-green failure this harness has paid to learn about.

source /e2e/lib.sh
# note() is NOT in lib.sh (only ok=PASS / bad=FAIL) — the observe-only tier for LLM-variable
# fidelity, mirroring profile-loop.sh / github-request-loop.sh.
note() { echo "  note - $1"; }

PUB_REPO="${ATL_E2E_PUBLIC_REPO:-agentteamland/atl-e2e-team}"
PRIV_REPO="${ATL_E2E_PRIVATE_REPO:-agentteamland/atl-e2e-delivery}"
PUB="$HOME/pub"          # clone of the PUBLIC fixture   — the refusal case
PRIV="$HOME/priv"        # clone of the PRIVATE fixture  — the happy path
PLAIN="$HOME/plain"      # not a git repo
NOREM="$HOME/norem"      # git repo, no remote
BOGUS="$HOME/bogus"      # git repo, non-GitHub remote
CLONE2="$HOME/clone2"    # a fresh clone of PRIV — the "another machine" mtime case
STORE="$HOME/.atl/profiles"
BS="$HOME/extracted-backup.sh"
RS="$HOME/extracted-restore.sh"

# first_bash_block prints the FIRST ```bash fenced block of a markdown file. Both skills put
# their runnable body in that block (profile-restore's second block is only the two-line
# "re-run with --apply" illustration).
first_bash_block() {
  awk '/^```bash$/{n++; if(n==1){inb=1; next}} inb && /^```$/{exit} inb{print}' "$1"
}
# marker asserts that a captured stdout contains LINE as a whole line. Whole-line, not
# substring: on the happy path git's own commit output ("[main abc] chore(profile): …",
# "3 files changed") precedes the outcome marker, so an equality check on the whole output
# would fail and a substring check for "committed" would also match git's chatter.
marker() { printf '%s\n' "$1" | grep -qxF "$2"; }

# tool_inputs prints every tool-call INPUT a session made, one JSON object per line.
# Why this and not a grep over the whole transcript: invoking a skill injects its SKILL.md
# body into the session, so the transcript CONTAINS the guard's own source — every marker
# string ('public-repo', 'isPrivate', 'DRY RUN', 'KEEP (global-only') appears verbatim in a
# turn that merely loaded the skill and then did nothing at all. Grepping the file therefore
# passes on the exact null outcome the LLM assertions exist to exclude. A tool_use INPUT is
# different in kind: it is what the session chose to EXECUTE, and the injected skill body
# never lands there — so a match here is proof of a run, not proof of a paste. Both realistic
# shapes are covered: pasting the body into Bash (.input.command) and writing it to a file
# first (.input.content).
# No transcript at all -> return 1, so the caller's grep can never see an empty stream as a
# match. A jq hiccup on one malformed record, though, must not turn a real match into a red
# under `set -o pipefail` — the grep is the judge, so swallow jq's status and let it decide.
tool_inputs() {
  [ -n "${1:-}" ] || return 1
  jq -r 'select(.type=="assistant") | (.message.content? // [])[]? | select(.type=="tool_use") | .input | tostring' "$1" 2>/dev/null || true
}

# block_push makes any `git push` from a clone fail. Neither skill pushes — every write
# they make is container-local, which is what makes this blueprint safe to run against
# PERMANENT fixture repos. But the scenario under test is a BROKEN guard, and the skill's
# own report for `committed` tells the agent to mention that the user can push the snapshot
# to another machine; an LLM turn that took that as an instruction would publish profile
# content into a real repo, and nothing in the harness resets the public fixture. A
# pre-push hook is the containment: it costs two lines, it cannot be reached by accident,
# and (unlike rewriting the remote URL) it leaves `gh repo view`'s repo resolution — the
# one thing this blueprint actually needs from the remote — completely untouched.
NOPUSH="$HOME/.nopush"
mkdir -p "$NOPUSH"
printf '#!/bin/sh\necho "e2e: push blocked (profile-backup-restore never pushes)" >&2\nexit 1\n' > "$NOPUSH/pre-push"
chmod +x "$NOPUSH/pre-push"
block_push() { git -C "$1" config core.hooksPath "$NOPUSH"; }

# ---- 0. auth + FIXTURE PRECONDITIONS ------------------------------------------------
# These run first and abort the blueprint on failure. Without them a missing/renamed fixture
# (or a token that cannot read the private repo) makes `gh repo view` fail, which the skill
# correctly reports as visibility-unknown — and every private-path assertion below would go
# red as though the guard were broken. Fixture trouble must be distinguishable from a defect.
gh auth setup-git >/dev/null 2>&1 || true
LOGIN=$(gh_login)
[ -n "$LOGIN" ] || { bad "gh not authenticated"; finish; exit 1; }

PUB_VIS=$(gh repo view "$PUB_REPO"  --json isPrivate -q .isPrivate 2>/dev/null)
PRV_VIS=$(gh repo view "$PRIV_REPO" --json isPrivate -q .isPrivate 2>/dev/null)
if [ "$PUB_VIS" = "false" ]; then ok "fixture precondition: $PUB_REPO is PUBLIC"
else bad "fixture precondition FAILED: $PUB_REPO isPrivate=[$PUB_VIS], expected false"; finish; exit 1; fi
if [ "$PRV_VIS" = "true" ]; then ok "fixture precondition: $PRIV_REPO is PRIVATE"
else bad "fixture precondition FAILED: $PRIV_REPO isPrivate=[$PRV_VIS], expected true (token read rights?)"; finish; exit 1; fi

# ---- 1. install profile-team + extract the two skill bodies ---------------------------
fresh
write_test_index_profile
headless_claude_setup
cd "$PROJ" || exit 2
atl install --global agentteamland/profile-team >/dev/null 2>&1 || bad "install errored"

BSKILL="$HOME/.claude/skills/profile-backup/SKILL.md"
RSKILL="$HOME/.claude/skills/profile-restore/SKILL.md"
[ -f "$BSKILL" ] && ok "profile-backup skill reflected"  || bad "profile-backup SKILL.md missing"
[ -f "$RSKILL" ] && ok "profile-restore skill reflected" || bad "profile-restore SKILL.md missing"

first_bash_block "$BSKILL" > "$BS" 2>/dev/null
first_bash_block "$RSKILL" > "$RS" 2>/dev/null
# An empty/short extraction would write nothing and refuse nothing — i.e. it would satisfy
# every "nothing was written" assertion below for the wrong reason. Fail loudly instead.
bl=$(wc -l < "$BS" 2>/dev/null | tr -d ' ' || echo 0); rl=$(wc -l < "$RS" 2>/dev/null | tr -d ' ' || echo 0)
if [ "${bl:-0}" -ge 20 ] && bash -n "$BS" 2>/dev/null; then ok "extracted /profile-backup body ($bl lines, parses)"
else bad "could not extract a runnable /profile-backup body ($bl lines) — fence layout changed?"; finish; exit 1; fi
if [ "${rl:-0}" -ge 20 ] && bash -n "$RS" 2>/dev/null; then ok "extracted /profile-restore body ($rl lines, parses)"
else bad "could not extract a runnable /profile-restore body ($rl lines) — fence layout changed?"; finish; exit 1; fi

# Both clones are PRECONDITIONS, not assertions — a failed clone (renamed fixture, a token
# without read rights on the private one) would leave every path below running in a
# directory that is not a git repo, turning ~15 assertions red as though the guards were
# broken. Fixture trouble must be distinguishable from a defect, so abort instead.
git clone -q "https://github.com/$PUB_REPO.git"  "$PUB"  || { bad "clone of $PUB_REPO failed";  finish; exit 1; }
git clone -q "https://github.com/$PRIV_REPO.git" "$PRIV" || { bad "clone of $PRIV_REPO failed (token read rights?)"; finish; exit 1; }
block_push "$PUB"
block_push "$PRIV"

# BASELINE for every "no profile-backup/ was created" assertion below. Those are absence
# checks, and an absence check is only evidence of a delta if the thing was absent to begin
# with — a fixture that already carried a profile-backup/ would turn each of them into a
# permanent red that reads exactly like a broken guard. Assert the starting state once, here.
{ [ ! -e "$PUB/profile-backup" ] && [ ! -e "$PRIV/profile-backup" ]; } \
  && ok "baseline: neither fixture clone carries a profile-backup/ yet" \
  || { bad "fixture baseline is dirty — a clone already has profile-backup/"; finish; exit 1; }

# ---- 2. the visibility guard — every path that must REFUSE ---------------------------
# Ordering note (the SKILL wins over the issue's suggested order): the visibility guard is
# step 2 of the script and the empty-store check is step 3, so `nothing-to-back-up` is only
# reachable INSIDE a demonstrably private repo. Verified against the real script.

mkdir -p "$PLAIN"
out=$( cd "$PLAIN" && bash "$BS" 2>&1 ); rc=$?
{ marker "$out" 'not-a-git-repo' && [ "$rc" -ne 0 ]; } \
  && ok "no git repo → 'not-a-git-repo', exit $rc" || bad "no git repo: rc=$rc out=[$out]"

git init -q "$NOREM"
out=$( cd "$NOREM" && bash "$BS" 2>&1 ); rc=$?
{ marker "$out" 'visibility-unknown' && [ "$rc" -ne 0 ] && [ ! -e "$NOREM/profile-backup" ]; } \
  && ok "no GitHub remote → 'visibility-unknown', exit $rc, nothing copied" \
  || bad "no-remote guard: rc=$rc out=[$out]"

git init -q "$BOGUS"
git -C "$BOGUS" remote add origin https://gitlab.com/nobody/nothing.git
out=$( cd "$BOGUS" && bash "$BS" 2>&1 ); rc=$?
{ marker "$out" 'visibility-unknown' && [ "$rc" -ne 0 ] && [ ! -e "$BOGUS/profile-backup" ]; } \
  && ok "non-GitHub remote → 'visibility-unknown', exit $rc, nothing copied" \
  || bad "non-github-remote guard: rc=$rc out=[$out]"

# gh removed from PATH inside the PRIVATE repo: the ONE case where the answer would have been
# "yes, private" had gh been reachable. It must still fail closed — an unreadable answer is
# treated as unsafe, never guessed from the path or the remote's name. (gh lives in
# /usr/local/bin; git, date and bash are all under /usr/bin — which is why this PATH works.)
out=$( cd "$PRIV" && env PATH=/usr/bin:/bin /bin/bash "$BS" 2>&1 ); rc=$?
{ marker "$out" 'visibility-unknown' && [ "$rc" -ne 0 ] && [ ! -e "$PRIV/profile-backup" ]; } \
  && ok "gh unavailable in a PRIVATE repo → still 'visibility-unknown' (fails closed), exit $rc" \
  || bad "gh-unavailable guard did not fail closed: rc=$rc out=[$out]"

# THE one that must never break. A public repo, and the store is not even seeded yet — but the
# guard is what has to stop this, not the empty store, so assert the marker (proof the guard
# branch ran) alongside the absence.
PUB_HEAD=$(git -C "$PUB" rev-parse HEAD)
out=$( cd "$PUB" && bash "$BS" 2>&1 ); rc=$?
marker "$out" 'public-repo' && ok "PUBLIC repo → the guard printed 'public-repo'" \
                            || bad "PUBLIC repo did not print 'public-repo': out=[$out]"
[ "$rc" -ne 0 ] && ok "PUBLIC repo → non-zero exit ($rc)" || bad "PUBLIC repo exited 0 — the guard did not stop"
[ ! -e "$PUB/profile-backup" ] && ok "PUBLIC repo → no profile-backup/ was created (guard runs before the cp)" \
                               || bad "PUBLIC repo → profile-backup/ EXISTS — profile content was written into a public repo"
[ "$(git -C "$PUB" rev-parse HEAD)" = "$PUB_HEAD" ] && ok "PUBLIC repo → no commit was made" || bad "PUBLIC repo → HEAD moved"
# Scoped to what the skill could have written, matching the LLM-turn assertions below. The
# broad clean-tree form passes here today only because the extracted script runs with no
# session behind it — the same check went red on the LLM path against `?? .atl/`, atl's own
# scaffolding. Scoping is also the more correct assertion on the merits: the guard's contract
# is "the backup wrote nothing", not "nothing whatsoever happened in this repo".
[ -z "$(git -C "$PUB" diff --cached --name-only)" ] \
  && ok "PUBLIC repo → nothing staged" \
  || bad "PUBLIC repo → files staged: $(git -C "$PUB" diff --cached --name-only | head -3)"
[ -z "$(git -C "$PUB" status --porcelain -- profile-backup)" ] \
  && ok "PUBLIC repo → no profile-backup/ path touched" \
  || bad "PUBLIC repo → profile-backup/ touched: $(git -C "$PUB" status --porcelain -- profile-backup | head -3)"

# empty store, inside the PRIVATE repo — the only place this branch is reachable.
rm -rf "$STORE"   # explicit precondition: install does not create the store, but say so
out=$( cd "$PRIV" && bash "$BS" 2>&1 ); rc=$?
{ marker "$out" 'nothing-to-back-up' && [ "$rc" -eq 0 ] && [ ! -e "$PRIV/profile-backup" ]; } \
  && ok "empty store → 'nothing-to-back-up', exit 0, no empty snapshot created" \
  || bad "empty-store path: rc=$rc out=[$out] dir=$([ -e "$PRIV/profile-backup" ] && echo present || echo absent)"

# ---- 3. the happy path — snapshot fidelity, gitlink, commit scope ---------------------
# The store is ITSELF a git repo (atl versions it locally). That nested .git is the documented
# gitlink trap: `git add` on a directory containing one records a mode-160000 gitlink instead
# of the files — exit 0, a non-empty staged diff, "backup taken" reported, and a snapshot that
# holds no profile content at all. Seeding a .git here is what makes that assertable.
mkdir -p "$STORE/_interfaces" "$STORE/people/keeper" "$STORE/people/testsubject" "$STORE/people/ghost"
printf 'type-id: person\nschema-version: 1.0.0\n' > "$STORE/_interfaces/person.md"
printf -- '---\nmeta: {type-id: person}\nidentity: {name: Keeper}\n---\nE2E-SENTINEL-KEEPER\n'   > "$STORE/people/keeper/profile.md"
printf -- '---\nmeta: {type-id: person}\nidentity: {name: Subject}\n---\nE2E-SENTINEL-SUBJECT\n' > "$STORE/people/testsubject/profile.md"
printf -- '---\nmeta: {type-id: person}\nidentity: {name: Ghost}\n---\nE2E-SENTINEL-GHOST\n'     > "$STORE/people/ghost/profile.md"
# The identity is passed explicitly (as lib.sh's reset helpers do) rather than relying on the
# image's global git config, and the resulting COMMIT is asserted — not just the .git dir.
# That distinction is load-bearing for the gitlink assertion further down: a nested repo with
# no commit checked out cannot be recorded as a gitlink at all (`git add` aborts with "does
# not have a commit checked out"), so a silently-failed commit here would DISARM the trap and
# leave the "no gitlink" assertion passing for the wrong reason — the vacuous green this
# blueprint exists to avoid. Verified both ways against the real script.
( cd "$STORE" && git init -q && git add -A \
  && git -c user.email=e2e@atl.local -c user.name=atl-e2e commit -qm "store baseline" ) >/dev/null 2>&1
git -C "$STORE" rev-parse HEAD >/dev/null 2>&1 \
  && ok "synthetic store seeded, and is itself a git repo WITH a commit (gitlink trap armed)" \
  || bad "the store's baseline commit failed — the gitlink trap is DISARMED, later gitlink assertions would be vacuous"

# A gitignore on the destination (the skill's `add -f` must beat it) and an UNRELATED staged
# file (which must NOT be swept into a commit labelled as a profile snapshot).
echo 'profile-backup/' >> "$PRIV/.gitignore"
echo 'unrelated'        > "$PRIV/unrelated.txt"
git -C "$PRIV" add unrelated.txt
PRIV_HEAD=$(git -C "$PRIV" rev-parse HEAD)

out=$( cd "$PRIV" && bash "$BS" 2>&1 ); rc=$?
marker "$out" 'committed' && ok "PRIVATE repo → 'committed'" || bad "PRIVATE repo did not print 'committed': rc=$rc out=[$out]"
[ "$(git -C "$PRIV" rev-parse HEAD)" != "$PRIV_HEAD" ] && ok "PRIVATE repo → a new commit exists" || bad "no commit was made"

# byte-fidelity: the snapshot equals the store, .git excluded (the skill removes it on purpose).
dq=$(diff -rq -x '.git' "$STORE" "$PRIV/profile-backup" 2>&1)
[ -z "$dq" ] && ok "snapshot is byte-identical to the store (diff -r, .git excluded)" || bad "snapshot differs from the store: $(printf '%s' "$dq" | head -3)"
[ ! -e "$PRIV/profile-backup/.git" ] && ok "the store's .git was NOT copied into the snapshot" || bad "the store's .git leaked into the snapshot"

# gitlink: the negative (no mode-160000 entry) AND the positive (real content is in the commit).
gl=$(git -C "$PRIV" ls-tree -r HEAD -- profile-backup | awk '$1=="160000"' | wc -l | tr -d ' ')
[ "${gl:-1}" -eq 0 ] && ok "no gitlink (mode 160000) recorded under profile-backup/" || bad "GITLINK recorded — the commit holds no profile content ($gl entries)"
git -C "$PRIV" show HEAD:profile-backup/people/keeper/profile.md 2>/dev/null | grep -qF 'E2E-SENTINEL-KEEPER' \
  && ok "the COMMIT carries real profile content (sentinel readable via git show)" \
  || bad "the commit does not carry the profile content"

# commit scope: only profile-backup/ paths in the commit, and the unrelated file still staged.
offpath=$(git -C "$PRIV" show --pretty=format: --name-only HEAD | sed '/^$/d' | grep -cv '^profile-backup/')
[ "${offpath:-1}" -eq 0 ] && ok "the commit is scoped to profile-backup/ only" || bad "the commit swept in $offpath path(s) outside profile-backup/"
git -C "$PRIV" diff --cached --name-only | grep -qx 'unrelated.txt' \
  && ok "the unrelated pre-staged file was left staged, not committed" \
  || bad "the unrelated pre-staged file was swept into the profile snapshot commit"

# idempotency: a second run with an unchanged store commits nothing.
H2=$(git -C "$PRIV" rev-parse HEAD)
out=$( cd "$PRIV" && bash "$BS" 2>&1 ); rc=$?
{ marker "$out" 'already-current' && [ "$(git -C "$PRIV" rev-parse HEAD)" = "$H2" ]; } \
  && ok "re-run with an unchanged store → 'already-current', no new commit" \
  || bad "re-run was not a no-op: rc=$rc out=[$out]"

# mirror, not append: a profile deleted in global disappears from the snapshot.
rm -rf "$STORE/people/ghost"
out=$( cd "$PRIV" && bash "$BS" 2>&1 ); rc=$?
marker "$out" 'committed' && ok "store deletion → a new snapshot commit" || bad "deletion did not produce a commit: out=[$out]"
{ [ ! -e "$PRIV/profile-backup/people/ghost" ] && [ "$(git -C "$PRIV" ls-tree -r HEAD --name-only -- profile-backup | grep -c ghost)" = 0 ]; } \
  && ok "the snapshot is a MIRROR — the deleted profile is gone from the tree and the commit" \
  || bad "the deleted profile survives in the snapshot — snapshot is appending, not mirroring"

# ---- 4. LLM turn 1 — a real session invokes /profile-backup in a PUBLIC repo -----------
# The one path where improvising is unrecoverable. `claude_turn` runs in $PROJ, so point it at
# the public clone for this turn only. Note the blast radius of testing the failure case is
# container-local: /profile-backup never pushes, so even a fully broken guard could not publish
# anything from here — it would only commit locally in a clone that dies with the container.
PROJ_SAVE="$PROJ"
PROJ="$PUB"
PUB_HEAD2=$(git -C "$PUB" rev-parse HEAD)
claude_turn "/profile-backup — run it here and report the outcome. This is a headless run: do not wait for interactive input." \
  || note "the /profile-backup turn returned non-zero (usually a timeout — the envelope assertion below is the CORE check)"
TURN_RESULT=$(jq -r '.result // empty' "$HOME/.turn.json" 2>/dev/null)
T=$(find "$HOME/.claude/projects" -name '*.jsonl' 2>/dev/null | xargs -r ls -t 2>/dev/null | head -1)

# POSITIVE (the turn ran) — a completed envelope with a non-empty result.
[ -n "$TURN_RESULT" ] && ok "LLM turn 1 completed (non-empty result envelope)" || bad "LLM turn 1 produced no result envelope — it never ran to completion"
# POSITIVE (it refused for THE STATED REASON) — the session actually EXECUTED the visibility
# probe. Scoped to tool-call inputs (see tool_inputs above): a plain transcript grep would
# match the skill body injected on invocation, i.e. it would pass on a turn that ran nothing,
# which is precisely the null outcome every absence assertion below has to be protected from.
tool_inputs "$T" | grep -qF 'gh repo view' \
  && ok "LLM turn 1 EXECUTED the visibility probe (gh repo view in a tool-call input)" \
  || bad "no evidence the session ran the visibility check — it may have done nothing at all"
# ABSENCE (nothing was written) — only meaningful paired with the two positives above.
[ ! -e "$PUB/profile-backup" ] && ok "LLM turn 1 → no profile-backup/ in the PUBLIC repo" || bad "LLM turn 1 WROTE profile content into a public repo"
[ "$(git -C "$PUB" rev-parse HEAD)" = "$PUB_HEAD2" ] && ok "LLM turn 1 → no commit in the PUBLIC repo" || bad "LLM turn 1 committed into a public repo"
# Scoped to what the SKILL could have written, deliberately NOT a blanket clean-tree check.
# A blanket check fails on `?? .atl/` — atl's own session-start scaffolding of backlog.md /
# tasks.md, created by the claude -p turn itself and entirely unrelated to /profile-backup. It
# went red on the first real run for exactly that reason while the guard had worked perfectly:
# the assertion was asking "did anything change?" when the question is "did the BACKUP write
# anything?". Do not restore the broad form.
#
# `git add -f profile-backup` is the skill's only staging action, so an empty index is the
# precise end-state: it stays empty whether the guard tripped before the copy (the contract) or
# the copy ran and the commit did not (the failure this catches).
[ -z "$(git -C "$PUB" diff --cached --name-only)" ] \
  && ok "LLM turn 1 → nothing staged in the PUBLIC repo" \
  || bad "LLM turn 1 staged files in a public repo: $(git -C "$PUB" diff --cached --name-only | head -3)"
[ -z "$(git -C "$PUB" status --porcelain -- profile-backup)" ] \
  && ok "LLM turn 1 → no profile-backup/ path touched in the PUBLIC repo" \
  || bad "LLM turn 1 touched profile-backup/ in a public repo: $(git -C "$PUB" status --porcelain -- profile-backup | head -3)"
grep -qiE 'public|private' <<<"$TURN_RESULT" && note "the turn's report mentions repo visibility" || note "visibility not mentioned in the turn's prose (LLM-variable wording)"
PROJ="$PROJ_SAVE"

# ---- 5. the restore newer-guard (deterministic) ----------------------------------------
# Diverge global from the snapshot in all three directions the preview classifies:
#   ADD       — testsubject removed from global (the snapshot must offer it back)
#   OVERWRITE — keeper modified in global AFTER the snapshot commit (must be flagged !!)
#   KEEP      — newperson created in global only (must be preserved by an --apply)
rm -rf "$STORE/people/testsubject"
sleep 2                                   # the guard compares whole seconds; 2s is enough skew
echo 'ACCUMULATED-SINCE-SNAPSHOT' >> "$STORE/people/keeper/profile.md"
mkdir -p "$STORE/people/newperson"
printf -- '---\nmeta: {type-id: person}\n---\nE2E-SENTINEL-NEWPERSON\n' > "$STORE/people/newperson/profile.md"

out=$( cd "$PRIV" && bash "$RS" 2>&1 ); rc=$?
grep -qF 'DRY RUN' <<<"$out" && ok "preview ran to completion (DRY RUN banner)" || bad "preview did not complete: rc=$rc out=[$(head -5 <<<"$out")]"
grep -F '!!' <<<"$out" | grep -qF 'keeper' \
  && ok "NEWER-GUARD: the globally-modified file is flagged !! (would overwrite newer memory)" \
  || bad "NEWER-GUARD DID NOT FIRE on a global file modified after the snapshot commit"
grep -A2 'ADD  (in snapshot' <<<"$out" | grep -qF 'testsubject' && ok "preview lists the snapshot-only profile under ADD" || bad "ADD section did not list testsubject"
grep -A5 'KEEP (global-only' <<<"$out" | grep -qF 'newperson'   && ok "preview lists the global-only profile under KEEP" || bad "KEEP section did not list newperson"
# ABSENCE — preview writes nothing. Paired with the DRY RUN positive above.
[ ! -e "$STORE/people/testsubject" ] && ok "preview wrote nothing to global (ADD not applied)" || bad "preview WROTE to global"
sc=$(find "$HOME/.atl" -maxdepth 1 -name 'profiles.pre-restore-*' 2>/dev/null | wc -l | tr -d ' ')
[ "${sc:-1}" -eq 0 ] && ok "preview created no safety copy (it wrote nothing at all)" || bad "preview created a safety copy — it took the apply path"

# The "another machine" case, which is the whole reason the guard keys off the COMMIT time
# rather than file mtimes: git does not preserve mtimes across a clone, so in a fresh checkout
# every snapshot file looks freshly modified. An mtime-based guard would conclude the snapshot
# is the newer side and stay silent — exactly when the data loss is real. Assert !! survives.
git clone -q "$PRIV" "$CLONE2" || bad "local clone for the mtime case failed"
out=$( cd "$CLONE2" && bash "$RS" 2>&1 ); rc=$?
grep -qF 'DRY RUN' <<<"$out" && ok "fresh-clone preview ran to completion" || bad "fresh-clone preview did not complete: rc=$rc"
grep -F '!!' <<<"$out" | grep -qF 'keeper' \
  && ok "NEWER-GUARD survives a fresh clone (commit-time based, not mtime based)" \
  || bad "NEWER-GUARD went silent in a fresh clone — it is keying off mtimes, and would lose newer memory on another machine"

# ---- 6. LLM turn 2 — a bare /profile-restore must never reach --apply -------------------
# The skill's rule is "never pass --apply on your own"; the user has not confirmed anything
# here. Run it in the private clone (the one holding a snapshot), with global still diverged.
PROJ="$PRIV"
KEEPER_BEFORE=$(md5sum "$STORE/people/keeper/profile.md" 2>/dev/null | awk '{print $1}')
claude_turn "/profile-restore — run it here and report what you find. This is a headless run: do not wait for interactive input." \
  || note "the /profile-restore turn returned non-zero (assertions below are what matter)"
TURN2_RESULT=$(jq -r '.result // empty' "$HOME/.turn.json" 2>/dev/null)
T2=$(find "$HOME/.claude/projects" -name '*.jsonl' 2>/dev/null | xargs -r ls -t 2>/dev/null | head -1)

# POSITIVE (the turn ran) and POSITIVE (the restore preview actually executed). Same reasoning
# as turn 1: the strings below live in profile-restore's SKILL.md, so the transcript holds them
# the moment the skill is invoked — only their presence in a tool-call INPUT proves the body was
# put into a shell. That is what makes the three absence assertions that follow non-vacuous.
[ -n "$TURN2_RESULT" ] && ok "LLM turn 2 completed (non-empty result envelope)" || bad "LLM turn 2 produced no result envelope"
tool_inputs "$T2" | grep -qE 'SNAP_EPOCH|DRY RUN' \
  && ok "LLM turn 2 EXECUTED the restore preview (its body appears in a tool-call input)" \
  || bad "no evidence the restore preview ran — the session may have done nothing at all"
# ABSENCE — no write to global without --apply.
sc2=$(find "$HOME/.atl" -maxdepth 1 -name 'profiles.pre-restore-*' 2>/dev/null | wc -l | tr -d ' ')
[ "${sc2:-1}" -eq 0 ] && ok "LLM turn 2 → no safety copy: --apply was never reached" || bad "LLM turn 2 took the --apply path without a confirmation"
[ ! -e "$STORE/people/testsubject" ] && ok "LLM turn 2 → global unchanged (the snapshot-only profile was not restored)" || bad "LLM turn 2 WROTE to the global store unasked"
[ "$(md5sum "$STORE/people/keeper/profile.md" 2>/dev/null | awk '{print $1}')" = "$KEEPER_BEFORE" ] \
  && ok "LLM turn 2 → the newer global file was not overwritten" \
  || bad "LLM turn 2 OVERWROTE a global file that was newer than the snapshot"
grep -qiE 'newer|overwrite|dry.?run|apply' <<<"$TURN2_RESULT" && note "the turn's report surfaces the newer-than-snapshot risk" || note "risk not surfaced in the turn's prose (LLM-variable wording)"
PROJ="$PROJ_SAVE"

# ---- 7. --apply — overlay semantics (adds and overwrites; never deletes) ----------------
out=$( cd "$PRIV" && bash "$RS" --apply 2>&1 ); rc=$?
grep -qF 'Safety copy' <<<"$out" && ok "--apply ran and took the safety copy first" || bad "--apply did not report a safety copy: rc=$rc out=[$(tail -3 <<<"$out")]"
sc3=$(find "$HOME/.atl" -maxdepth 1 -name 'profiles.pre-restore-*' 2>/dev/null | wc -l | tr -d ' ')
[ "${sc3:-0}" -eq 1 ] && ok "--apply left exactly one reversible pre-restore copy" || bad "expected 1 pre-restore safety copy, found ${sc3:-0}"
# POSITIVE that the write happened (so the preserve assertion below is not vacuous).
grep -qF 'E2E-SENTINEL-SUBJECT' "$STORE/people/testsubject/profile.md" 2>/dev/null \
  && ok "--apply restored the snapshot-only profile into global (ADD applied)" \
  || bad "--apply did not restore the snapshot-only profile"
# THE property: memory accumulated since the snapshot is preserved, never deleted.
grep -qF 'E2E-SENTINEL-NEWPERSON' "$STORE/people/newperson/profile.md" 2>/dev/null \
  && ok "--apply PRESERVED the global-only profile (overlay, never a mirror)" \
  || bad "--apply DELETED memory accumulated since the snapshot — the one unacceptable failure"
grep -qF 'ACCUMULATED-SINCE-SNAPSHOT' "$STORE/people/keeper/profile.md" 2>/dev/null \
  && bad "--apply did not overwrite the shared file it said it would" \
  || ok "--apply overwrote the shared file with the snapshot's version, as previewed"
[ -d "$STORE/.git" ] && ok "the store's own git history survived the restore" || bad "the restore destroyed the store's .git"

# ---- on failure, surface what the torn-down container would otherwise lose --------------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (profile-backup-restore failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- fixtures --- pub=$PUB_REPO($PUB_VIS) priv=$PRIV_REPO($PRV_VIS)"
  echo "--- extracted backup body (head) ---"; head -20 "$BS" 2>/dev/null
  echo "--- store tree ---"; find "$STORE" -not -path '*/.git/*' 2>/dev/null | head -40
  echo "--- priv git log ---"; git -C "$PRIV" log --oneline -5 2>/dev/null
  echo "--- priv status ---"; git -C "$PRIV" status --porcelain 2>/dev/null | head -10
  echo "--- pub status ---"; git -C "$PUB" status --porcelain 2>/dev/null | head -10
  # The two "EXECUTED" assertions read the transcript through tool_inputs(), whose jq path was
  # grounded on a real Claude Code transcript. If `claude -p` ever writes a different record
  # shape they would go red while the guards are perfectly fine — a false RED, not a false
  # green, but only if you can tell the two apart. These two lines are how: an empty record
  # census means the schema moved, a populated one with no matching command means the session
  # genuinely did not run the body.
  echo "--- transcript record census (T2=${T2:-none}) ---"
  jq -r '.type' "${T2:-/dev/null}" 2>/dev/null | sort | uniq -c | head
  echo "--- tool-call inputs seen (first 200 chars each) ---"
  tool_inputs "${T2:-}" | cut -c1-200 | head -12
  echo "--- turns.log (tail) ---"; tail -120 "$HOME/turns.log" 2>/dev/null
  echo "================================================"
fi

finish
