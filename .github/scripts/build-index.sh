#!/usr/bin/env bash
#
# Regenerate cli/internal/index/index.json from the `atl-team` GitHub topic
# (publish design F1, topic-discovery). First-party teams (handle
# `agentteamland`, monorepo subpaths) are preserved from the current index;
# third-party repos tagged `atl-team` are discovered, their team.json read, and
# appended. `verified` = agentteamland (first-party) + a maintainer allowlist.
#
# Runs in CI (.github/workflows/index.yml). Needs gh (GH_TOKEN) + jq.
# Network/topic behaviour is exercised in CI; locally it is safe to run (an empty
# topic just preserves the first-party entries).
set -euo pipefail

INDEX="cli/internal/index/index.json"
ALLOWLIST=() # extra owners granted verified, beyond agentteamland

# 1. Preserve first-party entries (handle == agentteamland) from the current index.
firstparty="$(jq '[.teams[] | select(.handle == "agentteamland")]' "$INDEX")"

# 2. Discover third-party repos via the atl-team topic (skip the org's own repos).
thirdparty="[]"
repos="$(gh search repos --topic atl-team --limit 200 --json fullName,owner \
  -q '.[] | select(.owner.login != "agentteamland") | .fullName' 2>/dev/null || true)"
for repo in $repos; do
  tj="$(gh api "repos/$repo/contents/team.json" -q '.content' 2>/dev/null | base64 -d 2>/dev/null || true)"
  [ -n "$tj" ] || continue
  owner="${repo%%/*}"
  ref="$(gh api "repos/$repo/releases/latest" -q '.tag_name' 2>/dev/null || echo main)"
  verified=false
  for v in agentteamland "${ALLOWLIST[@]:-}"; do [ "$owner" = "$v" ] && verified=true; done
  entry="$(printf '%s' "$tj" | jq --arg repo "$repo" --arg owner "$owner" --arg ref "$ref" --argjson verified "$verified" '{
    handle: $owner, name: .name, version: (.version // "0.0.0"),
    description: (.description // ""), keywords: (.keywords // []),
    scope: (.scope // "project"), verified: $verified,
    source: { repo: $repo, subpath: "", ref: $ref }
  }')" || continue
  thirdparty="$(printf '%s' "$thirdparty" | jq --argjson e "$entry" '. + [$e]')"
done

# 3. Merge first-party + third-party and write.
now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
jq -n --arg now "$now" --argjson fp "$firstparty" --argjson tp "$thirdparty" \
  '{ schemaVersion: 1, generatedAt: $now, teams: ($fp + $tp) }' > "$INDEX.tmp"
mv "$INDEX.tmp" "$INDEX"
echo "index: $(jq '.teams | length' "$INDEX") teams ($(printf '%s' "$firstparty" | jq length) first-party + $(printf '%s' "$thirdparty" | jq length) third-party)"
