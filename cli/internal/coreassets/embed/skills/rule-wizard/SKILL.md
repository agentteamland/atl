---
name: rule-wizard
description: "A clarification wizard that uses option-based Q&A rounds before adding a rule. Captures missing details from context, presents alternatives as options, dynamically detects multiple rules, and finally adds the finalized rule(s) via /rule. 2 scopes: project (default), --global."
argument-hint: "[--global] <context — general topic of the rule, any language>"
---

# /rule-wizard — clarify a rule through option-based Q&A, then write it via /rule

A wizard that surfaces details easily overlooked when writing a rule directly with
`/rule` (edge cases, exceptions, alternative formulations, scope, motivation,
example variants) through **option-based Q&A rounds**. Once the discussion is
complete, it hands the finalized rule(s) to the `/rule` skill to write them.

> Related skill: [/rule](../rule/SKILL.md) — the rule writer invoked at the end
> of this skill.

---

## Parameter requirement

The skill is always invoked with a **context** argument: a short text (any
language) describing the general topic of the rule, the initial draft idea, or
the problem encountered.

- ✅ `/rule-wizard Logging usage in API`
- ✅ `/rule-wizard Worker should not connect to DB directly`
- ✅ `/rule-wizard Controllers should not write try-catch; global handler takes over`
- ❌ `/rule-wizard` (invoked without context)

**If no context is provided:** do not proceed. Ask the user concisely:

> "This skill requires an initial context. Please briefly describe the general
> topic or initial idea for the rule: `/rule-wizard <topic of the rule>`.
> Example: `/rule-wizard Error handling for Worker Jobs`."

Do not read any files or ask any questions until the user responds.

---

## Phase 1 — Understanding and preparation

### 1.1 Read existing rules (mandatory)

**Before** questioning, always read:

- `<project>/.atl/rules/coding-common.md` (if present)
- **All `.md` files** under `<project>/.atl/docs/coding-standards/` (these differ
  per project — dynamically list and read them)
- For `--global` scope: the relevant files under `~/.atl/rules/`

Purpose:
- **Duplication prevention:** does the same or a very similar rule already exist?
- **Conflict detection:** does the new rule contradict an existing one?
- **Extension opportunity:** can the new rule be added as a bullet within an
  existing rule?
- **Cross-reference:** rules to reference in the `Related` field at the final step.

### 1.2 Analyze the context

Extract from the user's context (silently, not shown in the response):

- **Probable scope:** which application(s)? Common or a single application?
- **Probable intent:** mandatory (must) / prohibitive (must not) / advisory (should)?
- **Affected layers:** Controller, Service, Repository, Consumer, Hub, Job?
- **Initial hypotheses:** raw ideas for Apply when, Why, Examples.
- **Similar existing rules:** note the IDs if any were found in 1.1.

### 1.3 Present the analysis summary

Summarize your understanding in a **short paragraph**. Example:

> "As I understand it, you want API controllers not to contain try-catch blocks,
> and error handling to be delegated to a global handler at the upper layer. This
> falls under `coding-standards/api.md` and complements the existing
> `no-logic-in-bridges` rule. I'll now clarify the details with a few questions."

Then proceed to Phase 2.

---

## Phase 2 — Option-based questioning

### Core principles (all binding)

1. **Every question uses the `AskUserQuestion` tool.** No plain-text open-ended
   questions. "Can you explain this?" is forbidden — always generate options.
2. **Each question has 2-4 options.** Platform limit is 4. If more reasonable
   options exist, split the question; never cap at "the best 4".
3. **An "Other" option is added automatically.** The user can enter free text;
   don't write it as an explicit option — the tool adds it.
4. **If there's a recommended option, place it first** and add `(Recommended)` to
   its label. Briefly justify it in the option's `description`.
5. **Match the user's language for questions and options.** `/rule` translates to
   English at the final step — this skill mirrors whichever language the user
   invoked it in.
6. **Maximum 4 questions per round.** If more are needed, split into rounds —
   answers from round one can feed round two.
7. **Don't re-ask fields clearly derived from context.** Instead ask a
   confirmation question: "I understood it as X — is that correct?" (binary:
   Correct / No, change it).
8. **Options must be distinct and clear.** If two are nearly the same, drop one.
   Options should be collectively exhaustive — for an "all of the above"
   situation, present it with `multiSelect: true`.

### Areas to cover

Clarify each area per rule through Q&A. Fields clearly derived from context →
confirmation questions; unclear ones → full questions.

