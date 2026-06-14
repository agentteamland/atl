---
knowledge-base-summary: "input, textarea, label, select, checkbox, radio, switch, slider, combobox, input-otp, file-upload. Capture user input. Label pairing, helper-text consistency, validation strategy boundary."
---
# Forms group

> Anchor: `#components-forms` in `detail.html`.
> Source-of-truth: `ds.json.components.{input, textarea, label, select, checkbox, radio, switch, slider, combobox, input-otp, file-upload}`.

**Definition.** Components that **capture user input**. Anything the user types, picks, drags, or toggles to declare a value lives here.

## Components

### input

Single-line text field. The default form primitive.

```json
"input": {
  "variants": {
    "default": { "background": "surface",          "border": "1px solid border", "text": "text.primary", "note": "Primary form field on cards." },
    "filled":  { "background": "surfaceContainer", "border": "1px solid transparent", "text": "text.primary", "note": "Softer field — used when surface itself is white." }
  },
  "sizes": {
    "sm": { "height": "36px", "fontSize": "bodySmall", "padding": "0 12px", "radius": "lg" },
    "md": { "height": "44px", "fontSize": "body",      "padding": "0 14px", "radius": "lg", "note": "Default — meets 44px tap target." },
    "lg": { "height": "52px", "fontSize": "body",      "padding": "0 16px", "radius": "lg" }
  },
  "states": ["idle", "focus", "filled", "error", "disabled"],
  "stateRules": {
    "focus":    "Border: 1.5px solid brand.700; shadow: 0 0 0 3px brand.100.",
    "error":    "Border: 1.5px solid feedback.error; helper text uses feedback.error color.",
    "disabled": "Background: surfaceContainerHigh; text: text.muted; cursor: not-allowed."
  }
}
```

### textarea

Multi-line text field. Uses input's variant tokens; adds `minRows`/`maxRows` and resize behavior.

```json
"textarea": {
  "variants": {
    "default": { "background": "surface",          "border": "1px solid border",  "text": "text.primary" },
    "filled":  { "background": "surfaceContainer", "border": "1px solid transparent", "text": "text.primary" }
  },
  "sizes": {
    "sm": { "minRows": 2, "maxRows": 4, "fontSize": "bodySmall", "padding": "10px 12px", "radius": "lg" },
    "md": { "minRows": 3, "maxRows": 6, "fontSize": "body",      "padding": "12px 14px", "radius": "lg", "note": "Default — comfortable for paragraph entry." },
    "lg": { "minRows": 5, "maxRows": 10,"fontSize": "body",      "padding": "14px 16px", "radius": "lg" }
  },
  "states": ["idle", "focus", "filled", "error", "disabled"],
  "resize": "vertical",
  "characterCounter": "Optional — show when maxLength enforced; format `{used}/{max}`; warn at 90%."
}
```

### label

Form-field label primitive. Not interactive on its own; pairs with input/select/checkbox/etc.

```json
"label": {
  "variants": {
    "default":  { "fontSize": "label",     "color": "text.secondary", "fontWeight": 500, "note": "Sits above the field, 6px gap." },
    "inline":   { "fontSize": "label",     "color": "text.secondary", "fontWeight": 500, "note": "Sits to the left of small fields (e.g., toggles)." },
    "required": { "suffix": " *",          "suffixColor": "feedback.error", "note": "The asterisk is suffix-only; never replace the entire label." }
  },
  "sizes": {
    "default": { "lineHeight": "20px", "marginBottom": "6px" }
  },
  "states": ["idle", "disabled"],
  "stateRules": {
    "disabled": "color: text.muted; the field below is also disabled."
  },
  "helperPairing": "Helper text sits below the field (4px gap), uses caption type, color text.muted (text.error if state=error)."
}
```

### select

Triggers a menu of options. Not the same as a native HTML `<select>` — visual treatment matches input, menu appears anchored below.

```json
"select": {
  "variants": {
    "default": { "background": "surface",          "border": "1px solid border",  "text": "text.primary", "note": "Triggers menu anchored below." },
    "filled":  { "background": "surfaceContainer", "border": "1px solid transparent", "text": "text.primary" }
  },
  "sizes": {
    "sm": { "height": "36px", "fontSize": "bodySmall", "padding": "0 12px", "radius": "lg" },
    "md": { "height": "44px", "fontSize": "body",      "padding": "0 14px", "radius": "lg", "note": "Default — matches input md." },
    "lg": { "height": "52px", "fontSize": "body",      "padding": "0 16px", "radius": "lg" }
  },
  "states": ["idle", "focus", "open", "selected", "error", "disabled"],
  "menu": {
    "background":  "surface",
    "elevation":   "medium",
    "radius":      "lg",
    "padding":     "4px",
    "itemHeight":  "40px",
    "maxHeight":   "320px",
    "anchorOffset": "4px below trigger"
  },
  "stateRules": {
    "open": "Trigger keeps focus border; menu animates in (decelerate, normal duration); arrow icon rotates 180°."
  }
}
```

