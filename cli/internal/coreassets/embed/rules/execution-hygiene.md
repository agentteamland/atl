# Execution hygiene

Universal execution defaults for any agent doing multi-step engineering work — inherited by every ATL team and every autonomous `claude -p` delivery worker (they load these from `~/.claude/rules/` exactly as an interactive session does). Three disciplines that keep execution clean. They **complement** the thinking discipline in [karpathy-guidelines](karpathy-guidelines.md) and the branch discipline in [branch-hygiene](branch-hygiene.md) — they do not duplicate them.

## 1. Subagent hygiene

Offload research, exploration, and parallel analysis to subagents — one focused task per subagent — and keep the main thread's context lean so the primary reasoning stays coherent.

A subagent **returns the conclusion, not the file dumps**: the parent keeps the finding or the answer, not the raw exploration noise the subagent waded through to get there. When you delegate a broad search or a multi-file read, you want the result back — not the transcript of everything it looked at.

- Reach for a subagent when answering means sweeping many files or directories and you only need the conclusion.
- One task per subagent — don't bundle unrelated investigations into a single call.
- This complements [karpathy-guidelines](karpathy-guidelines.md) §2 (Simplicity): a bloated working context is its own kind of complexity.

## 2. Autonomous bug-fix

Given a reproducible bug, an error, or a failing test, investigate and fix it yourself — read the log, the stack trace, the test, and the surrounding code; form a hypothesis; fix it; verify the fix — rather than bouncing it back to the user for hand-holding.

The user handing you a bug is handing you the goal, not asking you to confirm you understood it. Look at the evidence and solve it.

- The *method* is [karpathy-guidelines](karpathy-guidelines.md) §4 (Goal-Driven Execution): reproduce it with a test, then make the test pass. This rule is the *reflex* — don't wait to be walked through it.
- The one case to surface is genuine ambiguity (multiple plausible root causes, or no reproduction) — and even then, surface a diagnosis, not a request to be told what to do.

## 3. Atomic commits

Each commit is one coherent, verified unit of work. Commit as soon as a unit works rather than batching unrelated changes into one large commit — a reviewer, and a future `git bisect`, should be able to trace every changed line to a single intent.

**Boundary with [branch-hygiene](branch-hygiene.md):** branch-hygiene owns the *branch lifecycle* (don't develop on `main`, no orphan branches, delete merged branches). This rule owns *commit granularity* — what goes into one commit. The two compose: non-trivial work on its own branch (branch-hygiene), built out of atomic commits (here).

- Commit when a unit is verified working, not at arbitrary stopping points.
- Don't fold a refactor and a behavior change into one commit, or two unrelated fixes into one.
- Commit and push when asked, or when the project's own convention says so.
