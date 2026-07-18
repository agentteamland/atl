# profile-team

A **global-scope** first-party ATL team. It curates a shared, cross-project profile of
the entities in your world — the same entity is the same profile in every project.

- **Install:** `atl install agentteamland/profile-team`
- **Storage:** entity profiles live at `~/.atl/profiles/` (global — not per project).
- **How it learns:** drop a `<!-- profile-fact: … -->` marker in conversation; the
  `profile-curator` agent drains it into the right profile at session start.

Full documentation lives on the docs site: <https://docs.agentteamland.com/>.

Licensed under the MIT License — see [LICENSE](LICENSE).
