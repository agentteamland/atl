# Schema

There is no separate machine-readable JSON Schema file in ATL v2.

In v1, `team.json` was validated against a standalone `team.schema.json` (JSON Schema Draft 2020-12) checked in CI. v2 dropped that file. The `team.json` contract is now documented for humans, and the CLI enforces it minimally at install time.

## The contract lives at one place

**[`team.json`](/authoring/team-json)** is the full field reference — every field, its type, whether it's required, and what it means, with examples.

## What the CLI enforces

When you run `atl install`, the CLI does not run a JSON Schema validator. It checks three things:

- `team.json` parses as valid JSON.
- It has a `name`.
- Every asset declared under `agents[]`, `skills[]`, and `rules[]` exists on disk at its expected path.

If any of those fail, the install stops with an error. Anything else (extra fields, formatting) is ignored.

## Related

- **[team.json](/authoring/team-json)** — the field reference and examples.
- **[Glossary](./glossary)** — terms used across ATL.
