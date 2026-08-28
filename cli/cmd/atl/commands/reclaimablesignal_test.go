package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
)

// THE SIGNAL MUST COUNT WHAT gc RECLAIMS, NEVER WHAT IT REFUSES.
//
// The defect this pins was not a wrong number. It was a signal naming an action that could
// not change what the signal reported: it counted gains, gc retains gains, so running the
// named remedy left the count exactly where it was. Measured on a real machine, `atl gc
// --apply` reclaimed the one genuine orphan, reported `nothing to reclaim`, and the next
// session still said four.
//
// A signal has to name a state somebody can move out of. That is the property under test
// here, and it is testable because the two sets are decidable: gc sweeps `!Tracked &&
// !Owned` and retains everything else.
//
// # Why both arms, and why the silent one is the load-bearing one
//
// The firing arm only shows the signal can speak. The SILENT arm is the whole fix — a gain
// present, and nothing said about it — and it is the arm that would have gone red before
// the change while every other check stayed green.
func TestTheReclaimableSignalCountsOnlyWhatGCWouldSweep(t *testing.T) {
	t.Run("silent when everything found is a gain gc will keep", func(t *testing.T) {
		root := newGCFixture(t, "agents/api/children/learned.md")

		out := captureStdout(t, func() { reclaimableSignal(root) })
		if out != "" {
			t.Fatalf("the signal spoke about a file gc retains, so running its own remedy "+
				"cannot clear it and it will say the same thing every session — got %q", out)
		}
	})

	t.Run("fires when there is something gc would actually reclaim", func(t *testing.T) {
		root := newGCFixture(t, "skills/from-a-removed-team/SKILL.md")

		out := captureStdout(t, func() { reclaimableSignal(root) })
		if !strings.Contains(out, "reclaimable item(s)") || !strings.Contains(out, "atl gc") {
			t.Fatalf("expected a signal naming the reclaimable count and the command that "+
				"clears it, got %q", out)
		}
		// The old wording is what made a retained gain read as rubbish. A regression to it
		// would be invisible to the two arms above, which only count.
		if strings.Contains(out, "orphaned file(s) beside installed units") {
			t.Errorf("the signal is using the wording that described gc's RETAINED set: %q", out)
		}
	})
}

// newGCFixture builds a project whose .claude tree holds one extra file, and returns its
// root.
//
// A manifest claiming `agents/api/agent.md` is installed first, which is what makes the
// two cases differ: a file under that same unit is a SIBLING GAIN and gc retains it, while
// a file under any other unit is wholly unowned and gc sweeps it. The distinction is
// structural rather than a flag, so the fixture has to build it rather than declare it —
// the first version of this test passed a bool called `owned` and created nothing, so both
// arms produced an unowned file and the silent arm failed against correct code.
//
// HOME is redirected because gc.Scan reads the global layer too, and a real one would put
// this machine's own accumulated files into a unit test's subject.
func newGCFixture(t *testing.T, rel string) string {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	m := &manifest.Manifest{
		Handle: "acme", Name: "team",
		Files: map[string]string{"agents/api/agent.md": "sha"},
	}
	if err := m.Write(filepath.Join(root, ".atl")); err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"agents/api/agent.md", rel} {
		abs := filepath.Join(root, ".claude", filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("# fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
