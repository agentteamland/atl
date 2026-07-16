# delivery-team

A **project-scope** first-party ATL team (in-progress build). An Azure DevOps
work-item-driven, sprint/Scrum-based autonomous software-delivery org: role-agents
(intake · business-analyst · technical-analyst · project-manager · tech-lead ·
tester) + dynamic `developer` workers + a human Product Owner, driven by ceremony
skills and orchestrated by the deterministic `atl work dispatch` engine over
isolated `claude -p` workers in git worktrees.

- **Source of truth:** the Azure DevOps project — work-items (transient execution
  state) + the project wiki (durable knowledge). The team stays thin and resumable
  because state lives in Azure, not in a long-lived context.
- **Backend:** the work-item store, git host, and durable-knowledge store are a pluggable
  **backend**, behind a provider-neutral contract
  ([`knowledge/backend-interface.md`](knowledge/backend-interface.md)) with per-provider
  implementations under `backends/` — Azure (via the `azureDevOps` MCP) and GitHub (via `gh`).

> **Status:** under construction. Stone #1 (`atl work dispatch`) shipped; this stone
> adds the Azure operation-contract. Not yet in the install catalog.

Full documentation lives on the docs site: <https://docs.agentteamland.com/>.

Licensed under the MIT License — see [LICENSE](LICENSE).
