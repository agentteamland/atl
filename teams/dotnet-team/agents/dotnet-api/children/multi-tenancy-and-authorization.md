---
knowledge-base-summary: "The isolation boundary: a global query filter on the context is the boundary, an inbound guard is defence in depth, and scoping stays centralized rather than scattered into handlers. Reference the tenant through the context instance, never a captured service — a captured constant is funcletized once and frozen into the cached query plan, which is a real cross-tenant leak. Authorization is an endpoint policy, never a UI affordance. Reason about bypassing a filter from the query's ROOT, not a count of call sites — and remember the filter degrades OPEN where there is no request."
---

# Multi-tenancy and authorization

Two questions that look like one and are not:

- **Which rows may this caller see?** — tenancy. Enforced by a query filter, invisibly, on every query.
- **What may this caller do?** — authorization. Enforced by an explicit check at each endpoint.

Conflating them is how a data leak gets filed as a permissions bug and fixed in the wrong layer. A
correct policy on an endpoint whose query is unfiltered still leaks; a correct filter under an
endpoint with no policy still lets the wrong caller act.

Multi-tenancy is **optional**. A single-tenant project ships none of this — no tenant interface, no
filter, no guard, no tenant claim. Do not scaffold the machinery speculatively; the correctness cost
of a half-installed isolation boundary is higher than the cost of adding it later.

## The model

```
Identity  → one human, one credential. The authentication target.
   ↓ has many
Profile   → that identity's face inside ONE tenant. Carries the role.
   ↓ belongs to
Tenant    → the isolation unit. Its NAME is project-shaped.
```

**The role lives on the profile, not the identity.** One human can be an administrator in one tenant
and an ordinary user in another; putting the role on the identity makes that unrepresentable, and
every later fix is a schema migration plus a token-shape change. The token therefore carries the
resolved context — identity id, profile id, tenant id, role — so each request is self-contained.

The tenant's *name* is a project decision and it belongs on the project's `Architecture/` page, not
here. What is generic is the shape: two levels, role on the middle one.

**Uniqueness on profiles.** If the system can ever enrol one identity twice in the same tenant under
different roles, the key is `(identity, tenant, role)` — not the two-column form, which will reject a
flow the product actually wants. Two database-side traps ride with it, both belonging to
[persistence-and-migrations](persistence-and-migrations.md) but worth knowing before you design the
index:

- **NULL handling in a unique index is engine-specific, and the tenant column is exactly where it
  bites** (tenant-less platform rows). PostgreSQL and standard SQL treat NULLs as *distinct*, so
  those rows are not constrained by the index at all and need a second partial index over
  `(identity, role) WHERE tenant IS NULL`. SQL Server does the opposite — NULLs compare *equal*, so
  the index permits only one tenant-less row per identity, which may be the wrong constraint rather
  than the missing one. Check which engine you are on; do not carry the rule across.
- **De-duplicate inside the migration, before creating the index** — a window function partitioned by
  the key columns does group NULLs, unlike the index — or the migration aborts on any live database.

## Layer 1 — the query filter IS the boundary

A global query filter, applied to every tenant-scoped entity and derived from the authenticated
principal's tenant claim, is the boundary. Everything else on this page is defence in depth.

The filter shape that works:

```
!IsDeleted && (CurrentTenantId == null || (Guid?)e.TenantId == CurrentTenantId)
```

A **null tenant means see-all**, which is how a tenant-less platform administrator gets a
cross-tenant sweep for free, with no second code path. That branch is also the one dangerous thing
on this page, so it comes with a precondition: **an ordinary user must be refused a null tenant at
profile creation.** If a non-privileged profile can exist with no tenant, the see-all branch is
reachable by an ordinary caller and the boundary is gone.

Two mechanics inside the filter that any change must preserve:

**1. Reference the tenant through the context instance, never a captured service.**

```csharp
// RIGHT — reads the property off the DbContext instance, evaluated per query
e => !e.IsDeleted && (CurrentTenantId == null || (Guid?)e.TenantId == CurrentTenantId)

// WRONG — captures a service and freezes its value into the cached model
var tenantId = _currentUser.TenantId;
e => e.TenantId == tenantId;
```

A value captured from a service is *funcletized* — evaluated once, when the model is built — and the
model is then cached for the lifetime of the application. The filter is therefore frozen to whichever
caller happened to compile the query first, and every later caller reads that caller's tenant. This
is a real cross-tenant leak, it reviews as correct, and it does not reproduce in a single-tenant test
run. Put a comment at the line saying why the form is what it is; it is the kind of expression a
later "simplification" removes.

