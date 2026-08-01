package commands

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/queue"
)

// testChannels stands in for "core's channel plus one a team declared" — the
// set declaredChannels returns on a machine with profile-team installed.
var testChannels = []manifest.Channel{
	coreChannel,
	{Name: "profile-fact", Drain: "/profile-drain", Rule: "profile-capture", Describes: "durable entity facts"},
}

// tickLine renders one transcript record in the shape ExtractFlow reads.
func tickLine(role, text string) string {
	return `{"message":{"role":"` + role + `","content":[{"type":"text","text":` + strconv.Quote(text) + `}]}}` + "\n"
}

// TestCaptureWatchdogIsPerChannel is the regression for the watchdog's
// channel-blindness. The old predicate was `len(marker.Parse(s)) > 0`, which
// matches BOTH channels — so a session emitting learning markers on every turn
// reset the dry stretch continuously and the watchdog could never fire for a
// missed profile-fact, the exact omission it exists to detect.
//
// The fixture is that session: every assistant turn carries a learning marker,
// none carries a profile-fact. The learning stretch is therefore always 0 (it
// resets each turn) and must stay silent; the profile-fact stretch accumulates
// and must fire.
func TestCaptureWatchdogIsPerChannel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sess.jsonl")

	body := tickLine("user", strings.Repeat("a", 600)) +
		tickLine("assistant", "reply one <!-- learning: pools are shared -->") +
		tickLine("user", strings.Repeat("b", 600)) +
		tickLine("assistant", "reply two <!-- learning: retries need a cap -->") +
		tickLine("user", strings.Repeat("c", 600)) +
		tickLine("assistant", "reply three <!-- learning: mtime beats self-reported time -->")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := queue.Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	out := captureStdout(t, func() { captureWatchdogNotice(st, "/p/a", path, testChannels) })

	if !strings.Contains(out, "capture-watchdog (profile-fact)") {
		t.Errorf("profile-fact watchdog did not fire; output = %q\n"+
			"a channel-blind predicate lets learning markers mask a missed profile-fact", out)
	}
	if !strings.Contains(out, "/profile-drain") {
		t.Errorf("profile-fact nudge should point at /profile-drain; output = %q", out)
	}
	if strings.Contains(out, "capture-watchdog (learning)") {
		t.Errorf("learning watchdog fired, but every assistant turn carried a learning marker; output = %q", out)
	}

	// Fire-once: the same stretch must not nudge twice.
	again := captureStdout(t, func() { captureWatchdogNotice(st, "/p/a", path, testChannels) })
	if again != "" {
		t.Errorf("second call re-fired: %q", again)
	}

	// A profile-fact marker re-arms the channel's latch by advancing its ordinal,
	// while leaving too little dry stretch behind it to fire again.
	body += tickLine("assistant", "noted <!-- profile-fact:\n  entity: ahmet\n  fields:\n    identity.name: Ahmet\n-->")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if out := captureStdout(t, func() { captureWatchdogNotice(st, "/p/a", path, testChannels) }); out != "" {
		t.Errorf("fired after the stretch was closed by a profile-fact marker: %q", out)
	}
}

// TestCaptureWatchdogOptOut keeps the documented escape hatch honest.
func TestCaptureWatchdogOptOut(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sess.jsonl")
	body := tickLine("user", strings.Repeat("a", 1200)) +
		tickLine("assistant", "dry reply one") +
		tickLine("assistant", "") +
		tickLine("user", "short") +
		tickLine("assistant", "dry reply two")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := queue.Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	t.Setenv("ATL_NO_CAPTURE_WATCHDOG", "1")
	if out := captureStdout(t, func() { captureWatchdogNotice(st, "/p/a", path, testChannels) }); out != "" {
		t.Errorf("fired with ATL_NO_CAPTURE_WATCHDOG set: %q", out)
	}
}
