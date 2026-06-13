# atl — AgentTeamLand

The AgentTeamLand platform: install agent teams, keep them updated, circulate the gains your agents learn, and let the platform run itself in the background so you can focus on your project.

> **Status:** v2 rebuild in progress — private, pre-release, not yet at feature parity with v1. Goes public + v1 repos archive at cutover (see the decision doc).

## Monorepo layout

| Path | What |
|---|---|
| `cli/` | the `atl` binary (Go) — the deterministic plumbing layer |
| `core/` | global rules + skills _(ported in a later step)_ |
| `teams/` | first-party teams _(ported in a later step)_ |
| `docs/` | VitePress docs site _(ported in a later step)_ |
| `.atl/` | repo-local architecture reference |

## Why v2

See [`.atl/docs/atl-v2.md`](.atl/docs/atl-v2.md) for the full platform-restructure decision: monorepo consolidation (15 repos → 2), a first-class global/project scope axis, a bbolt durable learning queue that kills the re-report bug class, three-ring gain circulation, GitHub-backed self-serve publish, and an automation-mandatory reliability layer (doctor as an automatic self-heal daemon).

## What works today

The `cli/` scaffold builds and runs, and the learning **queue** — the substrate the whole self-driving loop rides on — is real:

```bash
cd cli
go build ./... && go test ./...

# the queue is live (durable, per-project, dedup'd, multi-channel):
go run ./cmd/atl learnings _enqueue learning "prefers Node for APIs"
go run ./cmd/atl learnings status
```

Everything else (`install`, `update`, `promote`, `publish`, `doctor`) is a scaffold stub whose deterministic logic is filled in next.

## License

MIT — see [LICENSE](LICENSE).
