---
knowledge-base-summary: "Same feature, different UX per platform. Mobile (single column, bottom nav, swipe), tablet (master-detail, split view), web (sidebar, hover, keyboard shortcuts). Responsive vs adaptive distinction. Touch vs mouse interaction differences."
---
# Mobile vs Tablet vs Web

## Core Principle

Same feature, different experience. Each platform has unique constraints and strengths. Design for each platform's native interaction patterns — don't just scale the mobile layout up or shrink the desktop layout down.

## Platform Characteristics

### Mobile (< 600px width)

**Constraints:**
- Small viewport, single column
- Touch input only (finger, not stylus)
- Variable network (cellular)
- Interruptions (calls, notifications)
- One-handed use common

**Strengths:**
- Always with the user
- Camera, GPS, sensors
- Push notifications
- Gesture-rich interaction
- Quick, focused tasks

**UX Patterns:**
- Single column layout, vertical scrolling
- Bottom navigation (3-5 tabs)
- Full-screen modals for focused tasks
- Pull-to-refresh for data updates
- Swipe actions (archive, delete, pin)
- Bottom sheets for contextual options
- Floating action button (FAB) for primary creation action
- Large touch targets (minimum 44x44pt)
- Thumb-zone friendly (primary actions within easy reach)

---

### Tablet (600px - 1024px width)

**Constraints:**
- Medium viewport, can do two columns
- Touch input (two-handed or stylus)
- Usually WiFi connection
- Landscape and portrait orientations

**Strengths:**
- More screen real estate than mobile
- Can show context + detail simultaneously
- Better for content consumption and creation
- Multitasking (split screen on some OS)

**UX Patterns:**
- Master-detail layout (list on left, detail on right)
- Split view: navigation + content side by side
- Sidebar navigation (collapsible)
- Tablet-optimized grid (3 columns for cards)
- Popovers instead of full-screen modals
- Drag and drop for reordering
- Multi-column forms (related short fields side by side)
- Both portrait and landscape must work

---

### Web / Desktop (> 1024px width)

