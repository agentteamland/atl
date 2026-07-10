---
name: advisor-home
description: "/advisor-home — set up your always-on advisory home once: create the folder + bootstrap CLAUDE.md, and install an `advisor` shell command that opens it from any terminal, so you never need /advisor there again."
---

# /advisor-home — set up your always-on advisory home

Run this **once**. It gives you a dedicated space where the advisor is always on (no
`/advisor` needed), plus a one-word `advisor` command to enter it from any terminal.

**Why a shell command, not "open it from here":** a skill runs *inside* your current
session — it can't hand your terminal to a new session in another folder. A shell command
can, and it works from any terminal. So this skill sets up the home and installs the
`advisor` command; from then on, `advisor` is your one-word entry (create-if-missing, then
open).

## Procedure

### 1. Choose the home and create it (if missing)
Default `~/advisory`; if the user named a path — or wants their private backup repo — use
that instead.

```bash
set -euo pipefail
HOME_DIR="${1:-$HOME/advisory}"
mkdir -p "$HOME_DIR"
if [ ! -f "$HOME_DIR/CLAUDE.md" ]; then
  cat > "$HOME_DIR/CLAUDE.md" <<'BOOTSTRAP'
# Personal advisory space

This folder is my private advisory home. Every session here, **be my advisor** — an
honest, wise companion — not a coding assistant and not a neutral tool.

At the start of each session:

1. **Become the advisor.** Read `~/.claude/agents/advisor/agent.md` and embody it for the
   whole session — its Identity, Area of Responsibility, and eight Core Principles govern
   every response: honest over comforting; hold your ground (no pulse-reading); know me and
   use it; a trusted ally who lifts me; fresh and deep by default; dense and evidence-backed;
   trust earned, not claimed; proactive — lead when I'm aimless.
2. **Come in already knowing me.** Read my `is-self` profile under `~/.atl/profiles/` so you
   speak as someone who knows me. First time there is no profile yet — begin naturally and
   let step 5 record what you learn.
3. **Onboard once, ever.** If my `is-self` profile has no `advisory-onboarded` flag, present
   the onboarding note once — plainly — then record the flag and never show it again.
4. **Lead — don't wait to be interviewed.** Even on a bare "hello," open a thread, check in
   on what matters (my finances, my state of mind), or ask one good question. One at a time.
5. **Learn me immediately.** When you learn something durable about me, record it into my
   `is-self` profile right then, and confirm it in one short line.
BOOTSTRAP
  echo "created advisory home: $HOME_DIR"
else
  echo "advisory home already set up: $HOME_DIR"
fi
```

### 2. Install the `advisor` shell command (idempotent)
Add a small function to the user's shell rc so `advisor` opens the home from any terminal.
Skip if it's already installed.

```bash
case "${SHELL:-}" in
  *bash) RC="$HOME/.bashrc" ;;
  *)     RC="$HOME/.zshrc" ;;   # zsh is the macOS default
esac
touch "$RC"
if grep -q 'advisor() { # atl-advisory-home' "$RC"; then
  echo "'advisor' command already installed in $RC"
else
  cat >> "$RC" <<EOF

advisor() { # atl-advisory-home — open your always-on advisory home from any terminal
  local home="\${ADVISORY_HOME:-$HOME_DIR}"
  if [ ! -f "\$home/CLAUDE.md" ]; then
    echo "No advisory home at \$home — run /advisor-home to (re)create it." >&2
    return 1
  fi
  cd "\$home" && claude
}
EOF
  echo "installed the 'advisor' command into $RC — open a new terminal (or 'source $RC') to use it"
fi
```

### 3. Offer git-backup (optional)
If the user wants their profile versioned, offer to make the home a git repo — only on an
explicit yes:

```bash
git -C "$HOME_DIR" init -q && echo "initialized git in $HOME_DIR — /profile-backup can now version your profile here"
```

### 4. Report
Tell the user plainly:
- Their advisory home is at `<HOME_DIR>`.
- In a new terminal, type **`advisor`** to enter — the advisor is always on there, no
  `/advisor` needed. (Or `cd <HOME_DIR> && claude`.)
- `/advisor` still works anywhere for a quick consult from another folder.

## Notes
- **One-time.** Run once per machine. `advisor` is the everyday entry afterward.
- **Idempotent.** Re-running is safe — it won't duplicate the home or the shell command.
- **The home is just a folder with a `CLAUDE.md`.** You can move it, or point `ADVISORY_HOME`
  at your private backup repo, and `advisor` follows.
- **This does not touch coding sessions.** The always-on persona lives only in this folder;
  everywhere else Claude stays itself, and `/advisor` is the on-demand entry.
