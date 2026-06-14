# `atl search`

Search the public registry.

## Usage

```bash
atl search <query>
```

`<query>` matches against team names, descriptions, and keywords. It's case-insensitive and substring-based; no regex.

## Example

```bash
atl search dotnet
```

```
Found 1 team(s) matching "dotnet":

  software-project-team@1.2.1 [verified]
    .NET 9 API + Flutter + React + full Docker stack
    https://github.com/agentteamland/software-project-team
    keywords: dotnet, csharp, flutter, react
```

Status badges (in brackets after `name@version`):

- **`[verified]`** — reviewed by AgentTeamLand maintainers. Expected to install cleanly and follow conventions.
- **`[community]`** — listed in the registry, not yet reviewed. Works, but use at your own risk.
- **`[deprecated]`** — still installable, but no longer maintained. Migrate when convenient.

## A query is required

`atl search` requires exactly one positional argument. Running it with no query exits with a usage error — to browse the full catalog, see [the registry on GitHub](https://github.com/agentteamland/registry/blob/main/teams.json) or use a broad keyword like `atl search team`.

## Offline behavior

The registry is cached locally after the first fetch. `atl search` works offline using the cached copy; it prints a note so you know the results may be stale.

## No results?

The registry is PR-driven and young — if your domain isn't covered, that's likely just "not yet." Consider:

- Using a Git URL directly: `atl install https://github.com/you/your-team.git`
- Publishing your own team: [Creating a team](/authoring/creating-a-team)
- Submitting it to the registry: [Registry submission](/authoring/registry-submission)

## Related

- [`atl install`](/cli/install) — install what you find.
- [Registry submission](/authoring/registry-submission) — get your team listed.
