# mock-mcp — a mock `azureDevOps` MCP server (delivery-team Layer-A)

A zero-dependency Node stdio MCP server that stands in for the real
`@azure-devops/mcp`, so the delivery-team ceremonies can run **without live Azure**.
This is the `①`/`②a` half of delivery-team **stone #9** (the Azure-free half); the
`③` real-Azure Layer-B is a separate, environment-gated lane.

## Why it exists

The delivery ceremonies (`/kickoff`, `/refine`, `/sprint-plan`, `/sprint-start`,
`/sprint-review`) reach Azure only through the `azureDevOps` MCP tool surface — never
through code (the adapter is a *documented contract*, `teams/delivery-team/knowledge/
azure-adapter.md`). So "mock the adapter" = stand up a fake `azureDevOps` MCP server
exposing the **same tool names** over a believable in-memory store, and repoint the
ceremony's `--mcp-config` at it. This is the `#3 adapter second implementation` the
test strategy (detail-spec #18) calls for — it proves the contract is real by running
the ceremonies against a second implementation of it.

## What it is

| File | Role |
|---|---|
| `server.js` | stdio JSON-RPC 2.0 loop — `initialize` / `tools/list` / `tools/call`. Zero deps. |
| `tools.js` | the **curated** adapter tool surface (azure-adapter.md §2) + handlers. Off-contract tools (`wit_update_work_item_comment`, `work_list_team_iterations`, `pipelines_*`…) are deliberately absent — calling one is an error, not a silent success. |
| `store.js` | the in-memory work-item store: work-items, a Scrum process template with a non-literal state→category map (§6 runtime resolution), iterations with velocity history, PRs, wiki. Persists to `MOCK_STORE`. |
| `mock.test.js` | auth-free unit tests (`node --test`) — the always-on backbone that proves the mock without a Claude token. |

## Running

```bash
node --test test/e2e/mock-mcp        # the auth-free unit backbone (no Claude)
```

It is wired into a ceremony (real-Claude, token-gated) run by the
`delivery-loop` e2e blueprint via a `.mcp.json` binding `azureDevOps` → this server.

## State persistence

Claude Code spawns a fresh stdio process per `claude -p` turn, but the ceremony
chain spans several turns. So the store persists to the file named by the
`MOCK_STORE` env var (the blueprint points it at `.delivery/mock-store.json`), and
each fresh process reloads it — mutations from `/kickoff` are visible to `/refine`,
and the blueprint asserts on the store file between steps.

## Scope (honest)

Layer-A drives the **ceremonies** against the mock. It does NOT cover: the developer
`claude -p` worker micro-loop, the `az-attach.sh` REST attachment upload, or the
per-unit review/merge/Done orchestration — those are the real-Azure Layer-B (`③`)
and its deferred Go seams, gated on a provisioned Azure test project.
