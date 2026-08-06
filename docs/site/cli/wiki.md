# `atl wiki`

Integrity checks for a project's `.atl/wiki` — the **current-truth** knowledge layer. The sibling of [`atl docs check`](/cli/docs) and [`atl skills check`](/cli/skills), over the knowledge base rather than the docs site or the assets.

Runs in any project that has a `.atl/wiki` directory. Outside one it does nothing and exits 0.

## Usage

```bash
atl wiki check   # validate reachability and link integrity across the wiki
```

## What it checks

| check | what it reports |
|---|---|
| `index-targets` | a `CLAUDE.md` link into `.atl/wiki` whose target is not on disk |
| `reachability` | a wiki page that `CLAUDE.md` does not link at all |
| `links` | a relative link inside a wiki page whose target is not on disk |

All three are **Fail**-level. There is no warning tier: every finding is a fact about a file, and a check that cannot be certain does not belong in this command.

## Why reachability is not tidiness

The `CLAUDE.md` index is what an agent's generated retrieval query draws its vocabulary from — measured at **100% recall@5 with the index loaded against 58% index-blind**. A page nothing links to is therefore not merely harder to find: it is outside the vocabulary the consult mechanism was measured to depend on, so it is unreachable by the very thing built to reach it.

## What it deliberately does NOT check

**Whether a page is still true.**

The docs site can be checked for drift because it has a ground truth to compare against — the code. A wiki page's claim (*"this repo is public"*, *"the store has no remote"*) has no single referent, so correctness is not mechanically decidable here.

That was measured rather than assumed. The obvious candidate — *does a repo path cited by a page still exist?* — runs at **76–90% false positive** over a real corpus, and the cause is structural rather than tunable: a knowledge corpus's genre is largely **documenting that something is dead**, which is byte-identical to **citing something dead**. The check fires hardest on the pages that did the correction work best.

So this command has two honest jobs: a **regression guard** on discoverability and link integrity, and a **prioritised target list** for the judgment half — which lives in the [`/observe`](/skills/observe) skill's current-truth lens, not here. That split is the CLI/Skill boundary: the CLI is deterministic, the skill is the judgment.

## Exit codes

- `0` — clean, or no `.atl/wiki` in this project
- non-zero — one or more findings, printed one per line
