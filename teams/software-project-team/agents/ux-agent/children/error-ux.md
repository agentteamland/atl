---
knowledge-base-summary: "Error as opportunity, not dead end. Error message anatomy (what happened + why + what to do). Network errors, form errors, 404, permission denied — each with specific recovery guidance. Retry patterns (automatic vs manual). Offline mode messaging."
---
# Error UX

## Core Principle

Every error is a crossroads. The user tried to do something and it didn't work. A good error experience acknowledges the problem, explains it in human terms, and provides a clear path forward. A bad error experience leaves the user stranded, frustrated, and likely to leave.

## Error Message Anatomy

Every error message has three parts:

1. **What happened** — describe the problem clearly
2. **Why it happened** — brief, honest explanation (when possible)
3. **What to do next** — the recovery action

```
What happened:  "Unable to save your changes"
Why:            "The server is temporarily unavailable."
What to do:     "Your changes are saved locally. We'll sync automatically when the connection is restored."
```

### Writing Rules

**Be specific, not generic:**
- BAD: "An error occurred"
- GOOD: "Unable to load your orders"

**Be human, not technical:**
- BAD: "Error 500: Internal Server Error"
- GOOD: "Something went wrong on our end. We're looking into it."

**Be helpful, not blaming:**
- BAD: "You entered an invalid value"
- GOOD: "Phone number should be 10 digits (e.g., 555-123-4567)"

**Be honest when appropriate:**
- BAD: "Please try again" (no explanation)
- GOOD: "Our servers are experiencing high traffic. Please try again in a few minutes."

## Error Types and Patterns

### Network Errors

**Connection Lost (device offline):**
```
+----------------------------------------------+
| [Wifi-off icon]                              |
|                                              |
| No internet connection                       |
|                                              |
| Check your WiFi or cellular data.            |
| We'll reconnect automatically.               |
|                                              |
| [Retry Now]                                  |
+----------------------------------------------+
```

- Show a persistent banner at the top when connectivity is lost
- Auto-dismiss the banner when connectivity returns
- Queue user actions during offline (when possible)
- Show "Offline" indicator in the status area
- Resume automatically when connection returns (no user action needed)

**Server Unreachable (server down):**
```
We're having trouble connecting to our servers.
This is usually temporary. [Try Again]
```

- Don't say "server error" — users don't know what a server is
- Offer manual retry button
- Automatic retry with exponential backoff (1s, 2s, 4s, 8s...)
- After 3 auto-retries, stop and let the user decide

**Timeout:**
```
This is taking longer than expected.
[Keep Waiting]  [Try Again]
```

- Show after 10-15 seconds of no response
- Give user choice: wait more or retry
- Don't show timeout for background operations (just retry silently)

---

### Form Errors

**Field-Level Validation:**
```
Email
[john@invalid_______]
! Email must include a valid domain (e.g., name@example.com)

Password
[****]
! Password must be at least 8 characters
  [x] 8+ characters  [ ] Uppercase  [x] Number  [ ] Special char
```

**Rules:**
- Show error below the specific field (not at the top of the form)
- Red border on the field + red error text below
- Error icon (!) before the message
- After submit with errors: scroll to and focus the first error field
- Clear the error when user corrects the input (on blur or on change)
- Don't clear all errors at once — let each field validate independently

**Cross-Field Validation:**
```
+----------------------------------------------+
| ! Passwords don't match                       |
+----------------------------------------------+
```
Show at the top of the form or between the related fields.

**Server-Side Validation:**
```
[Save]  ! Email is already registered. Did you mean to log in?
        [Log In Instead]
```
Show as inline error after the API responds. Provide a direct link to the alternative action.

---

### Not Found (404)

```
+----------------------------------------------+
|                                              |
|  [Illustration: lost/searching]              |
|                                              |
|  Page not found                              |
|                                              |
|  The page you're looking for doesn't exist   |
|  or has been moved.                          |
|                                              |
|  [Search]  [Go to Home]                      |
+----------------------------------------------+
```

**Rules:**
- Never show "404" as the only information
- Suggest alternatives: search, home page, recent items
- If the URL looks like a known pattern (e.g., /products/123), suggest related content
- Log 404s to identify broken links
- Include a search bar on the 404 page

---

### Permission Denied (403)

