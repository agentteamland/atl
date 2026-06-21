# `atl promote`

Lift your project's local gains into the user-global layer — ring 1→2 of gain circulation, the upward mirror of [`atl update`](/cli/update)'s fan-out.

## When to use it

You almost never run this by hand. `atl promote` runs **automatically** in the background [`atl tick`](/cli/tick), so gains an agent accumulates in one project flow up to your global layer on their own, and your *other* projects fan them out on their next tick. The ring closes without you thinking about it.

The manual command exists for when you want to lift right now — after a session where an agent grew a lot, or to confirm what would be lifted.

## What it lifts

When you install a team, ATL copies its agents, skills, and rules into the project. As you work, the [learning loop](/cli/learnings) grows those assets past their install baseline — overwhelmingly an agent's `children/` and its rebuilt knowledge base. Fan-out pulls global→project, refreshing files the project never touched while preserving the ones it did. **Promote does the inverse:** it lifts the files the project *did* evolve up into the global copy of the same team.

It's narrow and safe by design:

- **Only teams installed at both scopes.** There must be a global copy to lift into — a team installed only in the project is left alone (`atl promote` skips it silently).
- **Additive.** Files the project changed past their baseline, plus brand-new files under the team's own units (e.g. a freshly grown `children/` entry), are copied up. After a lift, both the project and global baselines advance to the promoted version, so the same gain is never lifted twice.
- **Conflict-safe.** If the global layer *also* changed a file independently, that's a true conflict: the project value wins, and the prior global value is archived under `~/.atl/history` first (content-addressed, reversible).
- **Pin-aware.** Files you [`atl pin`](/cli/pin) are kept project-only and never lifted.

A successful pass bumps the global generation counter, which is what tells your other projects to fan the gains out on their next tick.

## Usage

```bash
atl promote [handle/team]
```

With no argument, promote walks every team in the current project and lifts each one's eligible gains. Pass an optional `handle/team` reference (e.g. `agentteamland/software-project-team`) to restrict the pass to that single team.

`atl promote` takes no flags.

## Examples

Promote every eligible team's gains from the current project:

```bash
atl promote
```

```
atl promote: lifted 3 file(s) to the global layer
```

Restrict the lift to one team:

```bash
atl promote agentteamland/software-project-team
```

When a file the global layer also moved gets lifted, the conflict is reported and the prior value is archived:

```
atl promote: lifted 2 file(s) to the global layer (1 conflict(s) — project won, prior global archived to ~/.atl/history)
```

When the global layer is already current, promote is a quiet no-op:

```
atl promote: nothing to lift — the global layer is already current
```

## Related

- [`atl tick`](/cli/tick) — the in-session cadence that runs promote automatically.
- [`atl update`](/cli/update) — fan-out, the downward mirror (global→project).
- [`atl pin`](/cli/pin) — keep a file project-only so promote never lifts it.
