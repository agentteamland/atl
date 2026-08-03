---
knowledge-base-summary: "The pieces below a screen: props typed at the boundary and locals inferred, function components and hooks only, and the presentational/container split decided by **data-ownership** rather than folder taste — the leaf that owns no data is the one testable in isolation. Composition and variants over copy-paste, callbacks expressed as domain events rather than raw DOM events, feature-local vs shared placement by blast radius, styling through the project's system rather than ad-hoc values, and every user-facing string through the i18n layer from the first commit."
---

# Component Craft

Everything below a screen. The screen itself — its route registration and its four render branches —
is [screen-blueprint.md](screen-blueprint.md); this file is the craft for the pieces that screen is
built out of, and the whole aim is that they **compose and stay testable**.

## Function components and hooks only

No class components. The reason is not taste: the testing library's render model, the data-fetching
layer, and the rules-of-hooks lint all assume the hooks model. A class component drops out of every
one of them and forces each reader to context-switch. The consistency is load-bearing.

Share **behaviour** through a custom `use<Thing>` hook, not through a wrapper component. A hook
composes logic without adding a node to the tree; a wrapper added purely to share logic adds
indirection and a render layer for nothing. The `use` prefix is what makes the lint rule apply.

## Typed at the boundary, inferred within

A component's props type **is** its public contract — it is what lets another developer, or a test,
use the component without reading its body. Inside the body, lean on inference; annotating every
local adds noise and hides the contract in it.

- **Name it `<Component>Props`.** Predictable, greppable, readable at the call site.
- **Model optional-vs-required honestly.** A prop the component can render without is `?`; one it
  cannot is required. That pushes "did the caller pass it?" into the type system instead of into a
  runtime guard.
- **No `any`.** Where a value is genuinely untyped, `unknown` plus a deliberate narrow — `any`
  disables the checker exactly where a mistake is most likely.
- **Never widen a prop to make a test pass.** If a test needs a prop the type forbids, either the
  type is the spec and the test is wrong, or the type is wrong. Fix the real one.

### Callbacks are domain events, not DOM events

```ts
type UserCardProps = {
  user: User;
  onSelect?: (userId: string) => void;   // yes — what happened
  // onSelect?: (e: React.MouseEvent) => void;   // no — how it happened
};
```

The parent cares that a user was selected, not that a mouse was clicked. Leaking the DOM event
couples the parent to the child's markup, so the child can never change from a button to a row
without breaking every caller. Type real DOM handlers properly where you genuinely need one
(`React.MouseEvent<HTMLButtonElement>`) — the rule is about the component's own boundary.

## The presentational / container split — by data-ownership

A component either **owns data** (fetches it, holds it, decides with it) or it **only renders what it
is handed**. Keep the two apart, and decide the split by that question rather than by folder taste:

- The **presentational leaf** takes props and returns markup. No query hook, no store subscription,
  no fetch. That is precisely what makes it testable in isolation — render it with props, assert the
  output — and reusable in a second context.
- The **container** wires the data and passes it down.

Do not manufacture a container for a component with no data to own. The value here is testability and
reuse; apply it where those actually pay.

**The split also decides where the component gets registered**, which is the step most often
skipped ([screen-blueprint.md](screen-blueprint.md)). A presentational leaf is registered by its
parent's JSX — the parent renders it or it appears nowhere. A container that is a route target is
registered in the route table. In both cases something the app root reaches must name it, and a
component nothing names passes lint, type-check and the whole unit suite green while appearing on no
screen at all: its colocated test renders it in a composed tree of one, so "it renders" is proven
while "the app renders it" is never tested.

## Extend by props and variants, not by copy-paste

The signal to extract is mechanical: **the second time you copy JSX, extract it.** The signals to
split an existing component are the same — it grew past the point a reader can hold it, or it carries
three or more conditional branches that each belong to a different reader.

