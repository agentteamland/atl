# 🎨 Design System Team

> Build comprehensive design systems and UI prototypes inside any project. Local, file-based, browser-viewable. No daemon, no API key, no external service — your existing Claude Code session does all the AI work.

The team ships 2 agents (`ds-architect-agent`, `prototype-agent`) and 10 `/dst-*` skills that produce `.dst/{ds.json, prototype.json, *.html}` artifacts in any project. Outputs are static HTML pages you open in your browser; the JSON is the source of truth and HTML is regenerable. `/dst-handoff` bundles a prototype and briefs `flutter-agent` or `react-agent` to integrate the design into your real source tree.

`.dst/` commits cleanly to git. Tokens are referenced by name, never copied — editing a design system cascades to every linked prototype.

## 📚 Documentation

Full docs live at **[agentteamland.github.io/docs](https://agentteamland.github.io/docs/)**.

Most relevant sections:

- [Team page](https://agentteamland.github.io/docs/teams/design-system-team) — full agent + skill listing, target platforms (flutter / react-admin / react-public), workflows, troubleshooting
- [Concepts](https://agentteamland.github.io/docs/guide/concepts) — team / agent / skill / rule mental model
- [Children + learnings](https://agentteamland.github.io/docs/guide/children-and-learnings) — the knowledge-base pattern both agents in this team use
- [Install via `atl`](https://agentteamland.github.io/docs/cli/install) — `atl install design-system-team` (the legacy `/team install` was retired in `team-manager@2.0.0`)

## License

MIT.
