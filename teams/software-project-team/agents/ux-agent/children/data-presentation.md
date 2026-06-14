---
knowledge-base-summary: "How to present data effectively. Table vs list vs card grid decision criteria. Data density guidelines (admin = high, consumer = low). Sorting, filtering, and search UX. Pagination vs infinite scroll. Empty state design. Data visualization: when chart vs number vs table."
---
# Data Presentation

## Core Principle

Data is only valuable when the user can understand it and act on it. The right presentation format makes data scannable, comparable, and actionable.

## Table vs List vs Card Grid

### Table
**When:** Comparing multiple items across many attributes. Users need to scan columns and sort/filter.

**Best for:**
- Admin panels, dashboards, reports
- Comparing items (products, users, orders) with 4+ attributes
- Data that benefits from sorting and filtering
- Bulk actions (select multiple, batch operations)

**Rules:**
- Column headers are always visible (sticky header on scroll)
- Right-align numbers, left-align text
- Zebra striping or row hover for readability
- Sortable columns indicated with sort icon
- Truncate long text with tooltip on hover
- Action column on the right (edit, delete, view)
- Row click opens detail (entire row is clickable, not just a link)

**Mobile adaptation:** Tables don't work on mobile. Convert to cards or lists.

---

### List
**When:** Sequential items where one or two attributes are primary. Users scan top-to-bottom.

**Best for:**
- Messages, notifications, activity feeds
- Search results
- Simple item lists with primary text + secondary text
- Mobile interfaces

**Rules:**
- Each list item has: primary text, secondary text, optional avatar/icon, optional trailing action
- Dividers between items (subtle line or spacing)
- Pull-to-refresh on mobile
- Infinite scroll or "Load more" button
- Swipe actions on mobile (archive, delete, pin)

---

### Card Grid
**When:** Items are visually distinct or have images. Users browse and compare visually.

**Best for:**
- Products with images
- Portfolio/gallery items
- Dashboard widgets
- Team member profiles
- Project/workspace tiles

**Rules:**
- Consistent card size within a grid
- Image/visual takes prominence
- Title + 1-2 key attributes visible
- Card is clickable (entire card, not just a button)
- Responsive: 4 columns on desktop, 3 on tablet, 2 on mobile, 1 on small mobile
- Hover effect on web (subtle elevation or border)

## Decision Matrix

| Factor | Table | List | Card Grid |
|--------|-------|------|-----------|
| Many attributes per item | Best | Poor | Medium |
| Image-heavy content | Poor | Medium | Best |
| Mobile-friendly | Poor | Best | Good |
| Comparison across items | Best | Poor | Medium |
| Quick scanning | Good | Best | Medium |
| Bulk operations | Best | Medium | Poor |
| Dense data (admin) | Best | Good | Poor |
| Consumer browsing | Poor | Good | Best |

## Data Density

### High Density (Admin/Professional)
- More items per screen, smaller spacing
- Compact row height in tables
- Less whitespace between elements
- Information-dense cards
- Users are trained, they know what they're looking at
- Example: CRM contact list, inventory management, analytics dashboard

### Low Density (Consumer/Public)
- Fewer items per screen, generous spacing
- Larger touch targets
- More whitespace, visual breathing room
- Cards with large images and minimal text
- Users are casual, they need guidance
- Example: E-commerce product grid, social media feed, news app

### Medium Density (Productivity)
- Balance between information and readability
- Moderate spacing
- Key info visible, details on click/hover
- Example: Email inbox, task management, project boards

## Sorting UX

### Sort Controls
- Default sort should be the most useful (newest first, most relevant, alphabetical — depends on context)
- Sort indicator: arrow icon in column header (table) or sort dropdown (list/grid)
- Sort direction: toggle between ascending and descending on click
- Active sort column is visually highlighted
- Remember the user's sort preference (persist across sessions)

### Sort Options by Data Type
- Dates: newest first (default), oldest first
- Names: A-Z (default), Z-A
- Numbers: highest first or lowest first (depends on context — price: low to high for buyers)
- Relevance: only for search results (default for search)
- Status: custom order (active > pending > completed > cancelled)

## Filtering UX

### Filter Patterns

**Filter bar (inline):** Quick filters as chips or buttons above the content.
```
[All] [Active] [Pending] [Completed]    Status: [Dropdown]    Date: [Range Picker]
```
Best for: 2-5 filter options that users frequently switch between.

**Filter panel (sidebar):** Multiple filter categories in a dedicated panel.
Best for: Complex filtering with many categories (e-commerce: brand, size, color, price, rating).
- Collapsible sections per category
- Show active filter count
- "Clear all" button
- Apply button (for server-side filtering) or instant update (for client-side)

