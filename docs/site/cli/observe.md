# `atl observe`

The **deterministic half** of the proactive observer: report whether a sweep is due, and stamp the cursor the [`/observe`](/skills/observe) skill sets after it runs. The audit itself — ripe backlog triggers plus the grep-grounded, adversarially-verified latent-gap sweep — is the skill's LLM work. This is the CLI/Skill boundary: the CLI is deterministic, the skill is the judgment.

Runs in any project with an ATL decision surface (a `.atl/` directory). Outside one it does nothing and exits 0.

## Usage

```bash
atl observe            # report whether a proactive observer sweep is due
atl observe --record   # stamp HEAD as the last observer sweep (after an /observe run)
```

## What "due" means

A sweep is **due** when the project's `.atl/` decision + knowledge surface has moved since the last recorded sweep — a decision, a shipped unit, a journal entry — gated by a **~1-day runaway-guard** so it never re-signals more than once a day. That cadence fires the observer right when shipped-vs-designed drift is most likely: just after work lands.

At session start, `atl session-start` prints **"a proactive observer sweep is due — run /observe"** on this condition, so you don't have to remember to check. The signal is cheap (a git-log cadence check); the expensive LLM sweep only runs if you invoke `/observe` in response.

`--record` stamps HEAD + the time as the last sweep (in `~/.atl/observe-state.json`), which resets that runaway-guard for ~1 day. `/observe` calls it at the end of a sweep.

## Related

- [`/observe`](/skills/observe) — the LLM half: watch ripe backlog triggers and sweep for latent gaps (shipped-vs-designed, growth/scale risks, unshipped decisions, your setup), grep-grounded and adversarially verified.
- [`atl docs`](/cli/docs) / [`atl rules`](/cli/rules) — the sibling deterministic-CLI + LLM-skill backstops, on the same cursor mechanism.
