---
knowledge-base-summary: "User + Profile + Tenant model. Optional — disabled by default. User = identity (auth), Profile = context (which tenant, which role), Tenant = isolation boundary (name varies: Dealer, Seller, Organization). Three auth flows: standard login, tenant-scoped embed/auto-login, tenant-scoped domain. Global query filter + interceptor for automatic data isolation."
---
# Multi-Tenancy: User + Profile + Tenant Model

## Philosophy

Multi-tenancy is **optional**. When the project is not SaaS, none of these components exist — no Profile, no Tenant, no ITenantEntity, no filters. The codebase stays clean. When SaaS is enabled via `/create-new-project`, the full infrastructure is scaffolded.

## Core Model

```
User              → Identity (email, password, auth)
    ↓ has many
Profile           → Context (which tenant, which role)
    ↓ belongs to
Tenant            → Isolation unit (Dealer, Company, Seller, Workspace — name varies per project)
```

**User:** Single record, single email, single auth. One human = one user. Auth always happens against User.

**Profile:** The user's face within a specific tenant. Same user can have profiles in different tenants (e.g., a customer registered with multiple dealers).

**Tenant:** The data isolation boundary. Name and depth vary per project.

### Entity Structures

```csharp
// User — always present, auth target
public class User : BaseEntity, IAuditableEntity
{
    public required string Email { get; set; }
    public required string PasswordHash { get; set; }
    public string? FirstName { get; set; }
    public string? LastName { get; set; }
    public bool EmailVerified { get; set; }
    public DateTime? EmailVerifiedAt { get; set; }

    public ICollection<Profile> Profiles { get; set; } = [];
}

// Profile — user's context within a tenant
public class Profile : BaseEntity
{
    public Guid UserId { get; set; }
    public Guid? TenantId { get; set; }       // null = system admin (no tenant)
    public string Role { get; set; } = null!;  // tenant-scoped role
    public bool IsActive { get; set; } = true;

    public User User { get; set; } = null!;
    public TenantBase? Tenant { get; set; }
}

// TenantBase — abstract, project provides concrete implementation
public abstract class TenantBase : BaseEntity, IAuditableEntity
{
    public required string Name { get; set; }
    public required string Code { get; set; }  // unique, public identifier
    public TenantStatus Status { get; set; } = TenantStatus.Active;

    public ICollection<Profile> Profiles { get; set; } = [];
}

// Example: PG uses Dealer as tenant
// public class Dealer : TenantBase { ... }
// Example: Marketplace uses Seller
// public class Seller : TenantBase { ... }
```

### Per-Project Mapping

| Project Type | Tenant Entity | Example |
|-------------|---------------|---------|
| Simple B2C (pedometer app) | *(none — SaaS disabled)* | User only |
| B2B2C (prior SaaS reference) | `Dealer` | Dealer → Customer profiles |
| Marketplace (trendyol-like) | `Seller` | Seller → shop employees + buyers |
| Enterprise SaaS (slack-like) | `Organization` | Organization → members |

## Three Auth Flows

### Flow 1 — Standard Login

Normal web/mobile login. User enters email + password.

```
Client: POST /api/auth/login { email, password }
    ↓
API: Validate user credentials
    ↓
API: Fetch user's profiles
    ├── Single profile → auto-select → issue JWT
    └── Multiple profiles → return profile list
        ↓
Client: POST /api/auth/select-profile { profileId }
        ↓
API: Issue JWT (userId, profileId, tenantId, role)
```

**JWT Claims:**
```json
{
  "sub": "user-uuid",
  "profileId": "profile-uuid",
  "tenantId": "tenant-uuid",
  "role": "customer",
  "email": "user@example.com"
}
```

### Flow 2 — Tenant-Scoped Auto Login (Embed / iframe)

Tenant's own application opens the system. Tenant is already known. User is created if not exists, customer profile is created if not exists. No selection screen — automatic.

