---
name: dst-open
description: "/dst-open — open .dst/index.html in the user's default browser. Cross-platform (macOS open / Linux xdg-open / Windows start). No-op if .dst/ doesn't exist (suggest /dst-init first)."
---

# /dst-open Skill

## Purpose

Open the design studio in the user's default browser. Convenience wrapper around the OS-native "open URL" command.

## Flow

1. **Check `.dst/index.html` exists.** If not → print "No `.dst/` in this project. Run `/dst-init` first." and stop.

2. **Detect OS and pick command:**
   - macOS (`darwin`): `open .dst/index.html`
   - Linux: `xdg-open .dst/index.html`
   - Windows: `start .dst/index.html`

3. **Execute via Bash tool.** Don't wait for browser to close (it's interactive).

4. **Print confirmation:**
   ```
   Opened .dst/index.html in your browser.
   ```

## Edge Cases

- **No graphical environment** (e.g., remote SSH session) — `xdg-open` may fail silently. If exit code != 0, print: "Couldn't open browser automatically. Manual: file://<absolute-path-to-.dst/index.html>"
- **Specific file requested** — future arg `--ds <name>` could open `.dst/design-systems/<name>/detail.html` directly. Not in v0.1.0.

## What this skill does NOT do

- Doesn't refresh open browser tabs (browser handles that on F5)
- Doesn't re-render any HTML (that's other skills' jobs)
- Doesn't start a server (we ship static files only)

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
