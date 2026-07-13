# atl — contributor orientation

The AgentTeamLand v2 monorepo: the `atl` CLI (Go), the platform core (rules + skills), the first-party teams, and the docs site — one repo.

## Layout

| Path | What |
|---|---|
| `cli/` | the `atl` binary (Go) — the deterministic plumbing layer. Build/test: `cd cli && go build ./... && go vet ./... && go test ./...` |
| `core/` | global rules + skills, embedded in the binary (`cli/internal/coreassets`) and reflected to `~/.claude` on install / update / session-start |
| `teams/` | first-party teams — see `teams/*/team.json` (profile-team, personal-advisory-team, delivery-team) |
| `docs/site/` | the VitePress docs site (EN canonical + a `tr/` mirror) |
| `.atl/` | the v2 decision doc (`docs/atl-v2.md`); the full brainstorm + wiki knowledge base lives in the `agentteamland/workspace` maintainer repo |

## Conventions

- **Commits:** `type(scope): message` — `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`.
- **Every PR is gated by CI:** Go build-test + `scan-non-english.sh` (committed files are English; the `/tr/` mirror is the only exception) + the deterministic **docs-drift gate** (`atl docs check` + the VitePress build, which fails on dead links).
- **Releases** are a `vX.Y.Z` tag → `release.yml` → goreleaser → GitHub Releases (the install scripts pull from there). No brew/scoop/winget.
- **Editing `core/`?** Re-sync the embedded copy: `bash cli/internal/coreassets/sync.sh`, and commit the `embed/` changes in the same commit — `TestEmbedMatchesCore` fails otherwise.

## Docs-correctness (read before changing behavior)

The docs site is the one surface kept in sync with the code; every other repo's README is a redirect-stub by design. **A doc-affecting change** — a CLI command or flag, a skill / rule / agent, a concept, the install flow — **ships its docs-site update in the same PR.**

- `/create-pr` does this automatically: its docs-impact pass runs `atl docs check` (deterministic) plus a grep-grounded semantic pass, and the doc edits ride the same PR.
- Shipping outside `/create-pr`? Run `atl docs check` and update the affected pages by hand (the EN page **and** its `tr/` mirror).

The CI docs gate catches the mechanical half (coverage, EN↔TR parity, stale install instructions, dead links); the semantic half — does the prose still match the code? — is on you, with the `/docs-audit` skill as the full-site backstop (it auto-signals at session-start when a sweep is due). This is docs-sync v2; the deterministic checks live in `cli/internal/docscheck`.
