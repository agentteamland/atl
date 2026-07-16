---
knowledge-base-summary: "How I capture and attach verification evidence: attaching it to the work-item per the active backend's adapter (concept #12 — with the worker's env credential, never in argv), what evidence a reviewer and the PO actually need (reproducible, self-describing, tied to a criterion), reading it back per the active adapter, and why the proof — not my word — is what makes my green trustworthy downstream."
---

# Evidence Collection

My verdict is only as trustworthy as the proof behind it. A bare "pass" comment is a claim; an
attached screenshot of the web surface satisfying an acceptance criterion, or the test-run output
showing every gate green, is *evidence*. The `tech-lead` review (micro-loop step 7) and the human
PO (at sprint-review) both make decisions on my green — so the green has to be *inspectable*, not
taken on faith. This child is how I collect it and attach it correctly.

## Attaching evidence — per the active backend's adapter (concept #12)

Attaching an artifact (a screenshot, a test-result file) to the work-item is concept #12 in the
backend interface; the active backend's adapter (`backends/<backend>/adapter.md`, chosen once at
`/delivery-init` and cached in `.delivery/config.json`'s `backend` field, default `azure`) binds it
to a concrete mechanism, so I attach through that **one** documented path and never improvise a
transport. What I rely on, whichever backend is active:

- **I see `attach(work-item, file)`; the transport stays hidden in the adapter.** Whatever the
  backend actually does to record the bytes and tie them to the unit, I call the neutral operation
  and read the concept — I never hand-roll the raw mechanism.
- **The credential comes from my env, never the argv.** The attach path reads the same backend
  credential `atl work dispatch` already put in my environment for every other backend call. It
  rides an auth header, is never on the command line, and is never logged. I **never** write, echo,
  or pass a literal token — the same secret-hygiene the whole interface enforces (the credential is
  referenced by name, never stored).
- **It's worker-runnable and safe by construction.** I already have the credential and the network,
  so I can run it directly. It validates its inputs (the work-item id, that the file exists, the
  credential/target present), builds and parses request payloads safely (so a filename or comment
  with quotes/newlines/reserved chars can't corrupt the request), and retries transient errors with
  bounded backoff — the same resilience posture as every other backend call (concept: resilience).

The Go orchestrator stays zero-backend through all of this — the attach lives in the *team*, run by
me, not in `atl`.

## What evidence a reviewer and the PO actually need

Not "some files." Evidence earns its keep only if a downstream reader can act on it without re-running
my work:

- **Reproducible** — the evidence shows the *state and input* that produced the result, so the
  tech-lead can re-create it if they doubt it. A screenshot of a passing screen with no indication of
  what was entered proves less than one that captures the input too.
- **Self-describing filenames + comments** — I name the file for what it proves and pass a `comment`
  that ties it to a criterion: e.g. `attach 4217 over-balance-rejected.png "AC-2: transfer over
  source balance is rejected"`. Six months later, or at sprint-review, the attachment is legible on
  its own. (The attach path safely encodes the filename and comment, so descriptive
  names with spaces/punctuation are fine.)
- **Tied to a specific criterion or edge** — each piece of evidence maps to an acceptance criterion
  (the spec field's `## Acceptance Criteria` — concept #2) or a specific edge/regression I probed. Evidence that doesn't
  map to anything is noise; a criterion with no evidence is an unproven pass.
- **Both the pass and the guarded failure** — for a rejection/error-path criterion ("over-balance
  transfer is rejected"), the *proof* is a screenshot of the rejection happening, not of the happy
  path. I capture the surface *demonstrating the boundary holds*, because that's the criterion.
- **Enough, not everything** — I attach what substantiates the verdict, not every frame. A test-run
  log file plus one or two decisive surface screenshots per non-trivial criterion is the right
  weight; a hundred screenshots buries the signal.

**Worked example (generic).** For the "transfer between accounts" work-unit: I attach `test-run.txt`
(all unit/integration gates green), `concurrency-conservation.png` (the surface after two racing
transfers, balances still summing correctly — rank-1 evidence), and `over-balance-rejected.png` (the
rejection message on an over-balance attempt — the boundary criterion). Three attachments, each
tied to a ranked criterion, each self-describing. My verdict comment then references them by name.

## Reading evidence back

Reading an attachment *back* goes through the active backend's adapter (concept #12).
The tech-lead pulls my evidence this way during review; the `project-manager` pulls it
when assembling the `Sprints/Sprint-<n>-Review` durable-knowledge page. I own the attach; the
read-back is there for whoever weighs the verdict — I attach the proof, they inspect it.

## Where evidence lives, and where it doesn't

- **Evidence attaches to the work-item.** It is per-work-unit execution proof — transient state tied
  to this task's verification (work-items are transient execution state; the durable-knowledge store
  is current-truth — concept #9).
- **My verdict comment references it** by attachment name, so a reader goes from "pass on AC-2" to
  the exact screenshot in one hop.
- **I do not put evidence in the durable-knowledge store.** Worker-dispatch agents don't write it
  (concept #9); it's durable current-truth, not per-task proof. The durable-knowledge store records
  what the project *is*, not that one task passed.
- **The craft of evidence-collecting is role-craft** — it travels with me via `/drain` to this
  child. A durable lesson ("for mobile criteria, always attach the emulator screenshot *and* the
  device/orientation it ran on, because a reviewer can't reproduce the lease") generalizes into this
  file, never into a project's durable-knowledge store.

## Evidence checklist (per verdict)

- [ ] Test-gate output captured (all relevant gates, showing green — or the failure, on a fail
      verdict)
- [ ] A decisive surface screenshot per non-trivial criterion — web via preview/chrome-devtools,
      mobile via the emulator lease (see [`mobile-and-web-surfaces.md`](mobile-and-web-surfaces.md))
- [ ] Boundary/rejection criteria evidenced by the *guard holding*, not the happy path
- [ ] Each attachment self-describing (name + `comment` tie it to a criterion/edge)
- [ ] Attached per the active backend's adapter (concept #12) — no literal credential anywhere; the
      adapter handled the upload+link
- [ ] The verdict comment references every attachment by name; no unproven pass, no orphan evidence
- [ ] Read-back path confirmed available for the reviewer (per the active adapter, concept #12)
