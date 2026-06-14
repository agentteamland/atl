# Release pipeline (goreleaser → brew + scoop + winget)

How `atl` releases get from a git tag in [`agentteamland/cli`](https://github.com/agentteamland/cli) to a `brew install` / `scoop install` / `winget install` ready-to-use binary on every supported platform.

This page is for **maintainers**. If you only want to install `atl`, see [Install](../guide/install).

## The pipeline at a glance

```
[tag pushed in cli repo]                   git tag v1.1.4 && git push origin v1.1.4
        ↓
[GitHub Actions workflow fires]            .github/workflows/release.yml
        ↓
[goreleaser builds 6 binaries]             darwin/amd64, darwin/arm64,
                                           linux/amd64, linux/arm64,
                                           windows/amd64, windows/arm64
        ↓
[goreleaser publishes per channel]
   ├── GitHub Release (artifacts attached)
   ├── homebrew-tap repo (Formula/atl.rb auto-bumped + force-pushed)
   ├── scoop-bucket repo (atl.json manifest auto-bumped)
   └── winget-pkgs FORK (manifests/a/AgentTeamLand/atl/<version>/ added)
        ↓
[manual upstream PR — winget only]         see /contributing/winget-process
```

Every other channel (brew + scoop) is fully automated. winget requires a manual PR to `microsoft/winget-pkgs` for the version to actually appear in the upstream catalog.

## How it's tagged

Versions are managed by `cli`'s `internal/config/config.go`:

```go
// internal/config/config.go
const Version = "0.1.0-dev"  // ldflags-overridden at build time
```

The `dev` suffix is the working-tree default. Goreleaser overrides via ldflags using the git tag name when it builds:

```bash
go build -ldflags "-X 'github.com/agentteamland/cli/internal/config.Version=v1.1.4'"
```

So `atl --version` prints whatever tag the build was created from.

## Tag → release flow

After merging a PR that's release-worthy:

```bash
cd repos/cli
git checkout main && git pull
git tag v1.1.4         # use the version that's in main's config.go after merge
git push origin v1.1.4 # triggers .github/workflows/release.yml
```

The tag push:

1. Triggers GitHub Actions on `cli`
2. goreleaser cross-compiles 6 binaries
3. Publishes a GitHub Release with the binaries + auto-generated changelog from commit titles
4. Force-pushes Formula updates to [`agentteamland/homebrew-tap`](https://github.com/agentteamland/homebrew-tap)
5. Force-pushes manifest updates to [`agentteamland/scoop-bucket`](https://github.com/agentteamland/scoop-bucket)
6. Adds new manifests to our [winget-pkgs fork](https://github.com/agentteamland/winget-pkgs)

Within ~5 minutes of the tag push, `brew upgrade atl` and `scoop update atl` will pull the new version. The winget channel needs an upstream PR (see below).

## Channel: Homebrew (macOS + Linux)

[`agentteamland/homebrew-tap`](https://github.com/agentteamland/homebrew-tap) holds a single Ruby Formula at `Formula/atl.rb`. Goreleaser replaces the Formula on every tag — no manual edits needed.

Branch protection is intentionally NOT applied here (would block goreleaser's force-push).

The `Formula/atl.rb` carries:
- The current version
- Per-platform binary URLs (point at the cli repo's GitHub Release artifacts)
- Per-platform SHA-256 checksums

A user installing via:

```bash
brew install agentteamland/tap/atl
```

clones the tap repo, reads `Formula/atl.rb`, downloads the binary for their platform from the cli repo's GitHub Release, verifies the checksum, installs to `/opt/homebrew/bin/atl` (or `/usr/local/bin/atl` on Intel macs / Linuxbrew).

> **Brew tap stale formula caveat:** `brew upgrade atl` does NOT auto-refresh the third-party tap clone — only homebrew-core's central index is auto-refreshed. Use `brew update && brew upgrade atl` (or set `HOMEBREW_AUTO_UPDATE_SECS=1` to disable the 24h throttle). See the [brew-tap-stale-formula](https://github.com/agentteamland/workspace/blob/main/.atl/wiki/brew-tap-stale-formula.md) wiki entry for the detailed history of this gotcha.

## Channel: Scoop (Windows)

[`agentteamland/scoop-bucket`](https://github.com/agentteamland/scoop-bucket) holds a single JSON manifest at `bucket/atl.json`. Same auto-update mechanism as Homebrew — goreleaser overwrites on every tag.

A user installing via:

```bash
scoop bucket add agentteamland https://github.com/agentteamland/scoop-bucket
scoop install atl
```

adds the bucket as a remote, downloads `atl.json`, fetches the binary for Windows from the cli GitHub Release, installs to `~/scoop/apps/atl/`.

Branch protection: NOT applied (same reason as Homebrew tap).

## Channel: winget (Windows, official catalog)

[`agentteamland/winget-pkgs`](https://github.com/agentteamland/winget-pkgs) is a **fork** of [`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs). winget unlike the other channels does NOT pull from arbitrary repos — packages must be in Microsoft's official catalog.

The flow:

1. **Goreleaser auto-pushes** new manifests to OUR fork's `manifests/a/AgentTeamLand/atl/<version>/` directory
2. **Manual PR to upstream** is required: open a PR from our fork to `microsoft/winget-pkgs:master`
3. Microsoft's validation pipeline runs (cert check, MSIX/MSI/EXE format check, hash verification, etc.)
4. Microsoft reviewer merges (typically within a few days)
5. Once merged, `winget upgrade atl` pulls the new version

The upstream PR step is detailed in [winget upstream-PR process](winget-process). Until the upstream PR merges, winget users see the previous version — a typical lag of one or two `v*` tags.

The first three winget releases:

- v0.1.1 — [PR #361975](https://github.com/microsoft/winget-pkgs/pull/361975), merged 2026-04-24
- v0.2.0 — [PR #364841](https://github.com/microsoft/winget-pkgs/pull/364841), merged 2026-04-25
- v1.1.4 — [PR #367931](https://github.com/microsoft/winget-pkgs/pull/367931), pending review (intermediate v0.3.0 / v1.0.0 / v1.1.0 / v1.1.1 / v1.1.2 / v1.1.3 manifests preserved in our fork's branches but not separately back-PR'd, per winget convention).

## Why this is set up this way

The three-channel split is **standard for cross-platform CLIs**. Each platform has its own dominant package manager and users expect to install through their familiar tool. Goreleaser is the orchestrator — single config (`.goreleaser.yaml` in the cli repo), one tag triggers all three.

The two automated channels (brew + scoop) keep maintenance cost ~zero. The winget channel needs a manual PR per release because Microsoft requires it — that's the cost of being in the official Windows catalog.

The release-pipeline repos (`homebrew-tap`, `scoop-bucket`, `winget-pkgs`) are intentionally NOT branch-protected. Goreleaser force-pushes; protection would block it. See [Governance](../guide/governance) for which repos are protected and which aren't.

## Related

- [Install `atl`](../guide/install) — the user-facing install instructions for all three channels
- [winget upstream-PR process](winget-process) — the manual step in the winget pipeline
- [Governance](../guide/governance) — branch protection across the org
- The cli repo's [`.goreleaser.yaml`](https://github.com/agentteamland/cli/blob/main/.goreleaser.yaml) — the goreleaser config that orchestrates this whole pipeline
