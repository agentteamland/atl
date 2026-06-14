# FAQ

### What does `atl` stand for?

**A**gent**T**eam**L**and. The CLI, the org, and the ecosystem share the name.

### Does `atl` replace Claude Code?

No. `atl` is a delivery layer for the files Claude Code already reads from `.claude/`. You still run Claude Code exactly as before; `atl` just makes it easier to get a solid configuration in place.

### How is this different from just copying files between projects?

Three ways:

1. **Versioning.** Teams tag SemVer releases. You pin a version; updates are opt-in via `atl update`.
2. **Inheritance.** You can build on someone else's team without forking. When they ship improvements, you inherit them.
3. **Reach.** Publishing a team once makes it available to every Claude Code user who runs `atl install your-team`.

### Do I need to run `atl` to use Claude Code?

No. Claude Code works fine without `atl`. Use `atl` when you want a reproducible, shareable setup — solo projects with a hand-rolled `.claude/` are still perfectly valid.

### Can I install more than one team in the same project?

Yes. Each install adds its own copies into `.claude/`. If two teams ship an agent with the same name, the **second-installed** team's version wins (it overwrites the first copy). Use `atl list` to see which team owns each copy.

### Can I install teams from private Git repos?

Yes, as long as your Git client can clone the repo (SSH keys, PAT, etc.). `atl` doesn't manage Git credentials itself — it delegates to your shell's Git setup.

### What if my `atl` version is too old for a team?

The team's `team.json` can declare a `requires.atl` minimum. If your installed `atl` is older, you get a clear error telling you to upgrade. Use `brew upgrade atl` or whichever channel you installed via.

### How do I downgrade a team?

```bash
atl install team-name@1.2.0   # install specific version
```

This replaces whatever was installed with the pinned version.

### What happens to the cache if I delete a project?

Nothing — the cache (`~/.claude/repos/agentteamland/`) is shared across projects. Other projects keep working. You can reclaim disk by deleting a specific team's cache directory manually when you're sure no project uses it anymore.

### Can I install a team without running `atl` (by hand)?

Yes. Clone the team repo into `~/.claude/repos/agentteamland/<team-name>/` and create the copies in `.claude/` yourself. The CLI exists to automate that plus inheritance/excludes resolution; there's nothing magic about its output.

### Where does `atl` keep the list of installed teams?

In each project: `.claude/.team-installs.json`. Human-readable JSON. You can edit it if you know what you're doing — but `atl install` / `atl remove` is always safer.

### Does `atl` send any telemetry?

No. `atl` is a local tool: it clones from GitHub, fetches the registry, writes copies. There's no phone-home.

### Is this an Anthropic product?

No. AgentTeamLand is an independent open-source project that works with Anthropic's Claude Code. MIT-licensed. No commercial affiliation.

### How do I contribute?

- **Publish a team** to the registry — [Registry submission](/authoring/registry-submission).
- **Improve the CLI** — PRs welcome at [`agentteamland/cli`](https://github.com/agentteamland/cli).
- **Improve these docs** — PRs welcome at [`agentteamland/docs`](https://github.com/agentteamland/docs). Every page has an "Edit this page on GitHub" link.
- **Open issues** — bug reports and feature requests go on the relevant repo.

### My question isn't here.

Open an issue on [`agentteamland/docs`](https://github.com/agentteamland/docs/issues) with the `faq` label. If it's a common question, it'll end up on this page.
