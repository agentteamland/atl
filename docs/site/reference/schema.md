# Schema

The `team.json` schema is published at:

**[`agentteamland/core/schemas/team.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/team.schema.json)**

It follows **JSON Schema Draft 2020-12**.

## Quick reference

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `schemaVersion` | integer | ✅ | — | Currently `1`. |
| `name` | string | ✅ | — | Lowercase kebab-case. Must be unique in the registry. |
| `version` | string | ✅ | — | SemVer 2.0.0. |
| `description` | string | ✅ | — | One-sentence summary. |
| `author` | string | — | — | `"Name <email>"` format recommended. |
| `license` | string | — | `"MIT"` | SPDX identifier. |
| `keywords` | string[] | — | `[]` | Used for `atl search`. |
| `repository` | string | — | — | Git URL. |
| `homepage` | string | — | — | Docs / landing URL. |
| `agents` | object[] | — | `[]` | Each: `{ name: string, description: string }`. |
| `skills` | object[] | — | `[]` | Same shape as `agents`. |
| `rules` | object[] | — | `[]` | Same shape as `agents`. |
| `extends` | string | — | — | `"team-name"` or `"team-name@constraint"`. |
| `excludes` | string[] | — | `[]` | Parent item names to drop. |
| `dependencies` | object | — | `{}` | Map `"team-name"` → `"version-constraint"`. |
| `requires` | object | — | `{}` | `{ atl: string }` — minimum CLI version. |

## Using the schema in CI

Any JSON Schema validator works. The [Creating a team](/authoring/creating-a-team) page shows a full `ajv-cli` GitHub Actions workflow. The short version:

```bash
npm install -g ajv-cli
curl -sSfL https://raw.githubusercontent.com/agentteamland/core/main/schemas/team.schema.json -o team.schema.json
ajv -s team.schema.json -d team.json --strict=false
```

## Editor integration

If your editor supports JSON Schema via `$schema`, add this at the top of `team.json`:

```json
{
  "$schema": "https://raw.githubusercontent.com/agentteamland/core/main/schemas/team.schema.json",
  "schemaVersion": 1,
  ...
}
```

VS Code, JetBrains, and most JSON editors will give you autocomplete and inline validation.

## Versioning the schema

- **Additive changes** (new optional field) → no version bump.
- **Breaking changes** (remove field, change type, tighten required set) → `schemaVersion` bumps to `2`. The CLI continues to accept older `schemaVersion` values for a deprecation window (at least two minor CLI versions).

Current schema is `schemaVersion: 1`.

## Related

- **[team.json](/authoring/team-json)** — human-oriented reference with examples.
- **[Glossary](./glossary)** — terms used across the schema.
