# Creating a team

A team is a reusable bundle of AI **agents**, **skills**, and **rules** that anyone can install with `atl install`. This page walks through it end to end — from an empty directory to a catalogued team others can pull by handle.

## What a team is (and isn't)

A team is just a Git repository with a `team.json` file and some Markdown. When someone runs `atl install <handle>/<team>`, the CLI resolves the handle against the GitHub-backed catalog, fetches the repo as an ephemeral tarball, and copies the team's `agents/`, `skills/`, and `rules/` into the target's `.claude/` directory. That's it — no plugin system, no JavaScript runtime, no custom binaries. The whole thing is text files and copies.

A team can be:

- **One agent** (a single Markdown file with instructions Claude follows)
- **One or more skills** (slash commands that Claude can invoke)
- **Rules** (always-loaded instructions that shape behavior)
- **Any combination** of the above

A team becomes installable once it's **catalogued**: tag its public GitHub repo with the [`atl-team`](https://github.com/topics/atl-team) topic (or run [`atl publish`](/cli/publish) from it), and the generated index picks it up. From then on anyone can `atl install <handle>/<team>` — where the handle is the repo's GitHub owner. There is no registry repo and no submission PR.

---

## The full walk-through

Let's build a small real team from nothing. You'll create a `my-team` repo, add one agent, and get it ready to install.

### Step 1 — Create the team repo

```bash
mkdir ~/projects/my-team
cd ~/projects/my-team
git init -b main
```

The folder name doesn't have to match the team's catalog name — that's set in `team.json` below.

### Step 2 — Write `team.json`

This is the team's manifest. Minimum viable:

```json
{
  "schemaVersion": 1,
  "name": "my-team",
  "version": "0.1.0",
  "description": "Opinionated setup for Next.js + Tailwind projects.",
  "author": { "name": "Your Name", "url": "https://github.com/you" },
  "license": "MIT",
  "keywords": ["nextjs", "tailwind", "typescript"],
  "agents": [
    { "name": "web-agent", "description": "Reviews and builds Next.js pages." }
  ],
  "skills": [],
  "rules": []
}
```

Full field reference: [team.json](./team-json).

**Gotchas worth calling out:**

- `name` is the team's short-name. Once set, don't change it — users refer to it. Must be kebab-case (lowercase letters, digits, hyphens).
- `version` is SemVer (major.minor.patch). Bump it when you ship changes — `atl update` uses this to decide whether to pull.
- `author` is an **object**, not a string. At minimum `{ "name": "Your Name" }`. A plain string like `"author": "You"` fails to parse.
- `agents` is an array of **metadata**, not agent content. The actual agent Markdown lives under `agents/<name>/agent.md` (see Step 3).

### Step 3 — Add your agent

Every agent the `team.json` declares needs a directory under `agents/` using the **children pattern**:

```
my-team/
├── team.json
└── agents/
    └── web-agent/
        ├── agent.md              ← short: identity, scope, principles (<300 lines)
        └── children/             ← optional: deep-dive topics
            ├── routing.md
            ├── data-fetching.md
            └── testing.md
```

`agent.md` is the entry point — Claude reads it on every invocation. Keep it short. Put detailed patterns in `children/*.md`; the agent's `## Knowledge Base` section can link to them and Claude reads them on-demand.

Minimum `agent.md`:

```markdown
---
name: web-agent
description: "Reviews and builds Next.js pages."
---

# Web Agent

## Identity
I build and review Next.js pages for this project.

## Area of Responsibility (Positive List)
I ONLY touch:
- `app/` — Next.js App Router pages + layouts + routes
- `components/` — shared UI primitives
- `lib/` — data-fetching + utility functions

I do NOT touch:
- `api/` — that's the backend's concern
- Build config (`next.config.js`, `tsconfig.json`) without explicit approval

## Core Principles
1. Server components by default; client components only when interactive.
2. Co-locate styles with their component; no global CSS.
3. Loading UI for every async boundary.
```

That's a functioning agent. Add more detail in `children/` as the agent grows.

::: tip Deep dive
The children pattern is explained in [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md) and summarized in [Children + learnings](/guide/children-and-learnings). Key idea: `agent.md` stays short, topic-specific detail goes in `children/*.md` with one topic per file.
:::

