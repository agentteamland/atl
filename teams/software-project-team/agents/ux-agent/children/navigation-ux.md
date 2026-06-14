---
knowledge-base-summary: "When to use which navigation pattern: bottom nav, sidebar, tab bar, drawer, modal, breadcrumb. Decision matrix across mobile, tablet, and web. Maximum item counts, platform conventions, and common anti-patterns to avoid."
---
# Navigation UX

## Core Principle

Navigation is the skeleton of the app. Users must always know: where am I, how did I get here, and where can I go next. If the user feels lost, navigation has failed.

## Navigation Patterns

### Bottom Navigation Bar
**When:** Mobile app with 3-5 top-level sections that users switch between frequently.

**Rules:**
- Minimum 3 items, maximum 5
- Each item has icon + label (never icon-only for primary navigation)
- Active item is visually distinct (color, fill, size)
- Badge for unread/pending counts (number or dot)
- Each tab preserves its own navigation stack
- Tapping active tab scrolls to top / resets to root

**Good for:** Home, Search, Orders, Profile (consumer app). Dashboard, Projects, Messages, Settings (productivity app).

**Anti-patterns:**
- More than 5 items (use drawer or "More" tab)
- Hiding the bottom bar on scroll (disorienting)
- Actions in bottom bar (like "Add" button — use FAB instead)

---

### Sidebar Navigation
**When:** Web or desktop app with 6+ sections. Admin panels, dashboards, content management.

**Rules:**
- Collapsible (icon-only mode) for more content space
- Group related items under section headers
- Active item highlighted with background color and/or indicator bar
- Support nested items (expandable) for complex hierarchies
- Sticky — always visible, doesn't scroll with content
- User avatar and key actions (logout, settings) at bottom

**Good for:** Admin dashboards, CMS, analytics platforms, project management tools.

**Anti-patterns:**
- Sidebar on mobile (use bottom nav or drawer instead)
- Too many nesting levels (max 2 levels deep)
- Mixing navigation and actions in sidebar

---

### Tab Bar
**When:** Related views within a single screen. Switching between aspects of the same content.

**Rules:**
- 2-5 tabs maximum
- Swipeable on mobile (horizontal swipe between tab content)
- Scrollable tab bar if labels are long or tabs exceed screen width
- Tab labels are nouns (not verbs) — "Activity", "Members", "Settings"
- Badge on individual tabs for new content
- Tab content should be at the same hierarchy level

**Good for:** Product detail (Description, Reviews, Specs). User profile (Posts, Followers, Following). Settings sections.

**Anti-patterns:**
- Using tabs for sequential steps (use stepper instead)
- Tabs with very different content types (mixing list, form, and dashboard)
- More than 5 tabs without scrolling

---

### Drawer (Hamburger Menu)
**When:** Secondary navigation that doesn't need constant visibility. Infrequently accessed sections.

**Rules:**
- Opens from left edge (LTR) or right edge (RTL)
- Overlay type: covers content with scrim (dimmed background)
- Contains: user info, navigation items, settings, about, logout
- Close on item selection (navigate and dismiss)
- Close on scrim tap or swipe
- Never the primary navigation on mobile — users forget what's behind the hamburger

**Good for:** Settings, help, about, account management, secondary features.

**Anti-patterns:**
- Hiding primary navigation in drawer (kills discoverability)
- Deep nesting inside drawer
- Using drawer when bottom nav would work (fewer than 6 items)

---

### Modal / Dialog
**When:** Focused task that interrupts the main flow. Requires user attention and decision.

**Rules:**
- Has a clear title explaining what the modal is for
- Primary action button on the right (or bottom on mobile)
- Dismiss with X button, scrim tap (if non-destructive), or Escape key
- Non-scrollable content preferred (if content is long, use a full page instead)
- No navigation inside modals (no going deeper from a modal)
- Confirmation modal for destructive actions: "Delete this item?" with "Cancel" and "Delete" buttons

