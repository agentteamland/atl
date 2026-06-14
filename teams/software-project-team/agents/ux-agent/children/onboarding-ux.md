---
knowledge-base-summary: "First-time user experience design. Welcome screens, feature highlights (coach marks), empty states as onboarding, progressive feature reveal, skip-always-available principle, and setup wizard patterns for complex initial configuration."
---
# Onboarding UX

## Core Principle

The first experience shapes the user's entire perception of the product. Onboarding should make the user feel competent and confident, not overwhelmed or lectured. The best onboarding is an app so intuitive it barely needs onboarding at all.

## Onboarding Strategies

### 1. Welcome Screen

The first thing a new user sees after signup/login.

**Structure:**
```
[App Logo / Illustration]

Welcome to AppName!

Brief one-line value proposition.

[Get Started]
```

**Rules:**
- One screen, no carousel of 5 slides (users skip them)
- Single CTA: "Get Started" or "Let's Go"
- No feature dump — one sentence about what the app does
- Skip option if there's a setup wizard ahead
- Animation or illustration to set the tone
- Shown only once (track first-login state)

---

### 2. Feature Highlights (Coach Marks)

Overlay tooltips pointing to specific UI elements, explaining what they do.

```
+--------------------------------------------------+
|  [Profile]  [Search]  [Notifications]  [Menu]    |
|                         ^                         |
|                    +-----------+                  |
|                    | Tap here  |                  |
|                    | to see    |                  |
|                    | updates   |                  |
|                    +-----------+                  |
|                    [Next]  [Skip]                 |
+--------------------------------------------------+
```

**Rules:**
- Maximum 3-4 highlights per session (not 10)
- Each highlight points to ONE element with ONE sentence
- "Next" and "Skip" always visible
- Dim the rest of the screen (focus on the highlighted element)
- Never block the user from using the app
- Show on first visit to each section, not all at once
- Don't repeat if user dismissed them
- Context-sensitive: show a highlight when the user first encounters a feature, not at the start

**Anti-patterns:**
- 10-slide tutorial before the user sees the app
- Explaining obvious things ("This is the search bar")
- Blocking the user until they complete all steps
- Showing all highlights on the first screen regardless of context

---

### 3. Empty State as Onboarding

The most natural form of onboarding. Empty screens teach by inviting action.

**Pattern:**
```
    [Illustration of empty inbox]

    No messages yet

    Send your first message to a friend
    or join a group to start chatting.

    [Find Friends]  [Join a Group]
```

**Rules:**
- Every empty collection is an onboarding opportunity
- Illustration should be warm and encouraging (not a sad face)
- Message explains what will appear here once the user takes action
- CTA directly leads to the action that populates this screen
- Different from "error" — empty state is a beginning, not a problem

**Examples by context:**

| Screen | Empty Message | CTA |
|--------|---------------|-----|
| Dashboard | "Your dashboard is ready. Add your first widget." | [Add Widget] |
| Orders | "No orders yet. Browse our catalog to find what you need." | [Browse Products] |
| Team | "You're the first one here. Invite your team members." | [Invite Members] |
| Notifications | "All caught up! You'll see new notifications here." | (no CTA needed) |
| Search results | "No results for 'xyz'. Try different keywords." | [Clear Search] |

---

### 4. Progressive Onboarding

Reveal features as the user grows, not all at once on day one.

**Strategy:**
```
Day 1:  Core feature only (create, view, basic actions)
Day 3:  Introduce secondary features (filters, sorting, export)
Day 7:  Introduce advanced features (automations, integrations, shortcuts)
Day 14: Introduce power-user features (API access, bulk operations, templates)
```

**Implementation:**
- Track user activity milestones (first item created, 10th login, etc.)
- Show contextual tips when milestones are reached
- "Did you know?" tooltip or banner at relevant moments
- Feature discovery through subtle visual cues (new badge, pulse animation)
- Never interrupt the user's current task for a discovery prompt

**Triggers:**
| Milestone | Feature to Introduce | Method |
|-----------|---------------------|--------|
| First login | Core workflow | Empty state CTA |
| 5th item created | Bulk operations | Tooltip on list screen |
| 3rd day active | Keyboard shortcuts | "Pro tip" toast |
| 10th login | Templates | Banner suggestion |
| First export | Scheduled exports | Follow-up tooltip |

---

### 5. Setup Wizard

For apps that need initial configuration before they're useful.

**When to use:**
- App needs account-specific data to function (team name, timezone, preferences)
- Integration setup is required (connect services, import data)
- Role-based features need configuration (admin vs member)

**Structure:**
```
Step 1: Personal Info      "Tell us about yourself"
Step 2: Preferences        "How do you want to use AppName?"
Step 3: Integrations       "Connect your tools" (optional)
Step 4: Invite Team        "Bring your team" (optional)
Step 5: Ready              "You're all set!"
```

**Rules:**
- Keep required steps minimal (2-3 max)
- Mark optional steps clearly: "Skip for now"
- Progress indicator shows how close to done
- Allow completion later: "You can always change this in Settings"
- Don't ask for information you don't need yet
- Pre-fill where possible (detect timezone, suggest defaults)
- Each step should take less than 30 seconds

**Anti-patterns:**
- 10-step wizard before the user sees the app
- Requiring information that could be collected later
- No skip option on optional steps
- Repeating the wizard on next login if partially completed

## Skip Option

**Mandatory rule:** Every onboarding element must have a skip option.

- Welcome screen: "Skip" link or just the "Get Started" button
- Coach marks: "Skip All" on every step
- Setup wizard: "Skip" on optional steps, "I'll do this later" for all
- Feature tours: "Got it" to dismiss

**Why:** Users who skip are not "bad users." They might be:
- Experienced users who know what they're doing
- Returning users who already saw this
- In a hurry to complete a task
- Just exploring before committing

## Onboarding Metrics

Track these to know if onboarding works:

| Metric | What It Tells You |
|--------|-------------------|
| Completion rate | How many users finish onboarding |
| Skip rate per step | Which steps users find unnecessary |
| Time to first value | How long until user does the core action |
| Day-1 retention | Do users come back after onboarding |
| Day-7 retention | Did onboarding create lasting engagement |
| Feature adoption rate | Are introduced features actually used |

## Returning User Experience

Onboarding is not just for new users. Returning users need guidance too.

### After Absence
- "Welcome back!" with summary of what changed
- Highlight new features since last visit
- Restore their previous context (last viewed item, unfinished draft)

### After Major Update
- "What's new" dialog (1-3 bullet points maximum)
- Highlight changed UI with coach marks
- Link to full changelog for curious users

### Account State Changes
- After upgrade: "You now have access to [feature]. Here's how to use it."
- After downgrade: "Some features are no longer available. Here's what changed."
- After role change: "You're now an admin. Here are your new capabilities."