### checkbox

Binary or tri-state choice. Used in lists ("select all"), forms ("agree to terms"), filters ("include archived").

```json
"checkbox": {
  "variants": {
    "default":   { "borderColor": "border",        "tickColor": "text.inverse", "fillColor": "brand.primary",   "note": "Standard." },
    "filled":    { "borderColor": "transparent",   "tickColor": "text.inverse", "fillColor": "brand.primary",   "note": "On colored surfaces." }
  },
  "sizes": {
    "sm": { "size": "16px", "tickSize": "10px", "radius": "sm" },
    "md": { "size": "20px", "tickSize": "12px", "radius": "sm", "note": "Default." }
  },
  "states": ["unchecked", "checked", "indeterminate", "focus", "disabled"],
  "stateRules": {
    "checked":      "Background: brand.primary; tick icon visible.",
    "indeterminate":"Background: brand.primary; horizontal bar instead of tick. Used in 'select all' headers when partially checked.",
    "focus":        "2px outline brand.500, 2px offset.",
    "disabled":     "40% opacity; cursor: not-allowed."
  },
  "labelPairing": "Label sits to the right, 8px gap. Click on label toggles checkbox."
}
```

### radio

Single-choice from a small fixed set. Use checkbox instead if more than one can be selected; use select instead if more than ~6 options.

```json
"radio": {
  "variants": {
    "default": { "borderColor": "border", "dotColor": "brand.primary", "note": "Standard." }
  },
  "sizes": {
    "sm": { "size": "16px", "dotSize": "8px" },
    "md": { "size": "20px", "dotSize": "10px", "note": "Default." }
  },
  "states": ["unchecked", "checked", "focus", "disabled"],
  "stateRules": {
    "checked":  "Border keeps default; inner dot appears (brand.primary fill).",
    "focus":    "2px outline brand.500, 2px offset.",
    "disabled": "40% opacity; cursor: not-allowed."
  },
  "groupRules": "Always rendered as a `radio.group` — exactly one of N must be selectable. Single radio with no group is a code smell — use checkbox.",
  "labelPairing": "Label sits to the right, 8px gap. Click on label selects radio."
}
```

### switch

Binary on/off control. Visually distinct from checkbox — switch implies an immediate effect (e.g., toggling notifications on); checkbox implies a value to be saved later (e.g., agreeing to terms in a form).

> **Naming note:** the existing `ds.json.components.toggle` key is an alias for this component (legacy Material naming). New DS instances should use `switch`. The `toggle` key is kept for backward compatibility with primary DS during Faz 2.

```json
"switch": {
  "variants": {
    "default": { "trackOff": "neutral.300", "trackOn": "brand.primary", "thumb": "surface", "note": "iOS-style track + thumb." }
  },
  "sizes": {
    "sm": { "trackWidth": "32px", "trackHeight": "20px", "thumbSize": "16px" },
    "md": { "trackWidth": "40px", "trackHeight": "24px", "thumbSize": "18px", "note": "Default." }
  },
  "states": ["off", "on", "focus", "disabled"],
  "stateRules": {
    "on":       "Track: brand.primary; thumb slides to the right.",
    "focus":    "2px outline brand.500, 2px offset around track.",
    "disabled": "40% opacity; cursor: not-allowed."
  },
  "labelPairing": "Label sits to the LEFT of the switch (opposite of checkbox/radio); 12px gap. Trailing position implies the value is the result of the action."
}
```

### slider

Continuous numeric input. Used for values where exact precision is less important than range awareness — volume, brightness, distance.

```json
"slider": {
  "variants": {
    "default": { "trackBg": "neutral.300", "trackFill": "brand.primary", "thumb": "brand.primary", "note": "Single-thumb." },
    "range":   { "trackBg": "neutral.300", "trackFill": "brand.primary", "thumb": "brand.primary", "note": "Two thumbs — min and max." }
  },
  "sizes": {
    "sm": { "trackHeight": "4px", "thumbSize": "16px" },
    "md": { "trackHeight": "6px", "thumbSize": "20px", "note": "Default." }
  },
  "states": ["idle", "hover", "focus", "dragging", "disabled"],
  "stateRules": {
    "hover":    "Thumb scale: 1.1; subtle shadow (elevation.low).",
    "focus":    "2px outline brand.500, 2px offset around thumb.",
    "dragging": "Thumb scale: 1.15; tooltip with current value appears 8px above thumb.",
    "disabled": "40% opacity; cursor: not-allowed."
  },
  "tickMarks": "Optional — declare `step` and `showTicks: true`; tick marks render every `step` along the track."
}
```

### combobox

Searchable select. Filters as the user types, supports single-pick / multi-pick / inline-tag-create modes.

