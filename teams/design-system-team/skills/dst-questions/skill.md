---
name: dst-questions
description: "/dst-questions — manage the project's open product/design questions list (.atl/wiki/open-questions.md) and keep it pinned to CLAUDE.md so questions blocking screen / prototype work are impossible to miss. Modes: init (scaffold), sync (re-pin from current file), add, resolve, list."
argument-hint: "[init|sync|add <topic — blocks X> | resolve <topic> | list]"
---

# /dst-questions Skill

## Purpose

Many product / design questions stay open while screens are being built — revenue model, MVP slicing, target persona, vendor choices, legal constraints, etc. They're not bugs to file in an issue tracker; they're **decisions that block screen design until answered**, and they accumulate over time.

This skill standardizes a single home for them — `.atl/wiki/open-questions.md` — and **auto-pins the list into the project's `CLAUDE.md`** inside an HTML-comment marker block, so every Claude session sees them as part of auto-loaded project instructions and surfaces them when the work it's about to do depends on one.

The mechanism mirrors `brainstorm@1.1.0` active-brainstorm pinning: the file is the source of truth; the marker is the safety net.

## When this matters

A project that has `design-system-team` installed AND has open questions blocking screen work should always have:

- `.atl/wiki/open-questions.md` listing the questions + what each blocks
- A `<!-- open-questions:active:start --> ... <!-- open-questions:active:end -->` block at the top of `CLAUDE.md`

If either is missing or out of sync, run `/dst-questions sync`.

## Modes

### `/dst-questions init`

**Use when:** the project has no `open-questions.md` yet.

1. Detect project root (`cwd`).
2. Ensure `.atl/wiki/` exists (create if missing — but do NOT run `/wiki init` automatically; that's a separate skill).
3. Create `.atl/wiki/open-questions.md` from the template below if it does not exist. If it already exists, do nothing (use `sync` instead).
4. Run the `sync` mode immediately after to insert the pin into `CLAUDE.md`.

Template:

```markdown
# Open Questions

> Questions that block screen / prototype work until answered. Each entry names the question + what it blocks. Resolved questions are removed (the answered fact lands in the relevant wiki page or `docs/` instead).

## Active

<!-- list each as: - **Question / topic** — blocks <wiki-page-or-area> -->

(none yet — add via `/dst-questions add` or by editing this file directly)

## Conventions

- Add an entry the moment a deferral happens ("we'll decide this later"). Don't trust memory.
- An entry stays here until its answer is captured somewhere durable (wiki page, docs/, or the relevant brainstorm).
- When resolving, link the place where the answer landed. Then run `/dst-questions resolve <topic>` (or remove the line + run `/dst-questions sync`).
```

### `/dst-questions sync`

**Use when:** you (or anything else) edited `open-questions.md` directly. Re-renders the pin in `CLAUDE.md` from the current file content.

1. Read `.atl/wiki/open-questions.md`.
2. Extract the bullet list under `## Active`.
3. **If the list is non-empty:** insert/update the marker block in `CLAUDE.md`:

```markdown
<!-- open-questions:active:start -->
## ❓ Open product questions

These questions block screen / prototype work — address them when their topic comes up. Full list: [.atl/wiki/open-questions.md](.atl/wiki/open-questions.md)

- **<question / topic>** — blocks <area>
- **<question / topic>** — blocks <area>
<!-- open-questions:active:end -->
```

4. **If the list is empty (only the "(none yet…)" placeholder):** remove the marker block entirely from `CLAUDE.md` (including the H2 heading and intro line).

**Insertion rules** (mirror brainstorm marker):

- **If a `<!-- brainstorm:active:start -->` block exists in `CLAUDE.md`:** insert the open-questions block **immediately after** it. Active brainstorms take visual priority over open questions because they're a stronger interrupt signal.
- **Else:** insert near the top of the file, right after the H1 + opening description (before the first H2 heading).
- **Bullet count cap:** if more than 8 active questions, list the first 6 inline and add `- … + N more — see [open-questions.md](.atl/wiki/open-questions.md)`. Keeps `CLAUDE.md` lean.

### `/dst-questions add <topic — blocks X>`

**Use when:** a new question emerges in conversation and you want to capture it without leaving the chat.

1. Append the entry to the `## Active` section of `.atl/wiki/open-questions.md`. Format: `- **<topic>** — blocks <area>`.
2. Remove the placeholder line if it's still there.
3. Run `sync` automatically.

### `/dst-questions resolve <topic>`

**Use when:** a question is answered and the answer has landed somewhere durable.

1. Find the bullet matching `<topic>` in `.atl/wiki/open-questions.md` `## Active` section.
2. Remove that bullet.
3. If the section is now empty, restore the placeholder line.
4. Run `sync` automatically.
5. **Surface a reminder to the user:** "Answer should be captured in <suggested-wiki-page> — confirm before I close the question."

### `/dst-questions list`

**Use when:** quick read of current state. Read-only.

Print the active entries. No file changes.

## Idempotence

`init` and `sync` are both idempotent — running them multiple times is safe. `init` skips creation if the file exists; `sync` overwrites the marker block deterministically.

## When the skill is not available

The convention (file + marker) is described in `rules/open-questions-pinning.md` and is auto-loaded by every project that installs `design-system-team`. Agents (`ds-architect-agent`, `prototype-agent`) can maintain the file + marker manually if the skill isn't run — the rule tells them how. The skill is the convenience, the rule is the contract.

## See also

- `rules/open-questions-pinning.md` — the convention this skill enforces
- `brainstorm@1.1.0` (`agentteamland/brainstorm`) — the parallel marker pattern for active brainstorms; same insertion logic, same recovery rule

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
