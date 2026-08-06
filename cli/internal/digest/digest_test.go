package digest

import (
	"os"
	"testing"
	"time"
)

// project isolates the global layer so a test never touches the real ~/.atl.
func project(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return t.TempDir()
}

func add(t *testing.T, root, sweep, title, body string) bool {
	t.Helper()
	ok, err := Add(root, Finding{Sweep: sweep, Title: title, Body: body})
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

// A project that has never been swept has an empty digest, not an error — the
// session signal reads this on every start, in every project.
func TestAnUnsweptProjectIsEmpty(t *testing.T) {
	root := project(t)
	got, err := Load(root)
	if err != nil {
		t.Fatalf("Load on a fresh project: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d findings, want 0", len(got))
	}
	if n := UnreadCount(root); n != 0 {
		t.Errorf("UnreadCount = %d, want 0", n)
	}
}

func TestAddThenRead(t *testing.T) {
	root := project(t)
	if !add(t, root, "observe", "the store has no remote", "evidence") {
		t.Fatal("the first Add must report the finding as new")
	}
	if n := UnreadCount(root); n != 1 {
		t.Fatalf("UnreadCount = %d, want 1", n)
	}
	n, err := MarkRead(root)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("MarkRead marked %d, want 1", n)
	}
	if n := UnreadCount(root); n != 0 {
		t.Errorf("UnreadCount after read = %d, want 0", n)
	}
}

// The idempotence the store exists for. A sweep re-runs whenever its paths move
// and a latent gap does not stop being one because it was reported yesterday —
// so re-reporting must converge, not grow the digest.
func TestReReportingTheSameFindingDoesNotDuplicate(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "the store has no remote", "first wording")
	if added := add(t, root, "observe", "the store has no remote", "sharper wording"); added {
		t.Error("re-reporting the same finding must not report it as new")
	}
	got, _ := Load(root)
	if len(got) != 1 {
		t.Fatalf("digest holds %d findings, want 1", len(got))
	}
	if got[0].Body != "sharper wording" {
		t.Errorf("body = %q, want the refreshed wording", got[0].Body)
	}
}

// …and re-reporting must not make a finding the reader has already seen unread
// again. Without this, every sweep is a fresh interruption about the same thing,
// which is precisely the constant channel the digest exists to avoid.
func TestReReportingDoesNotResurrectAReadFinding(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "a latent gap", "body")
	if _, err := MarkRead(root); err != nil {
		t.Fatal(err)
	}
	add(t, root, "observe", "a latent gap", "body, re-worded")

	if n := UnreadCount(root); n != 0 {
		t.Errorf("UnreadCount = %d after re-reporting a READ finding, want 0", n)
	}
}

// Two sweeps may legitimately name the same title about different corpora, so
// the key spans both fields.
func TestTheKeySpansSweepAndTitle(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "same title", "a")
	if added := add(t, root, "skill-stocktake", "same title", "b"); !added {
		t.Error("the same title from a different sweep is a different finding")
	}
	if got, _ := Load(root); len(got) != 2 {
		t.Errorf("digest holds %d findings, want 2", len(got))
	}
}

// The key must NOT include the body: a sweep that re-words its evidence between
// runs is reporting the same finding, and keying on the body would make every
// rewording a new entry — the duplicate-growth failure by another route.
func TestTheKeyIgnoresTheBody(t *testing.T) {
	if NewID("observe", "t") != NewID("observe", "t") {
		t.Fatal("NewID is not stable")
	}
	a, _ := Add(project(t), Finding{Sweep: "observe", Title: "t", Body: "one"})
	if !a {
		t.Fatal("precondition")
	}
	if NewID("observe", "t") == NewID("observe", "different title") {
		t.Error("a different title must be a different finding")
	}
}

// Reading is not deciding. A finding the reader has seen but not acted on is
// still real, so the digest must keep it — a store that emptied itself on being
// read would drop exactly the ones that needed thinking about.
func TestReadingKeepsTheFinding(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "needs a decision", "body")
	if _, err := MarkRead(root); err != nil {
		t.Fatal(err)
	}
	got, _ := Load(root)
	if len(got) != 1 {
		t.Fatalf("digest holds %d findings after reading, want 1", len(got))
	}
	if got[0].Unread() {
		t.Error("the finding should be marked read")
	}
}

func TestDropRemovesOnlyTheNamedFinding(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "one", "a")
	add(t, root, "observe", "two", "b")

	ok, err := Drop(root, NewID("observe", "one"))
	if err != nil || !ok {
		t.Fatalf("Drop = %v, %v; want true, nil", ok, err)
	}
	got, _ := Load(root)
	if len(got) != 1 || got[0].Title != "two" {
		t.Errorf("after Drop the digest holds %+v, want only \"two\"", got)
	}

	// An unknown id must say so rather than report a silent success — a caller
	// that reports "dropped" for a finding that is still there is worse than one
	// that errors.
	if ok, _ := Drop(root, "nosuchid"); ok {
		t.Error("Drop of an unknown id must report false")
	}
}

// Newest first: a reader scanning a digest wants what just landed, and the
// session signal's count is meaningless without a stable order behind it.
func TestFindingsAreNewestFirst(t *testing.T) {
	root := project(t)
	old := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := Add(root, Finding{Sweep: "observe", Title: "older", AddedAt: old}); err != nil {
		t.Fatal(err)
	}
	add(t, root, "observe", "newer", "b")

	got, _ := Load(root)
	if len(got) != 2 || got[0].Title != "newer" {
		t.Errorf("order = %v, want newest first", []string{got[0].Title, got[1].Title})
	}
}

// A corrupt digest must not wedge the sweep that writes it or the signal that
// reads it. Losing a finding is recoverable — the sweep re-reports it — while a
// permanently failing read is not.
func TestACorruptDigestReadsAsEmptyAndIsRecoverable(t *testing.T) {
	root := project(t)
	add(t, root, "observe", "one", "a")
	p, err := Path(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := Load(root); err != nil || len(got) != 0 {
		t.Fatalf("Load on a corrupt digest = %v, %v; want empty, nil", got, err)
	}
	if !add(t, root, "observe", "one", "a") {
		t.Error("a corrupt digest must be rewritable, not permanently wedged")
	}
}

// Two projects must not answer for each other — a sweep fires in every project
// with an .atl/ surface, and one shared file would let whichever was opened
// first suppress the others' signal.
func TestProjectsAreIsolated(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a, b := t.TempDir(), t.TempDir()
	if _, err := Add(a, Finding{Sweep: "observe", Title: "only in a"}); err != nil {
		t.Fatal(err)
	}
	if n := UnreadCount(b); n != 0 {
		t.Errorf("project b sees %d findings from project a, want 0", n)
	}
	if n := UnreadCount(a); n != 1 {
		t.Errorf("project a sees %d of its own findings, want 1", n)
	}
}
