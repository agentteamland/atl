# winget upstream-PR process

How AgentTeamLand publishes new `atl` versions to the official Windows Package Manager catalog at [`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs).

This page is for **maintainers**. End users install via `winget install AgentTeamLand.atl` once a release lands in the upstream catalog.

## Why this is manual

The other two distribution channels (Homebrew + Scoop) are fully automated by goreleaser — every git tag in [`agentteamland/cli`](https://github.com/agentteamland/cli) auto-bumps the formula / manifest in our org's tap + bucket repos within ~5 minutes.

winget is different. Microsoft's official catalog is the **only** place winget reads from — there's no "add a custom bucket" mechanism like Scoop has. To publish to the catalog, you open a PR from your fork to `microsoft/winget-pkgs:master`, Microsoft's validation pipeline runs, a Microsoft reviewer merges. **There's no automation that bypasses this gate.**

So our pipeline has two phases:

1. **Auto** — goreleaser pushes new manifests to OUR fork at [`agentteamland/winget-pkgs`](https://github.com/agentteamland/winget-pkgs)
2. **Manual** — open a PR from our fork to `microsoft/winget-pkgs:master`

This page covers phase 2.

## What goreleaser puts in our fork

For each tagged `atl` release, goreleaser adds a directory under `manifests/a/AgentTeamLand/atl/<version>/` to our fork's master branch. The directory contains three YAML files per the winget manifest schema:

```
manifests/a/AgentTeamLand/atl/1.1.4/
├── AgentTeamLand.atl.yaml              # version manifest
├── AgentTeamLand.atl.installer.yaml    # installer metadata + URLs + SHA-256
└── AgentTeamLand.atl.locale.en-US.yaml # description, license, package name (en-US)
```

These files describe: which version, where to download the installer (points at the cli GitHub Release artifacts), what the SHA-256 is, license info, package name.

Goreleaser commits + force-pushes these into our fork's master branch automatically on every tag. **At this point, the upstream catalog still doesn't know about the version.**

## Opening the upstream PR

```bash
cd repos/winget-pkgs   # the workspace's clone of OUR fork

# Make sure our fork's master is current with what goreleaser pushed:
git checkout master
git pull origin master

# Create a PR branch:
git checkout -b atl-<version>
# (no edit needed — goreleaser already committed the manifests on master)

# Push the branch:
git push -u origin atl-<version>

# Open the PR to upstream microsoft/winget-pkgs:
gh pr create \
  --repo microsoft/winget-pkgs \
  --base master \
  --head agentteamland:atl-<version> \
  --title "New version: AgentTeamLand.atl version <X.Y.Z>" \
  --body "Adds manifests for AgentTeamLand.atl <X.Y.Z>"
```

Microsoft's validation pipeline kicks in immediately. Common checks:

- Manifest schema validation
- Hash verification (re-downloads the installer URL, computes SHA-256, compares)
- Installer format check (must be MSIX, MSI, APPX, MSIXBundle, APPXBundle, or .exe)
- Reachability of all URLs
- License compatibility
- Package metadata sanity

If validation passes, a Microsoft reviewer takes a look (typically within 24–72h). They merge → the version becomes available via `winget install AgentTeamLand.atl` once Microsoft's CDN propagates (usually within minutes of merge).

## What if validation fails

The validation bot leaves a comment on the PR with the failure reason. Common ones:

- **Hash mismatch** — re-check that the SHA-256 in the installer manifest matches what goreleaser computed. Usually means goreleaser uploaded after our fork's manifest was committed; pull our fork's master fresh, re-PR.
- **Unreachable URL** — check the cli GitHub Release exists with the expected filename. Sometimes goreleaser has uploaded the manifest before all artifacts finished uploading.
- **MSIX/EXE rejection** — `.exe` is supported but the installer must satisfy [winget's installer requirements](https://github.com/microsoft/winget-pkgs/blob/master/doc/manifest/schema/1.6.0/singleton.md). For `atl` we ship a portable `.exe` that goes through validation cleanly; if a future installer format changes this, validation will catch it.

If a fix is needed, edit the manifests in our fork's branch, push, the bot re-validates automatically.

## Lag between cli release and winget availability

Typical: 1–3 days from cli tag to winget catalog visibility.

Detailed timeline:

1. **t = 0**: tag pushed to cli repo
2. **t + 5min**: goreleaser finishes; manifests in our fork's master branch; brew + scoop already updated
3. **t + 10min**: maintainer opens upstream PR (manual step)
4. **t + 10min to 24h**: Microsoft validation pipeline runs
5. **t + 24h to 72h**: Microsoft reviewer merges (sometimes faster, sometimes slower depending on queue)
6. **t + ~72h**: `winget upgrade atl` pulls the new version

This lag is **expected and documented** on the [Install page](../guide/install) — the docs explicitly note "winget may lag one or two `v*` tags behind." Users who need the absolute latest are pointed at the PowerShell one-liner or scoop.

## History — past upstream PRs

| atl version | PR | Status | Notes |
|---|---|---|---|
| v0.1.1 | [#361975](https://github.com/microsoft/winget-pkgs/pull/361975) | merged 2026-04-24 | First release in catalog |
| v0.2.0 | [#364841](https://github.com/microsoft/winget-pkgs/pull/364841) | merged 2026-04-25 | Second release |
| v0.3.0 / v1.0.0 / v1.1.0 / v1.1.1 / v1.1.2 / v1.1.3 | _(not separately back-PR'd)_ | preserved in fork | Per winget convention, intermediate versions are kept in the fork's branches but not separately PR'd to upstream — the next "real" upstream PR carries them implicitly |
| v1.1.4 | [#367931](https://github.com/microsoft/winget-pkgs/pull/367931) | pending review | Catches upstream catalog up to current shipping version |

## Related

- [Release pipeline](release-pipeline) — the full goreleaser flow (brew + scoop + winget)
- [Install `atl`](../guide/install) — user-facing instructions, including the "winget may lag" note
- [Microsoft's winget-pkgs README](https://github.com/microsoft/winget-pkgs) — upstream rules and format
- [winget-cli releases](https://github.com/microsoft/winget-cli/releases) — the client side that pulls from the catalog
