# `atl search`

Search the team catalog — the GitHub-backed index that [`atl install`](/cli/install) resolves against.

## Usage

```bash
atl search [keyword]
```

`[keyword]` matches against team handles, names, descriptions, and keywords. Matching is case-insensitive and substring-based; no regex. Run `atl search` with no keyword to browse the whole catalog.

## Example

```bash
atl search example
```

```
1 team(s) matching "example":

  acme/example-team@1.0.0
    An example team: a small stack of agents, skills, and rules for a fictional project.
    keywords: example, full-stack, starter
    install: atl install acme/example-team
```

Each result shows:

- the `<handle>/<name>@<version>` reference (the handle is the team's GitHub owner — ownership is authorship),
- the description and keywords,
- the exact `atl install` command to copy.

The **`[verified]`** badge marks teams reviewed by AgentTeamLand maintainers (`agentteamland/*` plus a maintainer allowlist). Its absence just means the team is self-published — not that it's unsafe.

## Browse the whole catalog

Omit the keyword to list every catalogued team:

```bash
atl search
```

## Offline behavior

`atl search` never blocks on the network. It resolves the index offline-first: the network-refreshed cache at `~/.atl/index.json` when present, otherwise the copy embedded in the binary. The cache is refreshed out of band (by `atl update`), so results stay current without `search` ever waiting on a fetch.

## No results?

The catalog is generated from public GitHub repositories tagged with the [`atl-team`](https://github.com/topics/atl-team) topic, and it's young — if your domain isn't covered yet, that's likely just "not yet." To get a team listed, tag its repo with `atl-team` (or run `atl publish` from the team repo) and the catalog picks it up. See [Creating a team](/authoring/creating-a-team).

## Related

- [`atl install`](/cli/install) — install what you find.
