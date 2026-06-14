---
knowledge-base-summary: "The primary production unit. Step-by-step review process: read coding-standards, scan changes, check each category, produce report. Defines the review report format with severity levels (critical/warning/suggestion) and the master checklist per category."
---
# Review Blueprint

This is the primary production unit of the Code Reviewer agent. Every code review follows this process.

## Review Process

### Step 1: Read Project Standards

Before reviewing any code, read these files (if they exist):

```
.atl/docs/coding-standards/{app}.md    -> Project-specific conventions
CLAUDE.md                                  -> Project overview and rules
.atl/agent-memory/code-reviewer-memory.md -> Past learnings
```

These files define the rules. Review against them, not personal preference.

### Step 2: Scan the Changes

Get the full picture before diving into details:

1. List all changed files -- understand the scope
2. Identify the feature/fix being implemented
3. Check if the change spans multiple layers (Domain, Application, Infrastructure, Api)
4. Note any new files vs modified files

### Step 3: Category Review

Run through each review category in order of priority:

| Priority | Category | Children File | Severity Default |
|----------|----------|---------------|-----------------|
| 1 | Security | security-scan.md | Critical |
| 2 | Error Handling | error-handling-review.md | Warning |
| 3 | Performance | performance-check.md | Warning |
| 4 | SOLID Principles | solid-check.md | Suggestion |
| 5 | Naming & Conventions | naming-review.md | Suggestion |
| 6 | Logging | logging-review.md | Suggestion |
| 7 | API Design | api-review.md | Warning |

For each category, read the corresponding children file and apply its checklist to the changed code.

### Step 4: Produce the Report

Generate a structured review report using the format below.

## Review Report Format

```markdown
# Code Review Report

**Scope:** {Brief description of what was changed}
**Files Reviewed:** {count}
**Date:** {date}

## Summary

| Severity | Count |
|----------|-------|
| Critical | {n}   |
| Warning  | {n}   |
| Suggestion | {n} |

## Critical Findings

### [CR-{n}] {Title}
- **File:** `{path}:{line}`
- **Category:** {Security|Performance|ErrorHandling|...}
- **Problem:** {What is wrong}
- **Why it matters:** {Why this is a problem}
- **Suggested fix:** {How to fix it}

## Warnings

### [WR-{n}] {Title}
- **File:** `{path}:{line}`
- **Category:** {category}
- **Problem:** {What is wrong}
- **Why it matters:** {Why this is a problem}
- **Suggested fix:** {How to fix it}

## Suggestions

### [SG-{n}] {Title}
- **File:** `{path}:{line}`
- **Category:** {category}
- **Observation:** {What could be improved}
- **Suggested improvement:** {How to improve it}

## Verdict

{APPROVE | APPROVE_WITH_COMMENTS | REQUEST_CHANGES}

{One paragraph summary of the overall quality and what needs to happen next}
```

## Severity Definitions

### Critical
Must be fixed before merge. The code is broken, insecure, or will cause data loss/corruption.

Examples:
- SQL injection vulnerability
- Auth bypass (missing authorization)
- Data leak (returning entity instead of DTO)
- Secrets hardcoded in code
- Unbounded query that will crash with real data

### Warning
Should be fixed before merge. The code works but has significant quality or maintainability issues.

Examples:
- N+1 query (works but slow)
- Missing validation on user input
- Generic exception catch swallowing errors
- Missing logging in handler
- Inconsistent error response format

### Suggestion
Nice to have. The code is fine but could be better.

Examples:
- Naming doesn't match convention (but is readable)
- SOLID principle could be applied more cleanly
- Logging message could be more descriptive
- Code could be simplified

## Master Checklist

Before submitting a review, verify you have checked:

- [ ] Read project coding standards
- [ ] Security scan completed (injection, auth, data leak, secrets)
- [ ] Error handling reviewed (no try-catch in handlers, no swallowed exceptions)
- [ ] Performance checked (N+1, unbounded queries, async, allocations)
- [ ] SOLID principles checked (SRP, OCP, LSP, ISP, DIP)
- [ ] Naming reviewed against project conventions
- [ ] Logging reviewed (completeness, templates, no sensitive data)
- [ ] API design reviewed (REST, auth, rate limiting, idempotency)
- [ ] Report generated with all findings categorized
- [ ] Verdict assigned (APPROVE, APPROVE_WITH_COMMENTS, REQUEST_CHANGES)
