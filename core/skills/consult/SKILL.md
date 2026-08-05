---
name: consult
description: Check what this project already recorded before you answer, propose, or edit — you write the search query yourself from the situation, rather than waiting for the user to ask a question. Use when you are about to assert how something works, propose a design, pick between options, edit a file whose conventions you are not certain of, or hit an error that smells familiar. Also use when the user's message is an instruction rather than a question ("merge it", "cut the release") but the work it names has a recorded procedure.
---

# /consult — ask the record before you answer

The project's knowledge base is not a library you visit when someone asks a
question. It is what this project already learned, and most of the time nobody
will tell you to go and read it.

**Measured on this system:** of 49 real prompts over two days, only 2-3 were
questions the corpus could answer. 53% were plain instructions. The per-prompt
hook that embeds the user's sentence therefore answers a question nobody asked
about 95% of the time, and 88% of what it offers is never opened.

So the trigger is not the user's grammar. It is **your** situation.

## When to consult

Not on every turn — a constant channel stops being read, and this repo has that
measured. Consult when one of these is true:

- **You are about to assert how something works.** Especially in a sentence
  starting "X already does…", "we decided…", "the convention here is…".
- **You are about to propose or choose.** A design, an approach, one option over
  another. The most expensive failures in this repo are proposals refuted by a
  page that already existed.
- **You are about to edit a file whose conventions you have not verified.**
- **You hit an error that feels like it has been hit before.** It usually has;
  most of the corpus is "this broke and here is why".
- **The instruction names work with a procedure** — a release, a merge, a sweep,
  an onboarding. "Cut the release" carries no question and has a checklist.
- **You are resuming**, and the thread's history matters more than its last message.

And a real permission, because a rule with no off-switch gets ignored wholesale:
**if none of those is true, do not consult.** A mechanical acknowledgement, a
status question about work in flight, a formatting fix — the record has nothing
to say and asking it is noise you are training yourself to skip.

## How to write the query — this is the part that decides the outcome

Run:

```
atl retrieve --query "<your terms>"
```

**Write the query in the vocabulary of the knowledge base, not in the user's
words.** This is not a style preference; it is the whole mechanism, and it is
measured. Same 12 questions, the real ranking path:

| query | recall@5 |
|---|---|
| the user's raw sentence (what the hook does) | 83% English · 75% Turkish |
| a query you wrote, **with the knowledge map in context** | **100%**, and the two languages score identically |
| a query you wrote, **without the map** | **58%** — worse than doing nothing |

The blind arm is the control, and it is the instruction: with the map you
produce `nomatch`, `tombstone`, `gitlink`, `generatedAt`; without it you invent
`failglob`, `nullopt`, `backup verification` — plausible, and absent from every
page. **The knowledge map in `CLAUDE.md` is already in your context. Read the
entry titles and annotations, take their words, and put those in the query.**

Three rules that fall out of that:

1. **Terms, not a sentence.** The index scores terms; grammar earns nothing.
2. **Keep identifiers exactly as written** — file names, commands, flags,
   symbols, error strings. They are the strongest signal there is.
3. **English, always** — the corpus is English by rule. If the conversation is in
   another language, the query is still English. This is why a query you write
   scores the same in both languages: it is born in the corpus's language, so
   there is no gap left to bridge.

## What comes back, and what to do with it

Up to five pages, each with a title, a path and an excerpt. Then:

- **Open the page** when its title matches what you are about to claim or decide.
  An excerpt is a pointer, not the page — the reasoning that would change your
  mind is in the body.
- **Say what you found, in one line**, when it changed your answer. Not for
  bookkeeping: the user cannot see this happen, and a memory that is never
  demonstrated is indistinguishable from one that does not exist.
- **`No page … matched`** is a real result, not a failure. It means the corpus is
  silent here — so either you are the first to hit this, or it is worth recording
  once you have solved it. Do not read that as permission to guess confidently.

## What this does not replace

The per-prompt hook still runs and still injects pages uninvited. The two are
different instruments — it guesses from the user's sentence, you ask from the
situation — and which one survives is a question for measurement, not for this
file. If the hook already surfaced the page you needed, you do not need to ask
again.
