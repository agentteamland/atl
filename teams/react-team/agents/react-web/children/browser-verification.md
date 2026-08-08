---
knowledge-base-summary: "The author-side gate for a UI change, and how to run it honestly. The minimum pass per touched route and per role that can reach it: load it in a real browser, navigate by clicking as well as by URL, no error boundary, no chunk-load failure, no 4xx/5xx, and the screen's own primary content asserted as a count or a DOM read rather than a screenshot. An empty screen is not a verified screen — the empty and populated branches are different code. Measurement traps that turn working UI into plausible findings (a stale console buffer, a pre-paint screenshot, a coordinate space that shifts under you, a depleted fixture), the positive control that makes an absence claim legitimate, and the fault-injection techniques for what a live run cannot reach — a control build, a stubbed transport layer, and forcing a race slow enough to sample."
---

# Browser Verification

This is the **author-side gate** for a change with a visible surface: the fast check I run before I
hand a unit off, on the one surface that can actually observe a UI change. It is Level-1 — my own
verification. The independent thorough pass belongs to a separate `tester` worker and I do not do it
here; the two levels exist because the mind that wrote the code wrote its self-test and shares its
blind spots.

The cardinal rule sits above everything below: **a surface that could not run is unverified, and
unverified is never a pass.** If the dev server will not start, the page errors on load, or the
browser tooling cannot drive the flow, I say so and name the criterion it leaves unproven. A false
green is the worst thing I can emit, because the tester's pass and the reviewer's gate are both built
on top of it.

## Two levels of instrument, and what each can prove

**Unit and component checks** — a Vitest run over React Testing Library, with the network stubbed at
the boundary (a request-mocking layer, not a mocked hook) — are the bottom of the pyramid: fast,
parallel, precise. Query by role, text and label rather than by test-id or class, drive interactions
with `user-event` rather than raw `fireEvent`, and assert on rendered output rather than on internal
state. They prove that a piece behaves when it is rendered.

What they structurally **cannot** prove is that the app renders it. RTL mounts a composed tree of
exactly one component, inside providers and a router the test itself declares. That is the whole
reason this page exists: the browser is the only instrument that sees the composed application.

Both run before the hand-off. `npm run typecheck` and the unit suite are necessary and not
sufficient — a green suite over an unregistered screen is green forever.

## The minimum browser pass

For **every route the change touches**, and for **each role that can reach it**:

1. **Load it in a real browser** — not a `curl`, not a source read. A lazily-imported route whose
   chunk fails to load surfaces **only** browser-side (a dynamic-import failure), while every API
   probe against the same deploy returns 200. Backend evidence is evidence about the backend.
2. **Confirm it does not fall into the error boundary.** A boundary that catches is a screen that
   rendered nothing.
3. **Confirm the console carries no chunk-load failure** (read the caveat in Trap 2 before believing
   a clean console).
4. **Confirm no request returns 4xx or 5xx — including chunk 404s.** A 404 on a code chunk is a
   deploy problem that looks like a routing problem.
5. **Confirm the screen's own primary content rendered**, as a **count or a DOM read**, never as a
   screenshot. "42 rows, 0 broken images, 1 request" survives a picture taken at the wrong
   millisecond; a picture does not.

**Navigate by clicking, not only by URL.** Clicking is the lazy-import path that a direct load can
bypass, and it is the path a user takes. A route that resolves by typed URL and is linked from
nothing is registered and still unreachable.

**An empty screen is not a verified screen.** The empty state and the populated state are drawn by
**different code**, and the interesting failures live in the populated branch. Create the data
first, then walk the screen. Whatever you created is a **cleanup debt** — record it as you go and
reverse it, or the next person inherits fixture rows as real ones. If the environment is shared,
assume a sibling worker is looking at the same data.

**Verify the path the user takes, not the one that is easiest to drive.** When a harness lets you
supply an argument the UI cannot supply, that argument is precisely the thing the UI might be
failing to supply. An endpoint driven directly with a hand-written payload can pass while the button
that is supposed to build that payload is broken.

**Enumerate the surfaces (screen x role) up front and tick them off.** A surface with no findings in
your notes is indistinguishable from one you never opened.

## State the claim at the depth you actually reached

A claim is rarely re-checked once written, so it must not be wider than the check that produced it.

- "Menus enumerated" is not "every screen opened with data present".
- "Source-verified" is not "runtime-verified" — say which one you did.
- A build that compiles is not a front end that survived an API change. Response interfaces are
  hand-written assertions; renaming a field server-side leaves `tsc` perfectly green while the value
  is `undefined` at runtime.
- **Never hang `&& echo "passed"` off a pipeline.** In a pipeline the exit status is the *last*
  command's, so `npx tsc … | tail -5 && echo "clean"` prints green when the compiler never ran at
  all. Redirect to a file, capture the status of the bare command, and print the code you got. A
  verification whose failure mode is indistinguishable from its success mode is not a verification.
