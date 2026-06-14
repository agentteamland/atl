---
knowledge-base-summary: "Form design principles covering input type selection, label placement, validation timing (on blur not keypress), error message specificity, required vs optional indication, multi-step forms with progress indicators, and auto-save vs explicit save decisions."
---
# Form UX

## Core Principle

Forms are conversations between the user and the system. Every question should be clear, every answer should be easy, and every mistake should be gentle to recover from.

## Input Type Selection

Choose the input type that requires the least effort from the user:

| Data Type | Best Input | Why |
|-----------|------------|-----|
| Free text (name, comment) | Text field | Open-ended, no constraints |
| Email | Email field | Shows @ keyboard on mobile |
| Phone number | Tel field | Shows numeric keyboard |
| Password | Password field | Masked input, show/hide toggle |
| Number (quantity, amount) | Number field | Numeric keyboard, stepper arrows |
| Date | Date picker | Prevents format errors |
| Time | Time picker | Prevents format errors |
| Yes/No (single toggle) | Toggle switch | Immediate, no submit needed |
| One from 2-4 options | Radio buttons | All options visible at once |
| One from 5+ options | Dropdown select | Saves space, searchable if many |
| Multiple from few options | Checkboxes | All options visible, multi-select |
| Multiple from many options | Multi-select dropdown or chip input | Saves space, searchable |
| Long text (description) | Textarea | Multi-line, resizable |
| File | File upload area | Drag-and-drop + click to browse |
| Color | Color picker | Visual selection |
| Range (price min-max) | Range slider | Visual, intuitive |
| Location | Map picker + address input | Visual + text fallback |

### Decision Shortcuts

- If there are 2 options: radio buttons or segmented control
- If there are 3-6 options: radio buttons (single) or checkboxes (multi)
- If there are 7+ options: dropdown with search
- If the answer is always yes/no: toggle switch (if instant effect) or checkbox (if requires submit)
- If the input has a specific format: use a specialized input (date picker, not text field for dates)

## Label Placement

### Top-Aligned Labels (Preferred for Mobile)
```
Email
[________________________]

Password
[________________________]
```
- Fastest completion time (eye moves straight down)
- Works on all screen widths
- Best for mobile
- Use this as the default

### Floating Labels (Material Design Pattern)
```
[________Email___________]  → label floats above on focus
```
- Saves vertical space
- Good for compact forms
- Label must remain visible when field has value

### Inline Labels (Placeholder as Label) -- AVOID
```
[Enter your email________]  → disappears on focus
```
- BAD: label disappears when user starts typing
- User forgets what the field is for
- Exception: search field (magnifying glass icon is sufficient context)

## Validation

### When to Validate

| Timing | Use When | Example |
|--------|----------|---------|
| On blur (leave field) | Format validation | Email format, phone format |
| On submit | Required fields, cross-field validation | Password match, all required filled |
| Real-time (debounced) | Availability check | Username availability (300ms debounce) |
| On input (each keystroke) | Character limit only | "42/280 characters" counter |

**Never validate on every keystroke** for format validation. The user hasn't finished typing yet. Showing "Invalid email" when they've typed "john@" is hostile UX.

### Error Message Rules

**Specific, not generic:**
- BAD: "Invalid input"
- GOOD: "Email must include @ and a domain (e.g., name@example.com)"

**Constructive, not accusatory:**
- BAD: "You entered an invalid phone number"
- GOOD: "Phone number should be 10 digits (e.g., 555-123-4567)"

**Location-aware:**
- Field-level errors appear below the field, in red/error color
- Form-level errors appear at the top of the form (for cross-field issues)
- Focus on the first field with an error after submit

**Error message anatomy:**
1. What is wrong: "Password is too short"
2. What is expected: "Must be at least 8 characters"
3. (Optional) How to fix: "Add more characters or use a passphrase"

### Inline Validation States

