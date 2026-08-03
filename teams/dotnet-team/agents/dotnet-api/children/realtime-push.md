---
knowledge-base-summary: "The server half of the push channel: hubs as bridges rather than logic, authenticating a connection whose transport cannot carry headers, group and connection tracking, and the delivery contract that keeps a push cheap — **persist the record first as the source of truth, then push best-effort**, short-timeout and swallowed on failure because the client's next fetch already covers it. Token lifetime is the sharp edge: an idle client generates no other traffic to refresh with, so expiry handling that looks safe on the server kills push precisely while nobody is watching."
---

# Realtime Push

The **server half** of a SignalR push channel. The client half — one connection per tab, invalidating
caches instead of patching them, the first-connect retry, relaxed polling while the channel is up —
belongs to the web specialist and is deliberately not restated here; the last section names the seam.

## The delivery contract — persist first, push best-effort

This is the one rule that makes a push cheap enough to add anywhere:

1. **Persist the record. It is the source of truth.** The row is what the client's next fetch will
   find, what a reconnecting client sees, and what a client that was offline gets.
2. **Then push, best-effort.** Bounded concurrency, a short timeout (single-digit seconds), failures
   **logged and swallowed**.

A push failure must never fail the write, roll back the transaction, or be retried into a queue,
because the persisted row already covers the miss. Inverting the order — push, then persist — buys a
few milliseconds and creates a notification the user saw once and can never find again.

The corollary a worker gets wrong most often: **a delivered push is not a delivery guarantee.**
Server-to-client sends have no acknowledgement. If something must be seen, it is a row, and the push
is only the hint that the row exists.

## What crosses the socket, and what does not

**The socket carries signals; REST carries data.** The event says *what happened*; the client fetches
the detail over HTTP.

| Belongs on the socket | Belongs on REST |
|---|---|
| "something changed, refetch it" | the object itself, and any list |
| status transitions, progress ticks | CRUD, uploads, anything transactional |
| presence, typing, live counters | anything paginated, filtered or sorted |
| a small notification the client can render as-is | anything the client needs an answer to |

Three decision rules, in order of how often they settle it:

- **If the client needs a response, use REST.** A hub send is fire-and-forget.
- **If it needs pagination, filtering or sorting, use REST.** A socket event has no query surface.
- **If the payload is routinely over a kilobyte, use REST.** Send the id and let the client fetch.

## The hub is a transport, not a decision-maker

A hub method **parses, forwards, returns** — and nothing else. No business rules, no authorization
decisions of its own, no reaching around the application layer for data. If you are writing an `if`
that makes a product decision inside a hub, it belongs behind the hub.

That rule holds in both deployment shapes, and the shape changes only *how* the hub reaches the
decision:

- **Hub inside the API host.** The hub calls the same application-layer entry point an endpoint would.
- **Hub in its own host.** The hub has no persistence stack at all and reaches the API over HTTP;
  the API pushes outward by calling an internal endpoint on the socket host, which fans out through
  `IHubContext`. That internal endpoint is **not** JWT-protected — it is system-to-system, guarded by
  a shared secret header — and it must be the *only* unauthenticated surface on that host.

Standing shape for a hub, either way:

```csharp
[Authorize]                                   // no anonymous hub access, ever
public sealed class NotificationsHub : Hub
{
    public override async Task OnConnectedAsync()    { /* join groups, track, log */ }
    public override async Task OnDisconnectedAsync(Exception? ex) { /* untrack, log */ }
}
```

**Errors out of a hub method:** throw `HubException` with a message that is safe for a client to see.
Any *other* exception type is replaced with a generic message before it reaches the client (unless
detailed errors are enabled, which is a development-only setting) — so an opaque client-side error
usually means an **unwrapped** exception, not a missing one. Log the real exception server-side and
rethrow it as a `HubException`.

## Authenticating a transport that cannot carry headers

A browser's WebSocket API provides no way to set request headers on the upgrade, so
`Authorization: Bearer …` is simply unavailable at connect time. The token therefore rides the
**query string**, and the server lifts it back into the normal JWT pipeline:

```csharp
options.Events = new JwtBearerEvents
{
    OnMessageReceived = ctx =>
    {
        var token = ctx.Request.Query["access_token"];
        // the path check is load-bearing — see below
        if (!string.IsNullOrEmpty(token) && ctx.HttpContext.Request.Path.StartsWithSegments("/hubs"))
            ctx.Token = token;
        return Task.CompletedTask;
    },
};
```

