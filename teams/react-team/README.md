# react-team

A **project-scope** first-party ATL team shipping one stack specialist: **`react-web`** — React +
TypeScript web-app craft for the delivery loop. Screens and routing, the server-state / client-state
split, the API client, forms and input, and the browser verification a green build cannot replace.

## How it is used

The specialist is **knowledge, not a worker**. `atl work dispatch` spawns the delivery-team's
`developer`; that worker becomes competent on this stack by loading `react-web`'s `children/`,
exactly where it would otherwise load a generic `packs/<area>/` stack pack. So this team ships no
delivery role-craft of its own — the worktree, the claim protocol, self-test, escalation and the PR
contract stay in `developer` and travel unchanged across every stack.

**The specialist declares what it is; the project's tech-lead binds it.** An `area:` tag is a
functional slice of *one* system — this project may call this slice `web`, the next `admin`, the
next `portal` — so a shipped agent cannot know which area it belongs to. The binding lives in the
area table on the tech-lead's `Architecture/` page. That is what lets two projects share an area
name and resolve to different stacks, and what lets this team install without delivery-team ever
learning a stack name.

Where it is bound it **replaces** that area's pack; the two are never layered. A worker reading both
would be arbitrating between two documents written by different hands about the same decision.

## Contents

| Path | What |
|---|---|
| `agents/react-web/agent.md` | identity, area of responsibility, core principles, and the auto-aggregated Knowledge Base index |
| `agents/react-web/children/` | the craft itself — one topic per file, including the **screen blueprint** (the production unit: decide → scaffold → **register in the route table** → open it in a browser and look) |

The grounded baseline is the client-rendered SPA (Vite, React Router, a client data layer); on a
meta-framework the routing and data-loading half is that framework's and the project's
`Conventions/` is the authority, which the agent states at the point it matters.

The `children/` are **seeds that mature**: durable React lessons land there through the capture →
`/drain` loop, so what one project pays for the next inherits.

Full documentation lives on the docs site: <https://docs.agentteamland.com/>.

Licensed under the MIT License.
