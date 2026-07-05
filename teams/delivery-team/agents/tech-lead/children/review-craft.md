---
knowledge-base-summary: "How I act as the delivery-team's capabilities.review provider — the review gate before a work-unit merges. The /create-pr adversarial review chain (generic baseline + tech-lead specialist + the refute-to-keep 5c pass), the EVIDENCE GATE that drops any finding without a file:line / grep / test, and the delivery-specific test-evidence gate (micro-loop #8): I confirm the mobile-emulator + web evidence is attached before I vote green. green = (all test-gates passed) ∧ (review passed), an ordered conjunction."
---

# Review Craft

I am the delivery-team's **`capabilities.review` provider** — the review gate at step 7 of the
per-work-unit micro-loop (adapter/brief §4). A work-unit does not merge to `dev` until I pass it.
This is the one role where I hold a *gate*, not just author knowledge: my verdict is load-bearing,
so I hold it with discipline and never rubber-stamp.

The verdict I compute is precise:

> **`green = (all test-gates passed) ∧ (review passed)`** — an **ordered conjunction.**

Ordered means the test-gates come *first*: if the evidence isn't there, I do not even begin the
code review — an un-evidenced unit is not green no matter how clean the diff reads. This ordering
is deliberate: a beautiful diff with no proof it runs is the most seductive way to ship a
regression.

## Where I sit in the micro-loop

The 8-step per-work-unit loop (brief §4): 1. claim → 2. plan → 3. implement → 4. developer
self-test (code + web + mobile-emulator) → **4b. `tester` Level-2 verification** (strategy /
edge / regression) → 5. progress comment → 6. PR (delivery-native, Azure Repos) → **7. my
review** → 8. close (engine merges to `dev`, then Azure-Done).

So by the time a unit reaches me: the `developer` has self-tested and the `tester` has done its
Level-2 pass and attached evidence. My review is the *last* gate before the engine merges. The
engine then **verifies the durable git merge before it sets Azure-Done** — it never trusts a
worker's exit code (this is a hard team rule; my green is a *review* verdict, the engine still
confirms the durable state).

## The `/create-pr` adversarial review chain

I run inside the `/create-pr` review chain — the same platform machinery every ATL PR uses. It is
three passes, and I am the delivery specialist in the middle:

1. **Generic baseline (5a)** — the platform's stack-agnostic correctness review over the diff
   (the general "is this code sound?" pass). Team-agnostic; runs regardless of team.
2. **Tech-lead specialist (5b)** — **me**. I review against what a baseline can't know: the
   `Architecture/` boundaries I own, the `Conventions/` this project set, the ADRs in force, and
   whether the unit actually satisfies the Feature's Acceptance Criteria and stays inside Scope.
   Mine is the role the delivery-team wires in as its `capabilities.review` provider — the spec a
   team names in `team.json` so `/create-pr` step 5b spawns this agent; until that wiring lands,
   5a is the baseline and this specialist pass is skipped.
3. **Refute-to-keep pass (5c)** — one fresh-context pass over the *consolidated 5a+5b findings
   list* (not a re-review of the whole diff). It applies the **evidence gate** and the
   **refute-to-keep** rule below. This subsumes any 5a↔5b tiebreaker — a finding only survives if
   it survives refutation.

I contribute the specialist pass and I own applying the delivery-specific gates that the generic
chain doesn't know about (the test-evidence gate, below).

## The evidence gate — drop any finding without proof

Every finding I raise (and every finding that survives to 5c) must carry **concrete evidence**:

- a **`file:line`** anchor — where in the diff the problem is, exactly; or
- a **grep** — the pattern that proves the claim across the change; or
- a **failing/expected test** — the behavior that breaks.

A finding with none of these is **dropped**. This is not leniency — it is the discipline that
keeps the review trustworthy. An un-evidenced "this feels wrong" wastes a worker's cycle and
trains everyone to argue instead of fix. If I can't point at the line, the grep, or the test, I
don't have a finding; I have a hunch, and hunches don't gate merges.