For visual variation, a single base plus a typed variant map beats a fork:

```ts
const buttonVariants = cva(base, {
  variants: { variant: { primary: '…', ghost: '…' }, size: { sm: '…', md: '…' } },
  defaultVariants: { variant: 'primary', size: 'md' },
});

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & { isLoading?: boolean };
```

Two things this buys that an if/else chain does not: the variant prop is **type-checked** against the
declared set, and the variant builder can be **exported** so a link can be styled as a button without
duplicating the class list. `class-variance-authority` is the common implementation and a reasonable
default; the project's `Conventions/` decides.

For structural variation, compose rather than configure: `children` for free-form content, named
`ReactNode` slots (`actions`, `header`) for structured content, a render prop when the caller must
control an item's markup. A component growing a tenth boolean flag is asking to be a slot.

## Placement by blast radius

The question is not "is this reusable?" — it is **"who breaks when I change it?"**

- **Feature-local** (`features/<feature>/`) — used only inside that feature. Safe to change; the
  blast radius is one folder. This is the correct default, including for something that *looks*
  general. A second consumer promotes it; a guess does not.
- **Shared** (`components/`) — used across features. Changing it is a whole-app change and should
  feel like one.

Colocate a feature's component, hook, types and test together rather than splitting them by file
type across the tree: a change to a feature touches all of them at once, so adjacency means one
folder per change instead of a scavenger hunt. One component per file, file named for the component
in PascalCase, test colocated as `<Component>.test.tsx` so a moved component moves its test.

The project may pin a different tree (a monorepo layout, an app-router convention, an atomic-design
taxonomy). **The project's structure wins** — this is the default for a project that has not said.

## Style through the project's system

Whatever the project uses — a utility framework, CSS modules, a styled API — the discipline is the
same and it is about **values, not syntax**:

- **Never hardcode a colour, spacing or radius value.** Use the project's tokens and scale. A literal
  value is invisible to a theme change and will be wrong the day the palette moves.
- **Never build a class string by concatenation.** Use the project's merge helper so a conditional
  class and a caller's `className` override compose predictably instead of colliding.
- **No inline style objects** for anything the system can express — they escape the system entirely.
- If the project supports a dark mode, every colour decision needs its counterpart. A component that
  looks right in one theme and unreadable in the other is not finished.

Read the project's existing components before writing a new one. Matching what is there matters more
than any default in this file.

## Every user-facing string through the i18n layer, from the first commit

Retrofitting localization is a whole-file sweep that reliably misses the strings inside conditional
branches, in `aria-label`s, in `placeholder`s, and in error text — precisely the ones a reader will
not notice are untranslated. Route them through the layer as you write them, even when the project
ships one language today.

- Group keys by feature namespace; add the key to **every** supported locale in the same commit, or
  the missing one only surfaces to the user who speaks it.
- Format dates and numbers with `Intl` and the active locale — never by hand, and never with a
  hardcoded separator.
- Where the API sends **message keys** rather than final text, resolve them through the same layer
  with the raw key as a last-resort fallback ([api-client.md](api-client.md)). Never render a key to
  a user; never build a second translation path for server strings.

## Accessibility floor

These are cheap, commonly missed, and each is a real defect rather than a nicety:

- **Icon-only controls need an accessible name** (`aria-label`). Without one the control is nameless
  to a screen reader and to most test queries.
- **Anything clickable is reachable and operable by keyboard.** If you put a click handler on a
  non-interactive element, you have taken on the role, the tab index and the Enter/Space handling —
  which is the argument for using a real `button` instead.
- **Semantic elements over styled `div`s**, and a visible focus state. Both come free with the
  semantic element and must be rebuilt by hand without it.
- **Never nest interactive elements** — a button inside a button, an anchor inside an anchor. It is
  invalid, the nested control's behaviour is undefined across browsers, and React reports it at
  render time regardless of whether the element is visible at your current viewport.
