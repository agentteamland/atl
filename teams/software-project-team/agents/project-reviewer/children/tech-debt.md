---
knowledge-base-summary: "Scan for TODO/FIXME/HACK comments, code duplication, complex methods, missing tests, hardcoded values, and deprecated API usage. Categorize by severity and effort. Every debt item should be tracked in the backlog."
---
# Technical Debt Assessment

## TODO/FIXME/HACK Comment Scan

### How to Scan

```bash
# Find all TODO, FIXME, HACK, TEMP, WORKAROUND comments
grep -rn "TODO\|FIXME\|HACK\|TEMP\|WORKAROUND\|XXX" src/ --include="*.cs" --include="*.ts" --include="*.tsx" --include="*.dart"
```

### Categorization

| Marker | Meaning | Severity |
|--------|---------|----------|
| TODO | Planned work, not yet done | Amber |
| FIXME | Known bug, needs fixing | Red |
| HACK | Temporary workaround, known to be wrong | Red |
| TEMP | Temporary code, should be removed | Amber |
| WORKAROUND | Working around a limitation | Amber |
| XXX | Requires attention | Amber |

### What to Check for Each

1. **Is it tracked?** Does a corresponding item exist in `.atl/backlog.md`?
2. **How old is it?** Check git blame -- TODOs older than 3 months are likely forgotten.
3. **Is it blocking?** Does the TODO affect correctness or security, or is it just a nice-to-have?
4. **Is it actionable?** Does the comment explain what needs to be done, or is it just "TODO: fix this"?

### Report Format

```
**File:** {path}:{line}
**Marker:** {TODO|FIXME|HACK}
**Comment:** "{the comment text}"
**Age:** {days since added, from git blame}
**Tracked:** {Yes (backlog item) | No}
**Severity:** {Red|Amber}
```

## Code Duplication Detection

### What to Look For

- **Copy-pasted handlers:** Two handlers that do almost the same thing with minor differences.
- **Repeated query patterns:** The same database query appearing in multiple handlers.
- **Duplicate validation rules:** The same validation logic in multiple validators.
- **Repeated mapping code:** Entity-to-DTO mapping written inline in multiple places instead of a shared mapper.

### How to Detect

```bash
# Look for suspiciously similar method signatures
grep -rn "async ValueTask.*Handle" src/ --include="*.cs"

# Look for repeated patterns
grep -rn "await _db\." src/ --include="*.cs" | sort | uniq -c | sort -rn | head -20
```

### Severity

- Same code in 2 places: Amber (DRY violation but manageable)
- Same code in 3+ places: Red (one change = multiple places to update = bugs)

## Complex Methods

### What to Check

- **Line count:** Methods longer than 30 lines are a smell. Longer than 50 is a problem.
- **Nesting depth:** More than 3 levels of indentation (if inside if inside foreach) is hard to follow.
- **Parameter count:** Methods with more than 5 parameters should use a request/options object.
- **Multiple return paths:** More than 3 return statements in a method makes it hard to reason about.

### How to Detect

```bash
# Find long files (potential complex methods)
find src/ -name "*.cs" -exec wc -l {} \; | sort -rn | head -20

# Find deeply nested code (multiple levels of braces)
grep -rn "                        " src/ --include="*.cs" | head -20
```

### Severity

| Metric | Amber | Red |
|--------|-------|-----|
| Method length | 30-50 lines | 50+ lines |
| Nesting depth | 3-4 levels | 5+ levels |
| Parameters | 5-7 | 8+ |
| Return paths | 3-4 | 5+ |

## Missing Tests for Critical Paths

### What to Check

Critical paths that MUST have tests:

| Critical Path | Why |
|--------------|-----|
| Authentication (login, register, token refresh) | Security: broken auth = compromised system |
| Authorization (role checks, ownership) | Security: broken auth = data leak |
| Payment/financial operations | Business: wrong calculation = money loss |
| Data validation | Integrity: invalid data = corruption |
| Core business rules | Business: wrong behavior = wrong outcomes |

### How to Detect

1. Identify critical handlers by scanning for auth, payment, and core business features
2. Check if corresponding test files exist
3. Check if test coverage is meaningful (not just testing that the handler doesn't throw)

### Severity

- No tests on auth/payment: Red
- No tests on core business logic: Amber
- No tests on CRUD operations: Green (suggestion)

## Hardcoded Values

### What to Look For

- **Configuration values in code:**
  ```csharp
  // BAD: Should be in appsettings
  var connectionString = "Host=localhost;Database=mydb";
  var timeout = 30;
  var maxRetries = 3;
  ```
- **Feature flags as boolean constants:**
  ```csharp
  // BAD: Should be configurable
  const bool EnableNewFeature = false;
  ```
- **URLs in code:**
  ```csharp
  // BAD: Should be in configuration
  var apiUrl = "https://api.example.com/v1";
  ```

### How to Detect

```bash
# Find string literals that look like configuration
grep -rn '"http\|"Host=\|"Server=\|"localhost\|"127.0.0.1' src/ --include="*.cs"

# Find numeric constants that might be config
grep -rn "const int\|const double\|const float" src/ --include="*.cs"
```

## Deprecated API Usage

### What to Check

- Methods marked with `[Obsolete]` attribute being called
- Use of APIs that are deprecated in the current framework version
- Patterns that the project has moved away from but old code still uses

### How to Detect

```bash
# Find Obsolete attribute usage
grep -rn "\[Obsolete\]" src/ --include="*.cs"

# Check for compiler warnings about deprecated APIs
dotnet build 2>&1 | grep -i "obsolete\|deprecated"
```

## Debt Prioritization Matrix

| Effort \ Impact | High Impact | Medium Impact | Low Impact |
|----------------|------------|---------------|------------|
| Low Effort | Do Now | Do Soon | Do When Convenient |
| Medium Effort | Plan for Next Sprint | Backlog | Skip |
| High Effort | Plan as Project | Backlog | Skip |

## Review Checklist

- [ ] TODO/FIXME/HACK comments scanned and categorized
- [ ] Each marker checked for backlog tracking
- [ ] Code duplication identified
- [ ] Complex methods flagged (line count, nesting, parameters)
- [ ] Critical paths checked for test coverage
- [ ] Hardcoded values detected
- [ ] Deprecated API usage found
- [ ] All findings prioritized using effort/impact matrix
- [ ] Untracked debt items recommended for backlog
