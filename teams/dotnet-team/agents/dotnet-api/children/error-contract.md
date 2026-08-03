---
knowledge-base-summary: "The wire shape errors go out in, and the single place status is mapped. `ProblemDetails` everywhere, with the validation status carrying a field-keyed dictionary whose keys are the property names a client binds form fields on — a contract, not display text. Three traps: serialize by the RUNTIME type or a derived problem held in a base variable silently loses its dictionary; framework-generated responses (auth challenge, policy denial, unmatched route) never reach the exception handler at all; and a model-binding failure escapes as a 500 unless one central arm maps it."
---

# Error Contract

Every error this API emits is an RFC-7807 `ProblemDetails` body, and there is exactly **one place**
that decides which status a failure becomes. Two things make this topic worth its own page: the
validation shape carries a machine-readable dictionary that a client binds form fields on — so it is
a contract, not copy — and three separate paths reach the wire *without passing through* the place
you think owns the mapping.

## The shape on the wire

Two shapes, and only two:

- **Validation** — a plain problem body plus a field-keyed dictionary:
  `{"title":"Validation failed","status":422,"errors":{"Amount":["insufficient available funds"]}}`
- **Everything else** (400 / 403 / 404 / 409 / 429 / 500) — a plain `{title,status,detail}`.

Pick one status for validation and **pin it project-wide**. The framework's own
`ValidationProblemDetails` defaults to **400**; **422** is a common deliberate choice for
"well-formed but semantically rejected". Which one you pick matters far less than that it never
varies by endpoint — the client branches on it.

## One switch, and no try/catch in handlers

A handler throws a typed exception; it does not catch, does not wrap a failure in a result object,
and does not set a status code. A global `IExceptionHandler` (registered with `AddExceptionHandler<T>`
and `UseExceptionHandler()`) maps exception type → status → problem body, in one `switch`.

```csharp
// ✅ throw and let the seam map it
var order = await _db.Orders.FindAsync([id], ct)
    ?? throw new NotFoundException(nameof(Order), id);

// ❌ per-handler catching — a second owner for a cross-cutting invariant
try { ... } catch (Exception ex) { return Result.Failure(ex.Message); }
```

A typical mapping, which is a **project convention** rather than a framework behaviour — write it
down once in `Conventions/` and keep the switch the only copy:

| Exception | Status |
|---|---|
| `NotFoundException` | 404 |
| `ValidationException` (FluentValidation or hand-thrown) | the pinned validation status |
| `ForbiddenException` | 403 |
| concurrency-token mismatch | 409 |
| `BadHttpRequestException` (model binding) | **400** — see trap 3 |
| anything else | 500 |

Why one switch and not per-handler mapping: this is a cross-cutting invariant, and per-handler is
correct only while you own every path that can throw. You stop owning it the first time a filter, a
middleware, or a library throws inside your pipeline.

## The keys are the contract, not display text

The dictionary keys are the **property names** of the request model — PascalCase, exactly as the C#
property is spelled. A client maps them onto its form fields. That has three consequences a worker
must internalise before touching validation:

1. **Never change a key to improve how a message reads.** See the localization section — this is the
   single most common way to make every per-field message silently vanish.
2. **A key that has no counterpart on the client renders nothing.** Concurrency tokens, computed
   fields, and a validator whose property name differs from the client's field name all produce a
   message the user never sees. The server cannot fix that alone; it is why the client half always
   needs a catch-all banner (see *The other half*).
3. **Renaming a request-model property is a wire-breaking change**, the same as renaming a response
   field. It compiles, it passes, and it silently unhooks the client's per-field errors.

## Trap 1 — serialize by the RUNTIME type

`WriteAsJsonAsync<TValue>(value)` serializes by the **compile-time** type `TValue`, not the runtime
type. A derived `ValidationProblemDetails` (which owns the `Errors` dictionary) held in a variable
typed as the base `ProblemDetails` therefore serializes **only the base's properties** — and the
dictionary is dropped, app-wide, with no error anywhere.

```csharp
ProblemDetails problem = BuildProblem(exception); // runtime type may be ValidationProblemDetails

// ❌ serializes the BASE — `errors` disappears from every validation response
await context.Response.WriteAsJsonAsync(problem, cancellationToken: ct);

// ✅ pass the runtime type explicitly
await context.Response.WriteAsJsonAsync(problem, problem.GetType(), options: null, ct);
```

