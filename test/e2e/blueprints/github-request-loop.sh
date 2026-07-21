#!/usr/bin/env bash
# needs: gh+token
#
# github-request-loop — the delivery-team's /request mid-project intake ceremony, end to end on
# the GitHub backend with a real `claude -p` ceremony. /request is the front door for a request
# that arises mid-project (from the PO or surfaced by the team): it captures the request as a
# *candidate* (excluded from the ready frontier until accepted), triages it (weight-proportional),
# runs a BA→TA→tech-lead feasibility deliberation, and presents a reasoned YES/NO/DEFER/NEEDS-INFO
# verdict at an honest PO gate that is an active dialectic (refute-to-keep), recording a
# `**[Request Decision]**` sentinel — then, on the PO's accept, drops the candidate flag so the
# item enters the frontier (whereupon /refine would decompose it — proven separately).
#
# Two turns model the interactive gate: turn 1 captures + triages + deliberates + records a verdict
# and STOPS at the PO gate; turn 2 is the PO's accept, which flips the candidate into the frontier.
#
# Candidate state (concept #13) is a Projects v2 `candidate` Status option in production — but
# Status options are not API-settable (board-setup, UI-only, like Iteration), so this headless run
# exercises the adapter's LABEL fallback: the `candidate` label + the `atl-request:<slug>:<init>`
# intake-provenance key (concept #14) carry the state. That is the SKILL.md's documented behaviour
# when the Status option is absent.
#
# Two assertion tiers:
#   * CORE (ok/bad) — install (incl. the request skill reflected), a candidate issue with the
#     `candidate` + `atl-request:<slug>:po` labels, a `**[Request Decision]**` sentinel comment
#     carrying a `## Recommendation` verdict, and — after the PO accept — the candidate flag DROPPED
#     (the item entered the frontier).
#   * NOTE (ok/note) — the triage tier label, the specific verdict value, and any /refine hand-off
#     (LLM-variable fidelity).
#
# SETUP: same as the other gh blueprints — agentteamland/atl-e2e-delivery + a fresh Project per run.
# needs: gh+token.

source /e2e/lib.sh
note() { echo "  note - $1"; }
ge()   { [ "${1:-0}" -ge 1 ] 2>/dev/null; }

command -v node >/dev/null 2>&1 && ok "node present" || bad "node missing in the image"

gh auth setup-git >/dev/null 2>&1 || true
LOGIN=$(gh_login)
[ -n "$LOGIN" ] || { bad "gh not authenticated"; finish; exit 1; }
OWNER="${ATL_E2E_DELIVERY_OWNER:-agentteamland}"
REPO="$OWNER/atl-e2e-delivery"

reset_delivery_repo "$OWNER" \
  && ok "reset delivery fixture repo to baseline (main/dev/release)" \
  || bad "could not reset $REPO (does it exist? token 'repo' rights on $OWNER?)"
PROJNUM=$(reset_delivery_project "$OWNER")
[ -n "$PROJNUM" ] && ok "created a fresh GitHub Project #$PROJNUM" || bad "could not create the Project (token 'project' scope?)"

fresh
write_test_index_delivery
headless_claude_setup

rmdir "$PROJ" 2>/dev/null || true
git clone -q "https://github.com/$REPO.git" "$PROJ" || bad "clone of $REPO failed"
cd "$PROJ" || exit 2

atl install agentteamland/delivery-team >/dev/null 2>&1 || bad "install errored"
[ -f "$PROJ/.claude/skills/request/SKILL.md" ]      && ok "request skill reflected"       || bad "request skill missing"
[ -f "$PROJ/.claude/backends/github/adapter.md" ]   && ok "github adapter reflected"       || bad "github adapter missing"
[ -f "$PROJ/.claude/knowledge/backend-interface.md" ] && ok "backend-interface reflected"  || bad "backend-interface missing"

# ---- .delivery/ config (GitHub shape) + methodology --------------------------------
mkdir -p "$PROJ/.delivery"
cat > "$PROJ/.delivery/config.json" <<EOF
{
  "owner": "$OWNER",
  "repo": "atl-e2e-delivery",
  "projectNumber": $PROJNUM,
  "branchPair": { "dev": "dev", "release": "release" },
  "backend": "github",
  "methodology": "scrum",
  "credential": { "ref": "GH_TOKEN" }
}
EOF
cat > "$PROJ/.delivery/methodology.json" <<'EOF'
{
  "id": "scrum",
  "displayName": "Scrum",
  "roles": [
    { "name": "intake",            "binding": "agent", "dispatch": "in-session" },
    { "name": "business-analyst",  "binding": "agent", "dispatch": "subagent" },
    { "name": "technical-analyst", "binding": "agent", "dispatch": "subagent" },
    { "name": "project-manager",   "binding": "agent", "dispatch": "subagent" },
    { "name": "tech-lead",         "binding": "agent", "dispatch": "subagent" },
    { "name": "tester",            "binding": "agent", "dispatch": "worker" },
    { "name": "developer",         "binding": "agent", "dispatch": "worker", "instances": "dynamic" },
    { "name": "product-owner",     "binding": "human" }
  ],
  "artifactHierarchy": ["Epic", "Feature", "Pbi", "Task"],
  "workItemTypeMap": { "Pbi": null, "Task": null, "Bug": null },
  "cadence": { "unit": "sprint", "planningCeremonies": ["sprint-plan", "sprint-start"], "reviewCeremony": "sprint-review" },
  "capacityModel": { "velocityWindowN": 3, "unit": "storyPoints", "coldStart": "po-seed", "seedVelocity": null, "availabilityFactorDefault": 1.0 },
  "branches": { "dev": "dev", "release": "release" }
}
EOF
ok "seeded .delivery/config.json (github) + methodology.json"

