# Creating a team

A team is a reusable bundle of AI **agents**, **skills**, and **rules** that you install into a project with `atl install`. This page walks through everything you need from empty directory to installed team — including the "I'm building this on my laptop, haven't pushed anywhere yet" case.

## What a team is (and isn't)

A team is just a git repository with a `team.json` file and some Markdown. When you run `atl install`, the CLI clones that repo into your local cache and copies its contents into `.claude/` inside your project. That's it — no plugin system, no JavaScript runtime, no custom binaries. The whole thing is text files and copies.

A team can be:

- **One agent** (a single Markdown file with instructions Claude follows)
- **One or more skills** (slash commands that Claude can invoke)
- **Rules** (globally-loaded instructions that shape behavior)
- **Any combination** of the above
- **An extension of another team** (inherits all its pieces, overrides or excludes specific items) — see [Inheritance](./inheritance)

You can install a team from:

1. The public **AgentTeamLand registry** (`atl install software-project-team`)
2. Any **GitHub repo** (`atl install agentteamland/starter-extended` or a full URL)
3. **Any git URL** — GitHub, GitLab, Bitbucket, self-hosted (public or private via SSH keys)
4. **Your local filesystem** (`atl install ./my-team` or an absolute path) — no remote required, just a `git init` in the team dir

The last option is what most people reach for first when authoring a team of their own. We'll cover it in detail below.

---

## Part 1 — The full walk-through

Let's build a small real team from nothing. You're going to create a `my-team` directory on your machine, add one agent, and install it into a test project — without ever pushing to a git server.

### Step 1 — Create the team directory

```bash
mkdir ~/projects/my-team
cd ~/projects/my-team

git init -b main                    # atl requires a git repo for the install pipeline
```

