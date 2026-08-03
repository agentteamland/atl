---
knowledge-base-summary: "Schema-first forms: one schema drives both validation and the inferred type, with the submit path owning loading and error state. Mapping a server rejection back onto the field that caused it (and to a banner when it belongs to no field), keyed by what the backend actually emits. Locale-correct numeric entry as three inseparable layers — a text input plus a locale-aware parser that returns a rejectable value for ambiguous and empty input, a field seeded with the same formatter the surrounding text uses, and the parsed value echoed live. Plus the pre-filled-form guard: compare parsed values against the original record, never test for empty fields, or an untouched submit writes a change that never happened."
---

# Forms And Input

A form is the one place where the user's intent becomes a value the system will act on, and almost
every failure in this topic is **silent**: a wrong-but-valid value passes every check on both sides
of the network and is recorded as what the user meant.

My baseline is **a schema-driven form library with a resolver** — React Hook Form plus Zod is the
combination I assume, because it is the one this craft was earned on. Where the project's
`Conventions/` pins something else, the project wins; the shape below (schema drives type, submit
path owns its own state, server rejections map back onto fields) transfers to any of them, and I say
at the point of contradiction which default I am giving up.

## Schema first — one schema, two outputs

Write the schema before the JSX. It is the single source of truth for **both** the validation rules
and the TypeScript type; deriving the type from the schema is what stops the two from drifting.

```ts
export const createUserSchema = z.object({
  name:  z.string().min(2).max(100),
  email: z.string().min(1).email(),
  role:  z.enum(['admin', 'member', 'viewer']),
  bio:   z.string().max(500).optional(),
});

export type CreateUserForm = z.infer<typeof createUserSchema>;
```

Then wire it once:

```tsx
const form = useForm<CreateUserForm>({
  resolver: zodResolver(createUserSchema),
  defaultValues: { name: '', email: '', role: 'member', bio: '' },
});
```

Four rules with reasons, not taste:

- **`defaultValues` is always fully specified.** A field that starts `undefined` and later receives a
  value flips from uncontrolled to controlled, which React warns about and which loses the first
  keystroke in some component wrappers.
- **`noValidate` on the `<form>` element.** Otherwise the browser's own validation fires first, in
  its own language and its own styling, and the schema never runs.
- **Simple inputs go through `register()`; anything with its own internal state goes through
  `Controller`** — a date picker, a rich text editor, a custom select. The dividing line is whether
  the component exposes a native `ref` and fires native events.
- **Validation here is UX, never authority.** It gives the user fast feedback. The rule that must
  hold, holds server-side; the client reflects the server's answer, including its rejections. A
  form-level check is not an enforcement boundary and must never be the only one.

For a multi-step form, keep **one** form instance with `FormProvider` and validate per step with
`trigger(fieldsForThisStep)`. Deferring all validation to the final submit means the user learns on
step 4 that step 1 was wrong.

## The submit path owns loading and error state

The submit handler is where a form stops being a rendering problem and becomes a network problem.
Three states it owns, all of them visible:

```tsx
const { mutate, isPending } = useCreateUser();
// ...
<Button type="submit" isLoading={isPending} />
```

- **In flight.** The submit control is disabled and says so. A form that accepts a second click while
  the first request is open produces duplicate writes, and on a non-idempotent endpoint that is a
  real defect, not a cosmetic one.
- **Rejected.** Mapped below. Never swallowed.
- **Succeeded.** Reset, navigate, or close — and **clear any previous banner**. A success banner that
  is not cleared when the next attempt starts stays on screen through a subsequent *failure*, so the
  screen shows success and failure simultaneously and the user believes the first one.

## Mapping a server rejection back onto the field that caused it

A rejection has two destinations, and choosing wrongly loses information the user needs:

```ts
function onError(error: unknown) {
  const apiError = parseApiError(error);

  if (apiError.errors) {                       // field-keyed dictionary
    Object.entries(apiError.errors).forEach(([field, messages]) => {
      form.setError(field as keyof FormValues, { type: 'server', message: messages[0] });
    });
    return;
  }
  showBanner(apiError.message);                // belongs to no field
}
```

- **A field-keyed error goes to its field**, via `setError`. A toast for something the server said
  about one field makes the user hunt for it.
- **Everything else goes to a banner** — a conflict with no field attached, a permission refusal, a
  transport failure. Rendering it inline under an arbitrary field is worse than a banner.

**The key casing is the trap.** The dictionary's keys follow **the backend's own property naming**,
not JSON convention. A .NET API validating with property-name defaults emits PascalCase
(`{"errors":{"Amount":[…],"Email":[…]}}`), so a camelCase lookup finds nothing and the per-field
message **silently fails to render even though the rejection arrived correctly** — which reads as
"validation is broken" when it is a casing mismatch. Read one real rejection body in the network
panel before writing the mapping, and normalize in one place rather than at each call site.

Two more shapes worth handling explicitly because their generic message is useless: a **conflict**
response that means one specific field is already taken (map it to that field), and a
**concurrency/version** rejection that means the record changed underneath the form (the user needs
to be told to reload, not that "something went wrong").

## Numeric entry — three inseparable layers

This is the single most expensive thing on this page, and it is worth the length because the failure
is **silent, valid, and unrecoverable**.

`Number(raw.replace(',', '.'))` is the standard-looking parser for a comma-decimal locale, and it
**divides grouped input by 1000**. In any locale that groups with `.` and takes `,` as the decimal
separator — a large family including `de-DE`, `es-ES`, `pt-BR`, `tr-TR` — `Number('2.000')` is
**`2`**. And `2` is a perfectly valid number: it passes `Number.isInteger`, it passes
`Number.isFinite`, it passes every server-side range check. Nothing anywhere reports anything, and
whatever audit trail exists records the corrupted value as a deliberate user action.

