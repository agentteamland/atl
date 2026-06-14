# Install

`atl` ships as a single static Go binary (~7 MB, zero runtime dependencies). Pick the channel that matches your platform.

---

## macOS / Linux — Homebrew (recommended)

```bash
brew install agentteamland/tap/atl
```

Equivalent two-step form:

```bash
brew tap agentteamland/tap
brew install atl
```

Upgrading:

```bash
brew update && brew upgrade atl
```

### macOS / Linux — one-liner fallback

If you don't have brew (or prefer a stdlib install):

```bash
curl -fsSL https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.sh | sh
```

Downloads the latest `atl` binary, extracts it, and moves it to `/usr/local/bin/atl` (prompts for sudo if needed). Override with `ATL_INSTALL_DIR=$HOME/.local/bin` for a no-sudo install.

---

## Windows — PowerShell one-liner (recommended)

```powershell
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

That's the whole thing. Open PowerShell, paste, Enter. The script:

- Downloads the latest `atl.exe` from GitHub Releases
- Installs it to `%LOCALAPPDATA%\Programs\atl\` (no admin needed)
- Adds that folder to your **user PATH**
- Verifies the install by running `atl --version`

Zero admin rights, zero package-manager prerequisites, works on a fresh Windows machine.

::: tip Why this over scoop/winget?
If you're comfortable with a package manager, scoop and winget below are fine. But if you don't already have scoop or winget configured (and many devs doing "vibe coding" don't), the one-liner skips that whole rabbit hole.
:::

Upgrading: rerun the same command. It always pulls the latest release.

Pinning a specific version:

```powershell
$env:ATL_VERSION = 'v1.1.4'
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

### Windows — Scoop

For users who already have scoop installed:

```powershell
scoop bucket add agentteamland https://github.com/agentteamland/scoop-bucket
scoop install atl
```

Upgrading:

```powershell
scoop update atl
```

Don't have scoop yet? Use the PowerShell one-liner above — simpler and faster than installing scoop first.

### Windows — winget

```powershell
winget install agentteamland.atl
```

::: tip May lag the latest release
`atl` has been in the winget catalog since 2026-04-24. Each new release goes through a manual review step on `microsoft/winget-pkgs`, so winget may lag one or two `v*` tags behind. If you need the absolute latest, use the PowerShell one-liner or scoop above.
:::

### Windows — Manual ZIP

Most hands-on way, zero automation, works anywhere:

1. Open [**GitHub Releases**](https://github.com/agentteamland/cli/releases/latest) → download `atl_<version>_windows_amd64.zip` (or `arm64`).
2. Extract the ZIP to a folder you control, e.g. `C:\Users\<you>\tools\atl\`.
3. Add that folder to your user PATH: **Settings → System → About → Advanced system settings → Environment Variables → Path → Edit → New**, paste the folder path, OK.
4. Open a new PowerShell / cmd window → `atl --version` → should print the version.

Good for locked-down machines where PowerShell scripts are restricted.

---

## Manual download (any platform)

Grab a pre-built binary from [**GitHub Releases**](https://github.com/agentteamland/cli/releases/latest). Artifacts ship for:

- `darwin` (macOS): `amd64`, `arm64`
- `linux`: `amd64`, `arm64`
- `windows`: `amd64`, `arm64`

Extract, drop `atl` somewhere on your `PATH`, and you're done.

---

## Verify

```bash
atl --version
atl --help
```

You should see the installed version and the command list (`install`, `list`, `remove`, `update`, `search`, `setup-hooks`).

## What got installed

A single binary. `atl` keeps its team cache under:

- macOS / Linux: `~/.claude/repos/agentteamland/`
- Windows: `%USERPROFILE%\.claude\repos\agentteamland\`

Installed teams live as cloned Git repos under that cache and are copied into each project's `.claude/` directory.

## Recommended next step — auto-update hooks

Once `atl` is on your PATH, run:

```bash
atl setup-hooks
```

This wires Claude Code's SessionStart + UserPromptSubmit hooks so that every session (and every message, throttled to once per 30m) silently checks for team/core/`atl` updates in the background. You don't have to run `atl update` manually — your teams stay fresh automatically. See [`atl setup-hooks`](/cli/setup-hooks) for details.

## Next

- **[Quickstart](/guide/quickstart)** — install your first team.
- **[Concepts](/guide/concepts)** — teams, agents, skills, rules, inheritance.
- **[Creating a team](/authoring/creating-a-team)** — author your own team (public, private, or local).