```json
"combobox": {
  "variants": {
    "default":  { "background": "surface",          "border": "1px solid border", "selectionMode": "single", "note": "Single-pick — choose one." },
    "multi":    { "background": "surface",          "border": "1px solid border", "selectionMode": "multi",  "note": "Multi-pick — chips render below input as user picks." },
    "tags":     { "background": "surface",          "border": "1px solid border", "selectionMode": "multi", "allowCreate": true, "note": "Allow creating new options inline (Enter while typing)." }
  },
  "sizes": {
    "sm": { "height": "36px", "fontSize": "bodySmall", "padding": "0 12px", "radius": "lg" },
    "md": { "height": "44px", "fontSize": "body",      "padding": "0 14px", "radius": "lg", "note": "Default — matches input md." },
    "lg": { "height": "52px", "fontSize": "body",      "padding": "0 16px", "radius": "lg" }
  },
  "states": ["idle", "focus", "open", "selecting", "error", "disabled"],
  "menu": {
    "background":  "surface",
    "elevation":   "medium",
    "radius":      "lg",
    "padding":     "4px",
    "itemHeight":  "40px",
    "maxHeight":   "320px",
    "anchorOffset": "4px below trigger",
    "highlightMatch": "Bold the matched substring of each result"
  },
  "behavior": "Filters list as user types · keyboard ↑/↓ navigation · Enter selects · Escape closes · Backspace removes last chip in multi mode · empty results show 'No matches for {query}' (pulled from voice.samples.empty.searchEmpty)."
}
```

### input-otp

OTP / verification code input. Renders N slot boxes with auto-advance.

```json
"input-otp": {
  "variants": {
    "default": { "slotCount": 4, "mode": "numeric", "note": "Default — 4 digits." },
    "six":     { "slotCount": 6, "mode": "numeric", "note": "6 digits — most 2FA codes." },
    "eight":   { "slotCount": 8, "mode": "alphanumeric", "note": "Long codes — recovery, license keys." }
  },
  "sizes": {
    "sm": { "slotSize": "44px", "fontSize": "title",   "gap": "6px",  "radius": "lg" },
    "md": { "slotSize": "56px", "fontSize": "display", "gap": "8px",  "radius": "lg", "note": "Default — large enough to read at arm's length." },
    "lg": { "slotSize": "64px", "fontSize": "display", "gap": "10px", "radius": "xl" }
  },
  "states": ["idle", "focus", "filled", "error", "disabled"],
  "stateRules": {
    "focus": "Active slot border: 1.5px solid brand.700; shadow: 0 0 0 3px brand.100.",
    "error": "All slots border: 1.5px solid feedback.error; below: helper text feedback.error."
  },
  "behavior": "Type digit → auto-advance to next slot · Backspace → clear current and step back · Paste full code → splits across all slots · only the configured `mode` (numeric/alphanumeric) accepts input."
}
```

### file-upload

Drag-and-drop + click-to-browse upload zone. Lists uploaded files with per-item progress.

```json
"file-upload": {
  "variants": {
    "default":   { "selectionMode": "single",   "accept": "*", "note": "One file at a time." },
    "multiple":  { "selectionMode": "multiple", "accept": "*", "note": "Multi-file — file list renders below the zone." },
    "image":     { "selectionMode": "multiple", "accept": "image/*", "showThumbnails": true, "note": "Image-only — shows thumbnail previews instead of generic file rows." }
  },
  "sizes": {
    "sm": { "zoneHeight": "120px", "iconSize": "32px", "fontSize": "bodySmall", "padding": "16px" },
    "md": { "zoneHeight": "160px", "iconSize": "48px", "fontSize": "body",      "padding": "24px", "note": "Default." },
    "lg": { "zoneHeight": "240px", "iconSize": "64px", "fontSize": "body",      "padding": "32px" }
  },
  "states": ["idle", "dragOver", "uploading", "success", "error", "disabled"],
  "stateRules": {
    "dragOver": "Zone border: 2px solid brand.primary (dashed → solid); background: brand.50.",
    "uploading": "Per-file progress bar (uses progress.linear); cancel icon-button trailing each row.",
    "success":  "Per-file checkmark icon (feedback.success).",
    "error":    "Per-file error icon (feedback.error) + retry icon-button."
  },
  "behavior": "Drag&drop on zone · click anywhere on zone opens native picker · `accept` attribute filters MIME types in picker · `maxSize` and `maxFiles` validation runs before upload starts."
}
```

## Group-level rules

- **Pair every input with a label** — never rely on `placeholder` as a label substitute (placeholder disappears on focus and is invisible to screen readers as a label).
- **Helper text consistency.** Every form group should declare helper text typography once (`label.helperPairing`) so all fields look the same.
- **Validation messaging.** Use `patterns.formComposition.validation.strategy` to decide inline vs. summary; never mix the two on one form.
- **Native vs. custom.** `select` is custom (renders our own menu). For mobile prototypes, prototype-agent may render native pickers — that's a prototype-level decision, not a DS-level one.
- **`required` indicator is always a suffix.** Asterisk after the label, never an "(optional)" marker on every other field.
