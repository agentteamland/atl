# `/consult`

Check what the project already recorded — with a query the agent writes itself, from the situation, rather than one derived from whatever sentence the user typed.

## The problem it exists for

ATL already had a read path: a `UserPromptSubmit` hook that embeds the user's prompt on every turn and injects the top five knowledge pages. Measured over two days on a real project, that channel is close to inert — and not for the reason it looks like.

Of **49 real prompts**, only **2–3** were questions the knowledge base could answer. **53%** were plain instructions ("merge it", "cut the release"). The hook therefore answers a question nobody asked about 95% of the time, and **88%** of the pages it offers are never opened.

The failure is not ranking quality, and it is not a reader who ignores good results. The question was simply never asked.

## The mechanism

The trigger belongs to the **agent's** situation, not to the grammar of the user's message. "Cut the release" contains no question and has a recorded checklist; the agent is the one who needs it, and the agent is the one who has to ask.

So `/consult` inverts both halves of the old design:

| | per-prompt hook | `/consult` |
|---|---|---|
| who writes the query | the harness, from the raw prompt | the agent, from the situation |
| when it runs | every turn | when the agent judges it needs the record |
| what it optimises | coverage | precision |

Both still run. Which one survives is a question for measurement, not for this page.

## Measured

Against a committed answer key (`.atl/eval/retrieval-answer-key.json` — 12 questions, both languages, scored on the real ranking path):

| query | recall@1 | recall@5 |
|---|---|---|
| the raw prompt, English (what the hook does) | 50% | 83% |
| the raw prompt, Turkish | 42% | 75% |
| **a query the model wrote** | **58%** | **100%** |
| a query the model wrote, **blind to `CLAUDE.md`** | 25% | 58% |

Two things fall out, and the second is the important one.

**The language gap closes at the source.** English and Turkish score *identically* with a model-written query, because the query is born in the corpus's language. Nothing has to be translated after the fact.

**The win is the index, not the model.** The same model in a directory with no `CLAUDE.md` scores 7/12 — *worse than doing nothing*. Blind, it invents plausible vocabulary that appears on no page (`failglob`, `nullopt`); with the knowledge map in context it produces the corpus's own words (`nomatch`, `tombstone`, `gitlink`). This is why the skill instructs the agent to take its query terms from the always-loaded knowledge map, and why shrinking that map is now a change to measure rather than a cleanup to schedule.

## The CLI half

```
atl retrieve --query "<terms>"
```

Searches the project's index with an explicit query and prints the matches. It shares the ranking path with the hook — same floors, same `topK` — so the two can never diverge and remain comparable.

Unlike the hook it is **not** silent on failure. Fail-open is a property of something that fires uninvited; a tool the agent chose to call reports what it found, including `No page … matched`, which is a real answer rather than an error.

## Keeping it measurable

A model-invoked mechanism depends on the model remembering to invoke it — and a turn that consulted nothing looks exactly like a turn that did not need to. That risk is why the measurement ships in the same change rather than after it:

- Every consult writes a `consulted` outcome to the fire log, with the pages it returned.
- A `Stop` hook (`atl retrieve turn-end`) records one line per completed turn, writing nothing to the model's context. It exists to give consultation an honest **denominator** — "in how many turns did the agent check the record?" — instead of reads-per-offered-page, which is meaningless when pages are offered on every prompt.

Both surface in [`atl retrieve stats`](/cli/retrieve#stats-the-channel-s-own-numbers):

```
turns           40
  consulted      9   22.5%   (agent wrote its own query)
    no match     2           (asked, corpus had nothing — a gap, not a miss)
```

`no match` is worth reading separately: it means the agent asked and the corpus was silent, which points at a hole in what has been written down rather than a failure to look.

## When it fires

The skill's own guidance: consult before asserting how something works, before proposing or choosing, before editing a file whose conventions are unverified, on an error that feels familiar, when an instruction names work with a recorded procedure, and when resuming a thread whose history matters.

And explicitly **not** otherwise. A constant channel stops being read — that is the measured failure this skill exists to correct, and it applies to the correction too.
