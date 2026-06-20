# Communication Style

How every ATL agent and skill talks to the user. Clear communication is part of the work, not a nicety — a correct answer the reader cannot follow has failed.

Sections 1–3 apply to user-facing output (chat, explanations, summaries). Section 4 governs what you write to disk.

## 1. Write fluently in the user's language

Match the user's language and write it well — correct grammar, natural phrasing, not stilted "translated-sounding" text. Technical identifiers (API names, code symbols, flags) may stay in their original form.

## 2. Explain each technical term on first use

The first time a response uses a technical term, add a short plain-language gloss in parentheses — e.g. "idempotent (running it twice changes nothing)". The reader may not know the term; one short clause keeps them with you. No need to re-explain it later in the same response.

## 3. Don't drown the reader in jargon

Use technical terms when they earn their place, but keep the balance. Two failure modes, both wrong:

- **Over-explaining** — talking down to the reader, spelling out the obvious.
- **Jargon pile-up** — stacking terms until the text is unreadable and the point is lost.

Readability and focus come first. If a sentence needs three glosses to parse, rewrite it with fewer terms.

## 4. Committed artifacts are English-only

You speak the user's language (§1), but everything you **commit** is English: code, comments, Markdown, docs, commit messages, identifiers. The sole exception is an explicit localization mirror — files under a `/tr/` path (the Turkish docs mirror), which are translations by design.

This is the deliberate inverse of §1. The conversation adapts to the reader; a committed artifact is a shared, public, long-lived record that must read the same to every contributor, now and years later — so the chat's language must never leak into a file (not even a user quote in a brainstorm: translate it). A pre-push scan (`scripts/scan-non-english.sh`) enforces this mechanically.
