package commands

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/queue"
)

// The check set is what actually reaches the user — a check nobody wires runs
// nowhere, and dropping one from a call site compiles clean and breaks no test
// in the doctor package. Assert the whole set here, once, since both `atl doctor`
// and the session-start hook now run exactly this list.
func TestPlatformChecksWiring(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	st, err := queue.Open(filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	got := map[string]bool{}
	for _, r := range doctor.Run(platformChecks(st, t.TempDir(), time.Now())) {
		got[r.Name] = true
	}
	for _, want := range []string{"queue-backlog", "tick-freshness", "asset-integrity", "hooks-bound", "brainstorm-pins"} {
		if !got[want] {
			t.Errorf("the %q check is not wired into the platform check set", want)
		}
	}
}