### Step 4 — Commit

Commit your work — `atl` installs from a committed ref, not your working tree:

```bash
git add .
git commit -m "feat: initial team"
```

### Step 5 — Publish so others can install it

`atl install` resolves handles against the catalog, so a team has to be catalogued before anyone (including you) can install it by handle. Push the repo to a public GitHub repo and tag it with the `atl-team` topic:

```bash
# Push to a public GitHub repo under your account or org:
gh repo create you/my-team --public --source=. --push

# Tag it so the catalog indexes it:
gh repo edit you/my-team --add-topic atl-team
```

The index reindexes from public `atl-team`-tagged repos, so within a short window your team is discoverable as `you/my-team`.

### Step 6 — Install it

```bash
mkdir /tmp/demo-app && cd /tmp/demo-app
atl install you/my-team
# → atl: installed you/my-team@0.1.0 at project scope

atl list
# project:
#   you/my-team@0.1.0

ls -la .claude/agents/
# → web-agent.md
```

If the output matches, your team is installed. The agent is now available to Claude in `/tmp/demo-app/`.

The team installs at whatever scope its publisher declared (project by default — see the `scope` field in [team.json](./team-json)). Override per-install with `--global` or `--project`.

### Step 7 — Iterate

Edit files under `~/projects/my-team/`, bump the `version` in `team.json`, then commit and push. The catalog reindexes to the new version, and any project that has the team installed picks it up with `atl update`:

```bash
cd ~/projects/my-team
vim agents/web-agent/agent.md           # or any edit
# bump "version" in team.json, then:
git commit -am "tweak web-agent guidance"
git push

cd /tmp/demo-app
atl update
# → atl re-fetches the published version, refreshes unmodified copies
```

`atl update` refreshes copies you haven't modified and leaves your local edits in place.

### Step 8 — (Optional) Add skills and rules

**Skills** are slash commands. Each gets a `skills/<skill-name>/SKILL.md` with frontmatter:

```markdown
---
name: lint-page
description: "/lint-page <path> — run the project's lint config against a Next.js page file."
argument-hint: "<path-to-page>"
---

# /lint-page Skill

## Purpose
Lint a single Next.js page file using the project's ESLint + Prettier.

## Flow
1. Validate the path exists and matches `app/**/*.tsx` or `pages/**/*.tsx`.
2. Run `npm run lint -- --file <path>`.
3. Parse the output; if violations exist, print them with file:line:column citations.
4. Offer to auto-fix where safe.
```

Declare it in `team.json`:

```json
"skills": [
  { "name": "lint-page", "description": "/lint-page <path> — run lint against a page file." }
]
```

**Rules** are always-loaded Markdown files that shape Claude's behavior. Put them at `rules/<rule-name>.md`:

```markdown
# React 19 defaults

- Server components unless interactivity is needed
- Never use `"use client"` at the top of a shared lib
- `useActionState` replaces manual form-state boilerplate
```

Declare:

```json
"rules": [
  { "name": "react-19-defaults", "description": "Default to server components; avoid client boundary creep." }
]
```

After any change — agents, skills, or rules — bump the version, commit, and push, then `atl update` to pick it up.

### Step 9 — Where to go next

- **Add a scaffolder skill.** If your team is meant to spin up new projects, add a `/create-new-project` skill. See [Scaffolder spec](./scaffolder-spec).
- **Depend on another team.** If your team builds on someone else's, declare it under `dependencies` in `team.json` — `atl install` pulls the dependency alongside yours.
- **Earn the verified badge.** Teams reviewed by AgentTeamLand maintainers (and everything under `agentteamland/*`) show a `[verified]` badge in `atl search`. Its absence just means a team is self-published.

---

## Team layout reference

