# `atl list`

Show the teams installed in the current project.

## Usage

```bash
atl list
```

## Output

```
Installed teams in /home/you/projects/your-app:

  ✓ software-project-team@1.2.1
     effective: 13 agents, 3 skills, 1 rules

  ↑ starter-extended@0.2.0 [community]  (latest: 0.3.0)
     extends: software-project-team@^1.0.0
     effective: 12 agents, 2 skills, 1 rules
```

Per-team format:

- **`✓`** — installed at the latest known version (green check).
- **`↑`** — installed but the registry has a newer version (yellow up-arrow). The newest version is shown in parentheses; run `atl update` to refresh.
- **`name@version`** — the installed team name and version (from `team.json`).
- **`[community]`** — status badge appears when the team is community-status (not yet verified). Verified teams have no badge.
- **`extends:`** — only printed when the team has a parent. Shows the inheritance chain (root → ... → installed team).
- **`effective:`** — counts after inheritance resolution: parent + child resources, minus `excludes`, with child-override collisions collapsed.

## When the project has no teams

```
No teams installed in this project.
Run: atl install <team-name>
```

## Notes

- `atl list` reads from `.claude/.team-installs.json` and the global registry cache. It does NOT hit the network in the steady state — outdated detection (`↑`) uses the cached registry copy, which is refreshed by `atl update`.
- The command currently has no flags. Earlier docs referenced `--json` and `--chain`; those were never implemented. If you need scriptable output, parse `.claude/.team-installs.json` directly — it's stable JSON with `name`, `version`, `extendsChain`, and `effective` fields.

## Related

- [`atl install`](/cli/install)
- [`atl remove`](/cli/remove)
- [`atl update`](/cli/update)
- [Inheritance](/authoring/inheritance)