#### A) Scope — where should it be written?

**If `--global` is provided,** skip this question; the scope is set. If no flag,
ask:

**Example question:** "Where should this rule be written?"

**Option patterns:**
- Specific to this project (`<project>/.atl/`) — default
- Applies to all my projects (`--global` → `~/.atl/rules/`)

**Follow-up if project scope is selected:** "Which application does it cover?"
- Dynamically list existing files in `<project>/.atl/docs/coding-standards/`
- All applications (common) — `coding-common.md`
- One option per existing `coding-standards/{app}.md` file

#### B) Single-sentence rule statement (Rule)

**Example question:** "Which best expresses the essence of the rule?"

**Option patterns:** derive 3 alternative formulations from context, each a
different tone/restrictiveness:
- **Strict prohibition** ("X must never be done")
- **Advisory** ("Use Y for X")
- **Conditional** ("X may only be done when Y")

The user can write their own via "Other".

#### C) Motivation (Why)

**Example question:** "What is the primary motivation for this rule?"

**Option patterns (select based on context):**
- Lesson from a past mistake (specify which)
- Architectural consistency (single source of truth, single entry point, …)
- Testability
- Performance
- Security
- Readability / maintainability
- Regulation / compliance

Use `multiSelect: true` if multiple motivations apply. Place the primary first.

#### D) Apply when (trigger conditions)

**Example question:** "In which file/code patterns should this rule be triggered?"

**Option patterns:** derive 2-4 specific triggers from context. Each option holds
a **concrete file path or code pattern**:

- `api/Controllers/*.cs` — in controller actions
- `api/Services/*.cs` — in service methods
- `api/Consumers/*.cs` — in consumer handlers
- A specific attribute/pattern (e.g., `[HttpPost]`, `BackgroundService` derivatives)

Use `multiSelect: true` if multiple triggers can combine.

#### E) Don't apply when (exceptions) — optional

**Example question:** "Are there situations where this rule does not apply?"

**Option patterns:**
- No, no exceptions
- Yes: test code is an exception
- Yes: legacy/generated code is an exception
- Other: user specifies their own

If the user selects "No," the `Don't apply when` field is omitted from the final
text.

#### F) Examples

**Goal:** at least one ✅ correct and one ❌ wrong concrete example.

**Example question:** "Which ✅ correct example best represents the rule?"

**Option patterns:** derive 2-3 short code snippets or scenarios, each
illustrating the rule from a different angle. The user picks one or writes their
own via "Other". Same logic for the ❌ example, asked as a separate question.

#### G) Related rules — optional

**Example question:** "Should this rule reference an existing rule?"

**Option patterns:** list IDs of similar rules found in Phase 1.1 + a "None"
option. Example:

- `no-logic-in-bridges` (related — gains meaning together)
- `repository-owns-db-access` (distantly related)
- None, independent rule

### Suggested round planning

- **Round 1 (fundamentals):** Scope + Rule statement + Motivation — 3 questions
- **Round 2 (behavior):** Apply when + Exception + Example (✅) — 3 questions
- **Round 3 (polish):** Example (❌) + Related + (if needed) edge case — 2-3 questions

Rounds shrink when fields are clearly derived from context; extra questions are
added on ambiguity. Never force the same answer twice within a round.

---

## Phase 3 — Dynamic multiple-rule detection

The skill **starts with a single-rule assumption**. But if any signal below
appears during questioning, **immediately** ask a distinction question and adjust.

### Detection signals

1. **Two different applications are selected in Scope and their natures differ**
   (e.g., both API and Worker, but the rule means different things in API
   controllers vs. Worker BackgroundServices).
2. **The triggers selected in Apply when point to two unrelated code layers**
   (e.g., `api/Controllers/` + `api/Repositories/`).
