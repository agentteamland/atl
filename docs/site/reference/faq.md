# FAQ

### What does `atl` stand for?

**A**gent**T**eam**L**and. The CLI, the org, and the ecosystem share the name.

### Does `atl` replace Claude Code?

No. `atl` is a delivery layer for the files Claude Code already reads from `.claude/`. You still run Claude Code exactly as before; `atl` just makes it easier to get a solid configuration in place — and keeps it improving on its own.

### How is this different from just copying files between projects?

Three ways:

1. **Versioning.** Teams tag SemVer releases. `atl update` pulls the latest published version of each installed team, so improvements fan out to every install.
2. **Scope.** A team installs at a [scope](/guide/concepts#scope-global-and-project) — global (`~/.claude/`) or project (`<project>/.claude/`). When both carry an asset with the same name, the project copy shadows global. Hand-copied files have no such axis.
3. **A self-running learning loop.** As your agents work, they accumulate gains — new learnings, sharpened skills. `atl` captures them automatically into a durable queue, `/drain` folds them into agent knowledge bases, and `atl promote` / `atl publish` circulate them outward. Copying files never gets better on its own.

### Do I need to run `atl` to use Claude Code?

No. Claude Code works fine without `atl`. Use `atl` when you want a reproducible, shareable setup — solo projects with a hand-rolled `.claude/` are still perfectly valid.

### Can I install more than one team in the same project?

Yes. Each install adds its own copies into `.claude/`. If two teams ship an agent with the same name, the **most-recently-installed** team's version wins — it silently overwrites the earlier copy (`atl` does not currently warn about the collision). This is collision handling, not inheritance — each team is installed independently. Use [`atl list`](/cli/list) to see what's installed at each scope.

### Where do teams come from? Can I install from a private repo or a Git URL?

`atl install` is **catalog-only**. It takes a `<handle>/<team>` reference, resolves it against the GitHub-backed catalog (built from public repos tagged with the [`atl-team`](https://github.com/topics/atl-team) topic), fetches the source as an ephemeral HTTPS tarball, and copies the team's installable subtrees (`agents/`, `skills/`, `rules/`, `knowledge/`, `backends/`, `scripts/`, `packs/`) into the scope's `.claude/`.

There is no install from a private repo, an arbitrary Git URL, SSH, or a local path — those were v1. If you want a team installable, make its repo public and tag it `atl-team` (or run [`atl publish`](/cli/publish) from the repo). See [`atl search`](/cli/search) for how the catalog is queried.

### What if my `atl` version is too old for a team?

A team's `team.json` can declare a `requires.atl` minimum to signal the version it expects. To update the binary itself, run [`atl upgrade`](/cli/upgrade) — it downloads the latest stable release, verifies its checksum, and atomically swaps it in (`atl` also checks for and applies this automatically at session start). [`atl update`](/cli/update) refreshes installed teams and core assets, but never changes the binary version. Re-running the install script works too:

```bash
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

### What happens to `atl`'s on-disk state if I delete a project?

Nothing global is affected. `atl` keeps two kinds of state, and they're separate:

- **Team assets** live in Claude Code's own `.claude/` directories. Deleting a project deletes that project's `.claude/`; global assets in `~/.claude/` are untouched.
- **`atl`'s bookkeeping** lives under `~/.atl/` (global) and `<project>/.atl/` (project) — the cached catalog (`index.json`), the learning queue (`queue.db`), pins, per-team install manifests, downloaded embedder models (`~/.atl/models/`), and each project's retrieval index (`~/.atl/cache/retrieve/<project-slug>/`). The global `~/.atl/` survives; the project's `<project>/.atl/` goes with the project (its retrieval index lingers in the global cache — a harmless leftover you can delete).

There is no shared clone cache to clean up — sources are fetched as throwaway tarballs at install time, never kept on disk.

### Can I install a team without running `atl` (by hand)?

Yes — `atl` just automates a copy. Fetch the team repo, then copy its installable subtrees — `agents/`, `skills/`, `rules/`, `knowledge/`, `backends/`, `scripts/`, `packs/` — into the target `.claude/` yourself. There's no inheritance or excludes resolution to replicate, and no persistent cache to populate. The only thing you'd lose is the install manifest `atl` writes (see below), which `atl update` and `atl doctor` rely on for refresh and self-heal.

### Where does `atl` keep the list of installed teams?

In a per-team manifest, one JSON file per team per scope, at `<scope>/.atl/installed/<handle>__<name>.json` (`<scope>` is `~/.atl` for a global install, `<project>/.atl` for a project install). Each manifest records the team's `handle`, `name`, `version`, `scope`, `source`, install time, and a `files` map of every written path to its SHA-256 baseline. `atl update`'s auto-refresh and `atl doctor`'s integrity check read those baselines. Editing manifests by hand is unsupported — use `atl install` / `atl remove`.

### Does `atl` send any telemetry?

No. `atl` is a local tool: it fetches teams over HTTPS, reads the catalog, and writes copies into `.claude/`. There's no phone-home.

### Is this an Anthropic product?

No. AgentTeamLand is an independent open-source project that works with Anthropic's Claude Code. MIT-licensed. No commercial affiliation.

### How do I contribute?

- **Publish a team.** Tag the team's repo with the [`atl-team`](https://github.com/topics/atl-team) topic, or run [`atl publish`](/cli/publish) from the repo. The catalog picks it up automatically — there's no registry and no submission PR.
- **Improve the CLI, the core rules/skills, or these docs.** They all live in the [`agentteamland/atl`](https://github.com/agentteamland/atl) monorepo. PRs welcome; every docs page has an "Edit this page on GitHub" link.
- **Open issues.** Bug reports and feature requests go on [`agentteamland/atl`](https://github.com/agentteamland/atl/issues).

### My question isn't here.

Open an issue on [`agentteamland/atl`](https://github.com/agentteamland/atl/issues) with the `faq` label. If it's a common question, it'll end up on this page.
