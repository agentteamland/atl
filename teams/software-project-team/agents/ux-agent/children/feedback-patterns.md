---
knowledge-base-summary: "System feedback for every user action. Loading patterns (skeleton vs spinner), success patterns (toast, redirect, inline), error patterns (field-level, toast, full-page), confirmation before destructive actions, progress indicators (determinate vs indeterminate), and optimistic UI strategy."
---
# Feedback Patterns

## Core Principle

Every user action must produce a visible system response. Silence is the enemy of good UX. The user should never wonder: "Did it work? Is it loading? Did something go wrong?"

## The Feedback Matrix

| User Action | System Response | Pattern |
|-------------|----------------|---------|
| Tap/click a button | Visual press state + action | Button state change |
| Submit a form | Loading -> success or error | Loading + result feedback |
| Delete something | Confirmation dialog -> result | Confirmation + feedback |
| Navigate | Page transition | Animation/transition |
| Scroll to bottom | Load more content | Loading indicator |
| Pull down (mobile) | Refresh data | Pull-to-refresh animation |
| Long press / right click | Context menu | Menu appearance |
| Swipe (mobile) | Action reveal | Swipe action |

## Loading Patterns

### Skeleton Screen (Structured Loading)
```
[____________________]     ← title placeholder
[__________] [______]      ← metadata placeholders
[____________________]
[____________________]     ← content placeholders
[____________________]
```

**When:** The content structure is known. Lists, cards, profiles, detail pages.

**Rules:**
- Match the layout of the actual content (same shapes and positions)
- Animate with a shimmer effect (left-to-right pulse)
- Never show skeleton for more than 3-5 seconds (if longer, add a message)
- Transition smoothly from skeleton to real content (no flash)

**Best for:** Lists, cards, user profiles, product details, dashboards.

---

### Spinner (Generic Loading)
**When:** The content structure is unknown or the loading area is small.

**Rules:**
- Use for inline loading (button submit, refresh)
- Center in the loading area
- Add a text message for long operations: "Loading your data..."
- Never fullscreen spinner without explanation (user thinks the app froze)

**Best for:** Button loading state, small widget refresh, modal loading.

---

### Progress Bar (Determinate Loading)
**When:** Progress can be measured (file upload, multi-step process).

**Rules:**
- Show percentage or step count: "Uploading... 67%" or "Step 2 of 4"
- Linear progress bar for uploads/downloads
- Circular progress for compact spaces
- Never fake progress (don't animate if you can't measure real progress)
- Show estimated time remaining for long operations: "About 2 minutes remaining"

**Best for:** File uploads, data imports, installation wizards.

---

### Loading Decision Guide

| Scenario | Pattern | Duration |
|----------|---------|----------|
| Page load, known structure | Skeleton | < 3 seconds |
| Button submit | Spinner inside button | < 2 seconds |
| API call, unknown result shape | Centered spinner + message | < 5 seconds |
| File upload | Progress bar + percentage | Variable |
| Background sync | Subtle indicator (top bar) | Variable |
| Pull-to-refresh | Platform-native pull indicator | < 2 seconds |

## Success Patterns

### Toast / Snackbar (Transient)
```
+----------------------------------------------+
|  checkmark  Profile updated successfully     |
+----------------------------------------------+
```

**When:** Action completed, no further user action needed. User should know it worked but doesn't need to dwell on it.

**Rules:**
- Auto-dismiss after 3-4 seconds
- Position: bottom of screen (mobile) or top-right (web)
- Green/success color with check icon
- Optional "Undo" action for reversible operations (e.g., "Archived. [Undo]")
- Don't stack more than 2 toasts at once
- Accessible: announce to screen readers

**Best for:** Save, update, send, archive operations.

---

### Redirect (Navigation)
**When:** Action completes and the user should see the result in context.

**Rules:**
- Redirect after successful creation to the new item's detail page
- Redirect after successful deletion to the list page
- Show a brief toast or inline message on the destination page
- Don't redirect on every action (only when context changes)

**Best for:** Create new item (go to detail), complete checkout (go to confirmation), submit form (go to list).

---

### Inline Success (In-Place)
**When:** Action completes within the current context, no navigation needed.

**Rules:**
- Replace the action area with a success message
- Example: "Save" button -> "Saved!" text for 2 seconds -> back to "Save" button
- Or: checkmark animation on the button itself
- Use for auto-save confirmation