**2. Keep the comparison null-tolerant.** Lift the column to the nullable type (`(Guid?)e.TenantId`)
rather than unwrapping `CurrentTenantId.Value`. The tenant is now a per-query parameter, and a null
tenant — the platform-admin case above, and every out-of-request case below — would throw.

**Keep scoping centralized.** Do not scatter tenant-id comparisons into handlers. Two payoffs, and
both are load-bearing: a gap in any outer layer is then never a breach, and the tenancy model stays
cheap to change later, because there is one expression to change rather than a population of
hand-written predicates.

**The write side belongs to the same seam.** A step in the `SaveChanges` override stamps the tenant
on newly added tenant-scoped entities. Make the sentinel **throw** when there is no ambient tenant
and the entity is tenant-scoped: a loud failure on an unstamped insert is the good case, and the
contrast with the read side is the point — the read side degrades *silently* into see-all.

## Layer 2 — the inbound guard, and why an anonymous endpoint is not a hole

Middleware that compares the principal's tenant against the tenant resolved from the request, and
returns 403 on a mismatch. It is defence in depth **only** — never the thing you rely on.

It runs **after** authentication, which means an anonymous endpoint passes straight through it. That
is deliberate and correct: an anonymous public route is legitimate because Layer 1 still constrains
everything it reads, not because Layer 2 cleared it. If you ever catch yourself justifying an
anonymous route by what the guard does, the reasoning is inverted — the guard never saw the request.

Resolving the request's tenant:

- **Read the host the server actually saw**, not a client-supplied forwarding header, and do not
  enable forwarded-header processing unless a trusted proxy chain genuinely requires it. Otherwise
  the caller chooses their own tenant.
- **Privileged hosts are an exhaustive allow-list, never a computed predicate.** The rule this comes
  from is worth stating in general: *an attacker-controllable check that the server's own identity
  satisfies is no check at all.* A predicate like "this host looks like an infrastructure address"
  is satisfied by the server's own address, which a spoofed header can supply — so the check
  disables itself for anyone who guesses it.
- A host→tenant lookup is usually cached. That is fine as long as the input is not
  attacker-controllable, but state the staleness window explicitly, and remember a moved binding lags
  by exactly that long.

## Authorization is an endpoint check, always

Policies at the endpoint. A hidden menu item, a client-side route guard, a pre-filtered list — all
presentation. **A role whose only limit is a hidden tab reaches everything it knows the path to.**
The two layers are genuinely independent in both directions: hidden is not denied, and denied does
not need to be hidden. A reachable-but-unlinked admin route is authorized-and-invisible; a visible
menu with no server check is denied-nowhere.

The tier ladder is usually four rungs, and naming it prevents most mistakes:

| Tier | May do |
|---|---|
| platform admin | above any tenant; the only tier that may mutate a **global** row |
| tenant admin | the administrative surface of one tenant |
| ordinary user | the self-service surface; an admin token is refused here, deliberately |
| "has a tenant context at all" | a coarse gate, not a role |

Note the third rung: locking the self-service surface to the ordinary-user role — so an
administrator's token is refused there — is a real check, not pedantry. Without it, "is a user of
this system" and "may act as a customer" are the same predicate.

### A mutation on a global row is a higher tier than the endpoint group it sits in

A global catalog row is served to every tenant, so a tenant administrator who can rewrite it rewrites
what every other tenant's users see. The endpoint group being legitimately tenant-admin-accessible
does not change that — the group is about the surface, the tier is about the row.

**Split at the boundary; do not raise the whole endpoint.** Gating the entire endpoint to platform
admin is the wrong fix: it removes a capability tenants genuinely need and centralizes routine work
on an operator with no per-tenant knowledge. Gate **only the global row's mutation**, and leave the
tenant-scoped half — already covered by Layer 1 — open. The result is a coherent workflow rather than
an escalation.

### Acting on a tenant's behalf needs exclusivity, not membership

When a tenant administrator mutates a **global** row that represents someone (an identity, an
account), "the target has a profile in my tenant" is the wrong check. Any identity shared across two
tenants satisfies it, so the mutation reaches into the other tenant — a credential reset is a
takeover, a deactivation is a denial of service.

The correct check is **exclusivity**: the target must have a profile in the caller's tenant **and
none in any other** (a tenant-less platform profile counts as another). Shared identities are
platform-admin-only, by construction.

Where the existence of the row is itself information, answer **404 rather than 403** — a 403
confirms that the identifier is real.

## Bypassing the filter — reason from the query's ROOT set

`IgnoreQueryFilters()` is not itself a verdict, and **a count of call sites tells you nothing.** The
verdict comes from what the query's **root** entity is:

