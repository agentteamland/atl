# Teams

Teams are ATL's unit of distribution: a versioned package of agents, skills, and rules that installs into a project (or globally) with one command. Every team follows the [`team.json` contract](/authoring/team-json) and is discoverable through the team catalog.

> The catalog is generated from public GitHub repositories tagged with the [`atl-team`](https://github.com/topics/atl-team) topic. There is no registry to submit to — tagging the repo (or running [`atl publish`](/cli/publish)) is what lists a team.

## First-party teams: being rebuilt

The v1-era first-party teams (a full-stack software team and a design-system team) were **retired in July 2026** — they predated the v2 platform and were removed in favor of a deliberate rebuild on the v2 foundation rather than incremental patching. Their history is preserved in the [atl repository](https://github.com/agentteamland/atl).

The rebuild starts with **profile-team** (a shared user-profile layer that advisory-style teams build on), followed by a new software developer team. This page becomes the catalog browse page again as they ship.

## Browse and install

```bash
atl search                      # browse the catalog
atl search <keyword>            # find teams by name, description, or keyword
atl install <handle>/<team>     # install by reference
```

Teams install by `<handle>/<name>` reference. [`atl install`](/cli/install) resolves the ref against the GitHub-backed catalog, fetches the source as an ephemeral tarball over HTTPS, and copies the team's `agents/`, `skills/`, and `rules/` into the scope's `.claude/` directory. By default a team installs at the scope its publisher declares (project, global, or both); pass `--global` or `--project` to override. See [scopes](/guide/concepts#scope-global-and-project) for how the two layers interact.

Teams published under the `agentteamland/` handle (plus a maintainer allowlist) carry the **`[verified]`** badge in [`atl search`](/cli/search). The badge marks teams reviewed by AgentTeamLand maintainers; its absence on a self-published team does not mean the team is unsafe.

## Publish your own

Anyone can publish a team — see [Creating a team](/authoring/creating-a-team). A public repo with a valid `team.json` and the `atl-team` topic appears in the catalog within the hour.