Any location works. The folder name doesn't have to match the team's registry name — that's set in `team.json` below.

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
  "rules": [],
  "extends": null,
  "excludes": []
}
```

Full field reference: [team.json](./team-json).

**Gotchas worth calling out:**

- `name` is the **registry short-name**. Once set, don't change it — users will refer to it. Must be kebab-case (lowercase letters, digits, hyphens).
- `version` is SemVer (major.minor.patch). Bump it when you publish changes, even for local iteration — `atl update` uses this to decide whether to pull.
- `author` is an **object**, not a string. At minimum `{ "name": "Your Name" }`. Registry submissions need `url` too.
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

> 📘 **Deep dive:** the children pattern is explained in `agentteamland/core`'s [agent-structure rule](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md). Key idea: `agent.md` stays short, topic-specific detail goes in `children/*.md` with one topic per file.

### Step 4 — Commit

atl's install pipeline uses `git clone` under the hood, so the team dir must be a git repo with at least one commit:

```bash
git add .
git commit -m "feat: initial team"
```

No remote is needed at this point.

### Step 5 — Install into a test project

```bash
mkdir /tmp/demo-app && cd /tmp/demo-app
atl install ~/projects/my-team         # absolute path ✓
# or:  atl install ../my-team          # relative path ✓
# or:  atl install file:///Users/you/projects/my-team   # explicit file:// URL ✓

atl list
# → my-team@0.1.0    (effective: 1 agents, 0 skills, 0 rules)

ls -la .claude/agents/
# → web-agent.md → ~/.claude/repos/agentteamland/my-team/agents/web-agent/agent.md
```

If the output matches, your team is installed. The agent is now available to Claude in `/tmp/demo-app/`.

> 💡 **Why "agentteamland" in the cache path for my private team?**  atl uses one shared local cache directory regardless of where a team came from. Your private team sits next to the public ones in `~/.claude/repos/agentteamland/`; the name in `team.json` is what distinguishes them. This does NOT mean your team got pushed or shared with the org — it's just a cache convention.

### Step 6 — Iterate

Edit files under `~/projects/my-team/`, commit (`atl` reads commits, not the working tree), then refresh the install:

```bash
cd ~/projects/my-team
vim agents/web-agent/agent.md           # or any edit
git commit -am "tweak web-agent guidance"

cd /tmp/demo-app
atl update my-team
# → atl re-pulls, refreshes copies
```

A round-trip takes ~1 second. You can iterate rapidly against the test project.

### Step 7 — (Optional) Add skills and rules

**Skills** are slash commands. Each gets a `skills/<skill-name>/skill.md` with frontmatter:

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

**Rules** are globally-loaded Markdown files that shape Claude's behavior. Put them at `rules/<rule-name>.md`:

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

After any change — agents, skills, or rules — commit + `atl update` in the test project to pick it up.

### Step 8 — Where to go next

Now that your team works locally:

- **Keep it private.** You can use it forever this way — just point `atl install` at your local path or push it to a private git server (GitHub private repo, GitLab, self-hosted Gitea). No submission required.
- **Share with a team.** Push to a private repo and give teammates the URL: `atl install git@github.com:your-org/your-team.git`. They'll install via SSH (no registry involvement).
- **Extend an existing team.** See [Inheritance](./inheritance) — declare `"extends": "software-project-team@^1.1.0"` to inherit that team's agents, then override or exclude specific items.
- **Submit to the public registry.** Only if you want others to discover it by short name. See [Registry submission](./registry-submission).
- **Add a scaffolder skill.** If your team is meant to spin up new projects, add a `/create-*` skill. See [Scaffolder spec](./scaffolder-spec).

---

## Part 2 — Install modes explained

All of these work. Pick the one that fits where your team lives.

### Registry (public, verified)

```bash
atl install software-project-team
```

The short name is looked up in the [public registry](https://github.com/agentteamland/registry). Works only for teams that have been submitted and verified.

### GitHub `owner/repo` shorthand

```bash
atl install agentteamland/starter-extended
```

Expands to `https://github.com/agentteamland/starter-extended.git`. Handy for public GitHub teams that aren't in the registry.

### Full git URL (any host, public or private)

```bash
atl install https://github.com/you/your-team.git      # public, HTTPS
atl install git@github.com:you/your-team.git          # private, SSH
atl install ssh://git@gitlab.com/you/your-team.git    # GitLab SSH
atl install https://gitea.example.com/you/team.git    # self-hosted Gitea
```

For private repos, git credentials / SSH keys must be set up on the host (atl shells out to `git clone`, which inherits your git config). No atl-specific authentication.

### Local filesystem (no remote needed)

```bash
atl install ~/projects/my-team                   # absolute or home path
atl install ./my-team                            # relative
atl install file:///Users/you/projects/my-team   # explicit file:// URL
```

Requires:
- The target is a directory that contains a `team.json` at its root
- The directory is a git repo with at least one commit (`git init` + `git add . && git commit` is enough)

No remote push is required. Ideal for local iteration, private teams, or teams you're not ready to share yet.

> ⚠️ **Version:** local-filesystem install requires `atl` ≥ 0.1.4. Earlier versions only accepted URLs.

---

## Part 3 — Team layout reference

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
│   └── api-agent/
│       ├── agent.md
│       └── children/ ...
│
├── skills/                        ← one dir per skill
│   ├── lint-page/
│   │   └── skill.md              ← frontmatter (name, description, argument-hint) + body
│   └── run-e2e/
│       └── skill.md
│
├── rules/                         ← one .md per rule (flat, not dir)
│   ├── react-19-defaults.md
│   └── file-naming.md
│
├── schemas/                       ← optional: JSON Schemas your team validates
│   └── config.schema.json
│
└── .github/workflows/
    └── validate.yml               ← schema-validate team.json on every push
```

Every file under `agents/`, `skills/`, and `rules/` that `team.json` lists becomes a copy in the consumer's `.claude/` when they install. Files not listed are ignored.

---

## Part 4 — Schema validation in CI (recommended)

Drop this into `.github/workflows/validate.yml` to catch malformed `team.json` before it breaks an install:

```yaml
name: Validate team.json
on:
  push:
  pull_request:

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - run: npm install -g ajv-cli

      - name: Download schema
        run: curl -sSfL https://raw.githubusercontent.com/agentteamland/core/main/schemas/team.schema.json -o team.schema.json

      - name: Validate
        run: ajv -s team.schema.json -d team.json --strict=false
```

This is optional but catches most `team.json` mistakes at PR time instead of at install time.

---

## Part 5 — Private team workflows

There are three flavors of "private team", and they each have a cleanest path:

### 🟢 (a) Fully local, just me

You're the only user. The team lives on your laptop.

```bash
# Author it:
mkdir ~/projects/my-team && cd ~/projects/my-team
git init -b main
vim team.json agents/main-agent/agent.md
git add . && git commit -m "init"

# Install into any project:
cd ~/projects/my-real-app
atl install ~/projects/my-team

# Iterate (edit → commit → refresh):
cd ~/projects/my-team
vim agents/main-agent/agent.md
git commit -am "tighter scope"

cd ~/projects/my-real-app
atl update my-team
```

That's the entire workflow. No remote involved.

### 🟡 (b) Shared with a few teammates

You want 2–10 people to install it, but NOT the public.

```bash
# Push to a private GitHub (or GitLab / Gitea) repo:
gh repo create you/your-team --private --source=. --push

# Teammates install via SSH (their SSH keys must be added to the repo):
atl install git@github.com:you/your-team.git

# Updates:
atl update your-team
```

No registry submission; no discoverability by strangers. SSH works on any modern git host.

### 🔵 (c) Internal / corp, behind a self-hosted git server

Same as (b) but the URL points at your self-hosted Git:

```bash
atl install git@git.your-company.com:platform/your-team.git
```

atl doesn't care about the host; `git clone` handles authentication via whatever SSH keys or credential helpers you've configured.

### 🟣 (d) Public to the world

Submit to the [AgentTeamLand registry](https://github.com/agentteamland/registry) so anyone can `atl install your-team`. See [Registry submission](./registry-submission).

---

## Part 6 — Common pitfalls

**`atl install ./my-team` says "could not find team"**
→ `./my-team` isn't recognized as a local path. Check: is it a directory? Does it contain `team.json`? Is `atl` version ≥ 0.1.4? (`atl --version`)

**`Error: agent source missing: .../agents/foo/agent.md`**
→ Your `team.json` lists `agents: [{"name": "foo"}]` but the filesystem has `agents/foo.md` (flat) instead of `agents/foo/agent.md` (children pattern). atl requires the children-pattern structure.

**`Error: parse team.json: json: cannot unmarshal string into Go struct field TeamManifest.author`**
→ `author` must be an object, not a string. Change `"author": "You"` to `"author": { "name": "You" }`.

**Edited the team and ran `atl update`, no effect**
→ Did you commit? `atl update` pulls via git, so uncommitted edits don't flow. Commit the team, then `atl update`.

**Want to delete a team cleanly**
→ `atl remove my-team` in the project removes copies from `.claude/` but keeps the cached repo. To nuke the cache too: `rm -rf ~/.claude/repos/agentteamland/my-team`.

**Team uses `extends` but I changed the parent locally; `atl install` pulls the wrong parent**
→ `extends` is resolved via the registry / URL spec written in the child's `team.json`. It won't read your local parent unless the child's `extends` points at a local path. Pin extending with a local path like `"extends": "/abs/path/to/parent-team"` for fully-local chains.

---

## Part 7 — FAQ

**Do I have to push my team anywhere to use it?**
No. Point `atl install` at your local path (atl ≥ 0.1.4). Commits in your local git repo are all that's required.

**Do I have to submit to the registry?**
No. The registry is only for discoverability via short names. Most private / internal teams never submit.

**Can multiple teams coexist in one project?**
Yes — `atl install a && atl install b && atl install c`. Each team's items are copied into the shared `.claude/` directory. If names collide, atl warns you (child team wins over parent; later installs override earlier).

**What Markdown format does atl use?**
Plain Markdown with optional YAML frontmatter. Claude's agent and skill format is supported natively.

**Can I version skills independently of the team?**
Not today. Versioning is team-level via `team.json.version`. Per-skill versioning is a proposed future feature.

**Are there size limits?**
No hard limits. In practice team repos are under 10 MB. If you embed large binaries (CSS bundles, SVG sprites), mention it in the README so users know what they're pulling.

---

## See also

- [team.json field reference](./team-json)
- [Inheritance](./inheritance) — extending an existing team
- [Scaffolder spec](./scaffolder-spec) — adding `/create-new-*` skills
- [Registry submission](./registry-submission) — publishing publicly
- [`atl install` command](../cli/install) — full CLI reference
- [Reference schema](../reference/schema) — JSON Schema for `team.json`
