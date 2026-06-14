# Registry submission

Listing your team in the public registry gives it a short name (`atl install your-team` instead of `atl install https://github.com/…`). The registry is a single JSON file maintained by PR.

## What the registry is

[`agentteamland/registry`](https://github.com/agentteamland/registry) holds one file: `teams.json`. Every entry maps a short name to a Git URL plus metadata:

```json
{
  "teams": [
    {
      "name": "software-project-team",
      "url": "https://github.com/agentteamland/software-project-team",
      "status": "verified",
      "description": ".NET 9 API + Flutter + React + full Docker stack",
      "keywords": ["dotnet", "flutter", "react", "docker"]
    }
  ]
}
```

## Before you submit

Your team should be:

1. **Installable from its Git URL** — `atl install https://github.com/you/your-team.git` works end to end.
2. **Schema-valid** — `team.json` passes validation. Every repo already gets this check in CI if you followed [Creating a team](./creating-a-team).
3. **Documented** — the repo's `README.md` tells a user what this team is for and how to use it.
4. **Tagged** — at least one SemVer tag (`v0.1.0` or higher).

## Steps

1. **Fork** `agentteamland/registry`.

2. **Add your entry** to `teams.json` in the `teams` array. Keep the array alphabetized by `name`.

   ```json
   {
     "name": "your-team",
     "url": "https://github.com/you/your-team",
     "status": "community",
     "description": "One-sentence pitch. Shows up in atl search.",
     "keywords": ["what", "your", "team", "covers"]
   }
   ```

   Fields:
   - `name` — must match the `name` in your `team.json`.
   - `url` — the Git HTTPS URL (no trailing `.git` required).
   - `status` — new submissions start at `"community"`. Maintainers promote to `"verified"` after review.
   - `description` — user-facing one-liner. **10–200 characters** (`description.maxLength = 200` in the schema). Same value as your `team.json` `description`. Going over 200 is the most common reason a registry PR fails CI.
   - `keywords` — for `atl search` matching.

3. **Validate locally before you push.** The registry repo ships a script that runs the same offline checks CI runs:

   ```bash
   npm install -g ajv-cli ajv-formats   # one-time; only if you don't already have ajv
   ./scripts/validate.sh
   ```

   ::: tip Wire it into git push so you can never push an invalid registry
   ```bash
   git config core.hooksPath .githooks   # one-time per clone
   ```
   After this, every `git push` that touches `teams.json` or `schemas/` runs `./scripts/validate.sh` automatically and aborts the push if validation fails. Catches the 200-char `description` overflow locally instead of in a failed PR check.
   :::

4. **Open a PR.** CI will validate:
   - JSON schema conformance (including the `description` 10–200 char range)
   - The `url` returns 200 when fetching `team.json`
   - The fetched `team.json` validates against the team schema
   - `name` uniqueness (no duplicates in the registry)

5. **Wait for review.** Maintainers check that the team exists, installs, and does what it says. Verified status is granted after review.

## Status lifecycle

- **`community`** — listed and installable, not yet reviewed. This is where every submission starts.
- **`verified`** — reviewed by AgentTeamLand maintainers; expected to install cleanly and follow conventions. Maintainers promote via a follow-up PR.
- **`deprecated`** — no longer maintained. Still installable, but users see a warning. Typically set when the author archives the team repo or the team is superseded by a rewrite.

## Removing a team

If you want your team removed from the registry, open a PR deleting your entry. Users who already installed can keep using the Git URL directly.

## Updating an entry

Only metadata changes belong in the registry PR — **not versions**. Version resolution is dynamic: `atl` reads the team's own tags at install/update time. You only need a registry PR if your description, keywords, URL, or status change.

## Questions?

Open an [issue on `agentteamland/registry`](https://github.com/agentteamland/registry/issues).

## Related

- **[Creating a team](./creating-a-team)** — the pre-registry checklist.
- **[team.json](./team-json)** — the schema the registry validates against.
