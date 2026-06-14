---
knowledge-base-summary: "Check for outdated, vulnerable, and unused packages. NuGet and npm audit. Version conflicts and license compliance. Update strategy and risk assessment."
---
# Dependency Audit

## Outdated Packages

### How to Check (.NET)

```bash
# List all outdated NuGet packages
dotnet list package --outdated

# Inside Docker (if project only runs in Docker)
docker compose exec api dotnet list package --outdated
```

### How to Check (npm/Node.js)

```bash
# List outdated npm packages
npm outdated

# Or for a specific directory
cd src/frontend && npm outdated
```

### Severity Assessment

| Gap | Severity | Action |
|-----|----------|--------|
| Patch version behind (1.0.0 -> 1.0.3) | Green | Update at convenience |
| Minor version behind (1.0.0 -> 1.2.0) | Amber | Update within sprint |
| Major version behind (1.0.0 -> 2.0.0) | Amber | Plan migration, check breaking changes |
| 2+ major versions behind | Red | Urgent -- likely missing security patches |

## Known Vulnerabilities

### How to Check (.NET)

```bash
# Check for known vulnerabilities
dotnet list package --vulnerable

# With transitive dependencies
dotnet list package --vulnerable --include-transitive
```

### How to Check (npm)

```bash
# Check for vulnerabilities
npm audit

# JSON output for parsing
npm audit --json
```

### Severity Mapping

| Vulnerability Level | Report Severity |
|--------------------| --------------|
| Critical | Red |
| High | Red |
| Medium | Amber |
| Low | Green (suggestion) |

### Every vulnerability finding should include:

- Package name and version
- Vulnerability description (CVE if available)
- Affected functionality
- Fixed version (if available)
- Migration effort estimate

## Unused Packages

### How to Detect

1. **Check project files (.csproj):** List all PackageReference entries
2. **Search for usage:** For each package, search the codebase for its namespace
3. **Flag if unused:** If a package is referenced but never imported, it's dead weight

```bash
# List all package references in a project
grep -r "PackageReference" src/{Project}/{Project}.csproj

# Check if the package namespace is used anywhere
grep -r "using {PackageNamespace}" src/{Project}/
```

### Common Unused Package Indicators

- Package installed "for later" but never used
- Package replaced by another but not removed
- Test package in production project
- Debug/profiling package left in production

## Version Conflicts

### What to Check

- Multiple projects referencing different versions of the same package
- Transitive dependency conflicts (A needs PackageX 1.0, B needs PackageX 2.0)
- Framework version mismatches between projects in the same solution

### How to Detect (.NET)

```bash
# Check for version inconsistencies across projects
dotnet list package --include-transitive | grep -i "conflict\|mismatch"
```

### Resolution Strategy

- Use Directory.Build.props for centralized version management
- Use `<ManagePackageVersionsCentrally>` in newer .NET
- Pin transitive versions when conflicts arise

## License Compliance

### What to Check

- All packages use licenses compatible with the project's license
- No GPL-licensed packages in proprietary projects (unless approved)
- No packages with "unknown" or missing licenses

### Common License Compatibility

| License | Commercial Use | Copyleft | Notes |
|---------|---------------|----------|-------|
| MIT | Yes | No | Most permissive |
| Apache 2.0 | Yes | No | Patent grant included |
| BSD | Yes | No | Similar to MIT |
| LGPL | Yes | Partial | Must allow relinking |
| GPL | Depends | Yes | Must open-source if distributed |
| AGPL | Depends | Yes | Must open-source even for SaaS |

## Update Strategy

### Recommended Approach

1. **Patch updates:** Apply immediately (bug fixes, security patches)
2. **Minor updates:** Apply within 1-2 sprints (new features, no breaking changes)
3. **Major updates:** Plan as a dedicated task (breaking changes, migration guide)

### Risk Assessment per Package

| Factor | Higher Risk | Lower Risk |
|--------|------------|------------|
| Usage scope | Used everywhere | Used in one place |
| Criticality | Auth, database, API | Logging, formatting |
| Breaking changes | API surface changed | Internal only |
| Test coverage | No tests | Well-tested integration |

### Report Format for Each Package

```
**Package:** {Name} {CurrentVersion} -> {LatestVersion}
**Type:** {NuGet|npm}
**Projects:** {list of projects using it}
**Risk:** {Low|Medium|High}
**Action:** {Update now|Plan migration|Monitor}
**Notes:** {Breaking changes, migration guide link, etc.}
```

## Review Checklist

- [ ] All packages checked for updates (dotnet list package --outdated)
- [ ] Vulnerability scan completed (dotnet list package --vulnerable)
- [ ] Unused packages identified
- [ ] Version conflicts detected and documented
- [ ] License compliance verified
- [ ] Update recommendations provided with risk assessment
- [ ] Critical/high vulnerabilities flagged as Red status
