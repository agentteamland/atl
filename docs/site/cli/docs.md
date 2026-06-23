# `atl docs`

Check the documentation site for drift against the code — the deterministic, LLM-free half of docs-correctness. The semantic half (does the prose still match what the code does?) is the `/docs-audit` skill; this command is everything a machine can verify with zero false positives.

## Usage

```bash
atl docs check [--external] [--record-audit]
```

`atl docs check` finds the docs site by walking up from the working directory to the repo that holds `docs/site/.vitepress`. Outside such a repo it does nothing and exits 0 — safe to run anywhere (the pre-flight skip). Inside one, it runs every deterministic check and exits non-zero if any **failure-level** finding is present (warnings never fail the command).

| Flag | Effect |
|---|---|
| `--external` | Also check that external links resolve over HTTP. Slow, networked, and sensitive to transient outages, so it is opt-in and warning-only. |
| `--record-audit` | When the run is free of failures, stamp the current commit as the last-audited one (`~/.atl/docs-audit-state.json`). The `/docs-audit` backstop reads this to know whether a fresh sweep is due. |

## What it checks

Each finding is `[FAIL|warn] check · page — detail`. Failures break the CI gate; warnings are surfaced but never fail.

- **`coverage`** (FAIL) — every CLI command has a `cli/<name>.md` page and every `cli/*.md` maps to a shipping command; likewise every core skill ↔ `skills/<name>.md`. The command list comes from the live CLI itself, so a new command with no page is caught — no hand-maintained inventory to keep current.
- **`parity`** (FAIL) — every English page has a Turkish mirror under `tr/`.
- **`tokens`** (FAIL) — a narrow denylist of stale *instructions*: install commands for the retired Homebrew / Scoop / winget channels, written as live steps. Deliberately instruction-only — a bare historical mention (explaining a channel *was* retired) is not flagged. Concept-rename drift in prose is the `/docs-audit` skill's job, not this one's.
- **`links`** (warn) — internal relative links that don't resolve to a file. VitePress's own build is the authority on dead links; this is the fast preview, so the check can run without Node.
- **`flags`** (warn) — every long flag of a command appears somewhere in its doc page.
- **`external`** (warn, `--external` only) — external URLs return `< 400`.

## The CLI / Skill split

`atl docs check` is deterministic and zero-false-positive by design: it reports only drift a machine can prove (a missing page, an absent mirror, a stale install step). Anything that needs judgement — "does this paragraph still describe what the code does?" — is out of scope here and belongs to the `/docs-audit` skill, which is grep-grounded and adversarially verified. This is the same CLI (deterministic) / Skill (LLM) boundary the rest of the platform follows.

## Examples

A clean site:

```bash
$ atl docs check
atl docs: clean
```

Drift found:

```bash
$ atl docs check
  [FAIL] coverage · cli/export.md — command `atl export` has no docs page
  [warn] flags · cli/install.md — flag --force not documented
atl: 1 documentation drift item(s), 1 warning(s) — fix before shipping
```

## Related

- [`atl doctor`](/cli/doctor) — the sibling deterministic self-heal, for installed assets rather than docs.
- [Release pipeline](/contributing/release-pipeline) — where the docs-drift CI gate runs `atl docs check`.
- [CLI overview](/cli/overview)