```
Default:      [ Field label          ]     (neutral border)
Focused:      [ Field label          ]     (brand color border)
Valid:        [ Field label        checkmark ]  (green border + check icon)
Error:        [ Field label          ]     (red border)
              Error message here            (red text below)
Disabled:     [ Field label          ]     (grey, no interaction)
```

## Required vs Optional

**When most fields are required:**
- Mark optional fields with "(optional)" text after the label
- Don't mark required fields — it's assumed

**When most fields are optional:**
- Mark required fields with asterisk `*` and a legend at the top: "* Required"
- Less common scenario — rethink if you really need all those optional fields

**Rule of thumb:** If a field is truly optional, ask yourself if you even need it. Every field increases form abandonment.

## Multi-Step Forms

### When to Break Up a Form
- More than 7 fields on mobile (5 on a single viewport)
- Logically distinct sections (personal info, address, payment)
- Progressive context (step 2 depends on step 1 answer)
- Complex flow where the user needs to focus on one thing at a time

### Progress Indicator
```
Step 1          Step 2          Step 3          Step 4
[Personal] ---> [Address] ---> [Payment] ---> [Review]
  (done)         (current)      (upcoming)     (upcoming)
```

- Show step names, not just numbers
- Completed steps are clickable (go back to edit)
- Current step is highlighted
- Upcoming steps are visible but not clickable

### Multi-Step Rules
- Validate each step before allowing "Next"
- Save progress (user can leave and come back)
- Show "Back" and "Next" buttons (not just "Next")
- Last step is always "Review" — show a summary before final submit
- Final action button says what it does: "Place Order", not "Submit"

## Auto-Save vs Explicit Save

| Pattern | When to Use | Example |
|---------|-------------|---------|
| Auto-save | Editing existing content, notes, documents | Google Docs-style: save on pause |
| Explicit save | Creating new records, critical data | User profile: "Save Changes" button |
| Draft auto-save | Long forms that might be abandoned | Job application: draft saved automatically |

### Auto-Save UX
- Show save status: "Saving..." -> "All changes saved" -> "Last saved 2 min ago"
- Save on pause (debounce 1-2 seconds after last edit)
- No save button needed
- Conflict detection: "This was edited by someone else. Review changes?"

### Explicit Save UX
- "Save" button is disabled until changes are made
- Warn before navigating away with unsaved changes: "You have unsaved changes. Discard?"
- After save: brief success feedback (toast or inline "Saved!")
- "Cancel" reverts all changes

## Form Layout Guidelines

### Single Column (Preferred for Mobile)
Fields stack vertically. One field per row. Maximum readability and completion speed.

### Two Column (Tablet and Web Only)
Only for closely related short fields:
- First name | Last name
- City | State
- Start date | End date

Never put unrelated fields side by side.

### Field Sizing
- Match field width to expected content length
- ZIP code field should be short (5-6 characters)
- Email field should be full width
- Phone field should be medium width
- This gives visual cues about expected input length

### Button Placement
- Primary action on the right: [Cancel] [Save]
- Mobile: full-width primary button at the bottom
- Primary button is visually dominant (filled, brand color)
- Secondary button is subdued (outlined or text-only)
- Destructive actions are separated (different section or additional confirmation)

## Password Field UX

- Show/hide toggle (eye icon)
- Strength indicator (weak/medium/strong with color bar)
- Requirements listed below, checking off as user types:
  - [x] At least 8 characters
  - [ ] One uppercase letter
  - [x] One number
  - [ ] One special character
- Confirm password: validate match on blur of second field
- "Forgot password?" link near the password field on login forms

## Search Field UX

- Magnifying glass icon (universally understood)
- Placeholder text: "Search orders...", "Search by name or email..."
- Clear button (X) appears when field has content
- Recent searches shown on focus (if applicable)
- Search suggestions appear as user types (debounce 300ms)
- Keyboard: search/go button instead of Enter/Return on mobile
