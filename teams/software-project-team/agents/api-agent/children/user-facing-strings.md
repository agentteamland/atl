---
knowledge-base-summary: "Backend is English-only; UI apps (Flutter, React) localize. User-visible text crosses the boundary as `messageKey + placeholders + fallback` envelopes — inside `ProblemDetails.extensions` for errors, inside notification payloads for socket events, and as `templateKey + locale` for emails/push (which MailSender and the push dispatcher render from per-locale templates). Enum values render via `{entity}_{field}_{value}` convention on the UI side. This is the canonical reference — every other agent's i18n guidance defers here."
---
# User-Facing Strings — The API-to-UI i18n Contract

This is the **canonical reference** for how user-facing text crosses the backend/frontend boundary. Every other agent (flutter-agent, react-agent, worker-agent, socket-agent) implements the side of this contract that concerns them. When this file conflicts with anything written elsewhere, this file wins.

## The Principle

**Backend is English-only.** API responses, socket events, log lines, worker outputs — all English. No `Accept-Language` header parsing, no per-locale response bodies, no localized error messages in ProblemDetails titles or details.

**UI apps localize.** Flutter, React admin, React public website carry their own translation dictionaries and render the user's language using keys they receive from the backend. If the user speaks Turkish, the UI's job — not the API's — is to produce Turkish output.

**Emails and push notifications are the exception.** They ARE user-facing surfaces, but the backend renders them (not the UI). So MailSender and the push dispatcher hold per-locale templates and pick based on the user's stored locale. This is explicit scope for those two services, not a general backend i18n.

## Key Format

**Short-code keys, snake_case, feature-prefixed.** `{feature}_{element}_{variant}` or `{feature}_{action}_{element}`.

| Pattern | Example |
|---------|---------|
| `{feature}_title` | `tasks_title` |
| `{feature}_empty_title` / `_subtitle` | `tasks_empty_title` |
| `{feature}_error_{type}` | `tasks_error_load_failed` |
| `{entity}_{field}_{value}` (enum render) | `task_status_completed` |
| `validation_{rule}` | `validation_password_too_short` |
| `common_{action}` | `common_retry` |

Not ambiguous English sentences as keys (gettext-style). Not dotted hierarchies for the top-level — i18next's `common:actions.save` style is fine INSIDE the UI's own namespace, but keys that cross the API boundary are always flat snake_case.

## Placeholder Syntax

**ICU MessageFormat.** Flutter's ARB tooling speaks ICU natively; i18next supports it via the plugin. One syntax across both UIs means shared mental model.

```
"tasks_count": "{count, plural, =0{No tasks} =1{1 task} other{{count} tasks}}"
"task_greeting": "Hello, {name}!"
"task_distance": "{distance, number, ::precision-integer} km"
```

Placeholders are passed as a flat object: `{ count: 5, name: "Alice", distance: 3.5 }`.

## Envelope 1: ProblemDetails Extensions (errors)

Every error response from the API is a standard RFC 7807 `ProblemDetails` body. User-facing text rides in `extensions`:

```json
{
  "type": "https://example.com/errors/validation",
  "title": "Validation failed",
  "status": 422,
  "detail": "Password must be at least 8 characters.",
  "extensions": {
    "messageKey": "validation_password_too_short",
    "placeholders": { "min": 8 },
    "fallback": "Password must be at least 8 characters."
  }
}
```

**Field rules:**
- `title` and `detail` — English. These are for logs, dev tools, and machine consumers. Never shown verbatim to end users unless `messageKey` resolution fails AND `fallback` is empty.
- `messageKey` — required for anything a human sees in the UI. Omit only for 5xx server errors that should never surface text (UI falls back to a generic `common_error_generic`).
- `placeholders` — optional. Include every value referenced in the ICU format string.
- `fallback` — same English string as `detail`, but specifically marked as safe-to-render. The UI shows this if it has no translation for `messageKey`. Including it is cheap insurance against deploy skew between API and UI.

**ValidationException with multiple errors** — FluentValidation produces multiple field-level failures. Pack them in `extensions.errors`:

```json
{
  "status": 422,
  "extensions": {
    "errors": [
      { "field": "email", "messageKey": "validation_email_invalid", "placeholders": {}, "fallback": "Invalid email." },
      { "field": "password", "messageKey": "validation_password_too_short", "placeholders": { "min": 8 }, "fallback": "Password must be at least 8 characters." }
    ]
  }
}
```

**.NET mapping:**

```csharp
// In the global exception handler
public async ValueTask<bool> TryHandleAsync(
    HttpContext httpContext, Exception exception, CancellationToken ct)
{
    if (exception is ValidationException vex)
    {
        var problem = new ProblemDetails
        {
            Status = 422,
            Title = "Validation failed",
            Type = "https://example.com/errors/validation",
        };
        problem.Extensions["errors"] = vex.Errors.Select(e => new
        {
            field = e.PropertyName,
            messageKey = e.CustomState as string ?? e.ErrorCode,
            placeholders = e.FormattedMessagePlaceholderValues,
            fallback = e.ErrorMessage,
        });
        // ... write response
        return true;
    }
    // ... other exception types
}
```

FluentValidation rules should set `WithErrorCode("validation_password_too_short")` or stash the key in `CustomState` so the handler has a messageKey to put in the envelope.