**Best for:** Settings toggle, inline edit, quick update.

## Error Patterns

### Field-Level Error (Inline)
```
Email
[john@invalid_______]
! Please enter a valid email address (e.g., name@example.com)
```

**When:** A specific form field has invalid input.

**Rules:**
- Appears below the field
- Red/error color text and border
- Specific message (not generic "Invalid")
- Disappears when the user corrects the input
- Focus scrolls to and highlights the first error field

---

### Toast Error (Transient)
```
+----------------------------------------------+
|  !  Could not save. Please try again.        |
+----------------------------------------------+
```

**When:** Action failed but it's recoverable and doesn't need persistent attention.

**Rules:**
- Red/error color with error icon
- Auto-dismiss after 5 seconds (longer than success toasts)
- Include retry option if applicable: "Failed to send. [Retry]"
- Don't use for critical errors (user might miss it)

---

### Banner Error (Persistent)
```
+----------------------------------------------+
|  !  Connection lost. Changes will sync        |
|     when you're back online.          [Retry] |
+----------------------------------------------+
```

**When:** Ongoing issue that affects the entire page/app (network down, API unavailable).

**Rules:**
- Persistent — doesn't auto-dismiss
- Positioned at the top of the content area
- Dismissible by user (if informational) or auto-clears when resolved
- Include action if possible (Retry, Reconnect)

---

### Full-Page Error
**When:** Fatal error, the page cannot render at all.

**Rules:**
- Centered message with illustration/icon
- Clear explanation of what happened
- What the user can do: retry, go home, contact support
- Error code visible but not prominent (for support reference)
- Never show raw error messages or stack traces

## Confirmation Patterns

### Destructive Action Confirmation
```
+------------------------------------------+
|  Delete this order?                       |
|                                           |
|  This will permanently remove Order #1234 |
|  and all associated data. This action     |
|  cannot be undone.                        |
|                                           |
|              [Cancel]  [Delete]           |
+------------------------------------------+
```

**Rules:**
- Always confirm before: delete, cancel subscription, remove user, discard changes
- Title: question format ("Delete this order?")
- Body: explain the consequence
- Cancel button on the left (secondary style)
- Destructive button on the right (red/danger color)
- Button label matches the action ("Delete", not "OK" or "Yes")

### Non-Destructive Confirmation
- Send email: "Send to 42 recipients?" [Cancel] [Send]
- Publish: "Publish this article? It will be visible to everyone." [Cancel] [Publish]
- Don't over-confirm — simple saves, toggles, and navigation don't need confirmation

### Undo Instead of Confirmation
Sometimes "Undo" is better than "Are you sure?"
- Archive email: just do it, show "Archived. [Undo]" toast
- Move to trash: just do it, show "Moved to trash. [Undo]" toast
- Undo window: 5-10 seconds
- Use when: action is reversible and users do it frequently

## Progress Indicators

### Determinate (Measurable)
- File upload: progress bar with percentage
- Multi-step wizard: step indicator (Step 2 of 5)
- Data processing: percentage or items processed ("Processing 145 of 500 records")

### Indeterminate (Unknown Duration)
- API call: spinner or pulsing animation
- Search: "Searching..." with animated indicator
- Processing: "Please wait while we process your request..."

### Rules
- Never show indeterminate progress for operations longer than 10 seconds without explanation
- For operations > 30 seconds, show estimated time
- For operations > 2 minutes, allow background processing with notification on completion
- Show what's happening, not just "Loading...": "Uploading photos...", "Generating report..."

## Optimistic UI

### Pattern
Update the UI immediately as if the action succeeded, then handle the actual result:

```
User taps "Like" →
  Immediately: heart fills, count increments
  Background: API call to save like
  If API fails: revert heart, revert count, show error toast
```

### When to Use
- Social actions (like, follow, bookmark)
- Toggle operations (mark as read, star, pin)
- Simple updates (rename, reorder)
- Network is usually reliable and the action usually succeeds

### When NOT to Use
- Financial transactions (payments, transfers)
- Destructive actions (delete, cancel)
- Actions with side effects (send email, publish)
- Actions requiring server validation (submit order)

### Rules
- Revert immediately and silently on failure (don't show error for a state the user barely noticed)
- For frequent failures, reconsider: maybe the action shouldn't be optimistic
- Log failures for debugging even if the user doesn't see them
