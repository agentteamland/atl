# `atl list`

Show the teams installed at each scope.

## Usage

```bash
atl list          # human-readable, grouped by scope
atl list --json   # the same set as JSON, for a script or a skill
```

`atl list` takes no arguments. It reads the install manifests under each layer's `.atl/installed/` directory — no network access.

## Output

Teams are grouped by [scope](/guide/concepts#scope-global-and-project), with each team printed as a two-space-indented `<handle>/<name>@<version>` line:

```
global:
  acme/example-team@1.0.0
project:
  acme/proto-team@0.3.0
```

A team installed at both scopes appears under each. The `<handle>` is the team's GitHub owner, `<name>` and `<version>` come from its `team.json`.

A team that declares a **review agent** (`capabilities.review` in its `team.json`) shows it alongside its version:

```
project:
  acme/proto-team@0.3.0  (review: code-reviewer)
```

## `--json`

Emits the same set as an array, which is how [`/create-pr`](/skills/create-pr) discovers which teams offer a domain reviewer for its review chain:

```json
[
  {
    "handle": "acme",
    "name": "proto-team",
    "version": "0.3.0",
    "scope": "project",
    "reviewer": "code-reviewer"
  }
]
```

`reviewer` is omitted for a team that declares none, which is the common case. With no teams installed the output is `[]`, never `null`.

This is deliberately **not** the raw manifest: a manifest also carries a per-file checksum map running to hundreds of entries, and a caller asking *which teams are installed, and what does each declare?* should not have to read past it.

::: tip A team installed before v2.26.0
`reviewer` is recorded at install time, so a team installed before the field existed shows none until [`atl update`](/cli/update) refreshes its manifest.
:::

## When nothing is installed

If no teams are installed at either scope:

```
atl list: no teams installed
```

## Related

- [`atl install`](/cli/install) — install a team.
- [`atl remove`](/cli/remove) — remove a team.
- [`atl search`](/cli/search) — find teams to install.
