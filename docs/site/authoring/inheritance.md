# Inheritance

A team can extend another team. This is how you reuse an existing team's work without forking it.

## The basics

```json
{
  "name": "my-team",
  "version": "0.1.0",
  "extends": "software-project-team@^1.0.0"
}
```

That's all it takes. When someone runs `atl install my-team`:

1. `atl` resolves the parent (`software-project-team@^1.0.0`) from the registry.
2. The parent is installed (recursively — its own `extends` chain resolves too).
3. The child's own agents/skills/rules are installed on top.

The user ends up with every agent from the parent **and** every agent from the child.

## Override by name

If the child ships an agent with the same name as a parent's agent, the child wins. No merging of files — it's a full replace.

```json
{
  "extends": "software-project-team@^1.0.0",
  "agents": [
    { "name": "api-agent", "description": "Custom API conventions for our stack." }
  ]
}
```

At install time, `agents/api-agent.md` in the child overwrites the copy pointing to the parent's copy.

The same rule applies to skills and rules: name collision ⇒ child wins.

## Exclude what you don't want

```json
{
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"]
}
```

`ux-agent` from the parent is never copied in. The parent file still exists in the cache (the parent team is installed normally) — the child just elects not to surface it.

You can exclude agents, skills, or rules by name.

## Unlimited depth

Chains can go as deep as you like:

```
grandchild-team
  └─ extends  child-team@^1.0.0
      └─ extends  software-project-team@^1.0.0
```

At install time, the load order is **deepest ancestor first, current team last** — so the closer-to-you team wins every naming collision. Same mental model as class inheritance.

## Circular detection

`A extends B extends A` is caught at install time and fails with the full chain in the error message. No silent infinite loops.

## Single parent only

A team has at most one `extends`. No multiple inheritance, no diamond problem. If you need bits from two teams, install both — or `extends` one and declare the other under `dependencies`.

## Version constraints

The value of `extends` is `name` or `name@version-constraint`. Supported constraints:

| Syntax | Matches |
|---|---|
| `software-project-team` | latest |
| `software-project-team@^1.0.0` | `>=1.0.0 <2.0.0` — **recommended** |
| `software-project-team@~1.2.0` | `>=1.2.0 <1.3.0` |
| `software-project-team@1.2.3` | exactly `1.2.3` |

Caret is the default recommendation — you inherit patch fixes and new minors, but a breaking `2.0.0` won't sneak in unnoticed.

## Precedence, in order

When `atl install` resolves copies, later entries override earlier ones:

1. Ancestors (deepest first, nearest last)
2. Current team
3. Applied excludes (drop anything listed)

## A worked example

Parent `software-project-team@1.0.0`:

```
agents: api-agent, flutter-agent, react-agent, ux-agent, ...
skills: create-new-project, verify-system
rules:  commit-style
```

Child `my-team@0.1.0`:

```json
{
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"],
  "agents": [
    { "name": "api-agent", "description": "Override — my API conventions." },
    { "name": "analytics-agent", "description": "New — Mixpanel + GA instrumentation." }
  ]
}
```

After `atl install my-team`:

```
.claude/agents/
├── api-agent.md         → my-team/agents/api-agent.md         (override)
├── flutter-agent.md     → software-project-team/agents/flutter-agent.md
├── react-agent.md       → software-project-team/agents/react-agent.md
├── analytics-agent.md   → my-team/agents/analytics-agent.md   (new)
└── (no ux-agent)                                              (excluded)
```

Skills and rules follow the same process.

## When to extend vs fork

- **Extend** when you agree with 70 %+ of the parent's decisions and want upstream updates.
- **Fork** when you disagree with fundamental decisions (different language, different architecture) — at that point, extending is more cost than it saves.

## Next

- **[team.json reference](./team-json)** — every field.
- **[Creating a team](./creating-a-team)** — full walkthrough.
