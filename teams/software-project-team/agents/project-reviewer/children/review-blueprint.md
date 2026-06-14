---
knowledge-base-summary: "The primary production unit. Full project review process: architecture scan, dependency audit, tech debt assessment, documentation check, config review. Produces a health report with RAG status (Red/Amber/Green) per category. Master checklist for complete reviews."
---
# Review Blueprint

This is the primary production unit of the Project Reviewer agent. Every project health review follows this process.

## Review Process

### Step 1: Understand the Project

Before reviewing anything, read these files to understand what the project is supposed to be:

```
CLAUDE.md                                    -> Project overview, tech stack, architecture
README.md                                    -> Setup, usage, public documentation
.atl/docs/                                -> Detailed documentation
.atl/backlog.md                           -> Known deferred items
.atl/agent-memory/project-reviewer-memory.md -> Past review learnings
```

These define the intended state. The review measures actual state against intended state.

### Step 2: Architecture Scan

Read `children/architecture-review.md` and check:

1. Does the project structure match the documented architecture?
2. Are layer boundaries respected (no cross-layer imports)?
3. Is dependency direction correct (always inward)?
4. Are features organized consistently?
5. Are host applications thin (no business logic)?

### Step 3: Dependency Audit

Read `children/dependency-audit.md` and check:

1. Are packages up to date?
2. Are there known vulnerabilities?
3. Are there unused packages?
4. Are there version conflicts?
5. Are licenses compatible?

### Step 4: Technical Debt Assessment

Read `children/tech-debt.md` and check:

1. Count and categorize TODO/FIXME/HACK comments
2. Identify complex methods (high line count, deep nesting)
3. Spot code duplication
4. Find missing tests for critical paths
5. Detect hardcoded values and deprecated API usage

### Step 5: Documentation Check

Read `children/documentation-check.md` and check:

1. Does CLAUDE.md match the actual project?
2. Is README complete and accurate?
3. Are API endpoints documented (Swagger)?
4. Are there features without documentation?
5. Is documentation stale (references things that no longer exist)?

### Step 6: Configuration Review

Read `children/configuration-review.md` and check:

1. Is .env.example complete?
2. Are there hardcoded config values in code?
3. Are health checks in place?
4. Is Docker Compose correct?
5. Are secrets properly managed?

### Step 7: Scalability Assessment

Read `children/scalability-assessment.md` and check:

1. Will database queries scale with data growth?
2. Is caching used effectively?
3. Are services stateless (can add replicas)?
4. Is connection pooling configured?
5. Are resource limits set?

### Step 8: Produce the Health Report

Generate a structured report using the format below.

## Health Report Format

```markdown
# Project Health Report

**Project:** {name}
**Date:** {date}
**Reviewer:** project-reviewer

## Overall Health: {GREEN | AMBER | RED}

## Category Summary

| Category | Status | Findings | Critical |
|----------|--------|----------|----------|
| Architecture | {RAG} | {count} | {count} |
| Dependencies | {RAG} | {count} | {count} |
| Technical Debt | {RAG} | {count} | {count} |
| Documentation | {RAG} | {count} | {count} |
| Configuration | {RAG} | {count} | {count} |
| Scalability | {RAG} | {count} | {count} |

## Architecture

### Status: {RED | AMBER | GREEN}

{Paragraph summary}

#### Findings

- **[AR-{n}] {Title}** ({severity})
  - **Location:** {path or area}
  - **Expected:** {what should be}
  - **Actual:** {what is}
  - **Impact:** {what could go wrong}
  - **Recommendation:** {how to fix}

## Dependencies
{Same finding format}

## Technical Debt
{Same finding format}

## Documentation
{Same finding format}

## Configuration
{Same finding format}

## Scalability
{Same finding format}

## Recommendations

### Immediate (Red Items)
1. {Action item with specific location and fix}

### Short-term (Amber Items)
1. {Action item}

### Long-term (Green Improvements)
1. {Action item}

## Backlog Items Discovered
{List any new items that should be added to .atl/backlog.md}
```

## RAG Status Definitions

### RED -- Requires Immediate Attention
The project has critical issues that affect correctness, security, or stability. These must be addressed before shipping.

Examples:
- Known security vulnerabilities in dependencies
- Architecture violations that break data isolation
- Missing auth on endpoints that need it
- Secrets committed to source control

### AMBER -- Needs Improvement
The project works but has significant quality or maintainability concerns. These should be addressed in the next sprint.

Examples:
- Several outdated dependencies (but no vulnerabilities)
- Documentation doesn't match reality in several places
- Technical debt is accumulating without tracking
- Missing health checks

### GREEN -- Healthy
The category is in good shape. Minor improvements may be suggested but nothing urgent.

Examples:
- Architecture matches documentation
- All dependencies are current
- Technical debt is tracked in backlog
- Documentation is accurate

## Master Checklist

Before submitting a health report, verify:

- [ ] CLAUDE.md and project docs read and understood
- [ ] Architecture scan completed (layers, boundaries, direction)
- [ ] Dependency audit completed (outdated, vulnerable, unused)
- [ ] Technical debt assessed (TODOs, complexity, duplication, tests)
- [ ] Documentation checked (accuracy, completeness, freshness)
- [ ] Configuration reviewed (env, Docker, health checks, secrets)
- [ ] Scalability assessed (queries, caching, statelessness, limits)
- [ ] RAG status assigned per category
- [ ] Overall health status determined
- [ ] Immediate/short-term/long-term recommendations listed
- [ ] New backlog items identified and documented
