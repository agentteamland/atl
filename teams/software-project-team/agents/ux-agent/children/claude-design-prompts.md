---
knowledge-base-summary: "How to author the prompt that seeds Claude Design (via `/design-screen`). Six elements in 5–8 sentences: identity, platform, states, actions, references (token file path), constraints (what NOT to invent). Pilot evidence: well-formed prompt yielded 1 iteration + 4/5 fidelity. Skip implementation primitives (TextFormField, hex colors), skip emotional storytelling — Claude Design reads tokens for register, not adjectives. Quick template at the end of the file."
---
# Authoring Prompts for Claude Design

When the project uses `/design-screen` (per the team's Claude Design workflow), ux-agent's flow documentation becomes the seed for Claude Design's prompt. The reference-project login pilot (2026-04-19) proved that **a 5–8 sentence prompt with the right inclusions yields 4/5 fidelity in 1 iteration** — and shorter or vaguer prompts force 2–3 iterations to recover.

This file documents what makes a high-yield prompt. The `/design-screen start` skill uses these patterns to construct prompts from your flow docs + AskUserQuestion answers; you should also follow them when writing standalone screen specs.

## The Anatomy of a Good Prompt

A high-yield Claude Design prompt has six elements, in this order:

```
[1. Identity]   — what app, what screen, who uses it
[2. Platform]   — Flutter mobile / React admin / React public, explicitly
[3. States]     — list every state to render
[4. Actions]    — what the user can do on this screen
[5. References] — token file path + (optional) related screens to echo
[6. Constraints]— what NOT to invent; e.g., "use only Material 3 icons"
```

5–8 sentences, dense. Skip storytelling; this isn't a brief for a designer-human, it's a constraint package for an LLM that already speaks design.

## A Worked Example (the pilot's actual login prompt)

```
Task management app's mobile login screen for iOS + Android. Single email + password
form with submit, plus inline error banner for failed attempts. States to
render: idle (form ready), submitting (button shows spinner, fields disabled),
error (banner above form with API error message). Below the form: "Forgot
password?" link and "Don't have an account? Register" row. Honor tokens in
flutter/lib/app/theme.dart — palette, typography, 4px spacing grid, 12px
radii. Use only Material 3 icons; do not invent custom SVGs. Add a small
EN/TR pill toggle in the top-right for language switching.
```

Result: 1 iteration, 4/5 fidelity, 25 min from Send → handoff.

Decompose what that prompt did right:

| Sentence | Element | Why it worked |
|----------|---------|---------------|
| "Task management app's mobile login screen for iOS + Android" | Identity + platform | Grounds the bundle in the right paradigm — touch targets, mobile chrome, etc. |
| "Single email + password form with submit, plus inline error banner" | Action set + structural hint | Bundle knows what fields and what error UX |
| "States to render: idle / submitting / error" | States explicit | Bundle ships every state ready, no follow-up needed |
| "Honor tokens in flutter/lib/app/theme.dart" | Reference | Bundle's tokens.jsx self-documents as derived from this file |
| "Use only Material 3 icons; do not invent custom SVGs" | Constraint | Without this, bundle had been likely to invent SVGs (constraint mostly worked; pilot still got 5 SVGs anyway, M3 swap was easy) |
| "Add a small EN/TR pill toggle in the top-right" | Action / detail | Bundle delivered exact two-pill toggle geometry |

What's NOT in that prompt — and shouldn't be:
- Component implementation details (`use TextFormField with InputDecoration...`) — Claude Design infers from theme.dart
- Hex colors (`primary should be #1A73E8...`) — bundle reads from token file
- Animation specs (`fade in over 300ms...`) — bundle picks defaults that flutter-agent will override with project's motion theme anyway
- Storytelling (`the user feels nervous before logging in...`) — Claude Design isn't a copywriter for emotional context

## The 6 Elements in Detail

### 1. Identity (1 sentence)

What app + what screen + who uses it. Be concrete:

```
✅ "Task management app's mobile login screen"
✅ "Admin dashboard's user list page"
✅ "E-commerce checkout review step"

❌ "A login screen" (no context)
❌ "Login screen for the app" (which app?)
```

### 2. Platform (1 phrase, embedded or separate)

State the platform explicitly even when it seems obvious. Mobile vs. web vs. tablet drives layout assumptions.

```
✅ "...for iOS + Android"
✅ "...desktop-first React admin"
✅ "...responsive React public site (mobile + desktop)"

❌ Omit entirely (bundle defaults to desktop, often wrong)
```

### 3. States (1 sentence, comma-listed)

Every state the screen supports. The bundle ships all of them as separate frames. If you don't list a state, you don't get it.

```
✅ "States to render: empty (no tasks yet), loading (initial fetch), error (network failed with retry), data (list of tasks)"

✅ "States: idle, submitting (button spinner + fields disabled), error (banner above form), success (toast + redirect)"

❌ "Standard states" (which?)
❌ Skip the line entirely (bundle gives you idle only)
```

### 4. Actions (1 sentence)

What the user can do. Include navigation targets if known.

```
✅ "Actions: submit form, click 'Forgot password?' (→ /auth/forgot), click 'Register' (→ /auth/register), toggle language (EN/TR)"

❌ "User can interact with the form" (too vague)
```

### 5. References (1 sentence)

Token file path is mandatory. Related screens to visually echo are optional.

```
✅ "Honor tokens in flutter/lib/app/theme.dart"
✅ "Honor tokens in web/admin/tailwind.config.ts; visual style should echo /dashboard/users existing screen"

❌ "Use the project's design system" (which file?)
❌ "Make it look like the rest of the app" (no anchor)
```

### 6. Constraints (1 sentence, optional but recommended)

Things to NOT do. Calibrates against bundle's tendencies.

```
✅ "Use only Material 3 icons; do not invent custom SVGs"
✅ "Stick to the 4px spacing grid; no arbitrary values"
✅ "Empty states use illustrations from /assets/illustrations/, not bundle-invented"

❌ Skip if you have no specific constraints — that's fine
```

## When to Add Each State

| State | When to include |
|-------|-----------------|
| **idle** | Always — default state |
| **loading** | If screen fetches data on mount |
| **submitting** | If screen has a form with async submit |
| **empty** | If screen displays a list/grid that can be empty |
| **error** | If screen can fail (network, server, validation) |
| **success** | If screen has a "completed" state distinct from data (e.g., post-submit confirmation before redirect) |
| **disabled** | If screen has actions that are conditionally locked |

Don't include states that don't apply. The bundle will dutifully render them anyway, but they clutter the iterations.

## Length Discipline

- Under 5 sentences → underspecified; expect 2–3 iterations
- 5–8 sentences → sweet spot per pilot
- Over 10 sentences → you're describing implementation; Claude Design will pick literal interpretations and you'll waste iterations correcting them

If your screen genuinely needs more than 8 sentences of spec, that's a sign the screen is doing too much. Consider splitting it (the bundle handles smaller, focused screens better than monolithic ones).

## Common Failure Modes

### Forgetting to name the platform
Claude Design defaults to web/desktop. A mobile screen prompt without "for iOS + Android" yields a tablet-sized layout you have to redo.

### Listing implementation primitives
"Use TextFormField with prefixIcon=Icons.mail" → bundle ignores it (it's a React-native bundle author, not a Dart writer) AND it crowds out the things bundle DOES read (tokens, states).

### Vague references
"Make it look like our other screens" → bundle has no anchor. Cite a specific token file or a specific existing screen by path.

### Over-constraining
"Use exactly 16px padding everywhere" → bundle uses 16px and your design rhythm gets flat. Better: cite the spacing grid and let bundle pick rhythm intentionally.

### Writing for humans, not for the model
"This screen is the user's first impression of our brand, so it needs to feel welcoming and trustworthy." → wasted tokens. Claude Design picks visual register from the project's existing tokens (which encode brand) — not from emotional adjectives.

## Iterating After the First Pass

If the first bundle isn't right, your follow-up prompt should:
1. Not re-state the original (Claude Design has the conversation context).
2. Pinpoint exactly what to change ("the EN/TR toggle should be in the app bar, not floating in the corner").
3. Give the desired alternative ("move it next to the title").

Don't say "make it nicer" or "fix the spacing." Specific deltas only.

## Working with `/design-screen start`

When the skill calls AskUserQuestion to gather requirements, your answers feed directly into the prompt. So:

- "States" question → don't write "standard ones"; write "empty, loading, error, data"
- "Actions" question → list every interactive element
- "Copy decisions" question → exact strings; the prompt will quote them

The skill writes the prompt, but its quality is bounded by your answers.

## Quick Template

```
[App name]'s [platform] [screen name] for [audience].
[Structural summary in 1 sentence: what's on the screen at a glance].
States: [list].
Actions: [list with navigation targets].
Honor tokens in [token file path][; visual style should echo {existing screen path}].
[Optional constraints: e.g., "use only M3 icons", "stick to 4px grid"].
```

Fill in the brackets, ship it. 5–6 sentences, sufficient for most screens.
