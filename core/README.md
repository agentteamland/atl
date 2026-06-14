# core/

Global rules + skills for the ATL platform (the v1 `agentteamland/core` content).

**Status:** porting in progress (decision doc, migration sequencing step 3) — rewritten to the simplified concept set (unified agent-KB, no `memory`, shrunk docs-sync).

**Rules — ported (7):** `agent-structure`, `branch-hygiene`, `communication-style`, `karpathy-guidelines`, `knowledge-system`, `learning-capture`, `skill-selection-discipline`. From v1's 9, three were dropped from user-facing core: `version-check` (the CLI does it now via `session-start`/`update`/`doctor`), and `team-repo-maintenance` + `docs-sync` (maintainer-side, not shipped to users).

**Skills — ported (3):** `drain` (the learning-queue drain; absorbs v1's `save-learnings`), `create-pr` (ship-a-PR flow, v2-slimmed — `/drain` instead of save-learnings, docs-sync step dropped), `create-code-diagram`. Backlog: `wiki` (`/drain` already writes `.atl/wiki/`; init/query/lint are optional bench tools). Dropped: `save-learnings` (became drain), `docs-sync` (maintainer-side).

Distribution wiring (how core rules + skills reach `~/.claude`) is part of a later content-port step.
