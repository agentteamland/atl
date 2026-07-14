---
name: rule
description: "Add a new coding/architecture rule. The user describes it in natural language (any language), the skill writes it in English structured format to the correct file. 2 scopes: project (default), --global."
argument-hint: "[--global] <rule in natural language, any language>"
---

# /rule — capture a coding/architecture rule

Analyzes a rule expressed in natural language (any language) and writes it to the
correct file in English structured format. A rule is enforced knowledge: it must
be detailed enough to actually apply later.

---

## Two scopes (the v2 scope axis)

ATL has one scope axis — **project** (the default) vs **global** (`~/.atl`),
isomorphic with Claude Code's layering (project shadows global).

| Flag | Target | When |
|---|---|---|
| *(none)* | `<project>/.atl/` rule files | Rules specific to this project (default) |
| `--global` | `~/.atl/rules/` | Personal rules that apply to every project |

> **How the rule actually loads:** `.atl/rules/` is the durable source; `atl
> session-start` reflects `.atl/rules/*.md` into the matching `.claude/rules/`
> (the surface Claude Code loads), so a rule written **to `.atl/rules/`** takes
> effect from the next session. The `.claude/rules/` copy is derived — `atl gc`
> reclaims it only once you delete the `.atl/rules/` source. (App-specific files
> under `.atl/docs/coding-standards/` are on-demand reference, not always-loaded
> rules, and are not reflected.)

---

## Flow

### 1. Analyze the rule
Extract from the user's natural-language expression:
- **Topic:** what kind of rule? (coding, architecture, naming, error handling, …)
- **Scope:** which application(s) / areas does it affect?
- **Motivation:** why this rule? (If not explicit, derive a reasonable Why — if
  unsure, ask.)

### 2. Determine the target file

**Project scope (default):**

Look under the project's `.atl/docs/coding-standards/` directory and identify the
existing per-application files.

| Scope | File |
|---|---|
| Common to all applications | `<project>/.atl/rules/coding-common.md` |
| A specific application | `<project>/.atl/docs/coding-standards/{app}.md` (pick from existing files) |

If the project has no `.atl/rules/` or `.atl/docs/coding-standards/` yet, create
the directory and the file as needed.

**Global scope (`--global`):**

| Scope | File |
|---|---|
| General rule | `~/.atl/rules/{topic}.md` (append to the existing file, or create if none) |

### 3. Check existing rules
**Always read** the target file first. Three situations:
- **Entirely new rule:** add as a new section.
- **Extending/updating an existing rule:** update in place — no duplication.
- **Conflict:** two rules contradict — ask the user; do not assume.

### 4. Write in structured format
In English, **detailed and clear** — not brief. An incomplete rule is more
dangerous than no rule at all.

```markdown
### {kebab-case-rule-id}
**Rule:** {Clear statement of the rule in a single sentence}

**Why:** {Motivation. What problem does it prevent? What principle does it support?
Include lessons from past mistakes if applicable. This field must not be left empty or vague.}

**Apply when:** {Under what circumstances does it apply — file paths, code patterns,
what types of changes? Be specific.}

**Don't apply when:** {(Optional) Explicitly state exceptions if any.}

**Examples:**
- ✅ Correct: {code example or concrete scenario}
- ❌ Wrong: {code example or concrete scenario}

**Related:** {(Optional) Related rule IDs}
```

### 5. Writing rules (CRITICAL)
- **Never assume.** If information is missing, ask.
- **Don't keep it short — explain.** Skipped detail = unenforced rule.
- **Capture edge cases.** Add `Don't apply when`.
- **Provide examples.** Both ✅ and ❌.
- **Assign a unique ID.** Read the file first to avoid conflicts.

### 6. Write and verify
- Update the target file with Edit.
- Give the user a brief summary: which file and which ID it was written to.

---

## Important rules

1. **Language:** the user may invoke the skill in any language; the skill always
   writes the rule in English.
2. **Ask if information is missing.** Never fill gaps on your own.
3. **No duplication.** Read existing rules first.
4. **Validate file paths.** Wrong scope → the rule goes to the wrong file.
5. **No format deviations.** All required fields must be filled: Rule, Why, Apply
   when, Examples.

