# dotnet-team

A **project-scope** first-party ATL team shipping one stack specialist: **`dotnet-api`** — .NET API
craft for the delivery loop. ASP.NET Core on the request path, EF Core behind it, and the four
surfaces that hang off them (a cache, a message broker, a realtime hub, and work that runs outside a
request).

## How it is used

The specialist is **knowledge, not a worker**. `atl work dispatch` spawns the delivery-team's
`developer`; that worker becomes competent on this stack by loading `dotnet-api`'s `children/`,
exactly where it would otherwise load a generic `packs/<area>/` stack pack. So this team ships no
delivery role-craft of its own — the worktree, the claim protocol, self-test, escalation and the PR
contract stay in `developer` and travel unchanged across every stack.

**The specialist declares what it is; the project's tech-lead binds it.** An `area:` tag is a
functional slice of *one* system — this project may call this slice `api`, the next `backend`, the
next `core-service` — so a shipped agent cannot know which area it belongs to. The binding lives in
the area table on the tech-lead's `Architecture/` page. That is what lets a Node project and a .NET
project share `area:api` and resolve differently, and what lets this team install without
delivery-team ever learning a stack name.

Where it is bound it **replaces** that area's pack; the two are never layered. A worker reading both
would be arbitrating between two documents written by different hands about the same decision.

## Contents

| Path | What |
|---|---|
| `agents/dotnet-api/agent.md` | identity, area of responsibility, core principles, and the auto-aggregated Knowledge Base index |
| `agents/dotnet-api/children/` | the craft itself — one topic per file, including the **endpoint blueprint** (the production unit: decide → scaffold → **register** → verify it is reachable) |

The `children/` are **seeds that mature**: durable .NET lessons land there through the capture →
`/drain` loop, so what one project pays for the next inherits.

Full documentation lives on the docs site: <https://docs.agentteamland.com/>.

Licensed under the MIT License.