The screen sets the trap itself: it renders every number grouped (`1.500`) while seeding the input
with `String(1500.5)` — `"1500.5"` — so two opposite conventions meet inside one visual field.

**`<input type="number">` does not save you.** Measured with real keystrokes in Chrome under a
dot-grouping locale:

| Typed | `.value` | `valueAsNumber` | `validity.badInput` |
|---|---|---|---|
| `1.000` | `"1.000"` | **1** | false |
| `1234,5` | `"1234.5"` | 1234.5 | false |
| `1.234.567` | `"1.234567"` | **1.234567** | false |

Nothing ever sets `badInput`. Every case is silently *valid*, which is exactly why this class
survives code review — and `1.234.567` becomes a number six orders of magnitude wrong rather than
blanking.

### Layer 1 — `type="text"` plus a locale-derived parser

Do not hard-code the separators. Ask `Intl` what this locale uses, so the parser and the formatter
rendering values beside it can never drift apart:

```ts
const parts = new Intl.NumberFormat(locale).formatToParts(12345.6);
const group = parts.find(p => p.type === 'group')!.value;    // "." in a dot-grouping locale
const dec   = parts.find(p => p.type === 'decimal')!.value;  // ","
```

Accept exactly two shapes — plain (`1234`, `1234,56`) and correctly grouped in its exact
`\d{1,3}(G\d{3})*` form (`1.234`, `1.234.567,89`) — and return **`NaN` for everything else**. `NaN`
is the point: the schema's existing numeric refinement already rejects it, so the user sees a
**visible** error instead of a silently reinterpreted number.

**The empty string must also yield `NaN`**, because `Number('')` is `0` and a blank field must never
become a real zero.

`1234.5` **becoming** an error under such a locale is correct, not a regression: it is genuinely
ambiguous there, and guessing is what caused the bug.

**Pointing this parser at a `type="number"` input is worse than doing nothing** — the browser has
already canonicalized the value, so the parser reads an already-normalized `"1.000"` as one thousand
and corrupts input that was previously merely wrong.

### Layer 2 — seed the field with the same formatter the surrounding text uses

`String(record.quantity)` under a header reading `1.500` teaches the user that this one field speaks
dot-decimal while everything around it does not. Seed with the locale formatter, and confirm the
values round-trip losslessly (format, then parse, and get the original back).

### Layer 3 — echo the parsed value live under the input

```
CURRENT   1.500
[ 2.000        ]
Will save: 2.000
```

This is the only layer that **does not assume the parser is right.** Layers 1 and 2 both trust the
parsing logic; layer 3 shows the user the number that will actually be sent, at the moment of
sending, so a future parser bug reads back as a different number instead of shipping silently.

### The three layers are one unit

Package them as a single field component so a new form cannot get two of the three. Before adding
any numeric input to an existing codebase, **grep for the shared parser, for `type="number"`, and for
`replace(',', '.')`** — this defect class arrives by copy-paste from the screen next door, and a
codebase that fixed it in one feature area very often still has it in another.

## The pre-filled-form guard

A form that pre-fills from an existing record breaks the usual "nothing was entered" guard: the
fields are **never empty**, so the guard never fires and an untouched submit sails through — writing
a change where none happened, which is worst wherever the trail is append-only.

**Compare the parsed submitted values against the original record**, field by field, never against
emptiness. Compare the *parsed* values, not the raw strings, or a re-formatted-but-identical number
reads as an edit.

## Verifying a form in the browser

Cross-reference [browser-verification.md](browser-verification.md) for the full gate; two things are
specific to forms and both are measurement traps that will waste a debugging round.

**A controlled form library ignores programmatic value sets.** Setting a field with a browser
automation tool's form-input helper, or with a synthetic type action, does **not** update the
library's internal state — so the submit handler reads empty values and no-ops, and the form looks
dead. Drive it through the native setter plus **bubbling** events, which is what the library's
subscription actually listens for:

```js
const setNativeValue = (el, value) => {
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
  setter.call(el, value);
  el.dispatchEvent(new Event('input',  { bubbles: true }));
  el.dispatchEvent(new Event('change', { bubbles: true }));
};
// set each field, then:
form.requestSubmit();
```

`requestSubmit()` runs the real submit path — validation, mutation, success handler — rather than
faking it.

**On a numeric field, programmatic setting and real typing disagree in opposite directions.**
Measured on a `type="number"` input under a dot-grouping locale: `el.value = "1234,5"` yields `""`
while real keystrokes yield `"1234.5"`. Any check on this field class written with the native setter
alone will mis-report it — assert the **captured request payload**, which is the only reading that is
about the system rather than about the harness.

And read the rejection path in the browser at least once: submit deliberately invalid data and
confirm the per-field message actually renders, since the casing mismatch above fails exactly here
and nowhere else.

## Pitfalls

- **A field validated only in the client.** The schema is UX; the server decides. If nothing
  server-side enforces the rule, the rule is not enforced.
- **A toast where a field error belongs.** The user is left hunting for which input the server meant.
- **A camelCase lookup into a PascalCase error dictionary.** The rejection arrives, nothing renders.
- **A submit control with no in-flight state.** Double submits on a non-idempotent endpoint.
- **A success banner that outlives its attempt**, so a later failure is displayed under a success.
- **`type="number"` on anything whose value matters.** Silent, valid, wrong.
- **A parser pointed at a canonicalized input.** Worse than the bug it was added to fix.
- **A "nothing changed" guard testing for empty fields on a pre-filled form.** It can never fire.
- **A schema and a hand-written type that drifted.** Derive the type; never declare it twice.
