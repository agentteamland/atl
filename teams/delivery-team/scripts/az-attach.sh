#!/usr/bin/env bash
# az-attach.sh — the delivery-team's ONE Azure REST carve-out (azure-adapter §9).
#
# Uploads a local file as a work-item attachment and links it — the single Azure
# operation the azureDevOps MCP can't do. Every other Azure call goes through the
# MCP; this keeps the transport split hidden behind one interface: a worker calls
#   az-attach.sh <work-item-id> <file> [comment]
# and never touches REST directly. The Go orchestrator stays zero-Azure — this
# lives in the team, run by the worker with the same PAT the MCP uses.
#
# jq builds and parses ALL JSON (never hand-concatenated) so a comment or filename
# with quotes / newlines / URL-reserved chars can't corrupt the request.
#
# Env (same names the worker's MCP already has; PAT never on the argv, never logged):
#   AZURE_DEVOPS_ORG       e.g. "contoso"          (required)
#   AZURE_DEVOPS_PROJECT   the project name/id     (required)
#   AZURE_DEVOPS_PAT       the PAT (Basic auth)     (required; falls back to PERSONAL_ACCESS_TOKEN)
#   AZURE_DEVOPS_API_VERSION   default 7.1
set -euo pipefail

WI="${1:-}"; FILE="${2:-}"; COMMENT="${3:-test evidence}"
ORG="${AZURE_DEVOPS_ORG:-}"
PROJECT="${AZURE_DEVOPS_PROJECT:-}"
PAT="${AZURE_DEVOPS_PAT:-${PERSONAL_ACCESS_TOKEN:-}}"
APIVER="${AZURE_DEVOPS_API_VERSION:-7.1}"

die() { echo "az-attach: $1" >&2; exit 2; }
command -v jq   >/dev/null || die "jq is required (JSON is built + parsed with jq, never hand-concatenated)"
command -v curl >/dev/null || die "curl is required"
[ -n "$WI" ] && [ -n "$FILE" ] || die "usage: az-attach.sh <work-item-id> <file> [comment]"
[[ "$WI" =~ ^[0-9]+$ ]] || die "work-item id must be numeric, got: $WI"
[ -f "$FILE" ] || die "file not found: $FILE"
[ -n "$ORG" ] && [ -n "$PROJECT" ] || die "AZURE_DEVOPS_ORG and AZURE_DEVOPS_PROJECT must be set"
[ -n "$PAT" ] || die "AZURE_DEVOPS_PAT (or PERSONAL_ACCESS_TOKEN) must be set"

BASE="https://dev.azure.com/${ORG}"
NAME="$(basename "$FILE")"
NAME_ENC="$(jq -rn --arg n "$NAME" '$n|@uri')"   # full percent-encode (& ? # + space non-ASCII)

# curl with bounded backoff on 429/5xx (best-effort; on exhaustion the worker
# re-invokes — the adapter §3 resilience contract lives in the MCP callers, not here).
# The PAT rides -u ":$PAT" (empty user) as a Basic auth HEADER — never the argv, never logged.
req() { # req METHOD URL CONTENT_TYPE [curl-data-args...]
  local method="$1" url="$2" ctype="$3"; shift 3
  local attempt=1 out code body
  while :; do
    out="$(curl -sS -w '\n%{http_code}' -u ":$PAT" -X "$method" -H "Content-Type: $ctype" "$url" "$@" || true)"
    code="${out##*$'\n'}"; body="${out%$'\n'*}"
    case "$code" in
      2*) printf '%s' "$body"; return 0 ;;
      429|5*) [ "$attempt" -ge 5 ] && die "HTTP $code after $attempt attempts: $body"
              sleep "$attempt"; attempt=$((attempt + 1)) ;;
      *) die "HTTP $code: $body" ;;
    esac
  done
}

# 1) upload the bytes → {"id":..., "url":"...attachments/<guid>"}
UP="$(req POST \
  "${BASE}/${PROJECT}/_apis/wit/attachments?fileName=${NAME_ENC}&api-version=${APIVER}" \
  "application/octet-stream" --data-binary "@${FILE}")"
ATT_URL="$(printf '%s' "$UP" | jq -r '.url // empty')"
[ -n "$ATT_URL" ] || die "no attachment url in response: $UP"

# 2) link the attachment to the work-item (JSON Patch add relation; jq builds the body safely)
PATCH="$(jq -n --arg url "$ATT_URL" --arg c "$COMMENT" \
  '[{op:"add",path:"/relations/-",value:{rel:"AttachedFile",url:$url,attributes:{comment:$c}}}]')"
req PATCH "${BASE}/_apis/wit/workitems/${WI}?api-version=${APIVER}" \
  "application/json-patch+json" --data "$PATCH" >/dev/null

echo "attached $NAME -> work-item $WI"
