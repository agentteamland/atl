# `/rule`

Add a coding or architecture rule. The user describes it in natural language (any language); the skill writes it in **English structured format** to the correct file.

For complex or ambiguous rules where multiple formulations are possible, use [`/rule-wizard`](/skills/rule-wizard) instead — it walks through option-based Q&A rounds before invoking `/rule` to write the final form.

Ships as a global skill in the [atl monorepo](https://github.com/agentteamland/atl).

## Two scopes

| Flag | Target | When |
|---|---|---|
| *(none)* | Project `.atl/` files | Rules specific to this project (default) |
| `--global` | `~/.atl/rules/` | Personal rules that apply to every project |

## Flow

### 1. Analyze the rule

From the user's natural-language statement, extract:

- **Topic** — coding, architecture, naming, error handling, etc.
- **Scope** — which application(s) does it affect
- **Motivation** — *why* this rule (if not stated, derive a reasonable Why; if uncertain, ask)

### 2. Determine the target file

**Project scope (default):**

| Applicability | File |
|---|---|
| Common to all applications | `.atl/rules/coding-common.md` |
| A specific application | `.atl/docs/coding-standards/{app}.md` (selected from existing files) |

**Global scope (`--global`):**

| Applicability | File |
|---|---|
| General rule | `~/.atl/rules/{topic}.md` (append if exists, create if not) |

If the rule applies to more than one but not all, the skill asks.

### 3. Check existing rules

**Always read** the target file. Three situations:

- **Entirely new rule** → add as a new section
- **Extending / updating an existing rule** → update in-place; do not duplicate
- **Conflict** (two rules contradict each other) → ask the user; do not assume

### 4. Write in structured format

Detailed and clear, in English. **An incomplete rule is more dangerous than no rule at all.**

```markdown
### {kebab-case-rule-id}
**Rule:** {Clear statement of the rule in a single sentence}

**Why:** {Motivation. What problem does it prevent? What principle does it support?
Include lessons from past mistakes if applicable. This field must not be empty or vague.}

**Apply when:** {Under what circumstances — file paths, code patterns,
what types of changes? Be specific.}

**Don't apply when:** {(Optional) Explicitly state exceptions.}

**Examples:**
- ✅ Correct: {code example or concrete scenario}
- ❌ Wrong: {code example or concrete scenario}

**Related:** {(Optional) Related rule IDs}
```

### 5. Writing rules (critical)

- **Never assume.** If information is missing, ask.
- **Don't keep it short — explain.** Skipped detail = unenforced rule.
- **Capture edge cases.** Add `Don't apply when` when applicable.
- **Provide examples.** Both ✅ and ❌.
- **Assign a unique ID.** Read the file first to avoid conflicts.

### 6. Write and verify

Update the target file via `Edit`. Give the user a brief summary: which file and which ID.

## Important rules

1. **Language:** The user may invoke the skill in any language; the skill always **writes the rule in English**.
2. **Ask if information is missing.** Never fill in gaps on your own.
3. **No duplication.** Read existing rules first.
4. **Validate file paths.** Wrong scope → wrong file.
5. **No format deviations.** All required fields filled: Rule, Why, Apply when, Examples.

## Related

- [`/rule-wizard`](/skills/rule-wizard) — option-based clarification wizard for ambiguous rules; invokes `/rule` at the end
- [Concepts: Rule](/guide/concepts#rule) — what rules are and how they're loaded

## Source

- Spec: [core/skills/rule/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/rule/SKILL.md)
