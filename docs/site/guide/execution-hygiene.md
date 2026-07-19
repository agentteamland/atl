# Execution hygiene

ATL agents share a small set of **execution defaults** — habits for *how* to do multi-step engineering work cleanly. They used to live only in one maintainer's personal setup; now they're a core rule, so every team and every autonomous delivery worker works the same disciplined way. This page is the user side of that posture.

## What's happening under the hood

The [`execution-hygiene` rule](https://github.com/agentteamland/atl/blob/main/core/rules/execution-hygiene.md) auto-loads in every session (and into the autonomous `claude -p` workers the delivery-team spawns, via the same global rule reflection). It codifies three habits that make an agent's work reviewable and trustworthy — the difference between "it produced a diff" and "it produced a diff a senior engineer would sign off on."

It **complements** the two rules next to it rather than repeating them: the [Karpathy guidelines](/guide/karpathy-guidelines) govern *thinking* before coding, and [branch hygiene](https://github.com/agentteamland/atl/blob/main/core/rules/branch-hygiene.md) governs the *branch lifecycle*. Execution hygiene is about the moment-to-moment mechanics in between.

## The three habits

**1. Subagent hygiene.** For a broad search or a multi-file investigation, the agent spins off a *subagent* — one focused task each — and keeps its own context lean. The subagent returns **the conclusion, not the file dumps**: you get the finding, not a transcript of everything it read. This keeps the main reasoning coherent (a bloated context is its own kind of complexity).

**2. Autonomous bug-fix.** Hand the agent a reproducible bug, an error, or a failing test and it **investigates and fixes it itself** — reads the log, the stack trace, the test, forms a hypothesis, fixes, verifies — instead of bouncing it back to ask you what to do. A bug report is a goal, not a request for confirmation. The one thing it *will* surface is genuine ambiguity — and even then, as a diagnosis, not a "what should I do?".

**3. Atomic commits.** Each commit is one coherent, verified unit of work — committed as soon as that unit works, not batched with unrelated changes. Every changed line traces to a single intent, so a review (and a future `git bisect`) stays legible. This is *commit granularity*; where those commits live is branch hygiene's job.

## Why it's a core rule

These are the disciplines that make ATL's **autonomous delivery** trustworthy: when a `claude -p` worker builds a backlog item unattended, it commits atomically, fixes its own test failures, and delegates cleanly — because the rule reaches it, not just an interactive session. Promoting them from one person's persona into the platform is what lets "the backlog delivers itself" mean *cleanly*.
