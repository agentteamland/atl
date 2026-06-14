# Dependency Versions — software-project-team

Centralized minimum-patched version pins for libraries the project agents reference. Every agent that mentions a versioned dependency should defer to this document for the canonical pin.

## Why this file exists

Each agent mentions packages it works with — `api-agent` lists `Microsoft.EntityFrameworkCore`, `react-agent` mentions Next.js, `rmq-agent` describes the email-consumer pattern using MailKit. Without a single source of truth, a CVE fix that lands via dependabot in one project leaves agent files in a stale-info state. Future projects opened by `create-new-project` then start with a known-vulnerable baseline.

This file is that single source of truth. When dependabot opens a security-fix PR and you merge it:

1. Update this file's **Min version** column for that package.
2. Add a one-line note in **Notes / CVEs** describing the fix.
3. Verify the agent file(s) referenced under **Referenced from** still describe the package correctly (no stale "version 14 has issue X" guidance after a major bump).

If `create-new-project` skill ever templates a `package.json` / `.csproj`, it must read minimum versions from this file — not hard-code its own.

## Pinned versions — .NET / NuGet

| Package | Min version | Range | Referenced from | Notes / CVEs |
|---|---|---|---|---|
| `Microsoft.EntityFrameworkCore` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | .NET 9 alignment. Floating minor for security patches. |
| `Microsoft.EntityFrameworkCore.Relational` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | |
| `Microsoft.EntityFrameworkCore.Design` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | EF migrations CLI. |
| `Npgsql.EntityFrameworkCore.PostgreSQL` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | EF Core PostgreSQL provider, .NET 9 line. |
| `Microsoft.Extensions.DependencyInjection.Abstractions` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | |
| `Microsoft.Extensions.Logging.Abstractions` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | |
| `Microsoft.Extensions.Localization.Abstractions` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | |
| `Microsoft.Extensions.Hosting` | `9.0.0` | `9.*` | rmq-agent/children/consumer-blueprint.md | Worker SDK consumer hosts pin this explicitly. |
| `Microsoft.Extensions.Options.ConfigurationExtensions` | `9.0.0` | `9.*` | api-agent/children/architecture-layers.md | |
| `MediatR` | `12.0.0` | `12.*` | api-agent/children/architecture-layers.md | CQRS dispatch. |
| `FluentValidation` | `11.0.0` | `11.*` | api-agent/children/architecture-layers.md | Request validation. |
| `FluentValidation.DependencyInjectionExtensions` | `11.0.0` | `11.*` | api-agent/children/architecture-layers.md | |
| `RabbitMQ.Client` | `6.8.1` | `6.*` | api-agent/children/architecture-layers.md, rmq-agent/* | **PINNED to 6.x** — 7.x is async-first and source-incompatible. Do not bump to 7 without an architectural plan. |
| `StackExchange.Redis` | `2.8.16` | `2.*` | api-agent/children/architecture-layers.md, redis-agent/* | |
| `BCrypt.Net-Next` | `4.0.0` | `4.*` | api-agent/children/architecture-layers.md | Password hashing. |
| `Microsoft.IdentityModel.Tokens` | `8.0.0` | `8.*` | api-agent/children/architecture-layers.md | JWT signing. |
| `System.IdentityModel.Tokens.Jwt` | `8.0.0` | `8.*` | api-agent/children/architecture-layers.md | |
| `AWSSDK.S3` | `3.7.0` | `3.7.*` | api-agent/children/architecture-layers.md | Object storage. Preferred over `Minio` package; works against MinIO and real S3. |
| **`MailKit`** | **`4.16.0`** | `4.*` (≥4.16.0) | rmq-agent/children/connection-management.md (mailsender pattern) | **CVE — STARTTLS Response Injection / SASL mechanism downgrade** (medium severity). Fixed in 4.16.0. MailKit lives inside the MailSender worker host (`Microsoft.NET.Sdk.Worker`). |

## Pinned versions — Node / npm

| Package | Min version | Range | Referenced from | Notes / CVEs |
|---|---|---|---|---|
| **`next`** | **`15.5.15`** | `^15.5.15` | react-agent/children/seo-meta.md (SSR/Next.js path), react-agent/children/dst-handoff.md, react-agent/agent.md | **15 alerts fixed in 15.5.15 including 2 CRITICAL**: (a) RCE in React flight protocol, (b) Authorization Bypass in Middleware. Plus 4 High + 8 Medium + 2 Low. Used by `react-public` target only. |
| `next-intl` | latest `4.x` | `^4.0.0` | react-agent/children/i18n.md | Open redirect (medium) — fixed in 4.x. |
| `react` | `19.0.0` | `19.x` | react-agent/agent.md | Pairs with Next 15 / shadcn / Vite. |
| `react-dom` | `19.0.0` | `19.x` | react-agent/agent.md | |
| **`vite`** | latest `7.x` | `^7.0.0` | react-agent/agent.md (Vite path for admin) | **CVE — Path Traversal in Optimized Deps `.map` Handling** (medium) — fixed in latest 7.x patch. Used by `react-admin` target. |
| **`esbuild`** | latest `0.25.x` | `^0.25.0` | react-agent (transitively via Vite) | **CVE — Dev server allows arbitrary cross-origin requests** (medium) — fixed in latest. |
| `i18next` | `23.x` | `^23.0.0` | react-agent/children/i18n.md | |
| `i18next-browser-languagedetector` | `8.x` | `^8.0.0` | react-agent/children/i18n.md | |
| `react-i18next` | `15.x` | `^15.0.0` | react-agent/children/i18n.md | |
| `tailwindcss` | `3.4.x` | `^3.4.0` | react-agent/* | Tailwind 3 line; 4 is breaking. |
| `eslint` | `8.57.x` | `^8.57.0` | react-agent/agent.md | Pre-flat-config; flat-config ecosystem still maturing. |

## Pinned versions — Flutter / Dart (pub)

Flutter package versions live alongside Flutter SDK constraint. Bump them in concert with Flutter SDK upgrades.

| Package | Min version | Range | Referenced from | Notes / CVEs |
|---|---|---|---|---|
| `flutter_riverpod` | `2.5.0` | `^2.5.0` | flutter-agent/agent.md | State management. |
| `go_router` | `14.0.0` | `^14.0.0` | flutter-agent/agent.md | Routing. |
| `dio` | `5.4.0` | `^5.4.0` | flutter-agent/children/dio-client.md | HTTP client. |
| `flutter_localizations` | (sdk) | bundled | flutter-agent/children/i18n.md | Bundled with Flutter SDK. |

## Reference data point — 2026-04-26

Current **Min version** pins were established by sweeping a reference project's closed dependabot alerts on 2026-04-26.

| Source bump | From → To |
|---|---|
| MailKit (NuGet, MailSender worker) | 4.8.0 → 4.16.0 |
| next (npm, public site) | 15.0.3 → 15.5.15 |
| **Total alerts fixed** | 19 (2 critical, 4 high, 11 medium, 2 low) |

Subsequent dependabot bumps from any project update this file.

## Bump checklist (when dependabot PR lands)

1. Read the security advisory (severity, attack vector, affected versions).
2. Check whether this file lists the package. If not — flag it; orphan dependency.
3. If listed: update the **Min version** to the patched release.
4. Add a one-line note under **Notes / CVEs** identifying the fix and severity.
5. Cross-reference the agent file(s) under **Referenced from** — does any version-specific guidance there go stale? Update if so.
6. Merge the PR.
7. Commit this file's update with the same PR (or a follow-up if the PR is auto-generated).

## Why floating ranges (`9.*`, `6.*`, `^15.5.15`)?

Floating minor ranges let dependabot deliver security patches without manual range edits. The **Min version** column is what new projects start with; the **Range** column is what's allowed without an explicit human bump.

Exceptions (do NOT float major):
- `RabbitMQ.Client` 6.x — 7.x is async-first and source-incompatible.
- `tailwindcss` 3.x — 4 is breaking.
- `eslint` 8.x — flat-config ecosystem not stable.