**Filter chips (applied filters):** Show active filters as removable chips.
```
Filters: [Status: Active x] [Date: Last 7 days x] [Clear all]
```
Always visible so users know what's applied.

### Filter Rules
- Show result count after filtering: "Showing 24 of 156 results"
- "No results" state with suggestion to broaden filters
- Filter state reflected in URL (shareable, bookmarkable)
- Reset/Clear all option always available
- Don't hide content behind too many required filters

## Search UX

### Instant Search vs Submit Search

**Instant search (search-as-you-type):**
- Results update as user types (debounce 300ms)
- Best for: small datasets, in-page filtering, autocomplete
- Show loading indicator while fetching results
- Minimum character threshold (usually 2-3 characters)

**Submit search (press Enter / tap Search):**
- Results appear after explicit submit
- Best for: large datasets, full-text search, complex queries
- Show search results on a dedicated results page

### Search Enhancements
- **Recent searches:** Show last 5-10 searches on focus (stored locally)
- **Suggestions:** Auto-complete as user types (from popular searches or indexed data)
- **Scoped search:** Let user narrow scope: "Search in Orders" vs "Search everywhere"
- **Search results:** Highlight matching text in results
- **No results:** "No results for 'xyz'. Try different keywords." + suggestions

## Pagination vs Infinite Scroll

### Pagination
**When:** User needs to reach a specific position, share a link to a specific page, or the dataset is very large.

**Rules:**
- Show: [< Previous] [1] [2] [3] ... [47] [48] [Next >]
- Show current page and total: "Page 3 of 48"
- Show result count: "Showing 21-30 of 478 results"
- Page size selector: 10, 25, 50, 100 (for admin/power users)
- Persist in URL: `?page=3&pageSize=25`

**Best for:** Admin panels, search results, data tables, anything where users need to reference "page X".

### Infinite Scroll
**When:** Browsing feed-like content, users don't need to reach a specific position.

**Rules:**
- Load next batch when user scrolls to 80% of current content (not at the very bottom)
- Show loading indicator at bottom while fetching
- "Back to top" button appears after scrolling down
- Preserve scroll position on back navigation
- Consider a "Load more" button as an alternative (gives user control)

**Best for:** Social feeds, news, activity logs, product browsing (consumer).

### Decision Criteria

| Factor | Pagination | Infinite Scroll |
|--------|-----------|----------------|
| User needs specific page/position | Yes | No |
| Link sharing to specific results | Yes | No |
| Content is browsable/feed-like | No | Yes |
| Footer needs to be reachable | Yes | No (footer is unreachable) |
| SEO matters | Yes | No |
| Admin/professional interface | Yes | No |
| Consumer browsing | Maybe | Yes |

## Empty State Design

Empty states are the first impression for new users. Make them welcoming and actionable.

### Anatomy of a Good Empty State
```
    [Illustration]
    
    No orders yet
    
    When you place your first order,
    it will appear here.
    
    [Browse Products]
```

1. **Illustration or icon** — visual, friendly, on-brand
2. **Title** — what is empty (not an error, just empty)
3. **Description** — why it's empty and what to do about it
4. **Call to action** — button to start (the most important part)

### Empty State Types

| Type | Title Example | CTA Example |
|------|---------------|-------------|
| First use | "No projects yet" | "Create your first project" |
| Search no results | "No results for 'xyz'" | "Try different keywords" |
| Filter no results | "No items match your filters" | "Clear filters" |
| Completed | "All caught up!" | "View completed items" |
| Permission | "You don't have access" | "Request access" |

### Rules
- Never show a blank screen. Every list/collection has an empty state.
- Empty state CTA should be the most important action the user can take.
- Use different messages for "no data ever" vs "no data matching filters."
- Tone is encouraging, not apologetic.

## Data Visualization

### When to Use Each Format

| Data Question | Best Format | Example |
|---------------|-------------|---------|
| What is the current value? | Single number (KPI card) | "Revenue: $45,200" |
| How has it changed over time? | Line chart | Monthly revenue trend |
| How do parts relate to whole? | Pie/donut chart (max 5 segments) | Revenue by category |
| How do categories compare? | Bar chart | Sales by region |
| What is the distribution? | Histogram | Order value distribution |
| What is the correlation? | Scatter plot | Price vs. quantity sold |
| What is the geographic spread? | Map | Users by country |

### Visualization Rules
- Every chart has a title that answers a question ("How has revenue changed?")
- Label axes clearly (include units)
- Use consistent colors across the dashboard
- Limit pie charts to 5 segments (group the rest as "Other")
- Show data labels on bars/segments for exact values
- Interactive: hover/tap for details (tooltip)
- Provide "Export" or "Download" option for data tables
- Consider color-blind safe palettes
- Always include a legend if using multiple series