- **The path check is not tidiness.** Without it, every request on the host starts accepting a
  bearer token from the query string — including endpoints that authenticate by another mechanism
  entirely. Scope the lift to the hub path prefix.
- **Clients do not build that query string by hand.** The SignalR clients take a token *factory* and
  append it themselves; server-side you only need the lift.
- **Validation parameters must match the API's exactly** — issuer, audience, signing key, and
  lifetime — or a token that is good on one is rejected on the other. Where they are configured from
  the environment, they are the *same* values, not parallel ones.
- **Security consequences of the query string:** it lands in access logs and proxy logs. Require TLS,
  scrub or exclude the parameter from request logging, and never log the raw token yourself. Log the
  user id and the connection id.

## Identity, groups, and knowing who is connected

- **`Context.UserIdentifier`** is what `Clients.User(id)` targets, and by default it comes from the
  name-identifier claim. If your tokens carry the user id under a different claim, register an
  `IUserIdProvider` — otherwise every user-targeted push silently reaches nobody.
- **Groups are the targeting primitive.** Name them `type:id` (`tenant:…`, `room:…`, `role:…`) — an
  unprefixed group named `42` is ambiguous the moment there are two kinds of thing. Group names are
  **case-sensitive**; normalize before use.
- **Join what is derivable from claims in `OnConnectedAsync`**; join anything that needs an access
  check through a hub method that *asks the application layer first*, never on the client's word.
- **Disconnect cleans up automatically** — SignalR removes a dropped connection from every group.
  Explicit removal in `OnDisconnectedAsync` is documentation, not a requirement.
- **You cannot enumerate a group.** SignalR has no "who is in this group" or "how many" API. If the
  product needs online status or a live count, track it yourself in a shared store: a set of
  connection ids per user (a user has many — tabs and devices), plus a cheap existence key. Remove
  the connection on disconnect and drop the user's keys only when the set empties.
- **A crashed host never fires `OnDisconnectedAsync`,** so external tracking accumulates ghosts.
  Choose a recovery: a TTL refreshed while the connection lives, or a sweep of that host's keys at
  startup. Pick one and write it down; "we'll notice" is not one.
- **Multiple server instances need a backplane.** Without one, a client on instance A never receives a
  group broadcast sent from instance B. This is invisible in development, where there is one process,
  and it is the most common "it worked locally" failure in this topic.

## Event naming and payload evolution

An event name and its payload are a **public API contract** with every client version currently
installed. Unlike an HTTP route, you cannot version the URL — the server pushes to whatever is
connected.

- **One casing convention, applied everywhere.** The name is a string matched exactly on both sides;
  a mismatch fails **silently** — no error, no log, nothing happens. Pin the convention (kebab-case is
  a common choice), and prefix by feature so a feature's events are greppable as a set.
- **Direction shows in the tense:** server→client is past (`order-confirmed`), client→server is
  imperative (`join-room`).
- **The payload is a typed record, never a bare primitive or a loose anonymous object** — a
  primitive gives the client nothing to evolve.
- **Carry a timestamp and an event id.** The timestamp orders events and identifies stale ones after
  a reconnect; the id lets a client de-duplicate a replay.
- **Never remove or retype a published field.** New fields are optional/nullable — an older client
  ignores them, and a newer client must tolerate their absence during a rolling deploy.
- **A genuinely breaking change is a new event name**, sent alongside the old one until the old
  clients are gone. Sending both is the cost of not breaking anyone.
- **Keep user-facing text out of the payload if the client localizes it** — send a key plus
  placeholders, the same envelope the error contract uses.

## Token lifetime — the sharp edge

The token is presented **at negotiate** and, by default, nothing re-checks it afterwards: a
connection outlives its own token until something drops it. Tightening that is correct, and it is
also where this topic bites hardest, because **the recovery lives on the client**.

Three facts, in the order they matter:

1. **Pin `ClockSkew` on the hub host to the API's.** The JWT validation default is **five minutes**.
   Leave it and the hub keeps accepting a token the API has already started rejecting — the two
   surfaces disagree about whether the user is logged in, for minutes, and nothing reports it.
2. **`CloseOnAuthenticationExpiration` closes the connection when its token expires** — set on the
   hub's connection options at `MapHub`. It is the right end state.
