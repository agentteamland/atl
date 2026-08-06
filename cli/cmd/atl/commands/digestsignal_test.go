package commands

import (
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/digest"
)

// The signal's contract, and the reason the digest is not just a file: it must
// name the ACTION and cite the RULE that owns the response — the same two
// properties the sweep signals are pinned on, because a signal that names
// neither is what left four sweeps waiting for a human in the first place.
func TestDigestSignalNamesTheActionAndCitesTheRule(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if _, err := digest.Add(root, digest.Finding{Sweep: "observe", Title: "a latent gap"}); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() { digestSessionSignal(root) })

	for _, want := range []string{"atl digest", "sweep-dispatch rule"} {
		if !strings.Contains(out, want) {
			t.Errorf("signal %q does not contain %q", out, want)
		}
	}
	if !strings.Contains(out, "1 sweep finding") {
		t.Errorf("signal %q does not report the count", out)
	}
}

// And the property that keeps it from becoming the noise it replaced: the
// signal carries the COUNT, never the findings. A signal that restated its
// payload every session is the constant channel this design exists to avoid.
func TestDigestSignalDoesNotCarryTheFindings(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if _, err := digest.Add(root, digest.Finding{
		Sweep: "observe",
		Title: "UNIQUETITLESENTINEL",
		Body:  "UNIQUEBODYSENTINEL",
	}); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() { digestSessionSignal(root) })

	for _, leaked := range []string{"UNIQUETITLESENTINEL", "UNIQUEBODYSENTINEL"} {
		if strings.Contains(out, leaked) {
			t.Errorf("the signal leaked its payload (%s): %q", leaked, out)
		}
	}
}

// Silence is the common case — every session in every project reads this, and a
// project with nothing waiting must print nothing at all.
func TestDigestSignalIsSilentWhenNothingWaits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if out := captureStdout(t, func() { digestSessionSignal(t.TempDir()) }); out != "" {
		t.Errorf("an empty digest printed %q, want silence", out)
	}
}

// A read finding is not a waiting finding: once shown, it must stop being
// announced, or the reader is interrupted about something they have seen.
func TestDigestSignalGoesQuietAfterReading(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	if _, err := digest.Add(root, digest.Finding{Sweep: "observe", Title: "seen"}); err != nil {
		t.Fatal(err)
	}
	if _, err := digest.MarkRead(root); err != nil {
		t.Fatal(err)
	}
	if out := captureStdout(t, func() { digestSessionSignal(root) }); out != "" {
		t.Errorf("a read finding still signalled: %q", out)
	}
}
