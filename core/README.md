# core/

Global rules + skills for the ATL platform (the v1 `agentteamland/core` content).

**Status:** porting in progress. The full rule/skill set lands in a later v2 step (decision doc, migration sequencing step 3) — rewritten to the simplified concept set (unified agent-KB, no `memory`, shrunk docs-sync). Added directly in v2 so far:

- `rules/communication-style.md` — how agents/skills talk to the user (fluent language, gloss each technical term on first use, no jargon overload)
- `skills/drain/` — the learning-queue drain skill

Distribution wiring (how core rules reach `~/.claude`) is part of the content-port step.
