# Scaffolder spec

A **scaffolder** is a team-scoped skill named `/create-new-project` that bootstraps a new project on the team's stack. This page defines the standard shape every scaffolder should follow.

::: tip Canonical source
The authoritative version of this spec lives at [`agentteamland/core/docs/scaffolder-spec.md`](https://github.com/agentteamland/core/blob/main/docs/scaffolder-spec.md). This page mirrors it; if they ever diverge, the `core` repo wins.
:::

## Why a spec?

Different teams will build very different scaffolders (a .NET + Docker stack vs. a Next.js + Sanity blog vs. a Python + Jupyter data project). But the **shape** of the UX should be consistent. Users who learn one team's scaffolder should feel at home using another.

## Skill location

Inside the team repo:

```
{team-repo}/skills/create-new-project/skill.md
```

And in `team.json`:

```json
{
  "skills": [
    { "name": "create-new-project", "description": "Scaffold a new project on this team's stack." }
  ]
}
```

When a user runs `atl install <team>`, the skill is copied into `.claude/skills/` automatically.

## The five phases

Every scaffolder MUST walk through these phases in order:

### Phase 1 — Gather information

Use the `AskUserQuestion` tool to collect what you need. Typical questions:

- Project name (skip if already passed as an argument)
- Which applications / modules / features to include
- Deployment targets, port offsets
- License choice
- Stack-specific toggles (SaaS? Multi-tenant? Framework version?)

**Rules:**

- Keep questions focused — 4–6 is plenty for most scaffolders.
- Provide sensible defaults; mark the top option as "Recommended".
- If the user passed an argument that answers a question, skip asking it.

### Phase 2 — Scaffold the project

Write every file the new project needs. For large scaffolds, delegate to specialized sub-agents in parallel — one per major concern (API, frontend, infra, mobile, etc.).

**Checklist:**

- Root files (`README.md`, `.gitignore`, language-specific lockfiles)
- Project configuration (`CLAUDE.md`, `.mcp.json`, `.env.example`)
- `.claude/` project directory (`agents/`, `skills/`, `rules/`, `docs/`, `brain-storms/`, `wiki/`, `journal/`, `backlog.md`)
- Source tree (everything that makes the app actually run)
- Container / deploy configuration if applicable

### Phase 3 — Build and start (optional)

If your stack has a build step (compile, `npm install`, `docker compose up`), run it:

- Compile; fail loudly on errors
- Start local services
- Wait for health checks to pass (30–60 seconds is typical)

Skip this phase for template-only scaffolders.

### Phase 4 — Verify (MANDATORY)

Invoke `/verify-system` as a `Skill` tool call. **This is non-negotiable.**

```
Skill(skill="verify-system")
```

The same team ships its own `/verify-system` that knows how to test the stack end-to-end. The scaffolder must:

1. Call the skill with the `Skill` tool (not inline bash).
2. Block on the result.
3. Fail visibly if verification does not pass.

### Phase 5 — Commit

After verification passes:

```bash
git init -b main
git add .
git commit -m "chore: initial scaffold via create-new-project"
```

Leave the remote unset — pushing to a remote is the user's decision.

## Output contract

At the end of a successful run, the user sees a **final report** with:

- Project path
- What was created (counts: files, services, agents, skills, rules)
- Verification result (✅ all passed / ❌ what failed)
- Next steps (how to open the project, how to run it, where the docs are)

## Failure modes

- **Phase 1 aborted** — user cancels mid-questionnaire. Clean exit, no files written.
- **Phase 2 error** — partial scaffold. The skill should roll back or leave the partial tree with a clear note.
- **Phase 3 build failure** — leave the scaffold in place; report the build error; do **not** proceed to Phase 4.
- **Phase 4 verification failure** — the scaffold stands; the user gets a clear list of what failed and guidance on how to fix.

## Onboarding UX

A first-time user who types `/create-new-project` before installing any team gets:

```
Skill not found: create-new-project
```

This is by design. `/create-new-project` is always stack-specific, so there is no generic global version. The user learns to install a team first, just like `npm create react-app` requires npm first.

Future work: `atl new-project <team> <name>` will dispatch straight into the team's scaffolder without requiring a pre-existing project directory.

## Related

- **[Creating a team](./creating-a-team)** — where scaffolder skills live.
- **[team.json](./team-json)** — how to register the skill.
- **[Concepts](/guide/concepts)** — what a skill is.
