# Release pipeline (goreleaser → GitHub Releases)

How an `atl` release gets from a git tag in the [`agentteamland/atl`](https://github.com/agentteamland/atl) monorepo to a ready-to-install binary on every supported platform.

This page is for **maintainers**. If you only want to install `atl`, see [Install](../guide/install).

## The pipeline at a glance

```
[tag pushed in the atl monorepo]      git tag v2.0.0 && git push origin v2.0.0
        ↓
[GitHub Actions workflow fires]       .github/workflows/release.yml
        ↓
[go test ./...]                       the cli module is tested before anything ships
        ↓
[goreleaser builds 6 binaries]        linux/amd64,   linux/arm64,
                                      darwin/amd64,  darwin/arm64,
                                      windows/amd64, windows/arm64
        ↓
[goreleaser publishes ONE channel]
   └── GitHub Release
         ├── per-platform archives (.tar.gz; .zip on Windows)
         ├── a checksums file (atl_<version>_checksums.txt)
         └── an auto-generated changelog grouped by commit type
```

v2 distributes through **GitHub Releases only**. There is no Homebrew, Scoop, or winget channel — those were retired in the v2 rebuild. Users install with the one-liner scripts (`install.sh` / `install.ps1`), which download the right archive straight from the latest GitHub Release.

## How it's versioned

The version is baked into the binary at build time via `cli/internal/buildinfo`:

```go
// cli/internal/buildinfo/buildinfo.go
var (
	Version = "dev" // ldflags-overridden at release build time
	Commit  = ""
	Date    = ""
)
```

`dev` is the working-tree default. goreleaser overrides all three via ldflags using the git tag, commit, and build date:

```
-X github.com/agentteamland/atl/cli/internal/buildinfo.Version={{.Version}}
-X github.com/agentteamland/atl/cli/internal/buildinfo.Commit={{.Commit}}
-X github.com/agentteamland/atl/cli/internal/buildinfo.Date={{.Date}}
```

So `atl --version` prints the tag the build was cut from.

## Tag → release flow

After merging a release-worthy PR to `main`:

```bash
cd repos/atl
git checkout main && git pull
git tag v2.0.0          # the version to release
git push origin v2.0.0  # triggers .github/workflows/release.yml
```

The tag push:

1. Triggers the `release` workflow on a `v*` tag (`.github/workflows/release.yml`).
2. Runs `go test ./...` in `cli/` — a failing test blocks the release.
3. Runs `goreleaser release --clean`, which cross-compiles the 6 binaries.
4. Publishes a GitHub Release with the archives, a checksums file, and a changelog auto-generated from commit titles (Features / Bug fixes / Documentation / Others).

Within a minute or two of the tag push, the new version is the latest GitHub Release — and the install scripts resolve to it automatically.

## The single channel: GitHub Releases

[`.goreleaser.yaml`](https://github.com/agentteamland/atl/blob/main/.goreleaser.yaml) defines one `builds` entry (the cli module, `dir: cli`, `main: ./cmd/atl`) and one `archives` entry:

- **Archives** — `atl_<version>_<os>_<arch>.tar.gz` for linux/darwin, `.zip` for windows. Each archive bundles the `atl` binary plus `README.md` + `LICENSE`.
- **Checksums** — a single `atl_<version>_checksums.txt` covering every archive, for verification.
- **Changelog** — generated from the GitHub commit history, grouped by conventional-commit type.

The release `header` embeds the install one-liners, so the GitHub Release page itself shows users how to install.

## How users install from it

The install scripts at [`scripts/install.sh`](https://github.com/agentteamland/atl/blob/main/scripts/install.sh) (macOS/Linux) and [`scripts/install.ps1`](https://github.com/agentteamland/atl/blob/main/scripts/install.ps1) (Windows):

1. Resolve the latest release tag via the GitHub API (or honor a pinned `ATL_VERSION`).
2. Detect OS + arch and build the archive name (`atl_<version>_<os>_<arch>.tar.gz`).
3. Download that archive from the release, extract `atl`, and drop it on the user's `PATH` (`ATL_INSTALL_DIR`, default `/usr/local/bin`).

No package manager, no tap, no central catalog — the release artifact is the source of truth. See [Install](../guide/install) for the user-facing instructions.

## Why a single channel

v1 fanned out to Homebrew, Scoop, and winget. Each added maintenance cost — winget in particular required a manual PR to `microsoft/winget-pkgs` per release, with a Microsoft review queue and fork-master discipline to police. The v2 rebuild collapsed distribution to the install scripts + GitHub Releases: one goreleaser config, one tag, one artifact set, zero per-release manual steps. goreleaser stays as the cross-compile + publish orchestrator; only the downstream package-manager pushes were dropped.

## Related

- [Install `atl`](../guide/install) — the user-facing install instructions.
- The monorepo's [`.goreleaser.yaml`](https://github.com/agentteamland/atl/blob/main/.goreleaser.yaml) — the goreleaser config that drives this pipeline.
- [`.github/workflows/release.yml`](https://github.com/agentteamland/atl/blob/main/.github/workflows/release.yml) — the tag-triggered workflow.