```
+----------------------------------------------+
|                                              |
|  [Lock icon]                                 |
|                                              |
|  You don't have access to this page          |
|                                              |
|  This content is restricted to team admins.  |
|  Contact your admin to request access.       |
|                                              |
|  [Request Access]  [Go Back]                 |
+----------------------------------------------+
```

**Rules:**
- Explain WHO has access (role, permission level)
- Provide a way to request access (if applicable)
- Don't expose the content (no partial views)
- If the user was logged in with the wrong account: "You're signed in as user@example.com. [Switch Account]"

---

### Rate Limiting (429)

```
Too many requests. Please wait a moment and try again.
You can retry in 45 seconds. [countdown timer]
```

**Rules:**
- Show a countdown timer if possible
- Don't make the user guess when to retry
- For login rate limiting: "Too many login attempts. Try again in 15 minutes."
- Consider showing a CAPTCHA after several failures instead of hard blocking

---

### Server Error (500)

```
+----------------------------------------------+
|                                              |
|  [Warning icon]                              |
|                                              |
|  Something went wrong                        |
|                                              |
|  We encountered an unexpected problem.       |
|  Our team has been notified.                 |
|                                              |
|  [Try Again]  [Contact Support]              |
|                                              |
|  Error reference: ERR-A7B3C                  |
+----------------------------------------------+
```

**Rules:**
- Never show the actual error message or stack trace
- Assure the user that the team knows about it
- Provide a reference code (not the technical error) for support
- Offer retry and support contact options
- Log everything server-side for debugging

---

### Conflict Error (409)

```
+----------------------------------------------+
|  This item was modified by someone else.     |
|                                              |
|  [View Their Changes]  [Overwrite with Mine] |
+----------------------------------------------+
```

**Rules:**
- Explain that someone else made changes
- Show a diff if possible
- Give user the choice: accept theirs, keep mine, or merge
- Never silently overwrite

## Retry Patterns

### Automatic Retry
**When:** Network errors, temporary server issues.

**Rules:**
- Exponential backoff: 1s, 2s, 4s, 8s, 16s (cap at 30s)
- Maximum 3-5 automatic retries
- Show "Retrying..." indicator during automatic retry
- After max retries, show manual retry button
- Jitter: add random milliseconds to prevent thundering herd

### Manual Retry
**When:** User-initiated action failed.

**Rules:**
- "Try Again" button clearly visible
- Button is specific: "Retry Payment", "Resend Message" (not just "Retry")
- Preserve user input (don't clear the form on error)
- Show what will be retried (the user's data is still there)

### No Retry
**When:** The error is permanent (invalid input, permission denied, resource deleted).

**Rules:**
- Don't offer retry if it will always fail
- Guide to the correct action instead
- "This item has been deleted by another user. [Go to list]"

## Offline Mode Messaging

### Entering Offline Mode
```
Banner: "You're offline. Some features may be limited."
```
- Show immediately when connection drops
- Non-blocking (user can continue using cached features)
- Show which features still work and which don't

### During Offline Mode
```
Queued: "Your message will be sent when you're back online."
Blocked: "This action requires an internet connection."
Cached: "Showing cached data from 10 minutes ago."
```

- Queue actions when possible (messages, edits, likes)
- Show queued action count: "3 changes pending sync"
- Block actions that can't be queued with clear explanation
- Show staleness indicator on cached data

### Returning Online
```
Banner: "Back online. Syncing your changes..."
Then: "All changes synced." (auto-dismiss after 3 seconds)
```

- Auto-sync queued actions
- Show sync progress if many items
- Notify if any queued actions failed during sync
- Remove the offline banner

## Error Prevention

The best error experience is no error at all. Prevent errors before they happen.

### Input Constraints
- Disable submit button until form is valid
- Input masks for formatted data (phone, credit card, date)
- Character counter for limited fields ("42/280")
- Dropdown/select instead of free text when options are known
- Auto-complete/suggest for known values

### Confirmation Before Risk
- "Are you sure?" before delete, cancel, discard
- Preview before publish, send, submit
- Summary before purchase, payment

### Graceful Degradation
- Show cached data when API fails (with staleness indicator)
- Disable features gracefully instead of crashing
- Queue actions for later instead of blocking
- Provide alternatives when primary path fails
