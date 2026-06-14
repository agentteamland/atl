---
name: design-screen
description: "Orchestrate the Claude Design loop for a new screen. /design-screen start \"<one-line description>\" → derives slug, gathers requirements, builds a token-aware prompt, writes intent.md (status: pending-design), hands prompt to user. /design-screen done <handoff-url> → finds the matching pending design (or asks if multiple), fetches the handoff bundle, extracts it, hands a self-contained brief to flutter-agent or react-agent for integration. State persists in .claude/design/{slug}/intent.md across sessions — full Phase A context survives compactions and restarts."
argument-hint: "start \"<description>\" | done <handoff-url>"
---

# /design-screen Skill

## Purpose

Coordinates the human-in-the-loop visual design phase that lives between text-only flow specs (ux-agent) and code (flutter-agent / react-agent). Wraps the Claude Design ↔ codebase round-trip in two commands so the workflow is repeatable and the context survives session boundaries.

This skill does NOT replace human visual iteration in Claude Design — that's the value-add. It removes the boilerplate around it: prompt construction, state persistence, bundle fetching, agent briefing.

> **Reference architecture:** `agentteamland/workspace/.atl/docs/design-workflow.md` (the canonical decision record). Pilot evidence: `<test-project>/.claude/design/login/pilot-findings.md` (2026-04-19).

## Two Modes

The first arg switches mode: `start` or `done`.

```
/design-screen start "<one-line description of the screen>"
/design-screen done <handoff-url-from-claude-design>
```

---

## Mode 1 — `start`

### Flow

1. **Derive slug.** Take the description, produce a kebab-case slug (e.g., "tasks empty state with start CTA" → `tasks-empty-state`). Keep it short (3–5 words). On collision with an existing `.claude/design/{slug}/` directory, append `-2`, `-3`, etc.

2. **Read project context.**
   - Tokens: locate `flutter/lib/app/theme.dart` (Flutter), `web/tailwind.config.ts` (React), or equivalent.
   - UX flow: scan `.atl/docs/coding-standards/`, `.atl/wiki/`, and any `flow-*.md` for related screen flows.
   - Existing design records: `.claude/design/*/intent.md` to detect related work.

