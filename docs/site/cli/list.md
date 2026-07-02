# `atl list`

Show the teams installed at each scope.

## Usage

```bash
atl list
```

`atl list` takes no flags or arguments. It reads the install manifests under each layer's `.atl/installed/` directory — no network access.

## Output

Teams are grouped by [scope](/guide/concepts#scope-global-and-project), with each team printed as a two-space-indented `<handle>/<name>@<version>` line:

```
global:
  acme/example-team@1.0.0
project:
  acme/proto-team@0.3.0
```

A team installed at both scopes appears under each. The `<handle>` is the team's GitHub owner, `<name>` and `<version>` come from its `team.json`.

## When nothing is installed

If no teams are installed at either scope:

```
atl list: no teams installed
```

## Related

- [`atl install`](/cli/install) — install a team.
- [`atl remove`](/cli/remove) — remove a team.
- [`atl search`](/cli/search) — find teams to install.
