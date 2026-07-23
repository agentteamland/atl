# `/observe`

The **proactive observer** — a force-multiplier on your own vigilance. Instead of *you* being the one who eventually notices "this only handles the self case, not the others" or "this file will silently truncate as it grows," `/observe` goes looking for that class of gap **first** and hands you a ranked, already-verified digest.

It has two dimensions, and by default runs both:

- **Trigger-watcher** — walks the deferred backlog's `_Trigger:_` conditions against what has actually accumulated (usage, recent work, the knowledge layer) and surfaces the **deferred items that are now ripe** — the "which one is ready?" watch, lifted off you.
- **Latent-gap auditor** (the load-bearing half) — a proactive audit for **untracked** gaps: shipped behavior that no longer matches design intent, things silently about to break as the project grows, decisions made but never shipped, and drift in your own global setup (`~/.atl/`).

Same discipline as [`/docs-audit`](/skills/docs-audit): findings are **grep-grounded** (no claim without a verbatim source quote) and **adversarially verified** (each candidate is challenged before it's kept). Multi-agent audits hallucinate ~40% of the time — that guard is what keeps the digest signal, not noise.

## When to run it

- When `atl` reports **"a proactive observer sweep is due — run /observe"** at session start.
- Any time you want to proactively check for ripe items or latent gaps.

Scope it with `--triggers-only` or `--gaps-only` to run just one dimension; the default is both.

## What it does

1. **Orient** — read the project's `CLAUDE.md`, recent `.atl/journal/`, and the deferral surface (`.atl/backlog.md`, or a delivery board when one is configured).
2. **Ripe triggers** — judge each deferred item's `_Trigger:_` against real evidence; an item is ripe only with a verbatim quote of the signal that fired it.
3. **Latent-gap sweep** — fan out finders across lenses (shipped-vs-designed, growth/scale, decided-but-unshipped, user setup), each grounding its findings with a source quote.
4. **Verify adversarially** — try to *refute* every candidate; keep only what survives, re-weigh severity.
5. **Surface** — a ranked digest of the verified flags, most-actionable first, each with its evidence and a suggested next step. An empty sweep is a valid result.
6. **Record** — `atl observe --record` stamps the cursor so the signal won't re-fire for ~1 day.

## Boundaries

- **It proposes, it doesn't auto-act.** A latent-gap finding often needs a decision (a brainstorm) — `/observe` surfaces it and lets you choose; it doesn't silently open PRs or create work items.
- **Honest bound.** It catches a *class* of gaps reliably — shipped-vs-designed mismatches, growth/scale risks, unshipped decisions, ripe triggers — enough to stop the recurring "you caught it first." It is **not** a guarantee to catch everything; it's a multiplier on your vigilance, not a replacement.
- **Advisor boundary.** It audits the advisor's *setup and output* from the outside; it never runs inside, or as, the advisor — that conversation stays pure.

## Related

- [`atl observe`](/cli/observe) — the deterministic half: the "sweep due?" signal and the cursor this skill stamps.
- [`/docs-audit`](/skills/docs-audit) — the same grep-grounded + adversarial discipline, aimed at the docs site.
