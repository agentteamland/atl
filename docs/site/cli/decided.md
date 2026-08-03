# `atl decided`

Search every surface that can hold a decision, and report honestly when there is nothing there.

```bash
atl decided "brief and stop"
```

## The question it answers

Before you write a rationale — *why the behaviour is this way, what it deliberately is not* — there is a question worth asking: **does the record actually hold the decision you are about to assert?**

It is easy to skip, because an authored justification reads as authoritative and nothing about it looks wrong. A claim that nobody ever decided survives review exactly as well as one that was.

`atl decided` is the cheapest possible way to check. One command, no state, no index — a direct search across the decision surfaces.

## What it searches

Widest-first, and only the roots that exist:

| root | what lives there |
|---|---|
| `.atl/docs` | settled decisions |
| `.atl/brain-storms` | the discussions that produced them, including rejected options |
| `.atl/wiki` · `.atl/journal` | current truth and history |
| `.claude` | installed team knowledge |
| `docs` · `cli` · `core` · `teams` | the site, and the code that implements decisions |

`cli/` is in the list deliberately. A decision is sometimes recorded **only in the code that implements it** — one real case was settled by a single line of a command's help text and by nothing else.

This is wider than what the per-prompt retrieval hook indexes, on purpose. Retrieval has to be selective; this is a direct search whose whole job is to be exhaustive, and where the expensive outcome is a false negative.

Vendored and generated trees (`node_modules`, `vendor`, `.git`, `worktrees`) are skipped.

## Reading a zero result

```
0 matches for "brief and stop"
searched: .atl/docs .atl/brain-storms .atl/wiki .claude docs cli core teams

That is a fact about this QUERY, not about the record. A decision written
in other words is indistinguishable from one that does not exist, so try
the synonyms before you draw a conclusion from this.
```

The empty case is the one this command exists for, which is why it prints the roots it searched instead of nothing at all.

But read the caveat literally. **A text search cannot tell you that no decision exists** — only that these words are not in these files. A term you did not think of looks exactly like a decision nobody made. Try the other wording before concluding.

The output also names only the roots that were actually present. A directory that does not exist is not searched and not claimed, so the zero result never implies coverage it did not have.

## Related

- [`atl learnings`](/cli/learnings) — the write side of the knowledge loop this searches.
- [Concepts](/guide/concepts) — how the decision surfaces relate to each other.

The per-prompt ranked surface (`atl retrieve`) is the selective counterpart: it may return nothing, and it is not exhaustive. This command is the one you reach for deliberately, when a zero result is the answer you need.
