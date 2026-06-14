---
knowledge-base-summary: "Motion principles: meaningful, quick, consistent. Easing curves, duration scale (micro to long). Reduced motion support. Common animation patterns (fade, slide, scale, shimmer). Platform implementation."
---
# Animation & Motion

Motion in the design system serves a purpose: it guides attention, provides feedback, and creates a sense of continuity. Motion is never decorative — every animation must earn its existence.

## Motion Principles

### 1. Meaningful
Every animation must have a reason:
- **Guide attention** — draw the user's eye to what changed
- **Provide feedback** — confirm an action was received (button press, submit)
- **Show relationships** — this element came from that element (expand, collapse)
- **Reduce cognitive load** — smooth transitions prevent disorientation

If an animation does not serve one of these purposes, remove it.

### 2. Quick
Users should never wait for an animation to finish before they can act. Most animations should be 200-300ms. If it feels slow, it IS slow.

### 3. Consistent
Same type of motion everywhere. If list items fade in, ALL list items fade in. If pages slide, ALL pages slide. Mixing motion types creates visual noise.

## Duration Scale

```
Token        Duration    Usage
──────────────────────────────────────────────────────────────────
micro        100ms       Micro-interactions: button press, toggle, checkbox
short        200ms       Small transitions: fade, color change, icon swap
medium       300ms       Standard transitions: panel open, card expand, page transition
long         500ms       Large transitions: full-screen overlay, complex layout shift
```

### Duration Rules

1. **Smaller elements = shorter duration** — a toggle (100ms) is faster than a modal (300ms)
2. **Appearing is faster than disappearing** — enter: 300ms, exit: 200ms
3. **Never exceed 500ms** — anything longer feels sluggish (exception: loading spinners, progress bars)
4. **Staggered lists**: 50ms offset between items, max 5-7 items animated (rest appear instantly)

## Easing Curves

```
Curve              CSS                          Flutter                        Usage
──────────────────────────────────────────────────────────────────────────────────────
Ease Out           cubic-bezier(0, 0, 0.2, 1)  Curves.easeOut                 Enter animations (appear)
Ease In            cubic-bezier(0.4, 0, 1, 1)  Curves.easeIn                  Exit animations (disappear)
Ease In Out        cubic-bezier(0.4, 0, 0.2, 1) Curves.easeInOut              Move between states
Ease Out Back      cubic-bezier(0.34, 1.56, 0.64, 1) Curves.easeOutBack       Playful enter (bounce)
Linear             cubic-bezier(0, 0, 1, 1)    Curves.linear                  Progress bars, spinners
```

### Easing Rules

1. **Ease Out for entering** — element starts fast, decelerates. Feels natural (like throwing a ball that comes to rest)
2. **Ease In for exiting** — element starts slow, accelerates. Gets out of the way quickly
3. **Ease In Out for state changes** — element is already visible and transforming in place
4. **Never use linear for UI transitions** — linear motion feels robotic (only for progress indicators)

## Common Animation Patterns

### Fade

The simplest and most versatile animation. Opacity: 0 to 1.

```
Enter:   opacity 0 → 1, duration short (200ms), ease out
Exit:    opacity 1 → 0, duration short (200ms), ease in
```

**When to use:** Appearing elements, content replacement, overlays, toasts.

**Flutter:**
```dart
AnimatedOpacity(
  opacity: isVisible ? 1.0 : 0.0,
  duration: Duration(milliseconds: 200),
  curve: Curves.easeOut,
  child: content,
)

// Or with FadeTransition for route transitions
```

**React/CSS:**
```css
.fade-enter {
  opacity: 0;
}
.fade-enter-active {
  opacity: 1;
  transition: opacity 200ms ease-out;
}
.fade-exit-active {
  opacity: 0;
  transition: opacity 200ms ease-in;
}
```

### Slide

Element moves in from an edge. Often combined with fade.

```
Slide Up (bottom sheet, toast):
  Enter: translateY(100%) → translateY(0), duration medium (300ms), ease out
  Exit:  translateY(0) → translateY(100%), duration short (200ms), ease in

Slide Right (page push):
  Enter: translateX(100%) → translateX(0), duration medium (300ms), ease out
  Exit:  translateX(0) → translateX(-30%), duration medium (300ms), ease in
```

**When to use:** Page transitions, bottom sheets, drawers, side panels, slide-in notifications.

**Flutter:**
```dart
SlideTransition(
  position: Tween<Offset>(
    begin: Offset(0, 1), // from bottom
    end: Offset.zero,
  ).animate(CurvedAnimation(
    parent: controller,
    curve: Curves.easeOut,
  )),
  child: content,
)
```

**React/CSS:**
```css
.slide-up-enter {
  transform: translateY(100%);
}
.slide-up-enter-active {
  transform: translateY(0);
  transition: transform 300ms cubic-bezier(0, 0, 0.2, 1);
}
```

### Scale

Element grows or shrinks. Used sparingly — mostly for dialogs and FABs.

```
Dialog enter:
  Enter: scale(0.9) + opacity 0 → scale(1) + opacity 1, duration medium (300ms), ease out
  Exit:  scale(1) + opacity 1 → scale(0.95) + opacity 0, duration short (200ms), ease in

FAB press:
  Press: scale(1) → scale(0.95), duration micro (100ms), ease in out
  Release: scale(0.95) → scale(1), duration micro (100ms), ease out
```