**Generalize it.** This is not an error-handling quirk; it is how `System.Text.Json` behaves for
*any* polymorphic-through-a-base-variable serialization. Any place that returns a derived type
through a base-typed variable — a DTO hierarchy, an event payload, a wrapper result — loses the
subtype's fields the same way. The symptom is always "the field is set in the debugger and absent on
the wire".

## Trap 2 — framework-generated responses never reach the handler

An `IExceptionHandler` only sees things that **throw**. These do not:

- the authentication challenge (401) from the JWT bearer handler
- an authorization-policy denial (403)
- an unmatched route (404)
- a bare result-object not-found (`Results.NotFound()`), and its siblings

They are produced by the status-code-pages / `IProblemDetailsService` path, so they keep the
framework's own titles long after every handler-produced message has been customized. Customize them
where they are actually built:

```csharp
builder.Services.AddProblemDetails(o => o.CustomizeProblemDetails = ctx =>
{
    // switch on ctx.ProblemDetails.Status — titles, type URIs, extensions
});
```

This is safe alongside a global handler that writes with `WriteAsJsonAsync` **directly**: that path
never goes through `IProblemDetailsService`, so the two do not overlap or double-apply. (If your
handler instead *delegates* to `IProblemDetailsService`, the customization runs for both — check
which shape you have before assuming.)

**There is no source string to grep for.** This gap has no textual footprint at all; the only way to
find it is to probe each status code against the running app and read the body. Budget that probe —
it is the single highest-yield ten minutes in this topic.

## Trap 3 — a model-binding failure escapes as a 500

A bad enum or a malformed date **in a query string** fails at the model-binding layer and surfaces as
`BadHttpRequestException`, which carries its own intended status (400). A catch-all `_ => 500` arm
overwrites that: the caller gets a server error for what is unambiguously a client mistake, and your
error monitoring fills with false 5xx.

One explicit arm in the central switch fixes every endpoint at once — that is the whole point of
having one switch.

### The twin one layer down — same symptom, different fix

A *well-formed* but zone-less date is the other half, and it looks identical from the outside:

- A date-only value (`?from=2026-07-01`) carries no offset, so the binder produces a `DateTime` with
  `Kind = Unspecified`. Nothing rejects it — it is a perfectly valid value.
- A provider that distinguishes zone-aware timestamps (PostgreSQL / Npgsql against
  `timestamp with time zone`) then refuses to compare it, and the request escapes as a **500**.

Both cases present as "a 500 out of a query-string date" and they need different fixes, so establish
**which layer threw** before reaching for either. The comparison-layer fix belongs to the persistence
topic (bind a day-granular type, or stamp the kind once at the top of the handler and use a half-open
upper bound); what belongs *here* is the diagnostic split and one contract lesson:

> **A failed request must never be readable as an empty result.** The 500's problem body has no
> items array, and a caller that reads absence as "zero rows" goes hunting for a bug in a query that
> is correct. That is a cost of the error shape, not of the query — which is why the contract is
> uniform and why the client is told to branch on status before it looks at the body.

## Localizing without breaking the keys

**The rule, whatever the mechanism: change the rendered message, never the key.**

With FluentValidation, set the display name **globally and chained** —
`ValidatorOptions.Global.DisplayNameResolver`, falling through to the previous resolver rather than
replacing it, so per-validator customization still works. That changes the `{PropertyName}` token
rendered inside the message and nothing else.

The trap is the per-rule `.WithName(...)`. **Verify what your FluentValidation version does to
`ValidationFailure.PropertyName`** before using it on a property rule: the explicitly key-renaming API
is `OverridePropertyName`, but `.WithName(...)` on a property rule has been **observed in a real
codebase to change the key that reaches the wire** — every per-field message silently disappeared
while the text itself looked perfectly translated. That is the worst failure shape available here:
the change looks correct in the response body, and the breakage is entirely on the client.

**The deliberate exception.** On an object-level rule (`RuleFor(x => x)`) the failure has no property
name at all, so it arrives with an empty key and cannot be routed anywhere. `.WithName(...)` there is
how such a failure *gets* a stable key, and that is legitimate and worth doing.