```
my-team/
├── team.json                      ← manifest (required)
├── README.md                      ← team docs (strongly recommended)
├── LICENSE                        ← usually MIT
│
├── agents/                        ← one dir per agent
│   ├── web-agent/
│   │   ├── agent.md              ← short: identity + scope + principles + knowledge-base index
│   │   └── children/             ← optional: deep-dive topics
│   │       ├── routing.md
│   │       ├── data-fetching.md
│   │       └── testing.md
│   └── backend-agent/
│       ├── agent.md
│       └── children/ ...
│
├── skills/                        ← one dir per skill
│   ├── lint-page/
│   │   └── SKILL.md              ← frontmatter (name, description, argument-hint) + body
│   └── run-e2e/
│       └── SKILL.md
│
└── rules/                         ← one .md per rule (flat, not dir)
    ├── react-19-defaults.md
    └── file-naming.md
```

Every file under `agents/`, `skills/`, and `rules/` that `team.json` lists becomes a copy in the consumer's `.claude/` when they install. Files not listed are ignored.

---

## How install works under the hood

When someone runs `atl install you/my-team`:

1. **Resolve.** The handle is looked up in the GitHub-backed catalog (generated from public `atl-team`-tagged repos). A team published from a monorepo subpath resolves to that subpath; a standalone team resolves to its own repo root.
2. **Fetch.** The team is downloaded as a ref-pinned HTTPS tarball into a temp directory — no `git` binary required. The temp directory is deleted after the install.
3. **Validate.** `atl` parses `team.json`, checks that it has a name, and confirms every declared agent/skill/rule actually exists on disk. Anything missing fails here.
4. **Write.** Agents, skills, and rules are **copied** into the scope's `.claude/` — `~/.claude` for a global install, `<project>/.claude` for a project install.
5. **Record.** A per-team manifest at `<layer>/.atl/installed/<handle>__<name>.json` records the source ref + per-file SHA-256 baselines that `atl update`'s auto-refresh and `atl doctor`'s integrity check rely on.

There is no persistent clone cache and no separate ATL asset store — team assets live under `.claude/`; ATL's bookkeeping (catalog cache, learning queue, pins, install manifests) lives under `~/.atl` and `<project>/.atl`.

---

## Common pitfalls

**`Error: agent source missing: .../agents/foo/agent.md`**
→ Your `team.json` lists `agents: [{"name": "foo"}]` but the filesystem has `agents/foo.md` (flat) where the children pattern expects `agents/foo/agent.md`. Match the declared assets to what's on disk.

**`Error: parse team.json: json: cannot unmarshal string into Go struct field TeamManifest.author`**
→ `author` must be an object, not a string. Change `"author": "You"` to `"author": { "name": "You" }`.

**Edited the team and ran `atl update`, no effect**
→ Did you commit, bump the version, and push? `atl update` pulls the published version, so uncommitted (or unpushed) edits don't flow. Commit + bump + push, then `atl update`.

**`atl install` says "team not found"**
→ The handle isn't in the catalog yet. The repo must be public and tagged with the `atl-team` topic (or have had `atl publish` run against it). Try `atl search` to confirm what's indexed.

**Want to delete a team cleanly**
→ `atl remove you/my-team` removes the team's manifest-recorded files at the scope (project by default; `--global` for the global layer) and prunes any now-empty directories.

---

## FAQ

**Do I have to push my team anywhere to use it?**
Yes. `atl install` resolves handles against the GitHub-backed catalog, so a team needs a public repo tagged `atl-team` (or one `atl publish` has run against) before it can be installed by handle.

**Can multiple teams coexist in one project?**
Yes — install as many as you like. Each team's items are copied into the shared `.claude/` directory. If two teams declare an item with the same name, the most recently installed one wins and `atl` prints a one-line warning.

**What Markdown format does atl use?**
Plain Markdown with optional YAML frontmatter. Claude Code's agent and skill format is supported natively.

**Can I version skills independently of the team?**
Not today. Versioning is team-level via `team.json`'s `version` field.

**Are there size limits?**
No hard limits. In practice team repos are under 10 MB. If you embed large binaries, mention it in the README so users know what they're pulling.

---

## See also

- [team.json field reference](./team-json)
- [Scaffolder spec](./scaffolder-spec) — adding `/create-new-project` skills
- [`atl install`](/cli/install) — full CLI reference
- [`atl publish`](/cli/publish) — circulate your team's accumulated gains upstream
- [Children + learnings](/guide/children-and-learnings) — the agent/skill knowledge-base pattern
