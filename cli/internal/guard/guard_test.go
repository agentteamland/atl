package guard

import (
	"strings"
	"testing"
)

func TestCatastrophe(t *testing.T) {
	cases := []struct {
		name    string
		cmd     string
		blocked bool
	}{
		// force-push — blocked, with the --force-with-lease escape hatch allowed.
		{"force-push long", "git push --force origin main", true},
		{"force-push short", "git push -f origin main", true},
		{"force-push cluster", "git push origin main -fv", true},
		{"force-push with-lease allowed", "git push --force-with-lease origin main", false},
		{"plain push allowed", "git push origin main", false},
		{"force-push after subcommand", "git fetch && git push --force", true},

		// reset --hard — blocked; softer resets allowed.
		{"reset hard", "git reset --hard HEAD~1", true},
		{"reset hard origin", "git reset --hard origin/main", true},
		{"reset soft allowed", "git reset --soft HEAD~1", false},
		{"reset mixed allowed", "git reset HEAD", false},

		// git clean -f — blocked; dry-run allowed.
		{"clean force", "git clean -fd", true},
		{"clean force long", "git clean --force -d", true},
		{"clean force xdf", "git clean -xdf", true},
		{"clean dry-run allowed", "git clean -n", false},
		{"clean dry-run cluster allowed", "git clean -nfd", false},

		// destructive SQL — blocked, case-insensitive.
		{"drop table", `psql -c "DROP TABLE users"`, true},
		{"drop table lower", `mysql -e "drop table users"`, true},
		{"drop database", `psql -c "DROP DATABASE prod"`, true},
		{"truncate table", `psql -c "TRUNCATE TABLE events"`, true},
		{"select allowed", `psql -c "SELECT * FROM users"`, false},

		// --no-verify — blocked (gate bypass).
		{"commit no-verify", "git commit -m wip --no-verify", true},
		{"push no-verify", "git push --no-verify", true},
		{"commit allowed", "git commit -m wip", false},

		// Segment scoping — a flag in one command of a chain must not leak into
		// another command's decision (regression for the whole-command-scan bug).
		{"push then rm -f not force", "git push origin main && rm -f stale.log", false},
		{"push then tar -xzf not force", "git push origin main && tar -xzf x.tgz", false},
		{"make -f then push not force", "make -f Makefile && git push origin main", false},
		{"push after fetch still force", "git fetch && git push --force", true},
		{"reset --hard in echo not blocked", "git reset --soft && echo --hard", false},
		// Force-clean must NOT be disarmed by an unrelated dry-run flag elsewhere.
		{"clean -fd then make -n still blocked", "git clean -fd && make -n", true},
		{"clean -fd then echo -n still blocked", "git clean -fd && echo done -n", true},
		{"clean -fd exclude -enode still blocked", "git clean -fd -enode", true},
		{"clean -fdn is a dry run", "git clean -fdn", false},

		// Refspec force (`+ref`) — the canonical force form, must be caught.
		{"refspec force", "git push origin +main", true},
		{"refspec force full ref", "git push origin +refs/heads/main", true},

		// Destructive SQL only fires for an actual client invocation, word-bounded.
		{"drop table in commit msg allowed", `git commit -m "docs: explain DROP TABLE"`, false},
		{"drop table in grep allowed", `grep "drop table" schema.sql`, false},
		{"drop tablet not matched", `psql -c "SELECT 'drop tablet'"`, false},
		{"no-verify in echo allowed", `echo "use --no-verify carefully"`, false},

		// ordinary commands — allowed.
		{"build", "go build ./...", false},
		{"test", "go test ./...", false},
		{"rm dir (built-in territory) allowed", "rm -rf node_modules", false},
		{"ls", "ls -la", false},
		{"empty", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reason, blocked := Catastrophe(c.cmd)
			if blocked != c.blocked {
				t.Fatalf("Catastrophe(%q) blocked = %v, want %v (reason=%q)", c.cmd, blocked, c.blocked, reason)
			}
			if blocked && reason == "" {
				t.Errorf("Catastrophe(%q) blocked with empty reason", c.cmd)
			}
		})
	}
}

func TestDecide(t *testing.T) {
	exists := func(string) bool { return true }
	missing := func(string) bool { return false }
	firstAlways := func(string) bool { return true }
	firstNever := func(string) bool { return false }

	cases := []struct {
		name      string
		in        Input
		exists    func(string) bool
		firstEdit func(string) bool
		want      Action
	}{
		{
			name:   "bash catastrophe denies",
			in:     Input{ToolName: "Bash", ToolInput: ToolInput{Command: "git push --force"}},
			exists: exists, firstEdit: firstNever, want: Deny,
		},
		{
			name:   "bash safe noop",
			in:     Input{ToolName: "Bash", ToolInput: ToolInput{Command: "go test ./..."}},
			exists: exists, firstEdit: firstNever, want: Noop,
		},
		{
			name:   "first edit of existing file nudges",
			in:     Input{ToolName: "Edit", ToolInput: ToolInput{FilePath: "/x/main.go"}},
			exists: exists, firstEdit: firstAlways, want: Context,
		},
		{
			name:   "second edit of file is silent",
			in:     Input{ToolName: "Edit", ToolInput: ToolInput{FilePath: "/x/main.go"}},
			exists: exists, firstEdit: firstNever, want: Noop,
		},
		{
			name:   "multiedit existing file nudges",
			in:     Input{ToolName: "MultiEdit", ToolInput: ToolInput{FilePath: "/x/main.go"}},
			exists: exists, firstEdit: firstAlways, want: Context,
		},
		{
			name:   "write to existing file nudges",
			in:     Input{ToolName: "Write", ToolInput: ToolInput{FilePath: "/x/main.go"}},
			exists: exists, firstEdit: firstAlways, want: Context,
		},
		{
			name:   "write new file is exempt",
			in:     Input{ToolName: "Write", ToolInput: ToolInput{FilePath: "/x/new.go"}},
			exists: missing, firstEdit: firstAlways, want: Noop,
		},
		{
			name:   "edit without file path is noop",
			in:     Input{ToolName: "Edit", ToolInput: ToolInput{}},
			exists: exists, firstEdit: firstAlways, want: Noop,
		},
		{
			name:   "read tool ignored",
			in:     Input{ToolName: "Read", ToolInput: ToolInput{FilePath: "/x/main.go"}},
			exists: exists, firstEdit: firstAlways, want: Noop,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Decide(c.in, c.exists, c.firstEdit)
			if got.Action != c.want {
				t.Fatalf("Decide(%+v) action = %q, want %q", c.in, got.Action, c.want)
			}
			if got.Action == Context && !strings.Contains(got.Reason, "grep") {
				t.Errorf("context nudge missing grep guidance: %q", got.Reason)
			}
		})
	}
}

func TestFirstEditFuncPerSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // windows

	a := FirstEditFunc("sess-A")
	b := FirstEditFunc("sess-B")

	if !a("/x/main.go") {
		t.Fatal("first edit in session A should report true")
	}
	if a("/x/main.go") {
		t.Fatal("second edit of the same file in session A should report false")
	}
	if !a("/x/other.go") {
		t.Fatal("first edit of a different file in session A should report true")
	}
	if !b("/x/main.go") {
		t.Fatal("first edit of the same file in a different session B should report true")
	}
}

func TestFirstEditFuncEmptySessionSuppressed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	f := FirstEditFunc("")
	if f("/x/main.go") {
		t.Fatal("empty session id must suppress the nudge (return false)")
	}
}
