# dotnet-team

A **project-scope** first-party ATL team shipping one stack specialist: **`dotnet-api`** — .NET API
craft. ASP.NET Core on the request path, EF Core behind it, and the four
surfaces that hang off them (a cache, a message broker, a realtime hub, and work that runs outside a
request).

## How it is used

The specialist is **knowledge, not a worker**. Any session working on a .NET backend becomes
competent on this stack by reading `dotnet-api`'s `children/`. So this team ships no process craft of
its own — no branch handling, no claim protocol, no PR contract. It is stack knowledge and nothing
else.

**The specialist declares what it is, never where it is used.** A project decides which parts of its
system this knowledge applies to; a shipped agent cannot know what a given project calls that slice.

## Contents

| Path | What |
|---|---|
| `agents/dotnet-api/agent.md` | identity, area of responsibility, core principles, and the auto-aggregated Knowledge Base index |
| `agents/dotnet-api/children/` | the craft itself — one topic per file, including the **endpoint blueprint** (the production unit: decide → scaffold → **register** → verify it is reachable) |

The `children/` are **seeds that mature**: durable .NET lessons land there through the capture →
`/drain` loop, so what one project pays for the next inherits.

Full documentation lives on the docs site: <https://docs.agentteamland.com/>.

Licensed under the MIT License.
