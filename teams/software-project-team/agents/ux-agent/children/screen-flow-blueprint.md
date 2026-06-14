---
knowledge-base-summary: "The primary production unit of this agent. How to map user journeys from entry points through decision points to outcomes. Flow diagram conventions, auth flow and purchase flow as worked examples, and a comprehensive checklist for completeness (reachability, back navigation, error paths, loading states)."
---
# Screen Flow Blueprint

This is the primary production unit of the UX Agent. Every new feature starts with a screen flow specification before any code is written.

## What is a Screen Flow

A screen flow maps the complete user journey for a feature. It defines:
- **Entry points** — how the user arrives at this flow
- **Screens** — what the user sees at each step
- **Decision points** — where the user makes choices or the system branches
- **Outcomes** — where each path ends (success, error, redirect)
- **Data requirements** — what data each screen needs and where it comes from

## Flow Diagram Convention

Use text-based flow diagrams with this notation:

```
[Screen Name]           → A screen the user sees
(Decision)              → A branching point (user choice or system check)
{Action}                → A system action (API call, redirect, calculation)
<State>                 → A UI state (loading, error, empty)
-->                     → Flow direction
|                       → Branch separator
```

### Example Notation

```
[Login Screen]
  --> {Submit credentials}
  --> <Loading>
  --> (Auth result?)
      | Success --> {Store tokens} --> [Home Screen]
      | Invalid credentials --> [Login Screen + error message]
      | Account locked --> [Account Locked Screen]
      | Network error --> [Login Screen + retry option]
```

## Screen Flow Template

Every screen flow spec must include these sections:

### 1. Overview
- Feature name
- One-line description
- Primary user goal (what are they trying to accomplish?)
- Trigger (what starts this flow?)

### 2. Entry Points
List every way a user can enter this flow:
- Direct navigation (menu, tab)
- Deep link (URL, push notification)
- Redirect (after another action completes)
- Context action (button on another screen)

### 3. Flow Diagram
The text-based diagram showing all paths.

### 4. Screen Inventory
For each screen in the flow:
- Screen name
- Purpose (one sentence)
- Data displayed (what the user sees)
- Actions available (what the user can do)
- Navigation targets (where each action leads)

### 5. State Definitions
For each screen:
- **Loading state** — what shows while data loads
- **Empty state** — what shows when there's no data
- **Error state** — what shows when something fails
- **Success state** — what shows when action completes

### 6. Edge Cases
- What if the user navigates back mid-flow?
- What if the session expires during the flow?
- What if the user force-closes and reopens?
- What if the user has no network?
- What if the data changes while viewing (another user edited it)?

## Worked Example: Auth Flow

```
FEATURE: Authentication
GOAL: User signs in to access the app
TRIGGER: App launch (not authenticated) or session expiry

ENTRY POINTS:
  1. App cold start (no stored tokens)
  2. Token refresh failure (session expired)
  3. Explicit logout then re-login
  4. Deep link requiring auth

FLOW:
[Splash Screen]
  --> {Check stored tokens}
  --> (Has valid refresh token?)
      | Yes --> {Refresh access token}
          --> (Refresh success?)
              | Yes --> [Home Screen]
              | No --> [Login Screen]
      | No --> [Login Screen]

[Login Screen]
  Actions: enter email + password, submit, forgot password, register
  --> (User action?)
      | Submit --> {Validate inputs locally}
          --> (Valid?)
              | No --> [Login Screen + field errors]
              | Yes --> {Call login API}
                  --> <Loading overlay>
                  --> (API result?)
                      | Success --> {Store tokens} --> [Home Screen]
                      | Invalid credentials --> [Login Screen + "Email or password incorrect"]
                      | Account locked --> [Account Locked Info Screen]
                      | Email not verified --> [Email Verification Prompt]
                      | Rate limited --> [Login Screen + "Too many attempts. Try again in X minutes"]
                      | Network error --> [Login Screen + "Connection failed. Check your internet."]
      | Forgot password --> [Forgot Password Screen]
      | Register --> [Registration Screen]

[Registration Screen]
  Actions: enter name + email + password + confirm password, submit, back to login
  --> (User action?)
      | Submit --> {Validate inputs locally}
          --> (Valid?)
              | No --> [Registration Screen + field errors]
              | Yes --> {Call register API}
                  --> <Loading overlay>
                  --> (API result?)
                      | Success --> [Email Verification Prompt]
                      | Email taken --> [Registration Screen + "This email is already registered"]
                      | Network error --> [Registration Screen + retry option]
      | Back --> [Login Screen]

[Email Verification Prompt]
  Shows: "Check your email" message with resend option
  --> (User action?)
      | Open email app --> {Launch email app intent}
      | Resend --> {Call resend API}
          --> (Cooldown active?)
              | Yes --> [Show countdown timer]
              | No --> {Send verification email} --> [Show "Email sent" confirmation]
      | Already verified --> [Login Screen]

[Forgot Password Screen]
  Actions: enter email, submit, back to login
  --> (User action?)
      | Submit --> {Call forgot password API}
          --> <Loading>
          --> [Success message: "If this email exists, we sent a reset link"]
      | Back --> [Login Screen]
```

### Screen Inventory for Auth Flow

