# 🏗️ Software Project Team

> 13 specialized AI agents for building production-grade software projects. Vertical Slice + Clean Architecture + Mediator pattern with .NET 9, PostgreSQL, RabbitMQ, Redis, Elasticsearch, Flutter, React, and full Docker.

The team scaffolds and maintains a complete production stack from a single command (`/create-new-project MyApp`). Every agent has a focused area of responsibility — API, sockets, workers, database, mobile, web, infra, code/project review, design system, UX — and a children-pattern knowledge base that grows with use.

Optional Claude Design integration (1.1.0+) inserts a visual-prototype phase between text-only flow specs and code, with handoff playbooks for `flutter-agent` and `react-agent`. UI-only i18n architecture (1.0.3+) keeps the backend English-only while UI apps localize via a `messageKey + placeholders + fallback` envelope.

## 📚 Documentation

Full docs live at **[agentteamland.github.io/docs](https://agentteamland.github.io/docs/)**.

Most relevant sections:

- [Team page](https://agentteamland.github.io/docs/teams/software-project-team) — full agent + skill listing, architecture overview, tech stack, settled patterns
- [Concepts](https://agentteamland.github.io/docs/guide/concepts) — team / agent / skill / rule mental model
- [Children + learnings](https://agentteamland.github.io/docs/guide/children-and-learnings) — the knowledge-base pattern every agent in this team uses
- [Install via `atl`](https://agentteamland.github.io/docs/cli/install) — `atl install software-project-team` (the legacy `/team install` was retired in `team-manager@2.0.0`)

## License

MIT.
