# ATL v2 — Platform restructure (decision)

**Decided 2026-06-12.** Closes the [`atl-v2`](../brain-storms/atl-v2.md) brainstorm. This is a structural + architectural decision; **no code ships under this doc alone** — it scopes the v2 rebuild. The brainstorm file holds the full reasoning, verbatim quotes, and per-item discussion; this doc is the distilled reference for implementation.

## Why v2

ATL v1 grew to 15 repos with a documented coordination tax (11-touchpoint new-rule checklist across 3 repos, git-tag↔team.json invariant bug class, 5-step post-merge ritual, the `sync/status/push-all` script family). Mesut: "it got scattered — too many repos and too much complexity." Two root causes: **repo topology** AND **conceptual surface area**. v2 attacks both.

## The north-star (acceptance criterion)

The user focuses only on their project's idea; ATL handles agent/skill/rule update + creation, learning, and all persistent memory (wiki/journal/agent-KB) automatically in the background. Every v2 decision serves this. Claude's honest assessment: ~80% sits solidly in the design; the remaining ~20% rides on closing the creation definition (done — reactive) and on implementation discipline around observability (the `atl doctor` commitment).

## Settled decisions (12 items + closure-gate additions)

**Topology & org**
1. **Org:** stay in `agentteamland` (CV/brand/link chain intact; archive pattern already established). No new org.
2. **Topology:** core monorepo `agentteamland/atl` (cli + core + first-party teams + docs + `.atl/`). 15 repos → 2 (`atl` + `.github`); `registry` survives only as a generated index.
3. **Repo name:** `atl` — product = repo = binary; Go module `github.com/agentteamland/atl`.
4. **Distribution:** GitHub Releases + `install.sh` (macOS/Linux) + `install.ps1` (Windows); update via `atl update`. brew/scoop channels + their repos die; goreleaser survives as release **builder** only.
5. **Migration:** greenfield `atl` built fresh (not lift-and-shift); `old-versions` flat snapshot for reference; v1 repos **archived** (not deleted — history/CV/reversibility preserved) + `.github` showcase-clean. Cut over at feature-parity.
6. **Workspace:** stays as the maintainer-meta hub (`.atl/` corporate memory + thin sync pulling only `atl` + `old-versions`); its 15-repo orchestrator role dies, its memory/control-room value survives simplified.

**Architecture spine**
7. **Scope axis (first-class):** two layers (user-global + project), isomorphic with Claude Code's own layering. Publisher declares default scope in the team manifest (`scope: global | project | both`); user overrides at install (`--global` / `--project`). Both-layer installs allowed; **project shadows global** on conflict. Granularity = team-level (single-agent teams are the idiom for global expert agents).
8. **Learning redesign:** silent in-conversation markers stay; rescan-and-filter dies. A hook-run step transfers markers into a **bbolt durable queue** (`~/.atl/queue.db`, per-project buckets) exactly once; processed items are **deleted** (re-report bug class structurally impossible). Generic multi-channel (`learning`, `profile-fact`, future). Pre-builds profile-team's drain substrate.
9. **Gain circulation (three-ring promote ladder):** `atl promote` lifts project-local gains to the user-global layer; `atl publish` pushes upstream (own team → re-publish; not-owned → best-effort "propose upstream" the owner accepts). Fan-out is **pull** (on `atl update`, unmodified project copies refresh from global, modified preserved — never push). Dissolves the "gains stay yours vs auto" binary: the user's own world circulates automatically (no gatekeeper); cross-author upstreaming stays a consenting handshake.
10. **Self-serve publish:** GitHub-backed index, zero hosted infra (option **C executed as A**: CLI verbs are the stable contract so a hosted backend can front it later without UX change). Identity = GitHub handle namespace (`mesut/my-team`); repo ownership = authorship. Open self-publish + optional maintainer-granted `verified` badge. Index supports monorepo-subpath sources (first-party) + standalone-repo sources (third-party). **registry-as-PR is dead.**
11. **CLI/Skill boundary:** deterministic plumbing (file I/O, git, SHA merge, queue transfer, install/update/promote/publish, index resolve, scope, hooks) in the **CLI**; LLM judgment (process a queue item → KB integration, profile inference, review, PR content) in **skills**. Verb inventory: `install / publish / promote / update / learnings status / doctor` + slimmed carryovers (`list / remove / config / setup-hooks / migrate`).
12. **Concept inventory:** `memory` **dissolves** into the user-global layer + profile data; `children` + `learnings` **unify** into one "agent knowledge base" (auto vs curated = lifecycle stage, not separate nouns); `docs-sync` **shrinks** to the EN→TR site flow (monorepo kills cross-repo drift); the knowledge model is documented **once** as two axes — (current-truth vs history) × (project vs agent vs user).

**Closure-gate additions (the reliability layer — Mesut's held-back points)**
- **Creation is reactive (v1):** a need arises → Claude authors the agent/skill/rule in-session → ATL persists + distributes (no fork+PR). Proactive auto-detection is out of v1 scope.
- **Doctor = automatic self-heal daemon, not a manual command.** Runs silently every session inside `atl session-start`: checks hooks/queue/loop, **self-heals what it can** (queue retry, fan-out re-run, hook re-bind), surfaces what it can't. `atl doctor` stays as on-demand diagnostic but is not the critical path. Safe repairs auto-apply; risky (content-mutating) repairs surface for confirmation.
- **Automation is mandatory, not opt-in:** hook installation is a required part of `atl install` (corrects v1's opt-in `setup-hooks` error). Residual blind spot (SessionStart hook never installed) minimized by mandatory install + opportunistic re-bind on any manual `atl` call.
- **In-session cadence (three-speed):** every-prompt cheap fan-out (generation-stamp guarded) + **5-10 min `atl tick`** (drain + doctor + fan-out) via **prompt-piggyback throttle** (UserPromptSubmit timestamp check — no daemon) + throttled/session-start network update. Interval tunable via `atl-config-system`. Closes the long-session compounding gap.

## Downstream effects

- **[`atl-scheduled-tasks`](../brain-storms/atl-scheduled-tasks.md): closed-absorbed.** ATL-internal scheduling solved by the in-session tick; generic team-scheduling is YAGNI (no concrete consumer) and, if ever needed, attaches as a declarative consumer-registry on the `atl tick` engine.
- **[`profile-team`](../brain-storms/profile-team.md):** resumes in the v2 landscape as a first-party team `atl/teams/profile-team`; its settled design already aligns with v2's queue + scope axis; the deferred profile-data-scope question (item 3.6) resolves under the new scope axis.
- **[`personal-advisory-team`](../brain-storms/personal-advisory-team.md):** resumes behind profile-team, in v2.

## Deferred to v2 implementation (not part of this decision)

- **CLAUDE.md content/structure design (global + local)** — a dedicated follow-up brainstorm opened once v2 implementation begins; inputs include Mesut's external-review suggestions, Claude's proactive suggestions, and a comparison against the current global/workspace CLAUDE.md + the item-7 marker-injection blocks.
