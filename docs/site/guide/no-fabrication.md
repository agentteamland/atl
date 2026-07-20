# No fabrication

An ATL agent that invents a plausible-looking value — a tool name, a work-item id, a "green" verdict it never actually verified — produces confident, wrong output that is expensive to catch downstream. This core rule forbids it: **resolve facts verbatim from the source, and if you can't resolve or verify something, stop and say so — never fake it.** This page is the user side of that discipline.

## What's happening under the hood

The [`no-fabrication` rule](https://github.com/agentteamland/atl/blob/main/core/rules/no-fabrication.md) auto-loads in every session (and into the autonomous `claude -p` workers the delivery-team spawns, via the same global rule reflection). It codifies an **output-integrity** habit: never manufacture a fact, identifier, or result you don't have.

It was distilled from the corpus itself — the same edge was independently re-derived across three teams and the core skills: the delivery `developer` ("never invent a tool name, a state literal, or a path"; "unverified is never a pass"), the backend contract ("a surface that can't be run → block, never fake-green"), the profile `curator` ("never fabricate a value"), the `advisor` ("never asserted without proof"), and `/rule` ("never assume — if information is missing, ask").

## What it means in practice

**Resolve identifiers verbatim.** A tool name, an API field, a state literal, a path, a work-item id, a version — read it from the source of truth and reproduce it exactly, rather than reconstructing a plausible one from memory.

**If you can't resolve it, surface the gap.** A missing analysis, an id that doesn't exist yet, a value the agent can't find — it says so, instead of filling the hole with a confident guess to keep moving.

**Never fake-green.** A test, build, or check that timed out, was skipped, or couldn't run is **unverified** — and unverified is never reported as "pass". The agent reports it as blocked/unknown with the evidence.

## Why it's a core rule

It is the honesty layer under ATL's **autonomous delivery**: an unattended `claude -p` worker that fabricates a merge state or fakes a green test would land broken work silently — the exact failure the deterministic gates exist to prevent. It **complements** the [Karpathy guidelines](/guide/karpathy-guidelines): those govern the *input* side (don't guess the requirement — ask); this governs the *output* side (don't fabricate the artifact you emit). The rule targets facts and results you claim to be real — it does **not** restrain genuinely generative work (drafting prose, brainstorming, clearly-labeled estimates), where invention is the task.
