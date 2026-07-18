# Command conventions — cobra structure, RunE, flags, honest output

A cobra CLI is a tree of `*cobra.Command` values wired together at startup. The craft here is
keeping each command **thin and honest**: it parses flags, delegates the real decision to a pure
helper, and returns an error rather than exiting. The through-line is that a command is glue —
almost nothing worth testing should live *inside* the closure.

## `RunE`, never `Run` — return errors, don't `os.Exit`

Give a command a `RunE func(cmd *cobra.Command, args []string) error`, not a `Run` that swallows the
error. Return failures; cobra prints them to stderr and sets a non-zero exit code in one place. Never
call `os.Exit` inside a command or a helper it calls — it can't be tested, and it bypasses cobra's
uniform error handling and any deferred cleanup.

```go
var thingCmd = &cobra.Command{
    Use:   "thing <name>",
    Short: "do the thing",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        force, _ := cmd.Flags().GetBool("force")
        if err := thingFlagConflict(force, /*…*/); err != nil {
            return err // fail loudly, before any work
        }
        return doThing(args[0], force)
    },
}
```

- Use cobra's `Args` validators (`cobra.ExactArgs(n)`, `cobra.MaximumNArgs`, `cobra.NoArgs`) instead
  of hand-checking `len(args)` — the usage error is then consistent and generated for you.
- Register flags in an `init()` on the command; read them in `RunE` with `cmd.Flags().GetBool/String`.

## Validate up front — reject contradictions, don't precedence-order them

When flags select mutually exclusive modes, reject more than one **before** doing any work — a clear
error beats silently running one and dropping the other, and it is critical when a branch is
irreversible (a destructive `--purge` must never win a silent precedence race over `--apply`). Put
the decision in a pure helper so it's unit-testable:

```go
// pure, no I/O → table-testable (see testing.md)
func thingFlagConflict(apply, undo, purge bool) error {
    if boolCount(apply, undo, purge) > 1 {
        return fmt.Errorf("conflicting modes: --apply, --undo and --purge are mutually exclusive")
    }
    return nil
}
```

Mirror any existing in-repo pattern (a repo that already validates `--global`/`--project` as mutually
exclusive is telling you the shape it wants) rather than inventing a new phrasing.

## Honest output through one pure helper

A user-facing line that can vary — success vs. offline vs. nothing-happened — is produced by a single
pure helper, so every variant is tested and the tool never asserts something it didn't do. Print the
helper's result; keep the `fmt.Println` in the command a one-liner.

```go
// the command:
fmt.Println(thingSummary(name, n, sc))

// the helper (pure → tested): only promises what actually happened
func thingSummary(name string, n int, sc scope.Scope) string {
    if n == 0 {
        return fmt.Sprintf("atl thing: %s — nothing to do (already in that state)", name)
    }
    return fmt.Sprintf("atl thing: %s — %d changed in %s scope", name, n, sc)
}
```

The anti-patterns this kills: a "reversible with `--undo`" promise printed when nothing was removed;
an "up to date" line printed when the network check couldn't even run. If the message can lie, route
it through a helper and test the truthful branch.

## Machine-readable output — `--json` parity

If sibling subcommands expose `--json`, a new sibling that emits data should too — tooling shouldn't
have to run a heavier command and re-derive. Add the flag, branch early in `RunE`, and marshal
through a pure helper that returns `"{}"`/`"[]"` for empty rather than `null`, so a caller always
gets a valid, stable shape (`encoding/json` sorts map keys — deterministic output tooling relies on).