**When to use:** Dialogs, popups, FAB press feedback, image zoom.

**Flutter:**
```dart
ScaleTransition(
  scale: Tween<double>(
    begin: 0.9,
    end: 1.0,
  ).animate(CurvedAnimation(
    parent: controller,
    curve: Curves.easeOut,
  )),
  child: FadeTransition(
    opacity: controller,
    child: dialog,
  ),
)
```

### Shimmer (Skeleton Loading)

Placeholder animation while content loads. A gradient sweeps across placeholder shapes.

```
Duration:   1500ms per sweep
Direction:  left to right
Gradient:   surface-container → surface-container-high → surface-container
Loop:       continuous until content loads
```

**When to use:** List loading, card loading, page loading. Replaces spinners for content-shaped areas.

**Flutter:**
```dart
// Use shimmer package or custom implementation
Shimmer.fromColors(
  baseColor: Theme.of(context).colorScheme.surfaceContainerHighest,
  highlightColor: Theme.of(context).colorScheme.surfaceContainerLow,
  child: Container(
    height: 16,
    width: 200,
    decoration: BoxDecoration(
      color: Colors.white,
      borderRadius: BorderRadius.circular(4),
    ),
  ),
)
```

**React/Tailwind:**
```html
<div class="animate-pulse">
  <div class="h-4 w-48 bg-neutral-200 dark:bg-neutral-700 rounded" />
  <div class="h-4 w-32 bg-neutral-200 dark:bg-neutral-700 rounded mt-2" />
</div>
```

### Staggered List

List items appear one after another with a slight delay between each.

```
Per item delay:    50ms
Max items to animate: 5-7 (rest appear instantly)
Individual item animation: fade + slide up (small, ~16px)
Individual duration: medium (300ms)
```

**Flutter:**
```dart
// Use AnimatedList or custom stagger
AnimatedBuilder(
  animation: controller,
  builder: (context, child) {
    return SlideTransition(
      position: Tween<Offset>(
        begin: Offset(0, 0.1),
        end: Offset.zero,
      ).animate(CurvedAnimation(
        parent: controller,
        curve: Interval(
          index * 0.1,  // stagger
          (index * 0.1) + 0.6,
          curve: Curves.easeOut,
        ),
      )),
      child: FadeTransition(
        opacity: CurvedAnimation(
          parent: controller,
          curve: Interval(index * 0.1, (index * 0.1) + 0.6),
        ),
        child: child,
      ),
    );
  },
)
```

## Reduced Motion

The design system MUST respect the user's reduced motion preference. This is an accessibility requirement.

### Detection

**Flutter:**
```dart
final reduceMotion = MediaQuery.of(context).disableAnimations;

Duration getAnimationDuration(Duration standard) {
  return reduceMotion ? Duration.zero : standard;
}
```

**React/CSS:**
```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
```

```tsx
function usePrefersReducedMotion(): boolean {
  const [prefersReduced, setPrefersReduced] = useState(
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  );

  useEffect(() => {
    const mql = window.matchMedia('(prefers-reduced-motion: reduce)');
    const handler = (e: MediaQueryListEvent) => setPrefersReduced(e.matches);
    mql.addEventListener('change', handler);
    return () => mql.removeEventListener('change', handler);
  }, []);

  return prefersReduced;
}
```

### What to Keep vs Remove

```
Keep (essential feedback):
  - Loading spinners (slow them down, don't remove)
  - Progress bars
  - Focus indicator transitions
  - Color changes (hover states)

Remove or reduce:
  - Slide transitions → instant or crossfade
  - Scale animations → instant
  - Staggered lists → all appear at once
  - Parallax → static
  - Auto-playing carousels → static, manual control
  - Decorative animations → remove entirely
```

## Motion Tokens (CSS Custom Properties)

```css
:root {
  --motion-duration-micro: 100ms;
  --motion-duration-short: 200ms;
  --motion-duration-medium: 300ms;
  --motion-duration-long: 500ms;

  --motion-ease-out: cubic-bezier(0, 0, 0.2, 1);
  --motion-ease-in: cubic-bezier(0.4, 0, 1, 1);
  --motion-ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
  --motion-ease-bounce: cubic-bezier(0.34, 1.56, 0.64, 1);
}

@media (prefers-reduced-motion: reduce) {
  :root {
    --motion-duration-micro: 0ms;
    --motion-duration-short: 0ms;
    --motion-duration-medium: 0ms;
    --motion-duration-long: 0ms;
  }
}
```

## Rules

1. **Every animation must have a purpose** — decorative motion is banned
2. **Duration maximum is 500ms** — nothing should take longer
3. **Ease Out for enter, Ease In for exit** — this is the default, deviate only with reason
4. **Respect prefers-reduced-motion** — mandatory, not optional
5. **Consistency** — same animation for the same type of action across the app
6. **Performance** — animate only `transform` and `opacity` (GPU-accelerated). Never animate `width`, `height`, `top`, `left` (triggers layout)
7. **Stagger limit** — maximum 5-7 items in a staggered list, rest appear instantly
8. **No animation on first load** — page content appears instantly on initial load, animations only on subsequent interactions
