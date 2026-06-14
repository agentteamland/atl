# core/

Global rules + skills for the ATL platform (the v1 `agentteamland/core` content).

**Status:** porting in progress (decision doc, migration sequencing step 3) — rewritten to the simplified concept set (unified agent-KB, no `memory`, shrunk docs-sync).

**Rules — ported (7):** `agent-structure`, `branch-hygiene`, `communication-style`, `karpathy-guidelines`, `knowledge-system`, `learning-capture`, `skill-selection-discipline`. From v1's 9, three were dropped from user-facing core: `version-check` (the CLI does it now via `session-start`/`update`/`doctor`), and `team-repo-maintenance` + `docs-sync` (maintainer-side, not shipped to users).

**Skills — porting:** `skills/drain/` done; `save-learnings` folded into drain. `wiki` / `create-pr` / `create-code-diagram` pending.

Distribution wiring (how core rules + skills reach `~/.claude`) is part of a later content-port step.
