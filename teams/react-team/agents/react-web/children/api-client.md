---
knowledge-base-summary: "The single HTTP layer: one configured client instance, auth-token injection, and a single-flight refresh-and-retry path that queues concurrent 401s instead of firing N refreshes. The boundary discipline that matters most — a hand-written response type is an assertion the compiler will never check, so responses are parsed at runtime and interfaces are diffed against the contract they mirror; a green `tsc` is never evidence a front end survived an API change. Plus the encoding traps: a client pinned to a JSON content type silently destroys a multipart body, and error responses carry a field-keyed dictionary whose key casing follows the backend's property names, not JSON convention."
---

# Api Client

Every request the app makes goes through **one** configured client instance. That instance is where
the base URL, the timeout, the auth header and the refresh path live, and it is the only reason those
concerns exist in one place instead of at three hundred call sites.

The rule that keeps it true: **a second ad-hoc instance, or a bare `fetch`, silently bypasses every
interceptor.** It will work in development, where the token is fresh, and fail on the first expiry.
If you need different behaviour for one call (a different content type, no auth), express it as a
per-request option on the shared instance — not as a second instance.

## Auth on the request path

The request interceptor reads the current token and sets the authorization header. Components never
attach a token by hand; if you find one that does, that is the bug, not the pattern.

Read the token from the store's **imperative getter**, not through a hook — an interceptor is not a
component and has no render to subscribe to. This is also why many projects keep the access token
outside the persisted slice of the auth store: the interceptor must be able to read it before
rehydration finishes. That split is deliberate and it is exactly what creates the guard race in
[routing-and-guards.md](routing-and-guards.md) — know that you are looking at one half of a designed
pair, and do not "tidy" it by moving the token into the persisted slice.

## Refresh must be single-flight

When a token expires, the app usually does not make one request — a screen mounts and fires five
queries at once, and all five come back 401. The naive response interceptor refreshes on each, which
means N refresh calls in flight. If the backend rotates the refresh token (a common and correct
design), the first call invalidates the token the other four are holding, four refreshes fail, and
the user is thrown to the login screen by an app that had a perfectly valid session.

The fix is a module-level flag plus a queue:

```ts
let isRefreshing = false;
let queue: Array<{ resolve: (t: string) => void; reject: (e: unknown) => void }> = [];

function flush(error: unknown, token: string | null) {
  queue.forEach(({ resolve, reject }) => (token ? resolve(token) : reject(error)));
  queue = [];
}
```

The response interceptor then reads:

1. Not a 401, or this request has already been retried once → reject. The **already-retried marker on
   the request config is load-bearing**: without it a request that 401s again after a successful
   refresh retries forever.
2. A refresh is already in flight → push a resolver onto the queue and return that promise; when it
   resolves, re-issue the original request with the new token.
3. Otherwise → mark the request retried, set the flag, refresh once, `flush` the queue with the new
   token, re-issue this request. Clear the flag in a `finally`, or one failed refresh wedges every
   subsequent 401 forever.
4. Refresh failed → `flush` with the error, clear the session, and go to login.

**Failed refresh is a one-way trip, and that direction is a diagnostic.** The failure path clears the
tokens and hard-navigates to login; there is no path back from it. So when someone reports a login
flash that *ends on an authenticated screen*, it cannot be this interceptor — do not spend a session
in here. That symptom is the hydration race in [routing-and-guards.md](routing-and-guards.md).

## The boundary the compiler does not guard

Front-end response types are, in almost every project, **hand-written interfaces that mirror a
backend type nothing checks them against**. A hand-written interface is an assertion, not a check:
declare a field the backend does not send and the compiler lets you read it, `undefined` at runtime,
with nothing reporting anything.

This was measured, not theorised. Two fields were renamed across an entire API; `tsc -b` reported
**zero errors** in the front end while every value that depended on them would have rendered
`undefined`. Worse, the same contract was declared three times in one tree, so fixing one copy fixed
one screen.

