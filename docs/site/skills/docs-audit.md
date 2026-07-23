# `/docs-audit`

Sweep the whole docs site for drift — the backstop of docs-sync v2. Change-time checks (the [`/create-pr`](/skills/create-pr) docs-impact pass + the deterministic CI gate) catch drift as it forms; `/docs-audit` is the net for what they miss: external-world link rot, a skipped docs-pass, accumulated drift.

It is **both** manually callable and auto-triggered. [`atl session-start`](/cli/setup-hooks) signals **"a full audit is due"** once doc-affecting commits (`docs/` `core/` `cli/`) have piled up since the last recorded audit, gated by a ~1-day runaway-guard — you run the skill then. The deterministic half is [`atl docs check`](/cli/docs); this skill adds the semantic judgment a machine can't.

## When to use it

- When `atl` reports **"a full audit is due — run /docs-audit"** at session start.
- Any time you want to sweep the whole docs site on purpose.

## How it works

### Deterministic first

The skill runs [`atl docs check`](/cli/docs) and fixes every **FAIL** (a missing page, an absent TR mirror, a stale install instruction) — mechanical, zero-false-positive. It never hand-audits what the CLI already proves.

Pass **`--external`** to also probe external-world links over HTTP (`atl docs check --external`) — slow and networked, so it's opt-in; the default sweep covers only in-site checks and prose-vs-code drift.

### Semantic, grep-grounded, adversarial

Then it sweeps each section of the site (`cli/`, `guide/`, `skills/`, `teams/`, …) and reads each page against the code, `SKILL.md`, or `team.json` it describes. Two guards keep the ~40% multi-agent-audit hallucination rate down:

- **Grep-grounded** — no drift is recorded without a verbatim source quote. A claim that can't be grounded in the code is dropped.
- **Adversarial** — each candidate finding is challenged ("is the prose actually right? is this deliberate historical contrast?") and dropped unless it survives.

Surviving fixes are applied to the EN page, the TR mirror is regenerated, and everything is opened as a **PR** for the maintainer to review — the autonomous draft, not a request for permission.

### Wider surfaces — the org profile and the demo

The docs site is the main target, but not the only surface the project publishes. When present, the sweep also audits, with the same grep-grounded discipline:

- **The org profile** — the landing page a newcomer sees first (the `.github` repo's `profile/README.md`, at github.com/agentteamland): retired teams still shown as "coming" or the shipped ones missing, a "full docs" link pointing at an archived site, removed install channels, stale command examples.
- **The demo** — the animated `assets/demo.gif`. It records real CLI output, so it goes stale on every release (version string, teams, commands); it's flagged when it no longer matches the current release, and **re-recorded after each release**.

### Records the audit

On completion the skill stamps the audit cursor (`atl docs check --record-audit`), which resets the runaway-guard so session-start won't re-signal for ~1 day.

## The CLI / Skill split

`/docs-audit` is the LLM half of docs-correctness. The deterministic half — coverage, parity, the stale-instruction denylist, link integrity — is [`atl docs check`](/cli/docs), which also runs as a CI gate on every PR. The skill never re-derives what the CLI proves; it spends LLM effort only on semantic prose-vs-code drift, grep-grounded and adversarially verified.

## Related

- [`atl docs`](/cli/docs) — the deterministic half this skill builds on.
- [`/create-pr`](/skills/create-pr) — its docs-impact pass is the change-time layer; `/docs-audit` is the backstop.
- [Skills overview](/skills/drain)

## Source

- Spec: [core/skills/docs-audit/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/docs-audit/SKILL.md)
