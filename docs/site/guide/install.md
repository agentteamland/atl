# Install

`atl` ships as a single static Go binary (~7 MB, zero runtime dependencies). One script per platform — no package manager to set up first.

---

## macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

That's the whole thing. The script:

- Resolves the latest release from GitHub
- Downloads the matching `atl` binary for your OS + arch (`darwin`/`linux`, `amd64`/`arm64`)
- Extracts it and moves it to `/usr/local/bin/atl` (prompts for `sudo` only if that directory isn't writable)

No-sudo install — point it at a directory you own:

```bash
ATL_INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

Upgrading: rerun the same command — it always pulls the latest release. (You rarely need to: once hooks are set up, `atl` keeps itself and your teams current in the background. See [auto-update hooks](#recommended-next-step-auto-update-hooks).)

Pinning a specific version:

```bash
ATL_VERSION=v2.0.0 curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

---

## Windows — PowerShell

```powershell
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Open PowerShell, paste, Enter. The script:

- Downloads the latest `atl.exe` from GitHub Releases (`amd64` or `arm64`)
- Installs it to `%LOCALAPPDATA%\Programs\atl\` (no admin needed)
- Adds that folder to your **user PATH**
- Verifies the install by running `atl --version`

Zero admin rights, zero package-manager prerequisites, works on a fresh Windows machine.

Upgrading: rerun the same command. It always pulls the latest release.

Custom install directory:

```powershell
$env:ATL_INSTALL_DIR = 'C:\Users\<you>\bin'
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Pinning a specific version:

```powershell
$env:ATL_VERSION = 'v2.0.0'
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

---

## Manual download (any platform)

For locked-down machines (or if you'd rather not pipe a script), grab a pre-built binary straight from [**GitHub Releases**](https://github.com/agentteamland/atl/releases/latest). Artifacts ship for:

- `darwin` (macOS): `amd64`, `arm64`
- `linux`: `amd64`, `arm64`
- `windows`: `amd64`, `arm64`

Extract the archive (`.tar.gz` on macOS/Linux, `.zip` on Windows), drop `atl` somewhere on your `PATH`, and you're done. On Windows, add the folder to your user PATH via **Settings → System → About → Advanced system settings → Environment Variables → Path**, then open a fresh terminal.

::: tip No brew / scoop / winget
`atl` v2 distributes through the install scripts and GitHub Releases only. The Homebrew, Scoop, and winget channels were retired in the v2 rebuild — the one-liner skips the package-manager setup entirely, and there's no third-party tap to keep in sync.
:::

---

## Verify

```bash
atl --version
atl --help
```

You should see the installed version and the command list: `install`, `update`, `promote`, `pin`, `unpin`, `publish`, `learnings`, `tick`, `session-start`, `setup-hooks`, `doctor`, `list`, `search`, `remove`.

## What got installed

A single binary. `atl` keeps its own state — the index cache, the durable learning queue, throttle stamps, and your global-layer gains — under:

- macOS / Linux: `~/.atl/`
- Windows: `%USERPROFILE%\.atl\`

Team assets (agents, skills, rules) are copied into Claude Code's own directory, where the editor picks them up:

- **Global layer:** `~/.claude/`
- **Project layer:** `<project>/.claude/` (this shadows the global layer for the current project)

So `.atl/` is ATL's operational store and `.claude/` is where the agents/skills/rules actually live for Claude Code to load.

## Recommended next step — auto-update hooks

Once `atl` is on your PATH, run:

```bash
atl setup-hooks
```

This wires Claude Code's hooks so the platform runs itself in the background:

- **`SessionStart` → `atl session-start`** — drains the previous session's learnings and runs `doctor` to self-heal.
- **`UserPromptSubmit` → `atl tick --throttle=10m`** — an in-session maintenance tick (throttled), so updates, fan-out, and learning capture happen without you lifting a finger.

In v2 this is meant to be on, not opt-in — automation is the point. You don't run `atl update` by hand; your teams and `atl` itself stay current automatically. See [`atl setup-hooks`](/cli/setup-hooks) for details.

## Next

- **[Quickstart](/guide/quickstart)** — install your first team.
- **[Concepts](/guide/concepts)** — teams, agents, skills, rules, and the global/project scope axis.
- **[CLI reference](/cli/overview)** — every command in detail.
- **[Creating a team](/authoring/creating-a-team)** — author and publish your own team.