gturn() { ( cd "$PROJ" && claude -p "$1" --dangerously-skip-permissions --output-format json ) >>"$HOME/turns.log" 2>&1; }

# ---- 1. /request — capture the candidate + triage + deliberate + verdict; STOP at the PO gate ----
gturn "/request. You are ALSO acting as the human product owner for this headless run — supply the request from these facts, do not wait for interactive input. THE REQUEST (from the PO): 'Add a health-check endpoint that returns the app version, so ops can verify a deploy.' Run the ceremony: capture it as a CANDIDATE with a stable request-slug (kebab-case) and initiator 'po' — the fresh Project has NO 'candidate' Status option (Status options are board-setup / UI-only), so use the adapter's LABEL fallback: open the candidate issue with a 'candidate' label + an 'atl-request:<slug>:po' label + a 'triage:<tier>' label. Triage it (weight-proportional). Adopt business-analyst -> technical-analyst -> tech-lead sequentially IN THIS SHARED CONTEXT (not isolated workers) for feasibility, mount a genuine anti-thesis (refute-to-keep), and form a reasoned YES/NO/DEFER/NEEDS-INFO verdict. Record a comment on the candidate whose FIRST LINE is the exact sentinel '**[Request Decision]**' with H2s: ## Recommendation (the verdict), ## Deliberation (thesis / anti-thesis / surviving position), ## PO Decision (leave as 'pending — awaiting PO'), ## Dissent On Record. STOP at the PO gate — do NOT make the accept/reject decision yet; the PO decides in the next turn." || bad "request turn 1 errored"

# CORE: a candidate issue exists with the candidate + intake-provenance labels
CAND=$(gh issue list --repo "$REPO" --state all --label candidate --json number -q '.[0].number' 2>/dev/null)
[ -n "$CAND" ] && ok "/request captured a candidate issue (#$CAND) with the 'candidate' label" || bad "no candidate issue with a 'candidate' label"
[ -n "$CAND" ] || { echo "!! no candidate — aborting before the PO-gate turn"; finish; exit 1; }

reqlab=$(gh issue view "$CAND" --repo "$REPO" --json labels -q '[.labels[].name | select(startswith("atl-request:"))] | length' 2>/dev/null || echo 0)
ge "$reqlab" && ok "candidate carries an 'atl-request:<slug>:po' intake-provenance label (concept #14)" || bad "candidate has no atl-request:<slug>:po label"

# CORE: the [Request Decision] sentinel with a verdict
dec=0; gh issue view "$CAND" --repo "$REPO" --comments 2>/dev/null | grep -q '\[Request Decision\]' && dec=1
[ "$dec" = 1 ] && ok "a **[Request Decision]** sentinel comment landed on the candidate (concept #15)" || bad "no [Request Decision] sentinel comment"
rec=0; gh issue view "$CAND" --repo "$REPO" --comments 2>/dev/null | grep -q '## Recommendation' && rec=1
[ "$rec" = 1 ] && ok "the decision record carries a ## Recommendation verdict" || bad "no ## Recommendation in the decision record"

# NOTE: the triage tier
tier=$(gh issue view "$CAND" --repo "$REPO" --json labels -q '[.labels[].name | select(startswith("triage:"))] | .[0] // ""' 2>/dev/null)
[ -n "$tier" ] && note "triage tier assigned: $tier" || note "no triage:<tier> label this run (LLM-variable)"

# ---- 2. PO gate — ACCEPT: drop the candidate flag so the item enters the frontier ----
gturn "/request — continue at the PO gate for candidate #$CAND. You are the human product owner. ACCEPT the request. Per the ceremony's accept path: remove the 'candidate' label from issue #$CAND so it leaves the candidate state and enters the ready frontier (concept #13), and update the '**[Request Decision]**' comment's '## PO Decision' section to record 'accept' and the convergence mechanism (concession). Do NOT run /refine or create PBIs — stop after the accept transition." || bad "request turn 2 (PO accept) errored"

# CORE: the candidate flag was dropped (item entered the frontier)
still=$(gh issue view "$CAND" --repo "$REPO" --json labels -q '[.labels[].name | select(. == "candidate")] | length' 2>/dev/null || echo 1)
{ [ "${still:-1}" -eq 0 ]; } 2>/dev/null \
  && ok "PO accept DROPPED the 'candidate' flag on #$CAND — it entered the ready frontier (concept #13)" \
  || bad "the 'candidate' label is still on #$CAND after accept — the frontier transition did not happen"

# NOTE: the decision record shows accept
acc=0; gh issue view "$CAND" --repo "$REPO" --comments 2>/dev/null | grep -iA3 '## PO Decision' | grep -qi 'accept' && acc=1
[ "$acc" = 1 ] && note "the [Request Decision] ## PO Decision records 'accept'" || note "accept not textually confirmed in ## PO Decision (LLM-variable wording)"

# ---- on failure, surface what the torn-down container would otherwise lose -----------
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (github-request-loop failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- turns.log (tail) ---"; tail -120 "$HOME/turns.log" 2>/dev/null
  echo "--- issues ---"; gh issue list --repo "$REPO" --state all --json number,title,state,labels 2>/dev/null
  echo "--- candidate comments ---"; [ -n "$CAND" ] && gh issue view "$CAND" --repo "$REPO" --comments 2>/dev/null | tail -60
  echo "=============================================="
fi

finish
