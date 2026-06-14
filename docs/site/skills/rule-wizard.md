# `/rule-wizard`

A clarification wizard that uses **option-based Q&A rounds** before adding a rule. Captures the details easily overlooked when writing a rule directly with [`/rule`](/skills/rule) — edge cases, exceptions, alternative formulations, scope, motivation, example variants — and dynamically detects when the conversation actually concerns *multiple* rules.

Once the discussion completes, the wizard hands the finalized text to `/rule` to write the rule(s) to the right file.

Ships as a global skill in [rule](https://github.com/agentteamland/rule).

## When to use which

| Skill | When |
|---|---|
| [`/rule`](/skills/rule) | The rule is clear; you can state it in one sentence with a known scope and a known why. |
| `/rule-wizard` | The rule is fuzzy. You know the area but not the exact wording. You want to be sure you didn't miss an exception, a related rule, or that you actually have *two* rules sneaking under one umbrella. |

## Required argument

`/rule-wizard` requires a **context** — a short text (any language) describing the topic, the initial idea, or the problem encountered:

- ✅ `/rule-wizard Logging usage in API`
- ✅ `/rule-wizard Worker should not connect to DB directly`
- ✅ `/rule-wizard Controllers should not write try-catch; global handler takes over`
- ❌ `/rule-wizard` — invoked without context, the wizard refuses and asks for one

## Three scopes

Same as [`/rule`](/skills/rule#three-scopes): project (default), `--global`, `--team`. When the flag is provided, the wizard skips the scope question.

## Phase 1 — understanding and preparation

### 1.1 Read existing rules (mandatory)

**Before** asking anything, read:

- `.claude/rules/coding-common.md`
- All `.md` files under `.atl/docs/coding-standards/` (dynamically listed)

Purpose:

- **Duplication prevention** — does the same or a very similar rule already exist?
- **Conflict detection** — does the new rule contradict an existing one?
- **Extension opportunity** — can it be added as a bullet to an existing rule?
- **Cross-reference** — rule IDs that can populate the final `Related` field

### 1.2 Analyze the context (silently)

From the user's context, internally derive:

- Probable scope (which app(s)? common or single?)
- Probable intent (mandatory `must` / prohibitive `must not` / advisory `should`?)
- Affected layers (Controller, Service, Repository, Consumer, Hub, Job?)
- Initial hypotheses for `Apply when`, `Why`, `Examples`
- IDs of similar existing rules (from 1.1)

### 1.3 Present the analysis summary

A short paragraph back to the user:

> "As I understand it, you want API controllers not to contain try-catch blocks, and error handling to be delegated to a global handler at the upper layer. This falls under `coding-standards/api.md` and complements the existing `no-logic-in-bridges` rule. I'll now clarify the details with a few questions."

Then proceed to Phase 2.

## Phase 2 — option-based questioning

### Core principles (all binding)

1. **Every question uses `AskUserQuestion`.** No plain-text open-ended questions.
2. **Each question has 2–4 options.** Platform limit is 4. If more reasonable options exist, **split** the question; never limit yourself to "the best 4."
3. **An "Other" option is automatic.** The tool adds it; do not write it explicitly.
4. **If there is a recommended option, place it first** with `(Recommended)` in the label and a brief reason in `description`.
5. **Match the user's language for questions and options.** `/rule` will translate to English at the final step — the wizard mirrors the user's language.
6. **Maximum 4 questions per round.** More than 4 → split into rounds; later rounds can use earlier answers.
7. **Don't re-ask fields clearly derived from context.** Use a confirmation question instead: "I understood it as X — correct?" (binary).
8. **Options must be distinct and clear.** If two options are nearly the same, remove one.

### Areas to cover

For each rule, clarify (via Q&A):

#### A) Scope — where to write?

(Skipped if `--global` or `--team` is set.)

Sub-question if project scope is selected: which application? (dynamically lists existing `.atl/docs/coding-standards/{app}.md` files + a "common" option).

Sub-question if team scope is selected: which agent's knowledge base? (lists installed-team agents + "team-wide rule" option).

#### B) Single-sentence rule statement

Three alternative formulations from context, each with a different tone / restrictiveness:

- **Strict prohibition** ("X must never be done")
- **Advisory** ("Use Y for X")
- **Conditional** ("X may only be done when Y")

User can write their own via "Other."

#### C) Motivation (Why)

Multi-select from a context-driven list: lesson from past mistake (specify), architectural consistency, testability, performance, security, readability, regulation. Primary motivation first.

#### D) Apply when (trigger conditions)

2–4 specific triggers from context. Each option contains a **concrete file path or code pattern** (e.g., `api/Controllers/*.cs` for controller actions). Multi-select if multiple triggers can stack.

#### E) Don't apply when (exceptions, optional)

Options like: no exceptions / test code is exempt / legacy or generated code is exempt / Other. If "no exceptions," the field is omitted from the final rule.

#### F) Examples

Two questions: ✅ correct example + ❌ wrong example. Each presents 2–3 short scenarios; user picks one or writes their own.

#### G) Related rules (optional)

Lists similar-rule IDs found in Phase 1.1 + a "None" option.

### Suggested rounds

- **Round 1 (fundamentals):** Scope + Rule statement + Motivation — 3 questions
- **Round 2 (behavior):** Apply when + Exception + ✅ Example — 3 questions
- **Round 3 (polish):** ❌ Example + Related + (if needed) edge case — 2–3 questions

Rounds shrink when fields are clearly derived; extra questions get added when ambiguity persists. The user is never forced to give the same answer twice in a round.

## Phase 3 — dynamic multiple-rule detection

The wizard **starts with a single-rule assumption.** But if any of these signals appear during questioning, immediately ask the user a distinction question:

| Signal | What it suggests |
|---|---|
| Two different applications selected in Scope and their natures differ | Two rules disguised as one |
| Apply-when triggers point to two unrelated code layers | Two rules |
| Rule statement combines two independent prohibitions ("X must not, and Y must not either") | Two rules |
| Examples can't be explained by a single rule | Two rules |
| Two independent justifications selected in Motivation multi-select | Two rules |

### Distinction question

```
"This context actually looks like two different rules. How should we proceed?"
```

Options:

- **(Recommended)** Add as two separate rules — clarify each one separately
- Keep as a single rule — expand the Rule statement to cover both clauses
- Focus on only one for now and handle the other later
- Misidentified — this is actually a single rule

### Post-decision flow

- **Two separate rules** → repeat Phase 2 independently for each. Don't mix question rounds.
- **Keep as a single rule** → re-ask the Rule statement question with combining formulations.
- **Focus on only one** → at end of Phase 4, offer "Shall we handle the other rule now?"
- **Misidentified** → return to normal flow, ignore the signal.

## Phase 4 — consolidation and final approval

### 4.1 Generate the final rule text

Compose a **natural-language** rule text in the user's language from collected answers. This text is the **input for the `/rule` skill** — `/rule` does the English translation + structured-format parsing.

The text must contain everything `/rule` will parse: Scope, Rule, Why, Apply when, Don't apply when (if applicable), Examples (✅ + ❌), Related (if applicable).

Example (TR-language wizard, English-final-rule):

> "Controller actions in the API project should not write try-catch blocks — error handling must be delegated to the global exception handler at the upper layer. This rule exists to preserve architectural consistency and to keep controllers as thin bridges; try-catch is the responsibility of services or the global handler. Applies to: all controller actions in `.cs` files under `api/Controllers/`. Test code is exempt. Correct example: `[HttpPost] public async Task<IActionResult> Create(CreateProductRequest req) { var result = await _productService.CreateAsync(req); return Ok(result); }` — no try-catch. Wrong example: writing `try { ... } catch (Exception ex) { return BadRequest(ex.Message); }` inside a controller. Related rule: `no-logic-in-bridges`."

### 4.2 Show the user; ask for approval

Show the final-rule text and ask via `AskUserQuestion`:

```
"Is this text the final version of the rule? Can I add it now with /rule?"
```

Options:

- **(Recommended)** Yes, add with `/rule`
- I need to correct part of the text — let me tell you which part
- I think there's a missing area — let's do another question round
- Cancel, do not add for now

### 4.3 Invoke `/rule`

When the user approves:

- **Single rule** → invoke `/rule <final text>`.
- **Multiple rules** → invoke each one **sequentially**, with brief progress notification: "First rule written (`{id}` → `{file}`). Moving to the second rule now."
- After each `/rule` invocation, summarize the result.

If the user requests correction → re-ask the relevant question, update the final text, return to 4.2.

If the user requests an additional round → run a question round for the missing area, return to 4.1.

If the user cancels → terminate cleanly. No file is written. Inform: "Rule was not written. You can start again with `/rule-wizard` whenever."

### 4.4 Final summary

Single message at the end:

- How many rules were written
- Each rule's ID + file
- Existing rules marked as related (if any)
- Reminder if a deferred rule from Phase 3 was set aside for later

## Critical principles

1. **Context is mandatory.** Skill does not work without an argument.
2. **Every question has options.** `AskUserQuestion` only.
3. **If 4 options aren't enough, split.** Never limit to "the best 4."
4. **Read existing rules first.** Mandatory prerequisite.
5. **Never assume.** Every field not clearly derived from context requires a question.
6. **Dynamically detect multiple rules.** Start single, ask when divergence signals appear.
7. **Final text in the user's language.** `/rule` translates to English.
8. **`/rule` is not invoked without approval.** The user sees and approves the final text.
9. **Incomplete field is worse than nonexistent field.** Required fields (Rule, Why, Apply when, Examples) must be fully represented.
10. **Skill can run repeatedly for multiple rules.** If split mode was selected in Phase 3, each rule goes through Phase 2–4 individually.

## Related

- [`/rule`](/skills/rule) — invoked at Phase 4 to write the finalized rule
- [Concepts: Rule](/guide/concepts#rule) — what rules are and how they're loaded

## Source

- Spec: [rule/skills/rule-wizard/skill.md](https://github.com/agentteamland/rule/blob/main/skills/rule-wizard/skill.md)
