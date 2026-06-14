---
knowledge-base-summary: "When to use push vs in-app vs email notifications. Urgency-based channel selection. Notification center design (bell icon, badge count, mark as read). Do-not-disturb respect. Notification grouping and batching strategies."
---
# Notification UX

## Core Principle

Notifications are a privilege, not a right. Every notification interrupts the user. It must earn that interruption by being timely, relevant, and actionable. When in doubt, don't notify.

## Channel Selection

### When to Use Which Channel

| Channel | When to Use | Urgency | User Action Needed |
|---------|-------------|---------|-------------------|
| Push notification | Time-sensitive, requires action soon | High | Yes, within hours |
| In-app notification | Informational, action at user's convenience | Medium | Yes, but not urgent |
| Email | Summary, receipt, can wait days | Low | Maybe, no rush |
| SMS | Critical/security (2FA, password reset) | Critical | Yes, immediately |
| No notification | User didn't ask for it, not relevant | None | No |

### Push Notifications

**Use for:**
- Order status changes (shipped, delivered)
- Payment events (payment failed, refund processed)
- Time-sensitive actions (appointment in 30 minutes)
- Security alerts (new login detected)
- Direct messages from other users

**Don't use for:**
- Marketing or promotional content (unless user opted in)
- Social activity that isn't directly about the user
- Informational updates that can wait
- Bulk notifications (5 separate order updates)

**Push notification anatomy:**
```
App Name                         2m ago
Title: Short, action-oriented
Body: One line of context. What to do.
```

**Rules:**
- Title: max 50 characters, tells the user what happened
- Body: max 100 characters, adds context or next step
- Deep link: tapping the notification opens the relevant screen (not the home screen)
- Collapsible: multiple notifications from the same app should stack/group
- Silent option: allow the user to receive without sound
- Action buttons: max 2 (e.g., "Reply", "Mark as Read")

---

### In-App Notifications

**Use for:**
- New messages or comments
- Task assignments or status changes
- Approval requests
- System updates (scheduled maintenance)
- Feature announcements (sparingly)

**Don't use for:**
- Anything the user didn't subscribe to
- Information that's already visible on the current screen

---

### Email Notifications

**Use for:**
- Transaction receipts and confirmations
- Weekly/monthly summaries or digests
- Account changes (password changed, email updated)
- Onboarding sequences
- Detailed information that the user may need to reference later

**Don't use for:**
- Real-time updates (too slow)
- Actions requiring immediate attention
- Duplicating every in-app notification

## Notification Center Design

### Bell Icon + Badge
```
[Bell icon with red dot or number badge]
```

**Rules:**
- Badge shows unread count (number) or just a dot (presence indicator)
- Number badge: cap at 99+ (never show "1,247")
- Dot badge: simpler, use when exact count doesn't matter
- Bell icon in the top navigation bar (consistent location)
- Tapping opens the notification panel

### Notification Panel

```
+------------------------------------------+
| Notifications                 [Mark All] |
|------------------------------------------|
| [Avatar] John commented on your post     |
| "Great analysis! I think we..."          |
| 5 minutes ago                     [dot]  |
|------------------------------------------|
| [Icon] Your order #1234 has shipped      |
| Expected delivery: Tomorrow              |
| 2 hours ago                              |
|------------------------------------------|
| [Icon] Payment of $49.99 processed       |
| Visa ending 4242                         |
| Yesterday                                |
|------------------------------------------|
|            [See All Notifications]       |
+------------------------------------------+
```

**Rules:**
- Most recent first
- Unread items visually distinct (bold text, dot indicator, background color)
- Each notification is tappable (deep links to relevant content)
- "Mark all as read" action at the top
- Group related notifications (see Notification Grouping below)
- Show relative time (5 min ago, 2 hours ago, Yesterday)
- Maximum 10-15 notifications in the panel, "See All" for the full list
- Empty state: "No new notifications. You're all caught up."

### Full Notification Page

Accessible from "See All" in the panel.

**Features:**
- Filter by type (messages, orders, system)
- Mark individual or all as read
- Delete/dismiss notifications
- Pagination or infinite scroll for history
- Date grouping (Today, Yesterday, This Week, Earlier)

## Notification Grouping

### Why Group
5 separate notifications about the same topic are noise. Grouped, they're information.

### Grouping Patterns

**By sender:**
```
John Smith (5 new messages)
  "Hey, are you free for lunch?"
  "Also, the report is ready"
  ... and 3 more
```

**By topic/thread:**
```
Order #1234 (3 updates)
  Shipped - Package is on its way
  Out for delivery
  Delivered
```

**By time window:**
```
Morning Digest (7 updates)
  3 new messages
  2 task assignments
  1 comment on your post
  1 system update
```

### Grouping Rules
- Group notifications from the same source within 15 minutes
- Show the latest message text, with count of grouped items
- Expanding a group shows all individual notifications
- Tapping a group opens the relevant context (conversation, order detail)

## Do Not Disturb

### Respect User Preferences

**Settings the user should control:**
- Push notifications on/off per category (messages: on, marketing: off)
- Quiet hours (e.g., 10 PM - 7 AM)
- Sound on/off
- Vibration on/off
- Badge count on/off

**Rules:**
- Never bypass DND for marketing/promotional notifications
- Security alerts (suspicious login, password reset) can bypass DND
- Let users set per-channel preferences (push, email, in-app independently)
- Default to minimal notifications — let users opt in to more, not opt out of too many
- Ask permission to send push notifications at a meaningful moment (not on first launch): "Want to know when your order ships?"

### Permission Request Timing

**Bad timing:**
- Immediately on first app launch (user hasn't experienced value yet)
- During onboarding flow (user is focused on setup)
- Right after signup (overwhelming)

**Good timing:**
- After user places first order: "Want to get shipping updates?"
- After user receives first message: "Turn on notifications so you don't miss replies"
- After user sets a reminder: "Enable notifications to receive your reminders"

## Notification Content Guidelines

### Writing Rules

**Be specific:**
- BAD: "You have a new notification"
- GOOD: "Jane commented on 'Q2 Report'"

**Be actionable:**
- BAD: "Your order status has changed"
- GOOD: "Your order shipped! Tap to track delivery."

**Be human:**
- BAD: "Transaction ID #892431 status: COMPLETED"
- GOOD: "Payment of $49.99 received. Thank you!"

**Be brief:**
- Title: what happened (max 50 chars)
- Body: context + next step (max 100 chars)
- No jargon, no codes, no technical details

### Tone by Category

| Category | Tone | Example |
|----------|------|---------|
| Security | Urgent, direct | "New login from Chrome on Windows. Was this you?" |
| Transaction | Confirming, clear | "Payment of $49.99 processed. View receipt." |
| Social | Warm, personal | "John liked your photo" |
| System | Neutral, informative | "Scheduled maintenance tonight 10 PM - 12 AM" |
| Marketing | Friendly, optional | "New features available. See what's new." |

## Notification Edge Cases

### Offline Behavior
- Queue notifications while offline
- Deliver when the user comes back online
- Don't deliver stale time-sensitive notifications (e.g., "Your table is ready" from 2 hours ago)

### Multiple Devices
- Push notification received on one device should be cleared on all devices
- Read state syncs across devices
- Allow user to choose which devices receive push notifications

### Uninstall / Logout
- Clear notification tokens on logout
- Stop sending push notifications to uninstalled devices
- Keep email notifications as fallback (if user opted in)
