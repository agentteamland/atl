package skillcheck

import (
	"path/filepath"
	"strings"
	"testing"
)

// skillWithBody lays out one team skill whose SKILL.md carries body as its
// fenced shell block, and returns the teams dir.
func skillWithBody(t *testing.T, body string) string {
	t.Helper()
	teams := filepath.Join(t.TempDir(), "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","skills":[{"name":"ship"}]}`)
	write(t, filepath.Join(base, "skills/ship/SKILL.md"),
		"---\nname: ship\ndescription: \"ship it\"\n---\n# Ship\n\n```bash\n"+body+"```\n")
	return teams
}

func shellFindings(t *testing.T, body string) []Finding {
	t.Helper()
	var out []Finding
	for _, f := range RunAll(Input{TeamsDir: skillWithBody(t, body)}) {
		if f.Check == "shell" {
			out = append(out, f)
		}
	}
	return out
}

// The exact bite: an unmatched glob is literal in bash but FATAL under zsh's
// default nomatch, and the Bash tool that runs a skill body is zsh (atl#331).
// Verbatim shape of the body that wiped a backup.
func TestShellFlagsBareGlobInForList(t *testing.T) {
	f := shellFindings(t, `set -euo pipefail
DEST="$REPO_ROOT/profile-backup"
mkdir -p "$DEST"
for entry in "$SRC"/* "$SRC"/..?*; do
  [ -e "$entry" ] || continue
  cp -R "$entry" "$DEST/"
done
`)
	if len(f) != 1 {
		t.Fatalf("expected the bare-glob for-list to be flagged once, got %+v", f)
	}
	if !strings.Contains(f[0].Detail, "line 11") || !strings.Contains(f[0].Detail, "nomatch") {
		t.Fatalf("finding should name the line and the cause, got %q", f[0].Detail)
	}
}

// The escalation: the same glob after a `rm -rf` is not a harmless error, it is
// data loss — the deletion is done and no outcome marker is ever printed.
func TestShellEscalatesGlobAfterDestructive(t *testing.T) {
	f := shellFindings(t, `set -eu
rm -rf "$DEST"
mkdir -p "$DEST"
for entry in "$SRC"/*; do cp -R "$entry" "$DEST/"; done
`)
	if len(f) != 1 || !strings.Contains(f[0].Detail, "follows `rm -r` at line 9") {
		t.Fatalf("expected the destructive-ordering escalation naming line 9, got %+v", f)
	}
	// A trap is the author handling the abort — no escalation then.
	g := shellFindings(t, `set -eu
trap 'echo failed' EXIT
rm -rf "$DEST"
for entry in "$SRC"/*; do cp -R "$entry" "$DEST/"; done
`)
	if len(g) != 1 || strings.Contains(g[0].Detail, "rm -r") {
		t.Fatalf("a trap should suppress the escalation, got %+v", g)
	}
}

// `[ … ] && cmd` bites ONLY in tail position — verified in sh, bash and zsh:
// as the last statement of the script the whole script exits non-zero, and as
// the last statement of a function the caller aborts under set -e.
func TestShellFlagsTestAndInTailPosition(t *testing.T) {
	f := shellFindings(t, `set -eu
echo "all the work happened"
[ -f "$MARKER" ] && echo "and the marker is there"
`)
	if len(f) != 1 || !strings.Contains(f[0].Detail, "line 10") {
		t.Fatalf("expected the trailing test-and to be flagged at line 10, got %+v", f)
	}

	fn := shellFindings(t, `set -eu
report() {
  echo "done"
  [ -f "$MARKER" ] && echo "marker"
}
report
echo "never reached when the test is false"
`)
	if len(fn) != 1 {
		t.Fatalf("expected the function-tail test-and to be flagged, got %+v", fn)
	}
}

// The false-positive guard, and the reason the check is narrow: POSIX exempts
// every AND-OR member but the last from errexit, so these all run clean in sh,
// bash and zsh. Flagging them would make the check noise, and a noisy check
// gets disabled.
func TestShellIgnoresNonTailAndGuardedTestAnd(t *testing.T) {
	for name, body := range map[string]string{
		"mid-script":  "set -eu\n[ -f \"$M\" ] && echo hit\necho after\n",
		"loop body":   "set -eu\nfor x in a b; do\n  [ \"$x\" = z ] && echo hit\ndone\necho after\n",
		"if branch":   "set -eu\nif true; then\n  [ -f \"$M\" ] && echo hit\nfi\necho after\n",
		"|| fallback": "set -eu\necho work\n[ -f \"$M\" ] && echo hit || true\n",
		"no errexit":  "[ -f \"$M\" ] && echo hit\n",
		"|| guard":    "set -eu\n[ -d \"$SNAP\" ] || { echo none; exit 0; }\n",
	} {
		if f := shellFindings(t, body); len(f) != 0 {
			t.Errorf("%s must not be flagged (it is errexit-safe), got %+v", name, f)
		}
	}
}

// Prose and data are not code. A heredoc body, a comment and a quoted string all
// reach the shell as text — matching inside them is pure false positive.
func TestShellIgnoresCommentsHeredocsAndQuotedText(t *testing.T) {
	f := shellFindings(t, `set -eu
# A glob-and-skip loop would be shell-dependent: for entry in "$SRC"/* aborts in zsh.
cat > "$RC" <<'EOF'
for entry in *; do
  [ -f "$entry" ] && echo "$entry"
done
EOF
echo "Undo: rm -rf \"$GLOBAL\" && mv \"$SAFETY\" \"$GLOBAL\""
`)
	if len(f) != 0 {
		t.Fatalf("comments, heredoc bodies and quoted text must not be scanned, got %+v", f)
	}
}

// A glob the shell never expands (inside quotes, or inside a command
// substitution the subshell owns) is not the bug this check is about.
func TestShellIgnoresNonExpandedGlobs(t *testing.T) {
	f := shellFindings(t, `set -eu
for f in $(git diff --name-only -- '*.md'); do echo "$f"; done
for i in 1 2 3; do echo "$i"; done
for a in "$@"; do echo "$a"; done
`)
	if len(f) != 0 {
		t.Fatalf("no unquoted glob is expanded here, got %+v", f)
	}
}

// Only fenced sh/bash blocks are shell. A ```text or ```markdown block that
// happens to contain a for-loop is documentation.
func TestShellOnlyScansShellFences(t *testing.T) {
	teams := filepath.Join(t.TempDir(), "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","skills":[{"name":"ship"}]}`)
	write(t, filepath.Join(base, "skills/ship/SKILL.md"),
		"---\nname: ship\ndescription: \"x\"\n---\n\n```text\nfor entry in \"$SRC\"/*; do echo x; done\n```\n")
	if f := ShellBodies("", teams); len(f) != 0 {
		t.Fatalf("a non-shell fence must not be scanned, got %+v", f)
	}
}

// ...but every shell fence IS scanned. The label set is load-bearing: the whole
// point is that the fence is a LABEL, so a body written under ```sh or ```zsh
// runs in exactly the same shell as one under ```bash.
func TestShellScansEveryShellFenceLabel(t *testing.T) {
	for _, lang := range []string{"bash", "sh", "zsh", "shell"} {
		teams := filepath.Join(t.TempDir(), "teams")
		base := filepath.Join(teams, "demo")
		write(t, filepath.Join(base, "team.json"), `{"name":"demo","skills":[{"name":"ship"}]}`)
		write(t, filepath.Join(base, "skills/ship/SKILL.md"),
			"---\nname: ship\ndescription: \"x\"\n---\n\n```"+lang+"\nfor entry in \"$SRC\"/*; do echo x; done\n```\n")
		if f := ShellBodies("", teams); len(f) != 1 {
			t.Errorf("a ```%s fence must be scanned, got %+v", lang, f)
		}
	}
}

// A shell block QUOTED inside a wider fence is a doc showing the anti-pattern —
// exactly what an authoring guide for this very rule would write. Flagging it is
// the false positive that gets a gate disabled.
func TestShellIgnoresShellBlockQuotedInsideAWiderFence(t *testing.T) {
	teams := filepath.Join(t.TempDir(), "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","skills":[{"name":"ship"}]}`)
	write(t, filepath.Join(base, "skills/ship/SKILL.md"),
		"---\nname: ship\ndescription: \"x\"\n---\n\n````markdown\nNever write this:\n\n```bash\nfor entry in \"$SRC\"/*; do echo x; done\n```\n````\n")
	if f := ShellBodies("", teams); len(f) != 0 {
		t.Fatalf("a shell block quoted inside a wider fence must not be scanned, got %+v", f)
	}
}

// An unbalanced fence must not swallow the rest of the file: the skip above is
// only safe because it needs a real closer.
func TestShellStillScansAfterAnUnclosedNonShellFence(t *testing.T) {
	teams := filepath.Join(t.TempDir(), "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","skills":[{"name":"ship"}]}`)
	write(t, filepath.Join(base, "skills/ship/SKILL.md"),
		"---\nname: ship\ndescription: \"x\"\n---\n\n````markdown\nan unclosed wider fence\n\n```bash\nfor entry in \"$SRC\"/*; do echo x; done\n```\n")
	if f := ShellBodies("", teams); len(f) != 1 {
		t.Fatalf("an unclosed non-shell fence must not blind the scanner, got %+v", f)
	}
}

// A bare ``` is as likely a stray closer as an opener, so it must never be
// treated as one: an UNPAIRED one read as an opener would skip forward to the
// next fence line — the ```bash opener itself — and swallow the block behind it.
func TestShellStillScansAfterAnUntaggedFence(t *testing.T) {
	teams := filepath.Join(t.TempDir(), "teams")
	base := filepath.Join(teams, "demo")
	write(t, filepath.Join(base, "team.json"), `{"name":"demo","skills":[{"name":"ship"}]}`)
	write(t, filepath.Join(base, "skills/ship/SKILL.md"),
		"---\nname: ship\ndescription: \"x\"\n---\n\n```\n\n```bash\nfor entry in \"$SRC\"/*; do echo x; done\n```\n")
	if f := ShellBodies("", teams); len(f) != 1 {
		t.Fatalf("an unpaired untagged fence must not blind the scanner, got %+v", f)
	}
}

// A here-STRING is not a heredoc. Misreading `<<<` as one puts the scanner into
// skip-until-terminator mode and silently drops every line after it — the check
// going quiet with no sign at all.
func TestShellHereStringDoesNotSilenceTheRestOfTheBlock(t *testing.T) {
	f := shellFindings(t, `set -eu
tr a-z A-Z <<< ready
for entry in "$SRC"/*; do cp -R "$entry" "$DEST/"; done
`)
	if len(f) != 1 {
		t.Fatalf("a here-string must not blind the rest of the block, got %+v", f)
	}
}

// The escalation reads the code, not the prose about it. `rm -rf` named in a
// comment or quoted in an echo has deleted nothing — attaching a data-loss
// clause to it would be a fabricated consequence in a Fail-level message.
func TestShellEscalationIgnoresMentionsOfRmInCommentsAndStrings(t *testing.T) {
	f := shellFindings(t, `set -eu
# The copy below replaces the old rm -rf "$DEST" approach.
echo "Undo: rm -rf \"$GLOBAL\" && mv \"$SAFETY\" \"$GLOBAL\""
for entry in "$SRC"/*; do cp -R "$entry" "$DEST/"; done
`)
	if len(f) != 1 {
		t.Fatalf("expected exactly the glob finding, got %+v", f)
	}
	if strings.Contains(f[0].Detail, "rm -r") {
		t.Fatalf("a recursive rm only mentioned in a comment or a string must not escalate, got %q", f[0].Detail)
	}
}
