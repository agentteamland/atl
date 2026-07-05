---
knowledge-base-summary: "How I capture and attach verification evidence: the scripts/az-attach.sh REST helper (adapter §9 — the ONE non-MCP op, upload-then-link with the worker's env PAT), what evidence a reviewer and the PO actually need (reproducible, self-describing, tied to a criterion), reading it back with wit_get_work_item_attachment, and why the proof — not my word — is what makes my green trustworthy downstream."
---

# Evidence Collection

My verdict is only as trustworthy as the proof behind it. A bare "pass" comment is a claim; an
attached screenshot of the web surface satisfying an acceptance criterion, or the test-run output
showing every gate green, is *evidence*. The `tech-lead` review (micro-loop step 7) and the human
PO (at sprint-review) both make decisions on my green — so the green has to be *inspectable*, not
taken on faith. This child is how I collect it and attach it correctly.

## The ONE REST carve-out — `scripts/az-attach.sh`

Uploading an attachment (a screenshot, a test-result file) is the **single** Azure operation with no
MCP tool (adapter §9 — the DRAFT once said "two"; Resolution #3 narrowed it to one). Every other
Azure call I make goes through the `azureDevOps` MCP; this one goes through a thin, uniform helper so
I never touch REST directly and the transport split stays hidden:

```
../../../scripts/az-attach.sh <work-item-id> <file> [comment]
```

From my worker context that relative path is
[`../../../scripts/az-attach.sh`](../../../scripts/az-attach.sh). What it does, and the contract I
rely on:

- **Two legs, hidden behind one call.** It `POST`s the bytes to `_apis/wit/attachments` (which
  returns an attachment URL), then links that URL onto the work-item as an `AttachedFile` relation
  (a JSON-Patch `add`). I see `attach(work-item, file)`; the two-step REST dance is inside the
  helper.
- **The PAT comes from my env, never the argv.** The helper reads the same `AZURE_DEVOPS_PAT` (and
  `AZURE_DEVOPS_ORG` / `AZURE_DEVOPS_PROJECT`) that `atl work dispatch` already put in my
  environment for the MCP. It rides Basic auth as a header, is never on the command line, and is
  never logged. I **never** write, echo, or pass a literal token — the same secret-hygiene the whole
  adapter enforces (config-and-methodology §2: the PAT is referenced by name, never stored).
- **It's worker-runnable and safe by construction.** I already have the PAT and the network, so I
  can run it directly. It validates its inputs (numeric work-item id, file exists, org/project/PAT
  present), builds and parses all JSON with `jq` (so a filename or comment with quotes/newlines/
  reserved chars can't corrupt the request), and retries 429/5xx with bounded backoff — the same
  resilience posture as the MCP callers (adapter §3).

The Go orchestrator stays zero-Azure through all of this — the carve-out lives in the *team*, run by
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
  its own. (The helper percent-encodes the filename and safely embeds the comment, so descriptive
  names with spaces/punctuation are fine.)
- **Tied to a specific criterion or edge** — each piece of evidence maps to an acceptance criterion
  (adapter §7 `## Acceptance Criteria`) or a specific edge/regression I probed. Evidence that doesn't
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

Reading an attachment *back* uses the MCP, not REST: `wit_get_work_item_attachment` (adapter §2/§9).
The tech-lead pulls my evidence this way during review; the `project-manager` pulls it (client-side)
when assembling the `Sprints/Sprint-<n>-Review` page. So the two directions split cleanly:
**upload = the `az-attach.sh` REST carve-out; read-back = the MCP.** I only own the upload leg; I
don't need REST to read.

## Where evidence lives, and where it doesn't

- **Evidence attaches to the work-item.** It is per-work-unit execution proof — transient state tied
  to this task's verification (adapter §8: work-items are transient execution state).
- **My verdict comment references it** by attachment name, so a reader goes from "pass on AC-2" to
  the exact screenshot in one hop.
- **I do not put evidence in the project wiki.** Worker-dispatch agents don't write the wiki
  (adapter §8); it's durable current-truth, not per-task proof. The wiki records what the project
  *is*, not that one task passed.
- **The craft of evidence-collecting is role-craft** — it travels with me via `/drain` to this
  child. A durable lesson ("for mobile criteria, always attach the emulator screenshot *and* the
  device/orientation it ran on, because a reviewer can't reproduce the lease") generalizes into this
  file, never into a project's wiki.

## Evidence checklist (per verdict)

- [ ] Test-gate output captured (all relevant gates, showing green — or the failure, on a fail
      verdict)
- [ ] A decisive surface screenshot per non-trivial criterion — web via preview/chrome-devtools,
      mobile via the emulator lease (see [`mobile-and-web-surfaces.md`](mobile-and-web-surfaces.md))
- [ ] Boundary/rejection criteria evidenced by the *guard holding*, not the happy path
- [ ] Each attachment self-describing (name + `comment` tie it to a criterion/edge)
- [ ] Attached via `scripts/az-attach.sh` (no literal PAT anywhere; helper handled the two-leg
      upload+link)
- [ ] The verdict comment references every attachment by name; no unproven pass, no orphan evidence
- [ ] Read-back path confirmed available for the reviewer (`wit_get_work_item_attachment`)