## Envelope 2: Notification Events (socket)

Notifications carrying user-facing text use the same envelope inside `Data`:

```csharp
public record NotificationEvent
{
    public string EventName { get; init; } = null!;
    public NotificationPayload Data { get; init; } = null!;
    public string TargetType { get; init; } = "all";
    public string? TargetUserId { get; init; }
    public string? TargetGroup { get; init; }
    public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
}

public record NotificationPayload
{
    public string MessageKey { get; init; } = null!;
    public Dictionary<string, object?> Placeholders { get; init; } = new();
    public string Fallback { get; init; } = null!;
    public object? Context { get; init; }   // entity IDs, references, not user-facing
}
```

Socket dispatches the JSON verbatim; the UI's socket handler resolves the envelope the same way it resolves ProblemDetails extensions.

**When text is NOT user-facing** — e.g., a socket event that only carries `{ orderId, newStatus }` for the UI to re-fetch — drop the envelope entirely. Only events with text that will be rendered need the envelope.

## Envelope 3: Email Jobs (RMQ → MailSender)

API publishes the EmailJob with a templateKey + placeholders + locale:

```csharp
public record EmailJob
{
    public string To { get; init; } = null!;
    public string TemplateKey { get; init; } = null!;        // e.g., "welcome", "password_reset"
    public Dictionary<string, object?> Placeholders { get; init; } = new();
    public string Locale { get; init; } = "en";              // user's stored locale
    public string? ReplyTo { get; init; }
    public string? Subject { get; init; }                    // only set if template doesn't define one
}
```

**Locale resolution** (in the publisher, i.e., the API handler):
1. User has a persisted locale (on the User or Profile entity) → use it.
2. User has no locale but the request carried an `X-User-Locale` header from the UI → use it.
3. Neither → default to `en`.

**MailSender side** holds templates per locale:

```
src/{ProjectName}.MailSender/
  Templates/
    en/
      welcome.subject.txt
      welcome.body.html
      password_reset.subject.txt
      password_reset.body.html
    tr/
      welcome.subject.txt
      welcome.body.html
      password_reset.subject.txt
      password_reset.body.html
```

MailSender resolution order: `{Locale}/{TemplateKey}.body.html` → fallback to `en/{TemplateKey}.body.html` if the locale file is missing. Template files use `{placeholder}` interpolation (not ICU — HTML templates don't need plural support).

**Adding a new email template** = add BOTH `en/` and `tr/` versions in the same commit. Missing translations are a lint failure in CI, not a silent fallback-to-English.

## Envelope 4: Push Notifications (Worker/API → dispatcher)

Same shape as emails, but with title + body keys and delivered via FCM/APNs:

```csharp
public record PushNotificationJob
{
    public string UserId { get; init; } = null!;
    public string TitleKey { get; init; } = null!;
    public string BodyKey { get; init; } = null!;
    public Dictionary<string, object?> Placeholders { get; init; } = new();
    public string Locale { get; init; } = "en";
    public Dictionary<string, string>? Data { get; init; }   // deep link, entity IDs
}
```

A PushDispatcher consumer resolves the keys against its own per-locale dictionary (same format as MailSender's templates, but `.txt` — push bodies are short), then calls FCM/APNs. FCM payloads already support `title` + `body` as plain strings; no need to send keys to the device.

## Enum Rendering

Backend returns enum values as strings: `{ "status": "completed" }`. The UI derives the translation key from the entity + field + value:

```
key = `{entity}_{field}_{value}` → task_status_completed
```

Flutter:
```dart
extension EnumLocalization on BuildContext {
  String enumLabel(String entity, String field, String value) =>
      l10n.lookup('${entity}_${field}_${value}') ?? '${entity}_${field}_${value}';
}

// usage
Text(context.enumLabel('task', 'status', task.status));
```

React (i18next):
```typescript
function useEnumLabel() {
  const { t } = useTranslation('common');
  return (entity: string, field: string, value: string) =>
    t(`${entity}_${field}_${value}`, { defaultValue: `${entity}_${field}_${value}` });
}
```

**Naming the enum keys:** snake_case, match the JSON value exactly. Backend sends `"in_progress"`, key is `task_status_in_progress`. Do not PascalCase in the key; wire format drives the key.

## Fallback Order (UI side)

When resolving `messageKey`:

1. Look up key in current locale's dictionary → use it if present.
2. Look up key in fallback locale (`en`) → use it if present.
3. Use `fallback` field from the envelope if provided.
4. Show the raw `messageKey` as last resort (makes missing keys visible to developers).

Never show a blank. Never swallow a missing key silently — log it client-side so it surfaces in dev/QA.

## Checklist (when writing a handler or publisher)

- [ ] Does this response carry user-facing text?
  - No → no envelope needed. Ship the data.
  - Yes → wrap in the appropriate envelope from above.
- [ ] Do all placeholders have corresponding keys in the ICU template?
- [ ] Is the `fallback` field an English sentence that a developer would accept if i18n fails?
- [ ] For emails/push: has the locale been resolved via the user's stored preference?
- [ ] For new keys: added to the appropriate `.arb` / `.json` dictionary in the UI repo in the same PR?
