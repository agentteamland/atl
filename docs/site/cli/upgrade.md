# `atl upgrade`

Update the `atl` binary itself to the latest stable release: resolve the newest published release, and if it is newer than the running build, download it, verify its checksum, and atomically replace this binary in place.

`atl upgrade` is the **manual** surface for keeping the binary current. You rarely need it — once the [hooks](/cli/setup-hooks) are set up, `atl session-start` runs the same check automatically (see [Automatic upgrades](#automatic-upgrades)). Reach for it to force an immediate update.

::: tip Binary vs. teams
`atl upgrade` updates the **binary**. [`atl update`](/cli/update) updates your installed **teams** and reflects the platform core into `~/.claude`. They are separate surfaces: the binary is a released artifact, teams are content.
:::

## Usage

```bash
atl upgrade
```

It takes no arguments and no flags.

## What it does

1. **Resolve the latest stable release.** Queries GitHub for the newest stable release (prereleases are excluded).
2. **Compare — only upgrade, never downgrade.** If the running build is not strictly older than the latest release, it is a no-op (`already up to date`). A `dev` build (an un-stamped local `go build`) is left untouched.
3. **Download + verify.** Fetches the release asset for your OS/architecture and its published `checksums.txt`, and verifies the download's SHA-256 **before** touching the install directory.
4. **Atomically replace the running binary.** Writes the new binary beside the current one and renames it into place, so an interrupted update never leaves a half-written binary. The running process keeps executing the old copy; the next invocation is the new version.

## Automatic upgrades

With the hooks installed, `atl session-start` performs this same check on your behalf — throttled to at most **once every 24 hours**. When a newer release exists, it starts the download and swap **in the background** (it never blocks the session) and prints a one-line notice; the new binary is active from the **next** session. This is automatic and mandatory — there is no per-project opt-in.

## Disabling it

Set `ATL_NO_SELF_UPDATE` (to any value) to disable self-update entirely — both the manual command and the automatic session-start check become no-ops. This is an emergency brake (for example, if a release is misbehaving); the feature is otherwise always on.

```bash
ATL_NO_SELF_UPDATE=1 atl upgrade   # → disabled, does nothing
```

## Platform notes

- **macOS / Linux** — full support (in-place atomic replace).
- **Windows** — a running `.exe` can't overwrite itself, so `atl upgrade` reports the newer version and asks you to rerun the [install script](/guide/install) instead; the automatic check notifies rather than swapping.
- **Permissions** — if the install directory isn't writable (for example a system path installed with `sudo`), the upgrade reports a clear error rather than silently escalating. Reinstall with the [install script](/guide/install), or point it at a user-writable directory with `ATL_INSTALL_DIR`.

## See also

- [`atl update`](/cli/update) — refresh installed **teams** (not the binary).
- [`atl setup-hooks`](/cli/setup-hooks) — wire the hooks that run the automatic check.
- [Installation](/guide/install) — the install script (the other way to upgrade the binary).
