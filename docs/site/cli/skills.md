# `atl skills`

Deterministic, LLM-free **content-quality checks** for the platform's own skills, agents, and team manifests — the sibling of [`atl docs check`](/cli/docs). Where docs-check validates the docs *site* against the code, skills-check validates the **assets themselves**.

This is a **maintainer-side** gate that runs against the monorepo's `core/` and `teams/` trees. Outside the monorepo it does nothing and exits 0 (the pre-flight skip), so end-user sessions never see it.

## Usage

```bash
atl skills check                      # validate frontmatter, team.json consistency, agent-KB children, skill shell bodies
atl skills check --record-stocktake   # stamp HEAD as the last-stocktaken commit (run by /skill-stocktake after a sweep)
```

## What it checks

The three structural checks are **zero-false-positive by construction** — the finding is a fact about the file (a frontmatter key is either there or it isn't). The fourth, `shell`, is a deliberately narrow pattern match over named constructs rather than a structural fact; it is measured clean across every skill in the repo, and what it does *not* cover is stated below. All four are safe to gate a PR on:

| Check | What must hold |
|---|---|
| **frontmatter** | Every skill's `SKILL.md` and every agent's `agent.md` carries a `name` + `description` frontmatter block. |
| **manifest** | Each `team.json`'s `agents[]` / `skills[]` names match the on-disk directories — **both directions** (nothing declared-but-absent, nothing on-disk-but-undeclared). |
| **children** | Every **shipped** agent-KB child (`teams/<team>/agents/<x>/children/*.md`) declares a non-empty `knowledge-base-summary` frontmatter — the KB-rebuild contract. |
| **shell** | No fenced shell block in a `SKILL.md` uses one of two known bash-only constructs — an unmatched glob is **fatal** under zsh, which is the shell that actually runs a skill body. The exact scope, and its limits, are below. |

### Why a skill's shell body is checked

A skill body is **executable**: the agent pastes the block into whatever shell its Bash tool runs, so the ` ```bash ` fence is a **label, not the executor** (on macOS it is zsh). zsh's default `nomatch` makes an unmatched glob **fatal** where bash leaves it literal — and one such abort landed *after* a `rm -rf`, wiping a backup and printing none of the skill's outcome markers. It shipped green through every gate, because the defect lived in a bash snippet inside a Markdown file.

Three constructs are flagged:

1. **A bare glob in a `for` list** (`for entry in "$SRC"/*`) — iterate a `find` result, or copy with `cp -R "$SRC/."`, instead.
2. **`[ … ] && cmd` as the last statement** of the script or of a function, under `set -e` — a false test makes the whole unit exit non-zero. Use `if … then … fi`, or add `|| true`. (Only *tail* position is flagged: elsewhere POSIX exempts every AND-OR member but the last from errexit, so the idiom is safe and is not reported.)
3. **Destructive ordering** — either of the above sitting after a `rm -r` with no `trap`, which is what turns a portability nit into data loss.

This is a **pattern match, not a parse**. `sh -n`, `bash -n` and `zsh -n` all accept every construct above — `nomatch` is runtime behavior — so a syntax check would catch none of it. The value is entirely in recognising known-unportable constructs.

**What it does not cover.** It knows those constructs; it does not decide whether a body is shell-agnostic in general. The nearest sibling it misses is a bare glob *outside* a `for` list — `cp -R "$SRC"/* "$DEST/"` aborts under zsh in exactly the same way. That is a deliberate bound, not an oversight: widening the glob rule to every command word was measured across the corpus and **every** hit was a false positive (a trailing comment ending in `?`, a `case` pattern, JavaScript quoted inside a shell fence), and a gate that cries wolf is a gate that gets switched off. Write the shell-agnostic form anyway — the check is a net under the rule, not the rule itself.

### The installed layer is checked at session start, not here

`children` above walks the copies authored in a PR. But `/drain` writes agent-KB children into the **installed** layer — `<project>/.claude/agents/<agent>/children/` and `~/.claude/agents/<agent>/children/` — which CI cannot see, so gating on it would gate on something the runner has no copy of.

That half is a **session-start warning** instead, and it fires in any project (not just the monorepo): if a child there carries no `knowledge-base-summary`, `atl session-start` says so. The `## Knowledge Base` section is derived *from* that frontmatter, so a child without one leaves a rebuild nothing to derive its entry from. It reports the breach — it cannot make `/drain` write the frontmatter; that contract lives in the skill's own prose (which also forbids a rebuild from dropping such a child instead of backfilling it).

`atl skills check` exits non-zero on any failure, so it **gates every PR in CI** alongside the docs-drift gate. The judgment half — does a skill obey its own documented flow? do two skills overlap? — is the job of the companion [`/skill-stocktake`](/skills/skill-stocktake) skill (LLM), not this deterministic net. That split is the CLI/Skill boundary: deterministic checks here, grounded judgment in the skill.

`--record-stocktake` stamps HEAD as the last-stocktaken commit (in `~/.atl` state) when the run is free of failures — the `/skill-stocktake` skill calls it at the end of a sweep to reset the session-start "a stocktake is due" signal, the sibling of `atl rules scan --record`.

## Related

- [`/skill-stocktake`](/skills/skill-stocktake) — the LLM half: obedience + redundancy, grep-grounded, change-aware
- [`atl docs check`](/cli/docs) — the sibling gate: docs-site drift (this one is asset content-quality)
- [`atl doctor`](/cli/doctor) — the runtime self-heal (this is a build-time quality gate)