3. **The Rule statement combines two independent prohibitions** ("X must not be
   done and Y must not be done either" — two independent clauses).
4. **The Examples scenarios can't be explained by a single rule** — each
   exemplifies a different principle.
5. **Two independent justifications are selected in Motivation multiSelect**
   (e.g., performance + security, each deserving its own rule).

### Distinction question

When any signal fires, ask via AskUserQuestion:

**Question:** "This context actually looks like two different rules. How should we
proceed?"

**Options:**
- **(Recommended)** Add as two separate rules — we'll clarify each separately
- Keep as a single rule — we'll expand the Rule statement to cover both clauses
- Let's focus on only one for now and handle the other later
- Misidentified — this is actually a single rule

### Post-decision flow

- **Two separate rules:** repeat Phase 2 independently per rule. Finalize the
  first, then the second. Don't mix rounds — each rule has its own answer set.
- **Keep as a single rule:** re-ask the Rule statement question with formulations
  that combine both clauses.
- **Focus on only one:** don't forget the other; at the end of Phase 4, offer to
  start a second round ("Shall we handle the other rule now?").
- **Misidentification:** return to normal flow, ignore the signal.

---

## Phase 4 — Consolidation and final approval

### 4.1 Generate the final rule text

Once questioning is complete, compose a **natural-language** rule text in the
user's language (matching the wizard) from the collected answers — this is the
**input for the /rule skill**.

It must contain everything `/rule` will parse:
- **Scope** (so it's clear which file it goes to)
- **Rule** (clear single-sentence statement)
- **Why** (motivation)
- **Apply when** (specific conditions)
- **Don't apply when** (if applicable)
- **Examples** (✅ and ❌)
- **Related** (if applicable)

**Example final text:**

> "Controller actions in the API project should not write try-catch blocks —
> error handling must be delegated to the global exception handler at the upper
> layer. This rule preserves architectural consistency and keeps controllers as
> thin bridges; try-catch is the responsibility of services or the global
> handler. Applies to: all controller actions in `.cs` files under
> `api/Controllers/`. Test code is exempt. Correct example: `[HttpPost] public
> async Task<IActionResult> Create(CreateProductRequest req) { var result = await
> _productService.CreateAsync(req); return Ok(result); }` — no try-catch. Wrong
> example: writing `try { ... } catch (Exception ex) { return
> BadRequest(ex.Message); }` inside a controller. Related rule:
> `no-logic-in-bridges`."

This can be one paragraph or a few sentences — but it is NOT compressed into
`Rule:` format; `/rule` does its own parsing.

### 4.2 Show to user and get approval

Show the generated final-rule text and ask via AskUserQuestion:

**Question:** "Is this text the final version of the rule? Can I add it now with
`/rule`?"

**Options:**
- **(Recommended)** Yes, add with `/rule`
- I need to correct a part of the text — let me tell you which part
- I think there's a missing area — let's do an additional question round
- Cancel, do not add for now

### 4.3 Invoking the /rule skill

When the user selects "Yes, add":

- **Single rule:** invoke `/rule <final text>`.
- **Multiple rules:** invoke each **sequentially**. Give a brief progress note
  between them:
  - "First rule written (`{id}` — `{file}`). Moving to the second rule now."
- **After each `/rule` invocation**, relay the result as a summary.

**If correction is selected:** the user states which part to fix. Re-ask the
corresponding (or closely related) question via AskUserQuestion, get the answer,
update the final text, and repeat 4.2.

**If additional round is selected:** run a new question round for the area the
user flagged as missing, then return to 4.1.

**If cancel is selected:** terminate cleanly. Write nothing. Tell the user: "Rule
was not written. You can start again with `/rule-wizard` whenever you want."

### 4.4 Final summary

When writing is complete, give a single summary message:

- How many rules were written
- Each rule's **ID** and which **file** it was written to
- Existing rules marked as **related**, if any
- Remind the user of any deferred rule from Phase 3 set aside for later

---

## Critical principles (summary)

1. **Context is mandatory.** No argument → ask the user for context.
2. **Every question has options.** AskUserQuestion is used; plain-text questions
   are never asked.
3. **If 4 options aren't enough, split.** 5+ reasonable options → split into two
   rounds. Never cap at "the best 4".
4. **Read existing rules first.** Mandatory prerequisite to catch duplication and
   conflicts early.
5. **Never assume.** Every field not clearly derived from context requires a
   question — even a confirmation question must be asked.
6. **Dynamically detect multiple rules.** Start single-rule, but ask the user when
   a divergence signal appears.
7. **Final text is in the user's language.** `/rule` translates to English — this
   skill mirrors the user's language.
8. **`/rule` is not invoked without approval.** Show the finalized text and get
   approval first.
9. **An incomplete field is worse than a nonexistent field.** The required fields
   (Rule, Why, Apply when, Examples) must be fully represented in the final text.
10. **The skill can run repeatedly for multiple rules.** If dynamic split mode was
    chosen in Phase 3, each rule goes through the Phase 2-4 cycle individually.