```
Tenant Backend: POST /api/embed/auth {
    tenantToken: "public-token",      // identifies the tenant
    tenantSecret: "secret",           // authenticates the tenant
    email: "customer@example.com",
    firstName: "John",
    lastName: "Doe",
    // ... additional optional fields
}
    ↓
API: Validate tenantToken + tenantSecret → resolve Tenant
    ↓
API: Find or create User by email
    ↓
API: Find or create Profile for this User + Tenant (role: customer)
    ↓
API: Issue JWT (userId, profileId, tenantId, role: customer)
    ↓
Response: { accessToken, refreshToken, expiresAt }
```

**Key behaviors:**
- User not found → create User (emailVerified = true, trusted source)
- User found but no Profile for this tenant → create Profile
- User found + Profile exists → use existing Profile
- All in a single endpoint call — no multi-step flow

### Flow 3 — Tenant-Scoped Domain

Custom domain or subdomain pins the tenant. Login screen exists but tenant info comes from URL/config.

```
Browser: https://dealer-name.app.com/login
    ↓
UI: Resolve tenant from subdomain/config
    ↓
Client: POST /api/auth/login { email, password, tenantCode: "dealer-name" }
    ↓
API: Validate credentials + filter profiles by tenant
    ├── Single profile for this tenant → auto-select → JWT
    └── Multiple profiles for this tenant → show selection
```

**Difference from Flow 1:** The tenant is pre-filtered. User only sees profiles within that tenant, not all their profiles across all tenants.

## Data Isolation

### ITenantEntity Interface

Every entity that belongs to a tenant implements this:

```csharp
public interface ITenantEntity
{
    Guid TenantId { get; set; }
}
```

### Global Query Filter

Automatic `WHERE TenantId = @currentTenantId` on all tenant-scoped queries:

```csharp
// In AppDbContext.OnModelCreating:
foreach (var entityType in modelBuilder.Model.GetEntityTypes())
{
    if (typeof(ITenantEntity).IsAssignableFrom(entityType.ClrType))
    {
        modelBuilder.Entity(entityType.ClrType)
            .HasQueryFilter(BuildTenantFilter(entityType.ClrType));
    }
}
```

### Tenant Interceptor

Automatically sets `TenantId` on new entities:

```csharp
// In SaveChanges:
var tenantEntries = ChangeTracker.Entries<ITenantEntity>()
    .Where(e => e.State == EntityState.Added);

foreach (var entry in tenantEntries)
{
    entry.Entity.TenantId = _currentTenant.TenantId
        ?? throw new InvalidOperationException("TenantId is required for tenant-scoped entities");
}
```

### ITenantContext

Resolves the current request's tenant:

```csharp
public interface ITenantContext
{
    Guid? TenantId { get; }
    Guid? ProfileId { get; }
    string? Role { get; }
}

// Implementation reads from JWT claims (set during auth)
public class JwtTenantContext : ITenantContext
{
    public Guid? TenantId => // from ClaimTypes "tenantId"
    public Guid? ProfileId => // from ClaimTypes "profileId"
    public string? Role => // from ClaimTypes "role"
}
```

## `/create-new-project` Integration

When SaaS is selected during project creation:

**Question 1:** "Tenant entity name?" → Dealer / Company / Seller / Organization / custom

**Question 2:** "Can a user belong to multiple tenants?" → Yes (multi-profile) / No (single profile)

**Question 3:** "Embed/auto-login needed?" → Yes (Flow 2 added) / No

**Question 4:** "Custom domain support?" → Yes (Flow 3 added) / No

When SaaS is NOT selected: User entity only. No Profile, no Tenant, no ITenantEntity, no filters. Clean codebase.

## Important Rules

1. **Auth always against User.** Login = email + password → User. Never against Profile or Tenant.
2. **JWT carries the full context.** userId + profileId + tenantId + role. Every request is self-contained.
3. **Global filter = safety net.** Even if handler forgets to filter, the query filter ensures tenant isolation.
4. **Interceptor = safety net.** Even if handler forgets to set TenantId, the interceptor sets it automatically.
5. **Profile switch without re-login.** User has valid session → can switch profile (different tenant) without entering password again. New JWT issued with different profileId/tenantId.
6. **Embed auth is trust-based.** Tenant's secret authenticates the request. User is auto-created/auto-verified. No email verification needed — the tenant vouches for the customer.
7. **TenantId null = system scope.** Profiles with null TenantId are system admins — they see all data (query filter doesn't apply).