| Screen | Purpose | Data | Actions | States |
|--------|---------|------|---------|--------|
| Splash | Check auth status | App logo | None (auto) | Loading only |
| Login | Collect credentials | Email + password fields | Submit, forgot, register | Default, loading, error |
| Registration | Create account | Name + email + password fields | Submit, back | Default, loading, error |
| Email Verification | Prompt verification | Instruction text | Resend, open email, verified | Default, cooldown |
| Forgot Password | Request reset | Email field | Submit, back | Default, loading, success |
| Account Locked | Inform lockout | Lock info + support contact | Contact support | Static |

## Worked Example: Purchase Flow

```
FEATURE: Purchase
GOAL: User buys a product
TRIGGER: User taps "Buy Now" on product detail

FLOW:
[Product Detail]
  --> {Tap Buy Now}
  --> (User authenticated?)
      | No --> [Login Screen] --> (after login) --> [Cart Review]
      | Yes --> [Cart Review]

[Cart Review]
  Shows: product, quantity, price breakdown
  --> (User action?)
      | Edit quantity --> {Update cart} --> [Cart Review updated]
      | Remove item --> {Confirm removal} --> (Cart empty?)
          | Yes --> [Empty Cart State] --> [Continue Shopping]
          | No --> [Cart Review updated]
      | Apply coupon --> {Validate coupon}
          --> (Valid?)
              | Yes --> [Cart Review + discount applied]
              | No --> [Cart Review + "Invalid or expired coupon"]
      | Proceed --> [Address Selection]

[Address Selection]
  Shows: saved addresses, add new option
  --> (Has saved addresses?)
      | Yes --> [Address list with select + add new]
      | No --> [Add Address Form]
  --> (User action?)
      | Select address --> [Payment Method]
      | Add new --> [Add Address Form] --> [Address Selection]

[Payment Method]
  Shows: saved payment methods, add new option
  --> (User action?)
      | Select method --> [Order Summary]
      | Add new --> [Add Payment Form] --> [Payment Method]

[Order Summary]
  Shows: product, address, payment, total
  --> (User action?)
      | Edit cart --> [Cart Review]
      | Edit address --> [Address Selection]
      | Edit payment --> [Payment Method]
      | Confirm --> {Process payment}
          --> <Loading: "Processing your order...">
          --> (Payment result?)
              | Success --> [Order Confirmation]
              | Payment failed --> [Order Summary + "Payment failed. Try another method."]
              | Network error --> [Order Summary + "Connection lost. Your order was not placed."]
              | 3D Secure required --> [3D Secure WebView]
                  --> (3D Secure result?)
                      | Authenticated --> {Complete payment} --> [Order Confirmation]
                      | Failed --> [Order Summary + "Authentication failed"]
                      | Cancelled --> [Order Summary]

[Order Confirmation]
  Shows: order number, estimated delivery, order details
  Actions: view order, continue shopping, share
```

## Flow Completeness Checklist

Before a screen flow is considered complete, verify every item:

### Reachability
- [ ] Every screen is reachable from at least one entry point
- [ ] No orphan screens (screens that nothing links to)
- [ ] Deep link routes defined for shareable screens

### Navigation
- [ ] Back navigation works from every screen (where does back go?)
- [ ] Hardware back button behavior defined (Android)
- [ ] Swipe-to-go-back behavior defined (iOS)
- [ ] Navigation state preserved when switching tabs

### States
- [ ] Every screen has a loading state defined
- [ ] Every screen that displays data has an empty state defined
- [ ] Every screen with API calls has an error state defined
- [ ] Success feedback defined for every user action

### Error Paths
- [ ] Network failure handled at every API call point
- [ ] Invalid input handled at every form
- [ ] Permission denied handled where applicable
- [ ] Session expiry handled mid-flow
- [ ] Rate limiting handled where applicable

### Edge Cases
- [ ] Flow interruption handled (force close, phone call)
- [ ] Multi-device scenario considered (started on phone, continues on tablet)
- [ ] Data staleness addressed (viewing outdated data)
- [ ] Concurrent modification addressed (two users editing same data)

### Accessibility
- [ ] Screen reader announcement defined for dynamic content changes
- [ ] Focus management defined after navigation (where does focus land?)
- [ ] Keyboard navigation order defined for web

### Platform
- [ ] Mobile layout defined (or confirmed as primary)
- [ ] Tablet layout differences noted (if applicable)
- [ ] Web layout differences noted (if applicable)

## Naming Conventions

- Flow file: `{feature-name}-flow.md` (e.g., `auth-flow.md`, `purchase-flow.md`)
- Screen names: PascalCase descriptive names (e.g., `[Cart Review]`, `[Order Confirmation]`)
- Decisions: short questions ending with `?` (e.g., `(User authenticated?)`)
- Actions: imperative verbs (e.g., `{Submit credentials}`, `{Validate inputs}`)

## Handoff to Implementation Agents

Once a screen flow is approved:

1. **Flutter Agent** receives: screen inventory, state definitions, navigation targets
2. **React Agent** receives: screen inventory, state definitions, navigation targets (web-specific)
3. **Design System Agent** receives: component requirements extracted from screen descriptions
4. **API Agent** receives: data requirements, action endpoints needed
