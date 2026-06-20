# Skill selection discipline

When you install multiple ATL teams in the same project, the assistant has multiple skill sets to choose from on every turn. ATL deliberately does NOT auto-activate teams based on prompt content — the assistant decides, with its judgment shaped by clear prompts from you. This page is the user side of that contract.

## What's happening under the hood

Each ATL team ships skills (slash commands the assistant can invoke). When you run `atl install software-project-team` and `atl install personal-advisory-team` (for example), both teams' skills become available in every Claude Code session in this project. The assistant sees the union of all skills and picks one — or none — per turn.

ATL's [`skill-selection-discipline` rule](https://github.com/agentteamland/core/blob/main/rules/skill-selection-discipline.md) requires the assistant to consider the full skill list, match prompt intent to each skill's purpose, and disambiguate when multiple skills could apply. The rule auto-loads in every session.

## Why ATL doesn't auto-trigger team skills

We considered three auto-activation mechanisms:
- **Per-team watch hooks** — every installed team probes every prompt; cost scales with team count.
- **Central dispatcher** — one ATL-side watcher hints which team is relevant; still requires inferential routing on every prompt.
- **Lazy team-load arbitration** — a small LLM call routes to the right team per prompt; deterministic but adds latency × cost on every turn.

All three were rejected on the same grounds: no candidate is fully deterministic AND every candidate adds non-trivial cost. Skill selection is the assistant's job; your prompt clarity is what makes the assistant's job reliable.

## What you can do

Three habits help the assistant pick correctly:

### 1. Be specific about your domain

A vague prompt forces the assistant to guess, and guessing is where wrong-team selection happens.

| Vague | Specific |
|---|---|
| "What should I do?" | "I have a meeting with Alex tomorrow — what should I ask?" (clearly personal) |
| "Take a look at this" | "Let's add validation to the API endpoint" (clearly software) |
| "Help me out" | "Let's review my finances this week" (clearly personal-advisory) |

The more your prompt names the domain, the less the assistant has to infer.

### 2. Name the skill if you know it

If you know which slash command to use, invoking it directly is faster and more reliable than waiting for the assistant to remember. Typing `/save-learnings` runs immediately; "save my learnings" requires the assistant to map intent to skill name.

### 3. Correct mid-turn if the wrong team gets picked

If the assistant starts answering with the wrong team's frame ("software response to a personal question" or vice versa), say so:

> "This isn't personal — it's software" — the assistant will switch and remember the correction within the session.

Mid-turn correction is normal and fast. A wrong skill running to completion is what costs time.

## A note on multi-team projects

Most ATL projects only need one team. Multi-team setups arise mainly when:

- You're using the assistant for personal advisory work AND for a specific software project, in the same `.atl/` (and you don't want two separate Claude Code sessions to switch contexts).
- You've extended a base team with team-specific add-ons (e.g., `starter-extended` extending `software-project-team`).

If you find yourself frequently fighting the assistant over which team a prompt belongs to, that's a signal to either (a) split into separate projects (each with one team), or (b) be more explicit in your prompts. ATL is designed to support multi-team setups, but it's not designed to be invisible — your prompts are part of the routing logic.

## Related

- **Rule source:** [`core/rules/skill-selection-discipline.md`](https://github.com/agentteamland/core/blob/main/rules/skill-selection-discipline.md) — the assistant-side rule that this page is the user-side counterpart of.
- **Auto-activation rationale:** [`auto-team-activation.md`](https://github.com/agentteamland/workspace/blob/main/.atl/brain-storms/auto-team-activation.md) — the workspace brainstorm that documents why per-team, central-dispatcher, and lazy-load arbitration were rejected.
- **Karpathy guidelines:** [`/guide/karpathy-guidelines`](/guide/karpathy-guidelines) — the broader behavioral principles that informed the rejection (Simplicity First).