3. **AskUserQuestion to fill gaps** (only ask what isn't already inferable):
   - **Target platform**: Flutter mobile / React admin / React public (if project has more than one and the description doesn't pin it down).
   - **States to render**: empty / loading / error / data / submitting / success / disabled / etc. (multiSelect; offer the typical set, let user add custom).
   - **Actions on this screen**: list (free text or multiSelect; mention things the screen lets the user do — submit, cancel, navigate, share, etc.).
   - **Copy decisions**: title, subtitle, CTA labels, error message tone (formal/friendly), placeholder text. Ask one question per copy slot if not in the description.
   - **Reference screens**: any existing screen the new one should visually echo.

4. **Construct the Claude Design prompt.** 5–8 sentences. Must include:
   - What screen + what app (so Claude Design grounds in identity).
   - Target platform explicitly.
   - States to render, listed.
   - User actions, listed.
   - Token reference path (e.g., "honor tokens in `flutter/lib/app/theme.dart`").
   - UX flow reference if one exists.
   - What NOT to invent (e.g., "use only icons available in Material 3"; pilot lesson).

5. **Write `.claude/design/{slug}/intent.md`** with `status: pending-design` and the schema below. **Detailed, not summarized** — verbatim user description, every AskUserQuestion answer as-given, full prompt text. A fresh Claude session reading this file later must be able to resume cold.

6. **Print the prompt to the user**, plus the next step:
   ```
   Prompt ready. Open https://claude.ai/design and paste this in. Iterate visually
   until you're satisfied. When you have the handoff URL (or you've downloaded
   the zip and have its file path), run:

       /design-screen done <handoff-url-or-zip-path>
   ```

### `intent.md` schema (mandatory shape, all sections required)

```markdown
---
status: pending-design
slug: {kebab-slug}
created: {ISO-8601 timestamp UTC}
description: "{user's verbatim one-liner from the start command}"
target: flutter | react-admin | react-public
---

# {Screen / surface display title}

## User's Original Description (verbatim)

> {exact quote of the description arg}

## Requirements (gathered via AskUserQuestion)

### States
- {as-given verbatim}

### Actions
- {as-given verbatim}

### Copy decisions
- title: "{verbatim}"
- subtitle: "{verbatim}"
- {other slots: verbatim}

### Target platform
{user-selected value}

### Reference screens
{list, or "none cited"}

## Token & flow references read

- tokens file: `{path}`
- ux flow doc: `{path or "none recorded"}`
- related design records: `{paths or "none"}`

## Derived prompt (sent to Claude Design)

```
{full prompt text — exactly what user pastes}
```

## Status log

- {ISO timestamp} — created (status: pending-design). Prompt handed to user.
```

**No paraphrasing in the body.** If the user said "users should be able to retry", do NOT write "supports retry" — write the verbatim quote. Detail discipline matters because the file is the only context bridge between Phase A and Phase B (sometimes weeks apart).

---

## Mode 2 — `done`

### Flow

1. **Locate pending designs.** Scan `.claude/design/*/intent.md`; collect every file with `status: pending-design`.

2. **Resolve which one is being integrated:**
   - **0 found** → Error: "No pending designs. Did you mean `/design-screen start \"<description>\"`?" Stop.
   - **1 found** → Use it. Tell the user: "Integrating design `{slug}` (created {date})." Proceed.
   - **N found** → `AskUserQuestion`: "Which pending design is this handoff for?" with options listing each `{slug} — {description} ({created date})`. Use the chosen one.

3. **Read the chosen `intent.md`.** Now you have full Phase A context: what the screen is, who it's for, what the prompt was, what tokens were referenced, every requirement.

4. **Fetch the handoff bundle.** The `done` arg is either:
   - A URL like `https://api.anthropic.com/v1/design/h/{id}?open_file=...` → use `Bash curl -L -o {tempdir}/bundle.zip "{url}"` to download.
   - A local file path to a downloaded zip → use it directly.

5. **Extract** to `.claude/design/{slug}/{YYYY-MM-DD}-v{N}-extracted/` (where N is the next available version number). Save the original zip alongside as `{YYYY-MM-DD}-v{N}-handoff.zip`.

6. **Save snapshot exports** if the bundle includes a renderable HTML or PDF (per Q4 versioning convention from the design-workflow doc): `.claude/design/{slug}/{YYYY-MM-DD}-v{N}.{html,pdf}`.

7. **Brief the implementation agent.** Pick:
   - `target: flutter` → `flutter-agent` (Agent tool, subagent_type=flutter-agent)
   - `target: react-admin` or `react-public` → `react-agent`

   **The brief MUST be self-contained.** Do not say "implement this bundle" — the pilot established that blind briefs are insufficient. Include in the brief:
   - The screen name and intent (from `intent.md`).
   - The verbatim Phase A requirements (states, actions, copy decisions).
   - The extracted bundle path.
   - Pointer to the agent's own `claude-design-handoff.md` children file (which spells out translation rules — SVG icon substitution, idiomatic state mapping, token resolution, ARB key extraction for Flutter / namespace placement for React).
   - Any project-specific conventions worth flagging (e.g., "this project uses Riverpod; map React state to Notifier patterns").

8. **Update `intent.md`:**
   - Frontmatter: `status: integrated`.
   - Append to status log: `{timestamp} — integrated (status: integrated). Handoff: {URL or zip path}. Extracted: {extracted-path}. Agent: {flutter-agent|react-agent}. Notes: {agent-output-summary or "see git diff"}.`

9. **Show the user a summary** of what changed (agent output highlights, files touched, fidelity match if observable, any frictions encountered).

---

## Important Rules

1. **State lives in files.** Never rely on conversation memory between `start` and `done` — the gap could be minutes or weeks. The intent.md file is the only contract.

2. **Detail over summary in intent.md.** Verbatim user input. Future-Claude readability is the test.

3. **Slug derivation never asks the user.** Pick a kebab-case slug from the description; collision suffix on conflict. Mirrors `/brainstorm` skill's auto-naming.

4. **No new agent invented.** This skill orchestrates; existing agents implement. The visual-iteration step is human-only by design.

5. **Self-contained brief to the implementation agent.** Pilot lesson: blind "implement this bundle" is insufficient. Always pass the intent + the agent's translation guidance pointer.

6. **Versioning is per-screen, not global.** Each screen's `.claude/design/{slug}/` accumulates v1, v2, v3 over time as the design iterates.

7. **Handoff bundle is React+HTML, not Dart.** Even for Flutter targets. Translation is part of the workflow; flutter-agent's `claude-design-handoff.md` covers the patterns. Do NOT promise the user "Dart code is coming directly from Claude Design."

8. **Don't promise visual perfection.** Pilot showed 4/5 fidelity is the realistic target. Brand glyphs, animation curves, subpixel color blends will diverge by ~1–5%. This is fine.

## What This Skill Does NOT Do

- Open `claude.ai/design` in a browser. (User does that manually after `start`. A future Phase 3 may add this via Chrome MCP if Claude Design's DOM stabilizes.)
- Iterate the design. (Human-in-the-loop; the value-add of the visual phase.)
- Decide platform target on user's behalf when ambiguous. (AskUserQuestion handles it.)
- Skip the AskUserQuestion gathering even when the description seems complete. (One question per slot is acceptable; better to over-capture than under-capture in intent.md.)
- Auto-trigger CI / lint / `flutter gen-l10n` after agent implementation. (Implementation agent's job per its own conventions.)

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
