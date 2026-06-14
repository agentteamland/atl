# `/rule`

Add a coding or architecture rule. The user describes it in natural language (any language); the skill writes it in **English structured format** to the correct file.

For complex or ambiguous rules where multiple formulations are possible, use [`/rule-wizard`](/skills/rule-wizard) instead — it walks through option-based Q&A rounds before invoking `/rule` to write the final form.

Ships as a global skill in [rule](https://github.com/agentteamland/rule).

## Three scopes

| Flag | Target | When |
|---|---|---|
| *(none)* | Project `.claude/` files | Rules specific to this project (default) |
| `--global` | `~/.claude/rules/` | Personal rules that apply to every project |
| `--team` | `~/.claude/repos/agentteamland/{team}/` files | Agent or team-rule files in the team repo |

For `--team`, active team is detected from installed `.claude/agents/` symlinks. Single team → used automatically; multiple teams → asks via `AskUserQuestion`.

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
| Common to all applications | `.claude/rules/coding-common.md` |
| A specific application | `.atl/docs/coding-standards/{app}.md` (selected from existing files) |

**Global scope (`--global`):**

| Applicability | File |
|---|---|
| General rule | `~/.claude/rules/{topic}.md` (append if exists, create if not) |

**Team scope (`--team`):**

| Related area | File |
|---|---|
| An agent's knowledge base | `~/.claude/repos/agentteamland/{team}/agents/{agent}.md` |
| Team-wide rule | `~/.claude/repos/agentteamland/{team}/rules/{topic}.md` |

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

### 7. Persisting team-scope rules

Team rules live under the team's local clone. Every public `agentteamland/{team}` repo is branch-protected, so direct push to `origin/main` is refused. Open a PR instead:

```bash
cd ~/.claude/repos/agentteamland/{team-name}
git checkout -b rule/{kebab-case-rule-id}
git add rules/{file}.md team.json
git commit -m "rule: {kebab-case-rule-id}"
git push -u origin rule/{kebab-case-rule-id}
gh pr create --fill
```

[`/create-pr`](/skills/create-pr) automates this if installed.

## Important rules

1. **Language:** The user may invoke the skill in any language; the skill always **writes the rule in English**.
2. **Ask if information is missing.** Never fill in gaps on your own.
3. **No duplication.** Read existing rules first.
4. **Validate file paths.** Wrong scope → wrong file.
5. **No format deviations.** All required fields filled: Rule, Why, Apply when, Examples.
6. **Team-scope rules ship via PR, not direct push.** Branch-protected; the skill writes locally and points at PR creation.

## Related

- [`/rule-wizard`](/skills/rule-wizard) — option-based clarification wizard for ambiguous rules; invokes `/rule` at the end
- [Concepts: Rule](/guide/concepts#rule) — what rules are and how they're loaded
- [team-repo-maintenance rule](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md) — discipline for shipping rule changes upstream

## Source

- Spec: [rule/skills/rule/skill.md](https://github.com/agentteamland/rule/blob/main/skills/rule/skill.md)