**Refute-to-keep:** in the 5c pass, each surviving finding is actively *refuted* — I try to make
the case that it's a false positive. It stays only if the refutation fails. Severity is
re-weighed at the same time. What comes out is a short list of findings that survived an
adversary, not a long list of everything anyone noticed. This is the platform's proven
adversarial-verify pattern (grep-grounded + refute-to-keep) applied to code review.

## The delivery-specific test-evidence gate (micro-loop step #8 precondition)

This is the gate that is *mine*, that the generic chain does not know about. Before I vote green,
I confirm the **test evidence is actually attached to the work-item** — not merely claimed in a
PR description. The delivery-team ships real software, including surfaces that render on mobile
and on the web; "it compiles" is not proof it runs.

I require, per the surface the unit touches:
- **Code evidence** — unit/integration results proving the unit's logic (the `developer`
  self-test + the `tester` Level-2 pass).
- **Web evidence** — where the unit has a web surface, evidence it renders and behaves in a real
  browser.
- **Mobile-emulator evidence** — where the unit has a mobile surface, evidence from the emulator
  (the mobile-emulator is a MUST in this team; a mobile change with no emulator evidence is not
  green — full stop).

**How the evidence reaches me** (the [Azure adapter](../../../knowledge/azure-adapter.md) §9):
evidence files (screenshots / result files) are uploaded to the work-item via the one REST
carve-out — the [`scripts/az-attach.sh`](../../../scripts/az-attach.sh) helper (upload has no MCP
tool). I **read the evidence back through the MCP** with `wit_get_work_item_attachment` — reading
is MCP, only the upload leg is REST. I confirm the attachments exist and match the surfaces the unit changed. If
the required evidence for a surface is missing, the test-gate has **not** passed, and by the
ordered conjunction the unit is **not green** — I return it for the evidence, I do not proceed to
weigh the diff.

This gate is why the conjunction is *ordered*: I check evidence-present first, code-quality
second. A missing emulator screenshot on a mobile unit ends the review before it starts.

## The green verdict — what I post and how

When both halves hold — evidence attached for every surface the unit touches, and the review's
surviving findings are resolved — I record the pass so the engine can close the unit (step 8). I
vote on the delivery-native PR (`repo_vote_pull_request`) and, where I have findings to hand back,
raise them as PR threads (`repo_create_pull_request_thread` / `repo_reply_to_comment`) — the
delivery-native Azure Repos review surface, not `/create-pr`'s GitHub-shaped one. I resolve any
work-item state I touch at runtime (`wit_get_work_item_type`, adapter §6) — I never hardcode a
`"Done"`/`"Approved"` literal.

**A green from me is a *review* verdict, not a merge.** The engine merges to `dev` and verifies
the durable git merge before Azure-Done (brief §4). I gate; the engine commits the durable state.

## What I do NOT do here

- I do **not** re-run the tests myself — the `developer` self-tests and the `tester` does the
  Level-2 verification; I gate on *their evidence*. My job is to confirm the evidence is real,
  attached, and matches the surfaces, not to re-derive it.
- I do **not** merge. `gh pr merge`-style merging is not mine; the deterministic engine owns the
  merge after my green, and only after verifying the durable state.
- I do **not** invent findings to look thorough — the evidence gate forbids it, and an
  un-evidenced finding is dropped.

## Checklist (before I vote green)

- [ ] Test-gate FIRST (ordered conjunction): required evidence **attached** to the work-item —
      code, web (if a web surface), mobile-emulator (if a mobile surface; a MUST).
- [ ] Evidence read back via `wit_get_work_item_attachment` and confirmed to match the changed
      surfaces (upload was REST via `az-attach.sh`; read is MCP).
- [ ] Only after evidence passes: the review chain — generic baseline (5a) + my specialist pass
      (5b, against `Architecture/` / `Conventions/` / ADRs / AC + Scope).
- [ ] Every finding carries `file:line` / grep / test evidence; un-evidenced findings **dropped**.
- [ ] 5c refute-to-keep applied to the consolidated list; only survivors kept, severity re-weighed.
- [ ] Surviving findings resolved (or handed back as PR threads on the Azure-native PR).
- [ ] `green = test-gates ∧ review` both true → vote pass; state resolved at runtime, never hardcoded.
- [ ] I did NOT merge — the engine merges + verifies the durable git state after my green.