> Rule of thumb: **`.WithName()` may assign a missing key; it may never rename a real one.**

Since this depends on a library version and on a distinction two APIs make badly, do not defend it
with care alone — **pin the keys with a test**. A test that asserts the exact key set a validator
produces turns the whole class of failure into a red build instead of a silent client regression.

### Two designs, same rule

Who renders the user's language is a project decision, and both designs keep the key untouched:

- **Server renders.** Localize at the chokepoints (the display-name resolver above; the exception
  constructor that builds the message). The wire carries finished text.
- **Client renders.** The wire carries a `messageKey` + `placeholders` + `fallback` envelope in the
  problem's extensions, and the client resolves it against its own dictionary. `title` and `detail`
  stay English for logs and machine consumers; `fallback` is the English sentence the client shows
  when it has no translation for the key, which is cheap insurance against deploy skew between API
  and client.

Pick one and record it in `Conventions/`. Running both is the failure the one-owner-per-invariant
principle exists to prevent.

## `detail` is for developers, not users

A global handler typically puts the whole `exception.ToString()` — stack trace included — into
`detail` in Development. Two rules follow:

- **Never surface `detail` on a 5xx** to an end user. A naive "fall back to `detail` if there's no
  message" ships a stack trace into a toast.
- **Never let Development-only content decide the contract.** The shape must be identical in both
  environments; only the *content* of `detail` differs.

## Cross-cutting text: count chokepoints, not call sites

Any sweep over user-facing text — error copy, log format, date or number rendering — presents as an
edit-count problem and is almost always a **structure** problem. The order that works is
**inventory → dedupe → find where the string is actually built → only then script the genuine
one-offs.**

The worked shape: a `NotFoundException` thrown from a hundred-plus call sites builds its message in
**none** of them. It builds it once, in the constructor. Editing that one constructor changes the
whole surface, no call site touched, and the next `throw` written afterwards inherits it for free.

This is a **correctness** argument before it is a cost one: one constructor edit cannot partially
apply — it is either in or it is not — whereas N hand-edits are N chances to miss one or to let two
drift. Before estimating such a change, grep for the constructor, factory, or base class the
occurrences funnel through; the honest estimate is the number of chokepoints.

Give the chokepoint a graceful fallback (an unmapped entity degrades to its raw name rather than
throwing), and note that a chokepoint is also the one place you can *see* which half is contract and
which half is copy — translate the rendered text, leave the programmatic identity alone.

## Verifying the contract

The gates that will *not* catch a broken error contract:

- **A unit test asserting the handler throws.** It proves the exception, never the wire shape.
- **A test that serializes the problem object directly.** It holds the value in its *derived* type,
  so trap 1 is invisible: the test passes and production drops the dictionary.
- **Any grep.** Trap 2 has no source string.

What does catch it:

- [ ] An in-process HTTP test against the **composed** app that reads the actual response **body**
      for a validation failure and asserts the **exact keys**.
- [ ] A runtime probe of each status code the app can emit — 401 (unauthenticated), 403 (policy
      denial), 404 (unmatched route *and* a not-found result), the validation status, 409, 500 —
      confirming every one is the agreed shape.
- [ ] A deliberate malformed query-string parameter, asserting **400** and not 500.
- [ ] After any validation-message change: the key-pinning test above.

## The other half

The client's consumption of this contract belongs to the web specialist, not here. State the seam so
neither side assumes the other's job:

- The client maps the dictionary keys onto form fields, converting case as its framework requires —
  **necessary but not sufficient**, because some keys have no matching field at all. It must also
  fill a banner with the first message unconditionally, or those errors render as nothing.
- The client suppresses `detail` on 5xx and shows a generic message.
- The client treats the dictionary as **optional** and falls back to the plain shape, so it stays
  forward-compatible with the day the server starts sending it.

## Related

- [architecture-and-layering](architecture-and-layering.md) — where the handler and the composition
  root live, and why the mapping switch is composition-root wiring.
- [persistence-and-migrations](persistence-and-migrations.md) — the comparison-layer half of the
  zone-less-date 500.
- [testing-and-verification](testing-and-verification.md) — the composed-app HTTP test this page's
  checklist depends on.
