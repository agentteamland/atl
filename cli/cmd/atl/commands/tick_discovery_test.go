package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/agentteamland/atl/cli/internal/transcript"
)

// seedSession writes one transcript under the slug Claude Code would use for a
// session whose working directory is cwd, recording that cwd the way a real
// transcript does, with one assistant turn carrying a learning marker.
func seedSession(t *testing.T, cwd, marker string) string {
	t.Helper()
	dir, err := transcript.ProjectDir(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"queue-operation","operation":"enqueue"}` + "\n" +
		`{"cwd":` + strconv.Quote(cwd) + `,"message":{"role":"user","content":"hi"}}` + "\n" +
		`{"cwd":` + strconv.Quote(cwd) + `,"message":{"role":"assistant","content":[{"type":"text","text":` +
		strconv.Quote("noted <!-- learning: "+marker+" -->") + `}]}}` + "\n"
	path := filepath.Join(dir, strings.ReplaceAll(marker, " ", "-")+".jsonl")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func queuedPayloads(t *testing.T, st *queue.Store, project string) string {
	t.Helper()
	items, err := st.Pending(project, queue.ChannelLearning)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.Payload)
		b.WriteByte('\n')
	}
	return b.String()
}

// The regression for atl#524. The queue is keyed by the repository root, but
// Claude Code files a session's transcript under the slug of the session's OWN
// working directory — a worktree under .claude/worktrees (the desktop default),
// or a subdirectory. Discovery read only the root's slug, so a session working
// anywhere but the root had its markers silently dropped: the tick reported
// "0 marker(s)" while they sat in a directory nothing read.
//
// The fixture is built so the root and the session's directory DIFFER. Seeding
// at the root — where both keys agree — is the condition that hid the defect.
//
// It also pins what discovery must NOT swallow: a sibling project whose slug the
// root's slug is a prefix of (ATL-foo vs ATL), and a nested repository inside the
// root's tree that is a project of its own.
func TestTickCapturesMarkersFromWorktreeAndSubdirectorySessions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ATL_NO_CAPTURE_WATCHDOG", "1")
	base := resolved(t, t.TempDir())
	root := filepath.Join(base, "ATL")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRepo(t, root)

	wt := filepath.Join(root, ".claude", "worktrees", "brave-otter")
	c := exec.Command("git", "worktree", "add", "-q", "-b", "wt-1", wt)
	c.Dir = root
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}
	game := filepath.Join(root, "game")
	nested := filepath.Join(root, "vendor-repo")
	sibling := filepath.Join(base, "ATL-foo")
	for _, d := range []string{game, nested, sibling} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	gitRepo(t, nested)

	seedSession(t, root, "from the root")
	seedSession(t, wt, "from the live worktree")
	seedSession(t, game, "from a subdirectory")
	// A worktree that has since been removed: its transcripts outlive it, so the
	// confirmation cannot ask git about a path that no longer exists.
	seedSession(t, filepath.Join(root, ".claude", "worktrees", "gone-heron"), "from a removed worktree")
	seedSession(t, sibling, "from the sibling project")
	seedSession(t, nested, "from the nested repository")

	t.Chdir(wt)
	project, err := projectKey()
	if err != nil {
		t.Fatal(err)
	}
	if project != root {
		t.Fatalf("projectKey() in the worktree = %q, want the root %q", project, root)
	}
	st, err := queue.Open(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	if _, _, _, _, _, _, err := drainProjectTranscripts(st, project, wt, testChannels); err != nil {
		t.Fatal(err)
	}
	got := queuedPayloads(t, st, project)

	for _, want := range []string{"from the root", "from the live worktree", "from a subdirectory", "from a removed worktree"} {
		if !strings.Contains(got, want) {
			t.Errorf("marker %q was not captured into the root's queue; queued:\n%s", want, got)
		}
	}
	for _, foreign := range []string{"from the sibling project", "from the nested repository"} {
		if strings.Contains(got, foreign) {
			t.Errorf("marker %q belongs to another project but was captured into %s; queued:\n%s", foreign, root, got)
		}
	}
}

// The watchdog measures "the live session". With transcripts drawn from several
// directories, "the newest transcript scanned" can be another session's — a
// worktree session would be nudged about a stretch it never had. The live
// session is the one in the caller's own directory, so a newer transcript in a
// sibling worktree must not be the one measured.
func TestCaptureWatchdogMeasuresTheCallersOwnSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	base := resolved(t, t.TempDir())
	root := filepath.Join(base, "ATL")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRepo(t, root)
	mine := filepath.Join(root, "game")
	other := filepath.Join(root, "tools")
	for _, d := range []string{mine, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// My session marked on every turn — its stretch is dry for nothing. The other
	// session, written AFTER mine, is long and has no markers at all.
	minePath := seedSession(t, mine, "my own marker")
	otherDir, _ := transcript.ProjectDir(other)
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"cwd":` + strconv.Quote(other) + `,"message":{"role":"user","content":"hi"}}` + "\n" +
		tickLine("user", strings.Repeat("a", 1200)) +
		tickLine("assistant", "dry one") +
		tickLine("user", strings.Repeat("b", 1200)) +
		tickLine("assistant", "dry two")
	otherPath := filepath.Join(otherDir, "other.jsonl")
	if err := os.WriteFile(otherPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(minePath)
	if err != nil {
		t.Fatal(err)
	}
	later := info.ModTime().Add(5e9)
	if err := os.Chtimes(otherPath, later, later); err != nil {
		t.Fatal(err)
	}

	st, err := queue.Open(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	out := captureStdout(t, func() {
		if _, _, _, _, _, _, err := drainProjectTranscripts(st, root, mine, testChannels); err != nil {
			t.Error(err)
		}
	})
	if strings.Contains(out, "capture-watchdog") {
		t.Errorf("the watchdog measured another session's stretch, not the caller's; output = %q", out)
	}
}
