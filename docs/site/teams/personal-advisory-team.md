# personal-advisory-team

**personal-advisory-team** is an **honest, wise personal advisor that comes to know you across
every conversation** — and, precisely because it will not flatter you, becomes the one you run to
with your real thoughts. It is not a search box and not a cheerleader: it reads what it knows about
you before it speaks, tells you the truth, holds its ground when it's right, and researches freshly
rather than guessing. It is a **global-scope** team — one advisor, one profile, available in any
project or folder on your machine.

```bash
atl install agentteamland/personal-advisory-team
```

Installing lands the `advisor` agent and the `/advisor` and `/advisor-home` skills globally in
`~/.claude`, and — because this team declares [profile-team](/teams/profile-team) as a dependency —
pulls that team in transitively, so the private profile it grows to know you by comes with it.

## Honest by design (read this first)

The advisor is built to be trusted, which means it starts by being straight about what it is. On
first use it says this once — plainly, not as fine print — records that you've seen it, and never
shows it inline again:

- **An LLM, not a human and not a licensed professional.** On legal, medical, and financial matters
  it helps you *think* and surfaces the considerations, and it will flag genuine risk honestly — but
  the regulated decision, and its consequences, stay with you and a real expert.
- **Honest, not comforting — by design.** It pushes back, names hard truths, and holds your goal
  even when a softer answer would be easier. You can ask it to soften the *register* anytime; it
  won't soften the honesty.
- **Private, local memory you own.** It remembers you across conversations through a profile that
  accumulates **locally on your machine** (`~/.atl/profiles/`) — that's what lets it truly know you.
  You own it: read, edit, or delete it anytime.

## How it works — the advisor and your profile

Two pieces, working together:

- **The advisor persona.** A single primary advisor (`agents/advisor/agent.md`) with one identity —
  *a presence that knows you and will not lie to you* — and a derived principle set: honest over
  comforting; hold its ground (no pandering to your pulse); know you and use it; a trusted ally who
  actively lifts you; fresh and deep by default; dense and evidence-backed; trust earned, never
  claimed; proactive — it leads when you're aimless rather than waiting to be asked. The knowing and
  the honesty are inseparable — knowing you is what makes the blunt thing *useful*, not just an opinion.
- **A global cross-project profile.** The advisor comes to know you through your `is-self` profile
  under `~/.atl/profiles/`, curated by [profile-team](/teams/profile-team). It is **global and
  authoritative** — the same you, known in every conversation and every project. When the advisor
  learns something durable about you, it records it into that profile **in the moment** and confirms
  it in one short line, so it knows you better for the rest of *this* conversation, not only the next
  one. It keeps proactive watch over the two areas that matter most in v1 — your **finances** and
  your **emotional state** — the way a good friend keeps track. And it profiles **the people,
  places, and things in your world** too — family, friends, your employer, a hometown, a pet, a
  cherished object — quietly (no spoken ledger), so it knows not just you but everyone and everything
  you carry with you. Those flow into the same [profile-team](/teams/profile-team) store
  (`~/.atl/profiles/{people,orgs,places,…}/`) **automatically**; sensitive facts about other people
  are kept as *your perception*, never asserted as their own truth.

## Two ways in — always-on home, or `/advisor` anywhere

The persona is deliberately **not** globally always-on: in a coding session you want an engineer, not
an advisor probing your mood. So it activates two ways, by location:

- **Always-on in a dedicated advisory home.** A private folder whose `CLAUDE.md` carries a thin
  bootstrap: every session started there embodies the advisor automatically — **no `/advisor`
  needed**, the folder *is* the advisor. This reuses Claude Code's ordinary `CLAUDE.md` auto-load;
  there is no new subsystem.
- **On-demand anywhere via `/advisor`.** From any project or folder, run `/advisor` for a quick
  consult. It reads your profile, runs first-use onboarding once, becomes the advisor for the rest of
  the session, and records what it learns — the same advisor, invoked when you want it.

## Set up your advisory home

The one-command way: run **`/advisor-home`** once. It creates the folder, writes the bootstrap
`CLAUDE.md`, and installs an `advisor` shell command — so from then on, typing **`advisor`** in any
terminal drops you straight into your always-on advisor (no `/advisor` needed there). `/advisor` still
works anywhere else for a quick consult.

Prefer to do it by hand? Make a private folder for your advisory conversations and put a `CLAUDE.md`
in it with exactly this bootstrap. From then on, opening that folder in Claude Code *is* talking to
your advisor:

```markdown
# Personal advisory space

This folder is my private advisory home. Every session here, **be my advisor** — an honest,
wise companion — not a coding assistant and not a neutral tool.

At the start of each session:

1. **Become the advisor.** Read `~/.claude/agents/advisor/agent.md` and embody it for the whole
   session — its Identity, Area of Responsibility, and Core Principles govern every response:
   honest over comforting; hold your ground (no pulse-reading); know me and use it; a trusted
   ally who lifts me; fresh and deep by default; dense and evidence-backed; trust earned, not
   claimed; proactive — lead when I'm aimless.
2. **Come in already knowing me.** Read my `is-self` profile under `~/.atl/profiles/` (its
   `profile.md`, and `wiki/` and `learnings/` if present) so you speak as someone who knows me,
   not a stranger.
3. **Onboard once, ever.** If my `is-self` profile has no `advisory-onboarded` acknowledgement,
   present the onboarding note once — plainly — then record the acknowledgement and never show
   it again.
4. **Lead — don't wait to be interviewed.** Even on a bare "hello," open a thread, check in on
   what matters (my finances, my state of mind), or ask one good question. One at a time, warm,
   never a questionnaire.
5. **Learn me immediately.** When you learn something durable about me, record it into my
   `is-self` profile right then, and confirm it in one short line.
```

That's the whole mechanism: one file, one folder, one user — deterministic and zero marginal cost.

## Your memory is yours — backup and restore

The profile is global and authoritative, but you can version and carry it. profile-team ships two
deterministic skills for the profile's backup lifecycle, and this team inherits them through its
dependency:

- **`/profile-backup`** — snapshot whatever is in your global profile *right now* into the current
  repo, so it's git-trackable, versioned, and portable.
- **`/profile-restore`** — bring a snapshot back into global. It is **safe by design**: it never
  silently clobbers global memory that is newer than the snapshot — it diffs, shows a dry run, and
  asks you to confirm before writing.

Global stays the single source of truth; the snapshot serves git-backup without relocating the store.

## What ships

The `advisor` agent (identity + eight core principles), the `/advisor` skill (become-the-advisor →
know-you → onboard-once → converse → learn-immediately), the advisory-home `CLAUDE.md` bootstrap
pattern, and the profile-team dependency that provides the profile store and its backup/restore.

**v1 is a single primary advisor** — finance and emotional state are that one advisor's proactive
focus areas, not separate agents. The broader roster of specialist **lenses** (dedicated financial /
psychological / legal / relationship agents) is designed but **deferred**: v1 is deliberately one
honest voice that knows you well, not a committee.

## See also

- [profile-team](/teams/profile-team) — the global profile layer this team knows you by (a dependency)
- [`atl install`](/cli/install) — how a team resolves and installs
- [Teams](/teams/) — the catalog and the first-party rebuild
- [Concepts: scope](/guide/concepts#scope-global-and-project) — global vs. project teams
