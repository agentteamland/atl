package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/transcript"
)

// sweepEnv points HOME and the working directory at temp dirs and seeds one
// transcript, returning the project root and the transcript path.
func sweepEnv(t *testing.T) (project, tpath string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	project = t.TempDir()
	t.Chdir(project)

	dir, err := transcript.ProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return project, filepath.Join(dir, "session.jsonl")
}

// declareChannel installs a minimal team manifest at the global layer declaring
// one capture channel, so a second channel is active alongside core's own.
func declareChannel(t *testing.T, name, drain, rule string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".atl", "installed")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"schemaVersion":3,"handle":"acme","name":"fake-team","channels":[` +
		`{"name":"` + name + `","drain":"` + drain + `","rule":"` + rule + `","describes":"facts"}]}`
	if err := os.WriteFile(filepath.Join(dir, "acme-"+name+".json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTurns(t *testing.T, path string, turns ...[2]string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, tn := range turns {
		if _, err := f.WriteString(`{"message":{"role":"` + tn[0] + `","content":"` + tn[1] + `"}}` + "\n"); err != nil {
			t.Fatal(err)
		}
	}
}

// runTranscript executes the command with the given flags and returns stdout+stderr.
func runTranscript(t *testing.T, flags map[string]string) (string, string) {
	t.Helper()
	cmd := learningsTranscriptCmd
	for _, name := range []string{"channel", "limit", "json"} {
		if f := cmd.Flags().Lookup(name); f != nil {
			if err := f.Value.Set(f.DefValue); err != nil {
				t.Fatal(err)
			}
		}
	}
	for k, v := range flags {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	t.Cleanup(func() { cmd.SetOut(nil); cmd.SetErr(nil) })
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("transcript: %v", err)
	}
	return out.String(), errb.String()
}

// The defect this closes: two consecutive sweeps used to read the same tail, so
// prose that arrived between them was read by neither. It must now be read by
// the second, exactly once.
func TestSweepResumesInsteadOfRereadingTheTail(t *testing.T) {
	_, tp := sweepEnv(t)
	writeTurns(t, tp, [2]string{"user", "before the baseline"})

	// First sweep baselines and reads the tail (the pre-cursor behavior).
	out, errs := runTranscript(t, map[string]string{"channel": "learning"})
	if !strings.Contains(out, "before the baseline") {
		t.Fatalf("first sweep did not read the tail:\n%s", out)
	}
	if !strings.Contains(errs, "first sweep for channel") {
		t.Fatalf("first sweep did not announce the baseline:\n%s", errs)
	}

	writeTurns(t, tp, [2]string{"assistant", "arrived after the baseline"})
	out, _ = runTranscript(t, map[string]string{"channel": "learning"})
	if !strings.Contains(out, "arrived after the baseline") {
		t.Fatalf("second sweep missed the new turn:\n%s", out)
	}
	if strings.Contains(out, "before the baseline") {
		t.Fatalf("second sweep re-read already-mined prose:\n%s", out)
	}

	// A third sweep with nothing appended yields nothing — the cursor held.
	out, _ = runTranscript(t, map[string]string{"channel": "learning"})
	if !strings.Contains(out, "no recent conversation flow") {
		t.Fatalf("third sweep re-emitted prose:\n%s", out)
	}
}

// The reason the cursor is per channel: /drain and /profile-drain sweep the same
// transcript for different channels and can run off the same turn. One advancing
// must not consume the other's unmined window.
func TestSweepCursorsAreIndependentPerChannel(t *testing.T) {
	project, tp := sweepEnv(t)
	declareChannel(t, "profile-fact", "/profile-drain", "profile-capture")
	writeTurns(t, tp, [2]string{"user", "baseline turn"})

	runTranscript(t, map[string]string{"channel": "learning"})     // baseline learning
	runTranscript(t, map[string]string{"channel": "profile-fact"}) // baseline profile-fact
	writeTurns(t, tp, [2]string{"user", "shared new material"})

	out, _ := runTranscript(t, map[string]string{"channel": "learning"})
	if !strings.Contains(out, "shared new material") {
		t.Fatalf("learning sweep missed the new turn:\n%s", out)
	}
	out, _ = runTranscript(t, map[string]string{"channel": "profile-fact"})
	if !strings.Contains(out, "shared new material") {
		t.Fatalf("the learning sweep consumed profile-fact's window:\n%s", out)
	}

	c, err := transcript.LoadCursor(project)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Known("learning") || !c.Known("profile-fact") {
		t.Fatal("both channels should have a recorded cursor")
	}
}

// A cursorless read is a look, not a sweep: it must never move a channel on.
func TestBareReadDoesNotAdvanceAnyCursor(t *testing.T) {
	project, tp := sweepEnv(t)
	writeTurns(t, tp, [2]string{"user", "baseline turn"})
	runTranscript(t, map[string]string{"channel": "learning"})

	writeTurns(t, tp, [2]string{"user", "unmined material"})
	if out, _ := runTranscript(t, nil); !strings.Contains(out, "unmined material") {
		t.Fatalf("bare read did not show recent prose:\n%s", out)
	}

	c, err := transcript.LoadCursor(project)
	if err != nil {
		t.Fatal(err)
	}
	off, _ := c.Offset("learning", "session.jsonl")
	info, err := os.Stat(tp)
	if err != nil {
		t.Fatal(err)
	}
	if off >= info.Size() {
		t.Fatal("a bare read advanced the learning cursor — an ad-hoc look must not consume a drain's window")
	}
	if out, _ := runTranscript(t, map[string]string{"channel": "learning"}); !strings.Contains(out, "unmined material") {
		t.Fatalf("the sweep lost material a bare read had shown:\n%s", out)
	}
}

// A transcript that appears after the baseline is a new session and must be
// mined from the start, not treated as already swept.
func TestSweepMinesATranscriptCreatedAfterTheBaseline(t *testing.T) {
	project, tp := sweepEnv(t)
	writeTurns(t, tp, [2]string{"user", "baseline turn"})
	runTranscript(t, map[string]string{"channel": "learning"})

	dir, err := transcript.ProjectDir(project)
	if err != nil {
		t.Fatal(err)
	}
	writeTurns(t, filepath.Join(dir, "later.jsonl"), [2]string{"user", "a whole new session"})
	out, _ := runTranscript(t, map[string]string{"channel": "learning"})
	if !strings.Contains(out, "a whole new session") {
		t.Fatalf("a transcript created after the baseline was skipped:\n%s", out)
	}
	_ = tp
}

// A truncated or replaced transcript must be re-read rather than skipped: an
// offset past the end is a corrupt position, and re-reading is the safe side.
func TestSweepRereadsATruncatedTranscript(t *testing.T) {
	_, tp := sweepEnv(t)
	writeTurns(t, tp, [2]string{"user", "original long turn that will be replaced"})
	runTranscript(t, map[string]string{"channel": "learning"})

	if err := os.WriteFile(tp, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTurns(t, tp, [2]string{"user", "short"})
	out, _ := runTranscript(t, map[string]string{"channel": "learning"})
	if !strings.Contains(out, "short") {
		t.Fatalf("a truncated transcript was skipped instead of re-read:\n%s", out)
	}
}

// A typo'd channel must fail loudly. The same reasoning as _enqueue's gate: a
// phantom channel would record a cursor no drain ever reads, so the sweep would
// look done while nothing was mined.
func TestSweepRejectsAnUndeclaredChannel(t *testing.T) {
	sweepEnv(t)
	cmd := learningsTranscriptCmd
	if err := cmd.Flags().Set("channel", "learnin"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("channel", "") })
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	t.Cleanup(func() { cmd.SetOut(nil); cmd.SetErr(nil) })
	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown channel") {
		t.Fatalf("a typo'd channel was accepted: err=%v", err)
	}
}
