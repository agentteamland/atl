# `atl pin` / `atl unpin`

Keep a project-local customization project-only. A **pin** marks a path under this project's `.claude` dir so [`atl promote`](/cli/promote) never lifts it — or its subtree — to the global layer.

A pin is a *declarative* opt-out, like a `.gitignore` entry: promotion still runs automatically: the pin only scopes it. `atl pin` records the exclusion; `atl unpin` clears it.

## When to use it

By default, a gain you make in one project (a tweaked agent, a new skill, a house-style rule) is eligible to be promoted to your global layer so it circulates to your other projects. Pin a path when that's *not* what you want — when the customization is deliberately specific to this one project and shouldn't leak elsewhere.

A pin does **not** touch fan-out. Fan-out already preserves any file you've changed locally, so a pinned divergence stays put on the receiving side regardless. Pin only stops the *upward* lift into global.

## Usage

```bash
atl pin [path]        # add a pin, or list pins when no path is given
atl unpin <path>      # remove a pin
```

`path` is relative to this project's `.claude` directory and slash-separated. It names either a single file or a subtree — point it at an agent/skill/rule unit to pin the whole thing:

```bash
atl pin agents/api-agent       # pins the whole api-agent subtree
atl pin rules/house-style.md   # pins a single rule file
```

Paths are normalized (leading `./`, surrounding slashes, and `.` / `..` segments are cleaned), so `./rules/house-style.md/` and `rules/house-style.md` record the same pin. A subtree pin covers every file nested under it — pinning `agents/api-agent` exempts `agents/api-agent/agent.md`, its `children/`, its `learnings/`, and so on.

Pins live in `<project>/.atl/pins.json` — one file per project, written atomically with the list kept sorted. A missing file just means "no pins."

`atl pin` and `atl unpin` are **project-scoped**: they always operate on the current working directory's project layer.

## Examples

List the current pins (no argument):

```bash
atl pin
```

```
atl pin — project-only paths (never promoted):
  agents/api-agent
  rules/house-style.md
```

With nothing pinned:

```
atl pin: no pins — every gain promotes to global
```

Pin a project-specific rule, then later allow it to circulate again:

```bash
atl pin rules/house-style.md
```

```
atl pin: rules/house-style.md is now project-only (won't be promoted)
```

```bash
atl unpin rules/house-style.md
```

```
atl unpin: rules/house-style.md will be promoted again
```

Re-pinning an existing pin, or unpinning something that was never pinned, is a no-op and says so (`… is already pinned` / `… was not pinned`); neither is an error.

## Notes

- **No flags.** Both commands take only the path argument: `atl pin` accepts zero or one, `atl unpin` requires exactly one.
- The pin key is the `.claude`-relative path, not an absolute or repo-root path. A path that resolves to a file `atl promote` would otherwise lift is what makes the pin effective.

## Related

- [`atl promote`](/cli/promote) — lifts project gains to global; honors pins.
- [`atl list`](/cli/list) — what's installed in this project.
