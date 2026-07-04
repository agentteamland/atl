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
- **Transport:** MCP-first — every role + worker reaches Azure through the
  `azureDevOps` MCP (`wit_*`/`repo_*`/`work_*`/`wiki_*`); the one operation the MCP
  can't do (attachment upload) is a thin REST carve-out. Contract:
  [`knowledge/azure-adapter.md`](knowledge/azure-adapter.md).

> **Status:** under construction. Stone #1 (`atl work dispatch`) shipped; this stone
> adds the Azure operation-contract. Not yet in the install catalog.

Full documentation lives on the docs site: <https://docs.agentteamland.com/>.

Licensed under the MIT License — see [LICENSE](LICENSE).