- **The root is not tenant-scoped** (an identity, a profile, a global catalog) → there is no tenant
  filter to strip, only a soft-delete one → **safe**.
- **A self-scoped handler already narrowed to the caller's own id from the token** → **safe**.
- **A tenant-scoped root** → **almost always wrong**.

**On the third case, first ask whether the call is needed at all.** The filter's null-tenant branch
already gives a platform administrator the full sweep *and* scopes a tenant administrator
automatically — so most filter-stripping on a tenant-scoped root is solving a problem the filter had
already solved, and **deleting the call is the whole fix**. This matters because the instinct on
discovering a leak is to *add* a scope variable, and a hand-rolled narrowing on a filter-stripped
path is precisely the construction that produces the next leak.

Where non-tenant data genuinely is needed, **keep the root query filtered** and resolve the extra
data in a **separate** unfiltered query, joined in memory. Never strip filters query-wide to reach
one related field.

### Once stripped, the hand-written comparison IS the boundary

With the filter gone, a `TenantId ==` comparison in the handler is no longer belt-and-braces — it is
the only thing standing there. So:

- **Re-audit every scope variable on that path for null.** A scope that resolves to `null` where the
  filter has been stripped means **every tenant**, not *my tenant*. This is reachable rather than
  theoretical wherever the tenant column is nullable with no constraint tying role to tenant, and the
  token omits the claim when it is null — both of which are ordinary, defensible design choices on
  their own.
- **Never copy a scope-narrowing idiom out of a neighbouring handler** without checking whether the
  neighbour still had its filter on. This is the mechanism behind the worst instance of the class:
  the copied line carried a comment describing it as defence in depth, and that comment was *true
  where it was written*. The same line is defence in depth or the entire boundary depending on
  whether the filter underneath it is still on.
- **A census is a snapshot, not current truth.** Re-run the search and re-classify by root set after
  any change that adds administrative surface. Reasoning from a remembered count — "we audited this,
  it was mostly fine" — is how the sites added since the audit go unclassified.

## Outside a request, the filter degrades OPEN

The principal is read off the HTTP context accessor, which is **null** in a hosted service, a
scheduled job or a queue consumer. So the current tenant is null, and null is the see-all branch. The
boundary opens by default, silently, in exactly the code that has no request to blame.

Three consequences for any out-of-request work:

- **Carry the caller's scope in the queued message** and re-apply it by hand. Carry the *filter*
  (a tenant id, or null meaning all), not a resolved list — the payload stays small and the work
  targets the set that is live at execution time.
- **Stamp the tenant explicitly on every insert.** The auto-fill sentinel throws when the ambient
  tenant is null, so relying on it there fails outright.
- **A root that is not tenant-scoped has no filter to degrade** — so a hand-written comparison there
  is the only boundary, which is the stripped-filter case above reached by a completely different
  route. Both are the same shape: no filter, one hand-written line, null means everyone.

The rest of the out-of-request contract — redelivery, idempotency, cancellation — is in
[messaging-and-background-work](messaging-and-background-work.md).

## Verifying a tenancy or authorization change

None of this is visible to the compiler, and a test suite with one tenant in it cannot fail on any
of it. Evidence means a live cross-tenant probe:

- **Two tenants, an actor in each.** Assert the read set of each is exactly its own, and assert the
  *write* is refused — the read half passing tells you nothing about the write half, and the write
  leaks are the expensive ones.
- **Prove the fix with a control.** Run the same request against the pre-fix build and observe it
  succeed, then against the post-fix build and observe it refused. An assertion that only ever ran
  after the fix cannot distinguish "fixed" from "was never reachable that way" — and a defence that
  was never demonstrated failing is not known to be a defence.
- **Sweep the whole surface, not your diff.** After a multi-phase tenancy or authorization change,
  audit every endpoint that touches the model, not only the ones you edited. Pre-existing defects in
  neighbouring handlers are not reachable by any amount of verification *of the code that changed* —
  which is exactly why they are still there.
- **Exercise the out-of-request path deliberately.** The see-all degradation cannot be observed from
  any request-path test, because there is always a request.

## Related

- [persistence-and-migrations](persistence-and-migrations.md) — where the filters are declared, the
  `SaveChanges` seam that stamps the tenant, and the index traps the profile uniqueness key creates.
- [messaging-and-background-work](messaging-and-background-work.md) — the out-of-request half in
  full: carrying scope in the message, redelivery, and idempotency by construction.
- [endpoint-blueprint](endpoint-blueprint.md) — where the policy is attached when a new endpoint is
  created, and why an unregistered one is the characteristic failure.