**Constraints:**
- Wide viewport (but users don't maximize browsers)
- Mouse + keyboard input
- Users multitask, switch tabs
- No push notifications (unless PWA)

**Strengths:**
- Largest viewport
- Hover states for progressive disclosure
- Keyboard shortcuts for power users
- Right-click context menus
- Multiple windows/tabs
- Copy/paste, drag/drop between apps
- URL-based navigation (bookmarkable, shareable)

**UX Patterns:**
- Sidebar navigation (permanent, collapsible)
- Multi-column layouts
- Data tables with sorting, filtering, pagination
- Hover tooltips for additional info
- Keyboard shortcuts (with discoverable cheat sheet)
- Right-click context menus
- Breadcrumbs for deep navigation
- Inline editing (click to edit, no separate page)
- Multi-select with Shift+click, Ctrl+click
- Content max-width (don't stretch to full 4K screen — 1200-1440px max)

## Same Feature, Different UX

### Example: Contact List

**Mobile:**
```
+---------------------------+
| Search contacts...        |
+---------------------------+
| [Avatar] John Smith       |
|          Marketing Lead   |
|          john@example.com |
+---------------------------+
| [Avatar] Jane Doe         |
|          Engineering      |
|          jane@example.com |
+---------------------------+
| ... (scroll)              |
| [+ FAB]                   |
+---------------------------+
```
Tap a contact -> full-screen detail page. Swipe left to call/email.

**Tablet:**
```
+------------+----------------------------+
| Contacts   | John Smith                 |
|            | Marketing Lead             |
| [Search]   |                            |
|            | Email: john@example.com    |
| John Smith | Phone: 555-1234            |
| Jane Doe   | Company: Acme Corp         |
| Bob Wilson  |                            |
| ...        | [Call] [Email] [Edit]      |
+------------+----------------------------+
```
Tap a contact -> detail appears on the right. No page navigation needed.

**Web:**
```
+------+------------------------------------------------------------------+
| Nav  | Contacts                               [Search] [+ Add Contact] |
|      |------------------------------------------------------------------|
| Home | Name          | Title           | Email           | Phone       |
| ... | John Smith    | Marketing Lead  | john@example... | 555-1234    |
| Cont | Jane Doe      | Engineering     | jane@example... | 555-5678    |
| ... | Bob Wilson    | Sales Manager   | bob@example...  | 555-9012    |
|      |------------------------------------------------------------------|
|      | Showing 1-25 of 142 contacts                   [< 1 2 3 4 5 >]  |
+------+------------------------------------------------------------------+
```
Click row -> inline expand or side panel. Hover row -> action icons appear. Right-click -> context menu.

### Example: Settings

**Mobile:**
- Single list of setting categories
- Tap category -> separate page with options
- Back button to return

**Tablet:**
- Left sidebar with categories
- Right area shows selected category's options
- No page navigation

**Web:**
- Sidebar or top tabs for categories
- Settings form inline
- Save button per section or auto-save

## Responsive vs Adaptive

### Responsive Design
Same components, fluid layout that adapts to screen width.

**How it works:**
- CSS breakpoints adjust layout (columns, spacing, sizing)
- Same HTML/component tree
- Content reflows naturally
- Images scale proportionally

**Best for:** Content-driven sites, blogs, marketing pages, simple apps.

### Adaptive Design
Different components or layouts per platform.

**How it works:**
- Detect platform/screen size
- Render different components (table on web, cards on mobile)
- Different navigation (sidebar on web, bottom nav on mobile)
- Different interaction patterns (hover on web, swipe on mobile)

**Best for:** Complex applications where mobile and desktop workflows differ significantly.

### Decision Guide

| Scenario | Approach |
|----------|----------|
| Marketing site | Responsive |
| Blog / content | Responsive |
| Simple CRUD app | Responsive with minor adaptive tweaks |
| Complex dashboard | Adaptive (different layouts per platform) |
| Data-heavy admin panel | Adaptive (table on web, cards on mobile) |
| Social/media app | Adaptive (different navigation, interactions) |

## Touch vs Mouse Differences

### Touch (Mobile / Tablet)
- **Tap** = Click (but less precise)
- **Long press** = Right-click (context menu)
- **Swipe** = No mouse equivalent (use for quick actions)
- **Pinch** = Zoom (maps, images)
- **Pull down** = Refresh
- **Drag** = Reorder, move
- Touch target minimum: **44x44pt** (Apple), **48x48dp** (Google)
- Spacing between targets: minimum 8pt to prevent mis-taps
- No hover state — everything must be accessible without hover

### Mouse (Web / Desktop)
- **Hover** = Preview, tooltip, reveal actions (not available on touch)
- **Click** = Primary action
- **Right-click** = Context menu
- **Double-click** = Edit, open
- **Scroll wheel** = Scroll, zoom (with Ctrl)
- **Drag** = Select text, move items, resize
- Cursor changes: pointer (clickable), text (editable), grab (draggable), not-allowed (disabled)

### Designing for Both
- Never rely on hover for essential information — it doesn't exist on touch
- Provide alternatives: hover tooltip on web -> tap-to-reveal or always-visible on mobile
- Hover to reveal actions on web -> swipe to reveal or long-press on mobile
- Right-click context menu on web -> long-press menu on mobile
- Keyboard shortcuts on web -> gesture shortcuts on mobile

## Orientation Handling

### Mobile
- Primary: portrait
- Support landscape for: video, photos, games, charts
- Lock to portrait for: forms, lists, reading
- Transition should be smooth (no layout jump)

### Tablet
- Both orientations equally important
- Portrait: more vertical space, good for reading and scrolling
- Landscape: master-detail layout, side-by-side content
- Test both orientations for every screen

### Web
- Fixed to landscape (horizontal)
- Handle narrow windows gracefully (responsive)
- Minimum supported width: 320px (mobile viewport in web browser)

## Platform-Specific Conventions

| Convention | iOS | Android | Web |
|------------|-----|---------|-----|
| Back navigation | Left edge swipe, top-left back | System back button, top-left arrow | Browser back, breadcrumb |
| Primary action | Top-right button | FAB (floating action button) | Top-right button or inline |
| Menu | Bottom sheet, action sheet | Bottom sheet, popup menu | Dropdown menu |
| Delete/destructive | Swipe left + red button | Swipe + action, or long-press menu | Button with confirmation dialog |
| Pull to refresh | Native pull indicator | Material pull indicator | Not standard (use refresh button) |
| Search | Pull down on list to reveal search bar | Persistent search bar or expand icon | Persistent search bar |
| Navigation | Tab bar at bottom | Bottom navigation bar | Sidebar or top navigation |
| Alert | Native iOS alert dialog | Material dialog | Browser dialog or custom modal |