- **Prove a new gate bites by injecting a deliberate error.** A gate you have never watched fail is
  a gate you have not tested.

## Measurement traps — when the instrument corrupts the measurement

Each of these produces a **plausible finding** rather than an error, which is what makes them
expensive: the output looks exactly like a defect report.

| Trap | What it looks like | What is actually happening |
|---|---|---|
| Programmatic value set | A form submits empty, the button "does nothing" | A controlled form library reads its own state, not the DOM `value` (see [forms-and-input.md](forms-and-input.md)) |
| Stale console buffer | Errors that are already fixed | The console read is a cumulative per-tab buffer |
| Pre-paint screenshot | A missing image or an empty region | The frame had not painted yet |
| Coordinate space | A dead button | The click landed somewhere else |
| Depleted fixture | An intermittent failure | The consumable the probe depends on ran out |
| Case-insensitive text match | A missing control | The regex never matched the label |
| Byte-level assertion via a decoder | A missing byte-order mark | The decode step removed it |
| Stale compiled config | A config change with no effect | A generated sibling shadows the source |

**Stale console buffer.** Treat the browser tooling's console read as a **persistent per-tab buffer**
that survives `console.clear()` and client-side navigation. Under Vite HMR a mid-edit reload
transiently breaks cross-file references, an error boundary catches the intermediate state, and those
errors linger — identifiable by a frozen `?t=` build-id on the module URLs, all pinned to the same
stale reload. For an honest zero-errors read, open a **fresh tab**, or read the console off a
production `vite build` which compiles the final source in one shot. Do not conclude "0 console
errors" from re-reading the tab you were editing on.

**Pre-paint screenshot.** A screenshot taken immediately after a navigation can capture a state that
has not painted. Read the DOM instead:

```js
[...document.querySelectorAll('img')].map(i => ({ complete: i.complete, px: i.naturalWidth }))
```

Two things before interpreting that: `complete: false` is usually **correct** for a `loading="lazy"`
image outside the viewport — it was never requested, which is the feature working — and
`complete && naturalWidth === 0` is the **only** real broken-image signal. A filter that omits the
`complete` half reports every still-loading image as broken.

**Coordinate space.** A browser-automation tool may interpret coordinates in CSS pixels or in
screenshot space (scaled by `devicePixelRatio`), and the space can change under you — measured: it
switched the moment a tab was fronted, so unchanged coordinates started landing at half position,
the click hit the page background, and nothing happened. That produced **three false dead-button
claims in a row** in one session, all of which worked once the coordinates were halved. Trust the
tool's own reported screenshot size, not the pixel dimensions of the image you are looking at. To
pin the mapping in one call:

```js
window.addEventListener('click', e => console.log('got', e.clientX, e.clientY), true);
```

**Depleted fixture.** This one is the *fixture*, not the tool. When a probe depends on a consumable
— a one-shot token, a rate-limited endpoint, a seeded row — a depleted consumable and a real defect
are indistinguishable from the status code alone. Two instances in one session: a rate limiter
silently starved the rig so a helper replayed an already-spent token, and a shell quoting bug emptied
the variable under test so the endpoint honestly rejected an empty value. Both were one keystroke
from being written up as behaviour. **Print a prefix of the value under test each round**, and reset
the limiter between rounds.

**Case-insensitive text match.** A JavaScript case-insensitive regex does not fold every locale's
capitals to ASCII. The dotted capital I (U+0130), used in Turkish and Azerbaijani, does not fold to
ASCII `i`, and `toLowerCase()` does not normalize it either — so a `/i…/i` pattern silently matches
nothing and the selector returns no element, which reads as "the button is missing". Select by
position, match an ASCII-safe substring of the label, or fold with an explicit locale.

**Byte-level assertion via a decoder.** `Blob.text()` UTF-8-decodes, and the decode step strips a
leading byte-order mark by spec — so testing a downloaded file for a BOM with `text()` is a
guaranteed false negative. Read the raw bytes:

```js
new Uint8Array(await blob.arrayBuffer()).slice(0, 3)   // [239, 187, 191] = EF BB BF
```

**Stale compiled config.** Vite resolves `vite.config.js` before `vite.config.ts`, and a `tsc -b`
build can emit a compiled `vite.config.js` next to the source. From that moment a frozen snapshot
takes precedence and **every subsequent edit to the `.ts` is silently ignored**, surviving a
container restart. Observed while verifying a dev proxy: the target was pointed at a dead host, then
deleted outright, and requests still reached the real service, with nothing in the logs saying why.
Before trusting any dev-server behaviour, confirm the artefact is absent — it is normally gitignored,
so deleting it is free.

## Take a positive control before calling anything dead or absent

**An absence is a claim about your instrument as much as about the page.** A "nothing happened" is
only evidence once you have proven the probe works at all:

