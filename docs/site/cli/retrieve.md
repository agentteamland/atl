# `atl retrieve`

The **read** side of ATL's knowledge loop. The write side is [learning capture](/guide/learning-marker-lifecycle) and [`/drain`](/skills/drain); this is what gets that knowledge back out.

It ranks the project's knowledge pages against a query — BM25 (lexical) fused with a local semantic embedder — and prints the top matches. It is **fail-open** everywhere: any error prints nothing and never blocks a prompt.

## Usage

```bash
atl retrieve --query "nomatch zsh glob"   # search with a query you wrote
atl retrieve stats                        # what the channel has actually done here
atl retrieve index                        # (re)build this project's index
atl retrieve warm                         # download the embedding model, warm the pipeline
```

Run bare with no subcommand, it is the `UserPromptSubmit` hook body: it reads the prompt from stdin and offers matches as context. You never type that form — [`atl setup-hooks`](/cli/setup-hooks) wires it.

## `--query` — the consult, and why it beats the hook

The hook embeds whatever sentence the user typed. `--query` lets the agent write the query itself, from its own situation. That difference is measured, on the same twelve questions:

| query | recall@5 |
|---|---|
| the user's raw prompt (what the hook does) | 83% English · 75% Turkish |
| written by the agent, **with the knowledge map in context** | **100%**, and both languages score the same |
| written by the agent, **without the map** | **58%** — worse than doing nothing |

The blind arm is the control, and it is the whole instruction: with the `CLAUDE.md` map loaded the agent produces the corpus's own vocabulary; without it, it invents plausible words the corpus has never used. So the query is written from the map, in terms, in English — see [`/consult`](/skills/consult), which is the skill that decides *when* to ask.

## `stats` — the channel's own numbers

The only place the retrieval channel is measurable. Every figure in the consult work came from here.

```
fires          440
  ranked       329   74.8%
    offered    319   72.5%   (1619 page refs)
    silent      10    2.3%
  suppressed   111   25.2%   (machine 104, short 7)
  translated    30    6.8%   (of the above — a query with no lexical hit)

turns           62
  consulted      8   12.9%   (agent wrote its own query)
    no match     1          (asked, corpus had nothing — a gap, not a miss)
```

Two lines deserve reading carefully:

- **`suppressed`** is healthy, not lost work. A quarter of prompts are machine-generated text or a few characters — questions nobody asked. A channel that cannot stay silent is not a signal.
- **`consulted`** counts turns where the agent asked *deliberately*. Its denominator is turns, which is why the `Stop` hook (`atl retrieve turn-end`) exists at all: without a turn marker the log records what was **offered** and nothing about what happened next.

## `index` and `warm`

`index` rebuilds the corpus index. You rarely need it — session-start rebuilds in the background when the corpus has moved, and a deleted page is detected now (a delete-only change used to leave the index serving a file that no longer exists).

A **cold** build is expensive — tens of minutes on a large corpus — and an incremental one is seconds, because chunks are keyed by `(path, text)`. `touch` therefore does not force a rebuild. A single-flight lock stops two builds racing; it is held by heartbeat rather than by a guess at how long a build takes.

`warm` downloads the embedding model and warms the pipeline, so the first real prompt does not pay for it.

## What it indexes

`.atl/{wiki,journal,docs}` and `.claude/{agents,knowledge,skills,backends,packs}`, plus `docs/` in a delivery-backed project and `teams/` in the repo that owns a team's **source**.

That last one is not the same as the installed copies: an installed copy can lag its source by a version, and in the owning repo the source is authoritative. Most sessions there are editing a team, and until it was added they could not retrieve what they were editing.

`.atl/brain-storms` is **deliberately excluded, permanently**: most brainstorms record rejected options by mandate, and a chunk split from its verdict reads exactly like a decision. Index the verdict, not the deliberation.

## Related

- [`/consult`](/skills/consult) — the skill that decides when to ask, and how to write the query
- [`atl setup-hooks`](/cli/setup-hooks) — wires the per-prompt hook and the turn marker
- [`atl wiki`](/cli/wiki) — integrity checks over the knowledge layer this searches
