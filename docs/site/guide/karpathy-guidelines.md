# Karpathy guidelines

Behavioral guidelines loaded into every Claude Code session by the platform. Reduces the common LLM coding mistakes — hidden assumptions, overengineering, drive-by edits, and vague execution.

Adopted into AgentTeamLand `core@1.1.0` on 2026-04-22 as a platform-wide rule, so every existing and future team inherits it automatically (no per-team duplication). Source: [forrestchang/andrej-karpathy-skills](https://github.com/forrestchang/andrej-karpathy-skills) — derived from [Andrej Karpathy's observations](https://x.com/karpathy/status/2015883857489522876) on LLM coding pitfalls. MIT-licensed.

## Why this is a platform-level rule

LLMs left to their own devices tend to:

- **Hide their confusion** instead of asking ("I'll just pick one")
- **Overengineer** ("here's a flexible abstraction in case you need it later")
- **Drive-by edit** adjacent code while making the requested change
- **Execute vaguely** without clear success criteria

These four patterns are observable failure modes — not vibes. Karpathy's principles target them directly. Loading them on every session means the friction shifts: they cost a little bit of speed on trivial tasks but save real time on non-trivial ones (fewer rewrites, fewer scope creeps, fewer "wait, why did you change that file too?" moments).

> **Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## The four principles

### 1. Think Before Coding

> **Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

> **Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Self-test: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

> **Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it — don't delete it.

When your changes create orphans:

- Remove imports / variables / functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

> **Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

| Vague | Goal-driven |
|---|---|
| "Add validation" | "Write tests for invalid inputs, then make them pass" |
| "Fix the bug" | "Write a test that reproduces it, then make it pass" |
| "Refactor X" | "Ensure tests pass before and after" |

For multi-step tasks, state a brief plan:

```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let the agent loop independently. Weak criteria ("make it work") require constant clarification.

## Working signals

You'll know these guidelines are taking effect when:

- **Diffs shrink.** Fewer unnecessary changes per PR.
- **Rewrites disappear.** Less back-and-forth from "actually, please simplify."
- **Clarifying questions come up front,** not after a wrong implementation.

You'll know they're being skipped when:

- The agent makes assumptions and apologizes for them later.
- The agent ships an abstraction with one call site.
- The agent's diff includes unrelated formatting changes.
- The agent's plan is "I'll figure it out as I go."

## How it ships to your project

The rule lives in [`core/rules/karpathy-guidelines.md`](https://github.com/agentteamland/core/blob/main/rules/karpathy-guidelines.md). Every project that has run [`atl install`](/cli/install) (any team) gets it auto-installed via the `core` cache, and `atl update` keeps it current via the [project-local copy refresh model](/cli/update#what-it-updates).

The rule is loaded into Claude's context on every session start (per the [knowledge-system architecture](https://github.com/agentteamland/core/blob/main/rules/knowledge-system.md)) — no per-prompt invocation needed.

## Going deeper

- Upstream paired examples (side-by-side wrong/right code for each principle): [EXAMPLES.md](https://github.com/forrestchang/andrej-karpathy-skills/blob/main/EXAMPLES.md)
- Original Karpathy thread: [@karpathy on X](https://x.com/karpathy/status/2015883857489522876)
- The rule's source-of-truth (what's loaded into your sessions): [core/rules/karpathy-guidelines.md](https://github.com/agentteamland/core/blob/main/rules/karpathy-guidelines.md)

## Related

- [team-repo-maintenance](/authoring/team-repo-maintenance) — governance discipline that pairs with these coding guidelines on shared repo work
- [Concepts: Rule](/guide/concepts#rule) — what rules are and how they're loaded
