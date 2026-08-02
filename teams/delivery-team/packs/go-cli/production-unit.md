# Production unit — adding a command

The thing this pack builds over and over is a **command**: a cobra command or subcommand, its
flags, the pure helper behind it, and the table test that pins that helper. This page is the
lifecycle for one, start to finish.

Two of its steps exist for one reason, and it is worth stating up front. The characteristic failure
of a scaffolded command is **not** a bad body — it is a **correct body that nothing calls**. A
command written but never mounted on its parent compiles, vets and tests completely green, and is
unreachable from the binary. Every gate this pack names passes. So the lifecycle does not end at
"tests pass"; it ends at "I ran the built binary and saw the command in its parent's help".

## 1. Decide before you write

Settle these first — renaming later means touching the file, the registration, the tests and any
docs page that names the command.

| Question | How to answer it |
|---|---|
| **New command, or a flag on an existing one?** | A flag if it modifies an existing action; a command if it *is* a different action. When in doubt, a flag — a command is a permanent surface. |
| **Top-level or under a group?** | Under a group when it belongs to a family that already exists. A group's members are discovered together in help, so grouping is the user's index. |
| **Which file?** | One file per top-level command or group, named after it. A subcommand lives in its group's file unless that file is already large. |
| **What is the pure decision inside it?** | Name it now. If you cannot state a decision worth testing, the command may be a thin wrapper and the test belongs a layer down. |
| **Does it write anything durable?** | If yes, say what and where — that determines both the verification in step 4 and whether the change needs a docs update. |

## 2. Scaffold

```go
var thingCmd = &cobra.Command{
	Use:   "thing <arg>",
	Short: "One line, lowercase, no trailing period — this is what `--help` lists",
	Long:  "A paragraph the user reads when they ask for help on this command specifically.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		// Validate contradictions BEFORE doing any work — especially when one branch
		// is irreversible. See command-conventions.md.
		if verbose && quietFlagAlsoSet(cmd) {
			return fmt.Errorf("--verbose and --quiet are mutually exclusive")
		}
		// The I/O lives here; the DECISION lives in a pure helper the test can reach.
		state, err := readState()
		if err != nil {
			return fmt.Errorf("read state: %w", err)
		}
		fmt.Println(describeState(state, verbose)) // <- the pure helper
		return nil
	},
}
```

`RunE`, never `Run`; return errors, never `os.Exit` inside a command. See
[command-conventions.md](command-conventions.md) for why, and
[code-conventions.md](code-conventions.md) for the pure-helper extraction.

## 3. Register it — the step that is easy to skip and invisible when skipped

A command only exists once it is mounted on a parent. Mount it in the same `init()` where its
siblings are mounted, in the file that owns that parent:

```go
func init() {
	thingCmd.Flags().Bool("verbose", false, "explain each step")
	rootCmd.AddCommand(thingCmd)          // top-level
	// or, for a subcommand:
	// groupCmd.AddCommand(thingCmd)
}
```

Read the parent's existing `init()` before writing yours and **match it**. If the siblings are
registered in one grouped `AddCommand(...)` call, add yours to that call rather than starting a
second one; if flags are declared before the mount, declare yours the same way. A registration that
looks unlike its neighbours is the one a reviewer skims past.

**Flags are part of registration.** A flag declared on the wrong command, or declared after the
parent has already been executed, silently does nothing. Declare flags on the command they belong
to, in the same `init()` that mounts it.

## 4. Verify the durable effect — run the binary, do not trust the test suite

The code-surface gate is necessary and **not sufficient** for this unit type. Build the binary and
observe the command where a user would find it:

```bash
go build ./... && go vet ./... && go test ./...   # necessary
go build -o /tmp/<tool> ./cmd/<tool>              # then actually look:
/tmp/<tool> --help            | grep '<thing>'    # top-level command is listed
/tmp/<tool> <group> --help    | grep '<thing>'    # or: subcommand is listed under its group
/tmp/<tool> <thing> --help                        # its own help renders, flags included
```

**The false green to name out loud:** a command that is written but never registered produces a
fully green `build` + `vet` + `test`. Its file compiles, its pure helper is covered by a passing
table test, and nothing anywhere fails — because no test drives the composed command tree, and
[testing.md](testing.md) deliberately steers away from building mocks that would. The binary is the
only place the omission is visible.

This is measured, not assumed. Dropping an unregistered command with a table-tested helper into a
real cobra CLI produces exactly:

```
build: PASS
vet:   PASS
ok      <module>/cmd/<tool>/commands    0.512s
$ <tool> --help | grep -c '<thing>'
0
```

Three green gates and an unreachable command. Nothing in the code surface can tell you.

Two more effects worth observing when they apply:

- **A flag** — `<tool> <thing> --help` must list it, and passing it must change behavior. A declared
  but unread flag also passes every gate.
- **A durable write** — if the command writes a file, a config key or a queue entry, run it and
  inspect the artifact. Do not infer the write from a return value.

Attach that output to the work-item as the Level-1 self-test evidence. "Tests passed" is not
evidence for this unit type; the help output is.

## 4b. Ship the test — the unit is not done without one

Step 4 proved the thing works **now**, by looking at it. This step is what keeps it working: a unit
is not complete until it ships a test covering the behaviour it added. The policy, identical across
packs, is [`testing-surfaces.md` §7](../../knowledge/testing-surfaces.md):

> **diff coverage >= 90%** of the lines this unit added or modified, **and** at least one test that
> goes RED when the change is reverted.

The second half is not ceremony. Coverage proves a line *ran*; it says nothing about whether anything
*checked* the result — a test that exercises the code and asserts nothing scores 100%. Confirming it
costs a minute: revert the change, run the test, see red, restore.

Coverage for this stack:

```bash
go test -coverprofile=cover.out ./... && go tool cover -func=cover.out
```

**The false green peculiar to this stack:** a table test that calls the command's `RunE` directly.
The body is covered, the assertions are real, and the command may still be unreachable because
nothing added it to the composed root. Step 3 is what catches that; this test does not, and its green
makes it look as though something did. Cover the body here, and let step 4 prove the wiring.

If 90% is genuinely unreachable because the new code is entangled with something untestable, record
the exception on the work-item — which lines, and why. Never a silent pass.

## 5. Common pitfalls

- **Registered on the wrong parent.** It runs, but under a name nobody expects. The help output in
  step 4 catches it; a test never will.
- **`os.Exit` inside `RunE`.** Bypasses cobra's central error handling and makes the path
  untestable. Return the error.
- **The decision left inside the closure.** If the branch you care about lives in `RunE`, no table
  test can reach it. Extract it, then test the extraction.
- **A user-facing line written twice.** Two call sites printing "up to date" drift. One pure helper,
  one string, every variant testable — see the honest-output convention in
  [pack.md](pack.md).
- **Docs not updated.** A new user-facing command or flag is a documented surface. If the project
  runs a docs gate, it will fail the PR; if it does not, the drift is silent. The brief names the
  gate when it applies.

## 6. Hand-off

The work-item is ready for the tester when: the code-surface gate is green, the **help output
showing the command in its parent** is attached, any durable write has been inspected rather than
inferred, and the docs surface is updated if the command is user-facing.

State plainly in the hand-off which of these you *observed* versus which you *assumed*. An
unverified claim is worse than an admitted gap — the tester's Level-2 pass is what gates the green,
and it can only check what it is told about.
