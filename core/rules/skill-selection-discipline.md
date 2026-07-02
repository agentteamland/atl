# Skill selection discipline

When responding to a user prompt, the agent must give explicit consideration to skill selection — particularly when the project has multiple teams installed.

## Why this rule exists

ATL projects can have multiple teams installed at once (e.g. a software team for codebase work + a personal-advisory team for life-advice work, both rooted in the same `.atl/`). Their skill sets don't overlap, but their *trigger surfaces* can — a prompt may sound applicable to either, neither, or both. Selecting the wrong skill produces bad output (a software answer to a personal question) or missed automation (a turn that should have invoked a skill but didn't).

Auto-activation (per-team watch hooks, central dispatchers, lazy-load arbitration) was considered and rejected on cost × determinism grounds: no candidate is both fully deterministic and cheap. Skill selection therefore stays the agent's responsibility, and this rule codifies the diligence that makes it reliable.

## What the agent must do

When the prompt could plausibly invoke a skill — i.e. it's not a trivial conversational turn — work through these in order:

1. **Survey the union of installed skills.** Don't default to the most-recently-used team. Yesterday's session was a software refactor; today's prompt is a personal question — the right skill probably lives in a different team.
2. **Match prompt intent to each candidate skill's `description` frontmatter.** That field is what the skill exists for. Don't infer from the skill's name alone — two skills can look similar by name yet cover different scopes.
3. **When more than one skill could apply, name them and disambiguate.** Either pick the strongest match with a one-line justification, or ask one clarifying question. Don't pick silently.
4. **When no installed skill applies, say so explicitly.** Silent non-invocation is indistinguishable from oversight; an explicit "no skill applies here, responding directly" closes the loop.

## Failure modes to watch

- **Recency bias** — defaulting to the team used in the previous turn, regardless of the current prompt's domain.
- **Team bias** — defaulting to the team installed first, or the one the user works on most.
- **Name-based confusion** — picking by skill name without reading `description`. Frontmatter exists for this reason.
- **Silent skip** — answering directly when an installed skill could have done the job. If you skip, name the skill you considered and why it didn't fit.

## When this rule does NOT apply

- **Trivial turns** (greetings, status questions, factual lookups that trigger no workflow).
- **User explicitly named a skill** (e.g. "/drain yap" — invoke as named, no second-guessing).
- **No teams installed** — only built-in Claude Code commands are in scope.

## How this rule interacts with other rules

- **Karpathy `Think Before Coding`** — step 1 (survey) is a domain-specific application of "state assumptions explicitly."
- **Karpathy `Goal-Driven Execution`** — when uncertain which skill applies, the disambiguation question (step 3) is the verification step before acting.
- **`learning-capture`** — when a non-obvious skill-selection decision gets corrected, that's worth a marker: `<!-- learning: <what the right skill was and why> -->` so future sessions don't repeat the miss.