- **For a click** — click an element *known* to respond, with the identical coordinate mapping. When
  the control also fails to react, the mapping is the suspect rather than the page. That is what
  retracted the three dead-button claims above.
- **For a zero count** — run the identical query on a sibling screen where a positive result is
  known to exist. A zero from a query that could never return non-zero is not a measurement.

## Fault injection — reaching what a live run cannot

Three techniques for branches a normal run will not enter. All three share one rule: **revert the
instrumentation, confirm the diff is empty, and name what the technique does not prove.**

**A control build — for any claim shaped "X no longer happens."** "No console errors" is weak: the
defect may never have rendered, the buffer may be stale, or the measurement may not exist. Run the
**unmodified** build on a second port beside the patched one and compare the **same measurable, on
the same page, with the same data** — e.g. `document.querySelectorAll('button button').length` for
nested-interactive-element hygiene, which gave a real 50 to 0 instead of an unfalsifiable "looks
clean". Two constraints: build-time-inlined environment variables must be passed to the throwaway
container explicitly, since it inherits nothing from the normal compose setup; and the build must run
where `node_modules` was installed — a Linux container install cannot be built from a macOS host,
because the bundler ships a platform-specific native binary and the resulting red is an environment
artefact, not a code failure. Do not report that red as a defect and do not "fix" it by reinstalling
on the host.

The same control-build shape answers **"is this a real bug or a development-mode artefact?"**
`<StrictMode>` double-invokes effects in development builds only, so any defect that rides on
mount to unmount to remount is invisible in production. Run the identical flow against dev and
against a production build and compare the request count. A related corollary: React's DOM-nesting
validation fires **at render time regardless of CSS visibility**, so responsive variants hidden by a
breakpoint class are still auditable at desktop width — you do not need to emulate a phone.

**Stub the transport — when the backend the change was written against is not deployed.** Every
endpoint 404s and the usual answer is "cannot verify until it merges". Patch
`XMLHttpRequest.prototype.open`/`send` in the page and synthesize contract-shaped responses:
instance-level `Object.defineProperty` on `readyState`/`status`/`responseText`/`response`, a stubbed
`getAllResponseHeaders()`, and a `setTimeout` that fires `onreadystatechange`, then `onload`, then
`onloadend`. An axios XHR adapter accepts this, so the **real** query layer, the real form library
and the real error-mapping paths run — including field-vs-banner routing on a validation rejection
and the forbidden-response branch — while writing **zero rows** to a shared database.

**The honesty constraint is mandatory: report it as "wiring verified, server round-trip
UNVERIFIED."** A stub that returns exactly what the front end expects cannot detect a contract
mismatch, which is precisely the failure such a change is most likely to have.

**Force a race slow enough to sample.** This is the inverse of the pre-paint trap: there the read
arrives too early, here every read arrives too late, because a scripted navigate-then-read waits for
load to finish and the window has already closed. Pin the single gating condition open — hold the one
hook every guard calls at its pre-ready value for about 1500 ms — and change the claim from a
snapshot to a **sequence**: on a protected deep link with a valid session, the app must read
splash, then content, and **never** login. Sample off a marker that occurs exactly once and only on
the splash, collecting into an array rather than logging, which also sidesteps the stale console
buffer entirely:

```js
const seq = [];
const t = setInterval(() => seq.push([location.pathname, !!document.querySelector('[data-splash]')]), 100);
// after ~2 s: clearInterval(t); seq
```

**Then revert and confirm `git diff` is empty before committing.** This is not tidiness: the
instrumentation *inverts* the invariant under test — a gate pinned open renders the splash
unconditionally and the app never shows content at all — so one surviving line ships the loudest
possible regression, produced by the very check that proved the fix. And be honest about the bound:
it proves the guard renders correctly *while the window is open*, not that the flash is gone in
ordinary use.

## Telling a load flake from a defect

A browser smoke suite creates a failure mode of its own: a red run that is not a defect. Its
signature is **three things together** — disjoint failure sets across runs (a real defect reproduces
on the same tests), every failure passing in isolation, and total runtime tracking machine load
rather than code. Any one alone is consistent with a genuine intermittent defect. Establish it by
re-running the failures individually and comparing failure sets, then take a clean full green.
"It is probably just flaky" is the same sin as shipping past a real failure, wearing a lab coat.

And note the scope limit: **a read-only suite measures render paths, not submit paths.** Its failure
count is a **lower bound** on a defect's blast radius, never its size. Confirm scope by grepping the
call sites, not by counting red tests.

## What travels with me, and what does not

The discipline here — the minimum pass, claim-at-the-depth-reached, the positive control, the
fault-injection techniques and their honesty constraints — is stack craft and travels. **Which**
screens exist, which roles reach them, how this project's dev server is started, and what its
evidence conventions are, are project facts: they reach me through the task at hand, and I never
pre-author them here. Where the evidence is attached is the project's convention, not anything I
own.