Three obligations follow, and they are cheap:

- **Parse at the boundary, once.** Put the runtime schema check inside the client wrapper — a
  method that takes an optional schema and parses the response before returning it — so validation
  costs one place, not every call site. A parse scattered into components is a parse that will be
  skipped.
- **Diff the interface against the contract it mirrors whenever you add or edit one**, and grep the
  whole tree for other copies of the same shape before you call the edit complete.
- **Never report a compiling front end as evidence it survived an API-contract change.** The
  instrument on this side is the browser ([browser-verification.md](browser-verification.md)).

A schema validator is usually already installed for form input; pointing it at responses is normally
a matter of using it, not adding a dependency. Where a runtime parse is genuinely too expensive for a
hot path, say so explicitly rather than leaving the boundary silently unchecked.

**Diagnostic reflex:** a React key warning that fires while a key demonstrably *is* being passed
usually means the value you keyed on is `undefined` — suspect the response type before the JSX.

## Two encoding traps

Both produce a correct-looking request, no error on either side, and a symptom that reads as a
backend bug.

### A JSON content type destroys a multipart body

An HTTP client created with a default `'Content-Type': 'application/json'` — the near-universal
setup — will **serialize a `FormData` body away to nothing**. The request arrives as a JSON object
with an empty value where the file was. Nothing throws. The server, binding a file from multipart,
fails to bind, and you go looking for the fault on the wrong side of the network.

Override the header on every multipart request:

```ts
const body = new FormData();
body.append('file', file);
await client.post(url, body, { headers: { 'Content-Type': 'multipart/form-data' } });
```

Measured on the axios 1.x family in a browser, across three builds of it: only a declared type
containing `application/json` takes the destructive branch. Declaring `'multipart/form-data'` does
**not** lose the boundary — the browser adapter strips the header and lets the browser serialize the
body with its own boundary (the library writes a boundary itself only in its Node adapter). Setting
the header to `undefined` is equivalent. **What is never correct is leaving the header off entirely**,
because then the instance default wins and the body is destroyed.

Two consequences worth knowing before you write the call:

- The override rides on the **per-request config**. A thin wrapper method that takes no config
  argument cannot express it — so either give the wrapper a config parameter or reach for the
  underlying instance directly for that one call. Reaching for a *different* instance is the wrong
  escape hatch: a second instance usually carries the same JSON default and no auth interceptor,
  which is strictly worse.
- If you meet either form in an existing tree, they are equivalent — do not "fix" one into the other.

This is version-bound behaviour of one client library. If the project uses a different client, or a
different major, confirm it the same way it was confirmed here: post a `FormData` and look at the
actual request body on the wire.

### A validation-error payload is keyed by the backend's property casing

A field-keyed error dictionary comes back keyed by the **backend's own property names** — PascalCase
from a .NET validator, snake_case from several others — not by the JSON casing you see in the success
body. A camelCase lookup finds nothing, so the per-field message **silently fails to render while the
response was perfectly correct**, and it reads as "validation is broken".

Read one real error body before writing the mapping. Do not infer the key casing from the success
response, and do not infer it from the DTO you wrote.

## Normalize errors once

Give the app one error shape — status code, a display message, and the optional field dictionary —
produced by one function at the client layer:

```ts
type ApiError = { statusCode: number; message: string; fields?: Record<string, string[]> };
```

Every consumer then branches on that instead of re-deriving it from the raw client error. The
consumer's job is routing, not parsing: a field-keyed entry goes onto the field that caused it, a
status the screen knows how to explain gets its own branch, everything else falls to a banner.
Mapping a server rejection back onto the form field is [forms-and-input.md](forms-and-input.md); this
file only guarantees the rejection arrives in one predictable shape.

Where the server sends **message keys rather than final text**, resolve them through the project's
i18n layer with the raw key as the last-resort fallback — never render a key to a user, and never
build a second translation path for API strings.