**Good for:** Confirmations, quick forms (add note, rename), alerts, previews.

**Anti-patterns:**
- Modal inside modal (inception problem)
- Complex forms in modals (use a full page)
- Modals that users need to compare with underlying content (use side panel)

---

### Bottom Sheet
**When:** Mobile-specific pattern for contextual actions or supplementary content.

**Rules:**
- Peek height: shows just enough to understand what's inside
- Expandable: drag up to reveal full content
- Dismissible: drag down or tap scrim
- Used for: action lists, filters, detail previews, sharing options
- Persistent bottom sheet: stays visible, content scrolls behind it (maps, media player)

**Good for:** Share options, filter panels, quick actions, mini player, map place details.

**Anti-patterns:**
- Using on web/desktop (use dropdown or side panel instead)
- Critical actions in bottom sheet (easy to dismiss accidentally)
- Very long content in bottom sheet (use full screen)

---

### Breadcrumb
**When:** Deep hierarchies where users need to understand their location and jump back to any level.

**Rules:**
- Shows full path: Home > Category > Subcategory > Current Page
- Each segment is clickable except the current page
- Truncate middle segments on mobile if too long (...between first and last)
- Always starts with Home/Root
- Separator character: `>` or `/` or chevron icon

**Good for:** E-commerce categories, file managers, documentation sites, CMS content trees.

**Anti-patterns:**
- Using breadcrumbs for flat navigation (no hierarchy to show)
- Breadcrumbs as the only back navigation (always provide a back button too)
- More than 5 levels (rethink the information architecture)

## Decision Matrix

| Scenario | Mobile | Tablet | Web |
|----------|--------|--------|-----|
| 3-5 top-level sections | Bottom nav | Bottom nav or sidebar | Sidebar |
| 6+ top-level sections | Bottom nav + drawer for extras | Sidebar | Sidebar |
| Related views of same content | Tab bar | Tab bar | Tab bar |
| Secondary/settings navigation | Drawer | Drawer or sidebar | Sidebar section |
| Contextual actions | Bottom sheet | Bottom sheet or popover | Dropdown menu |
| Focused task / confirmation | Full-screen modal | Center modal | Center modal |
| Deep hierarchy (categories) | Back button chain | Breadcrumb + sidebar | Breadcrumb + sidebar |
| Search-heavy app | Tab in bottom nav + full-screen search | Persistent search bar | Persistent search bar |

## Navigation State Management

### Back Stack
- Each major section (tab) maintains its own back stack
- Back button pops within the current section first
- At root of section, back exits the app (or shows "press again to exit")

### Deep Linking
- Every meaningful screen should have a URL/route
- Opening a deep link navigates directly, with a synthetic back stack
- Example: deep link to order detail creates stack: Home > Orders > Order #123

### State Preservation
- Switching between tabs preserves scroll position and state
- Returning from a modal preserves the underlying screen state
- Form data is preserved on back navigation (warn before discarding)

## Common Navigation Flows

### Master-Detail
List screen > Detail screen. Tap an item to see its detail. On tablet, show both side by side (split view). On mobile, push to new screen.

### Wizard / Stepper
Step 1 > Step 2 > Step 3 > Confirmation. Linear flow with progress indicator. Back goes to previous step. Each step validates before allowing next.

### Search-Filter-Result
Search bar > Filter panel > Result list > Detail. Search triggers results, filters narrow them, tapping a result opens detail.

## Anti-Patterns to Avoid

1. **Mystery meat navigation** — icons without labels, unclear what they do
2. **Too many navigation levels** — if the user needs more than 3 taps to reach content, flatten the hierarchy
3. **Inconsistent back behavior** — back should always be predictable
4. **Hidden navigation** — if users can't see it, they won't use it
5. **Navigation that changes context unexpectedly** — switching tabs shouldn't open a new screen, opening a link shouldn't leave the app without warning
6. **Dead ends** — every screen must have a way out
