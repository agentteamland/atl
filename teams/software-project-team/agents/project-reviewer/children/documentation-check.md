---
knowledge-base-summary: "Verify CLAUDE.md accuracy, README completeness, API documentation coverage, and documentation freshness. Documentation that doesn't match the code is a finding."
---
# Documentation Check

## CLAUDE.md Accuracy

### What to Check

CLAUDE.md is the project's source of truth. Every section must match reality.

| Section | Verification |
|---------|-------------|
| Technology list | Are all listed technologies actually used in the code? |
| Project structure | Does the directory tree match the actual structure? |
| Development commands | Do the listed commands actually work? |
| Port mappings | Are the documented ports correct in docker-compose? |
| Rules | Are the documented rules followed in the code? |

### How to Verify

1. **Technology list:** For each technology mentioned, search the codebase for evidence of its use (imports, config, Docker services).
2. **Project structure:** Compare the documented tree with `ls -R src/` output.
3. **Development commands:** Check docker-compose.yml for service names and ports.
4. **Port mappings:** Match documented ports against docker-compose port bindings.
5. **Rules:** Spot-check rules against actual code patterns.

### Common Findings

- Technology listed but not yet implemented (aspirational documentation)
- Directory listed but doesn't exist or is empty
- Port changed in docker-compose but not updated in CLAUDE.md
- Rule documented but consistently violated in code
- Service added to docker-compose but not documented

### Severity

- Technology/service mismatch: Amber
- Command that doesn't work: Red
- Port mismatch: Amber
- Rule violation: Amber (if minor), Red (if systemic)

## README Completeness

### Required Sections

Every README should have:

| Section | Content | Check |
|---------|---------|-------|
| Description | What the project does | Present and accurate? |
| Prerequisites | What you need installed | Complete? |
| Setup | How to get running locally | Steps work? |
| Development | How to develop (commands, tools) | Accurate? |
| Architecture | High-level structure overview | Matches reality? |
| Contributing | How to contribute (if open source) | Present? |

### Common Findings

- Missing setup steps (assumes knowledge)
- Outdated prerequisites (wrong version numbers)
- Development commands that don't work
- Architecture diagram that doesn't match current state
- No mention of required environment variables

## API Documentation (Swagger/OpenAPI)

### What to Check

1. **Coverage:** Are all endpoints documented in Swagger?
2. **Request/Response models:** Are request and response schemas accurate?
3. **Auth requirements:** Does Swagger show which endpoints need auth?
4. **Examples:** Are request/response examples provided?
5. **Error responses:** Are error response schemas documented?

### How to Verify

1. List all endpoints from the code (scan for MapGet, MapPost, MapPut, MapDelete)
2. Compare against Swagger output (hit `/swagger` endpoint)
3. Check if new endpoints were added without Swagger annotations

### Common Findings

- Endpoints exist in code but not in Swagger
- Swagger shows old response format (schema changed, Swagger not updated)
- Auth requirements not reflected in Swagger
- Missing response codes (only 200 documented, no 404/422/500)

## Missing Documentation for Features

### What to Check

1. **Scan for features:** List all feature folders in `Application/Features/`
2. **Check for docs:** Does each feature have corresponding documentation?
3. **Check for decision records:** Major features should have a brainstorm or ADR explaining why they were built this way

### Documentation Needs by Feature Type

| Feature Type | Documentation Needed |
|-------------|---------------------|
| Auth/Security | How auth works, token flow, role hierarchy |
| Business logic | What the rules are, edge cases |
| Integration | How external services are called, error handling |
| Background jobs | What they do, when they run, what happens on failure |
| Data model | Entity relationships, invariants |

## Stale Documentation

### How to Detect

1. **Check git blame on docs:** Documentation not modified in 3+ months while related code changed frequently.
2. **Search for references to non-existent things:** Documentation mentioning classes, methods, or endpoints that no longer exist.
3. **Compare dates:** If `README.md` was last modified 6 months ago but `src/` has daily commits, the README is likely stale.

### How to Scan

```bash
# Last modification date of documentation files
find . -name "*.md" -exec stat -f "%m %N" {} \; | sort -rn

# Search for references to files that don't exist
# (Manual: Read docs, note class/file names, verify they exist)
```

### Common Staleness Indicators

- Documentation references a technology no longer used
- Setup instructions mention a step that was automated
- Architecture docs show a component that was removed
- API docs describe endpoints that were renamed or deleted

## Documentation Quality Checks

### Style and Consistency

- Consistent heading levels
- Code examples are syntax-highlighted and runnable
- Links are not broken
- No placeholder text ("TODO: fill this in")
- Consistent terminology (don't call the same thing by different names)

### Completeness Criteria

A feature is considered "documented" when:
1. Its purpose is explained (what problem it solves)
2. Its usage is shown (how to use it)
3. Its configuration is listed (what settings it needs)
4. Its edge cases are noted (what can go wrong)

## Review Checklist

- [ ] CLAUDE.md technology list matches actual usage
- [ ] CLAUDE.md project structure matches actual directories
- [ ] CLAUDE.md development commands work
- [ ] CLAUDE.md port mappings match docker-compose
- [ ] CLAUDE.md rules are followed in code
- [ ] README has all required sections
- [ ] README setup instructions work
- [ ] All endpoints appear in Swagger
- [ ] Features have corresponding documentation
- [ ] No stale documentation (references to non-existent code)
- [ ] No placeholder text in documentation