3. **Turn it on only once every client can recover on its reconnect path.** This is the ordering
   lesson, and it was paid for: a client that is merely *idle* generates **no other traffic** — an
   unfocused tab pauses its polling — so nothing else refreshes its token for the socket. Without a
   refresh hook on the reconnect path, the client just re-presents the same expired token forever.
   Measured once, deliberately, on one-minute tokens: a client looped unauthorized negotiate attempts
   for **~16 minutes with zero refreshes**, recovering only when a focus event happened to trigger
   other traffic. Enabling expiry-close before that hook exists kills push **precisely while nobody
   is looking** — the failure is invisible by construction.

A client has **two independent retry surfaces** (its first-connect loop and its automatic-reconnect
policy) and a refresh hook on one does not cover the other. Confirm **both**, on **every** client,
before flipping the flag.

**Known, bounded cost once it is on:** between expiry and expiry + `ClockSkew` a reconnect still
succeeds with the old token and is closed again — a short burst of very short-lived connections per
client per expiry cycle, ending when validation starts rejecting and the refresh fires. Measure it
once and leave the number, with its token lifetime, in a comment at the `MapHub` call so the next
reader does not re-derive it.

> Generalize the ordering: **a server-side tightening whose recovery lives in the client ships after
> the client half, not before.** State the gate explicitly ("this flag is off until every client
> carries a reconnect-path refresh") or it gets flipped by someone reading only the server.

## Adding a push event, end to end

1. **Decide it belongs on the socket at all** — apply the boundary table above. Most "make it live"
   requests are answered by an invalidate-and-refetch signal, not by shipping the object.
2. **Define the payload as a typed record** with a timestamp and an event id; write the name into
   the project's event catalog in `Conventions/` in the same change.
3. **Pick the target deliberately** — caller, one user (all their connections), a group, a group
   except the caller, or everyone. Wrong target is the most common defect here, and it is a *silent*
   one: nothing errors when a broadcast reaches nobody.
4. **Persist first, then dispatch** through the one place that owns pushing (a notification service
   or the internal endpoint) — never `IHubContext` inline from a handler, or you have a second owner
   for the delivery contract.
5. **Register.** If it is a new hub: `MapHub<T>("/hubs/<name>")`, with the connection options, and
   the auth token lift's path prefix must cover that route. If it is a new event on an existing hub,
   the registration step is the **client subscription** — an event nothing listens for is exactly as
   dead as an unmapped hub.
6. **Verify by observation.** See below.

**The false green.** Everything in this list can be correct except step 5, and every gate stays
green: the record persists, the handler returns 200, the dispatch call succeeds (a push to a group
with no members is a **no-op, not an error**), the unit test passes. Nothing on the server can
distinguish "delivered" from "sent into the void". So the verification for a push event is not a
server assertion — it is **watching a real client receive it**:

- [ ] A client connected and authenticated (check the connection appears, and with the expected
      user identity — not just that it connected).
- [ ] The event arrived **at that client**, under the exact name the client subscribes to.
- [ ] The persisted record exists **independently** of the push having worked.
- [ ] The push failing (stop the hub host, or block it) leaves the write succeeding and the request
      returning normally — the best-effort contract, actually exercised.
- [ ] If there is more than one server instance in any environment: verified **across** instances, or
      the backplane is missing and you will find out in production.

## The other half — the client's job, not this agent's

Named so neither side assumes the other handles it. All of this belongs to the web specialist:

- **One connection per tab**, owned by the app shell.
- **Invalidate, do not patch** — a push means "this is stale, refetch"; REST stays the source of truth.
- **A first-connect retry loop.** The client library's automatic reconnect covers only a connection
  that was already established once — a client that boots while the hub is down is push-less for the
  whole session with no recovery short of a reload.
- **A refresh hook on *both* retry surfaces**, which is the precondition for the expiry-close flag above.
- **Relaxed polling while the channel is up, snapping back the moment it drops** — and left alone for
  data no push covers.
- **The regression bar:** with the hub unreachable the app must still render, authenticate and
  navigate, with nothing worse than a swallowed warning.

## Related

- [messaging-and-background-work](messaging-and-background-work.md) — the queued fan-out that
  produces most pushes, and why a background writer has no ambient identity.
- [error-contract](error-contract.md) — the same "key plus placeholders, never a raw sentence"
  envelope, on the HTTP side.
- [multi-tenancy-and-authorization](multi-tenancy-and-authorization.md) — why a group join that skips
  the server-side access check is not a boundary.
- [testing-and-verification](testing-and-verification.md) — why a real dependency is required for
  anything whose behaviour lives outside C#.
