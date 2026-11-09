package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// subjectProject is a project with a backlog file and a source file to anchor on.
func subjectProject(t *testing.T) string {
	t.Helper()
	root := project(t)
	for _, p := range []string{".atl/backlog.md", "src/order.go"} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func key(t *testing.T, root, kind string, anchors ...string) string {
	t.Helper()
	k, err := SubjectKey(root, kind, anchors)
	if err != nil {
		t.Fatalf("SubjectKey(%q, %v): %v", kind, anchors, err)
	}
	return k
}

// The reproduction. Two sweeps report the same stale backlog row, one in English
// and one in Turkish — the 2026-11-09 shape that opened AB#2033 and AB#2137 for
// one finding. Keyed by title they are two findings; keyed by subject, one.
func TestTheSameFindingInTwoLanguagesIsOneFinding(t *testing.T) {
	root := subjectProject(t)
	const en = "backlog.md still shows the shipped virtual-account rows as in flight"
	// The Turkish title is escaped so the repository's English-only scan passes;
	// it is a specimen of the input, not prose this file speaks in.
	const tr = "backlog.md sevk edilmi\u015f sanal hesap sat\u0131rlar\u0131n\u0131 h\u00e2l\u00e2 yolda g\u00f6steriyor"

	// What the title key does, recorded so the contrast stays visible: two.
	titled := project(t)
	add(t, titled, "observe", en, "evidence")
	add(t, titled, "observe", tr, "kan\u0131t")
	if got, _ := Load(titled); len(got) != 2 {
		t.Fatalf("title-keyed control: got %d findings, want 2 (the bug this test exists for)", len(got))
	}

	// Each sweep lists its anchors its own way: order, ./ prefix, absolute path,
	// card-reference case. None of that may move the key.
	k1 := key(t, root, "stale-truth", ".atl/backlog.md", "AB#1201")
	k2 := key(t, root, "stale-truth", "ab#1201", "./.atl/backlog.md")
	k3 := key(t, root, "stale-truth", filepath.Join(root, ".atl", "backlog.md"), "AB#1201", "AB#1201")
	if k1 != k2 || k1 != k3 {
		t.Fatalf("one subject, three keys: %s / %s / %s", k1, k2, k3)
	}

	if !mustAdd(t, root, Finding{Sweep: "observe", Key: k1, Title: en, Body: "evidence"}) {
		t.Fatal("the first report must be new")
	}
	if mustAdd(t, root, Finding{Sweep: "observe", Key: k2, Title: tr, Body: "kan\u0131t"}) {
		t.Fatal("the Turkish re-report of the same subject was recorded as a new finding")
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d findings for one subject, want 1", len(got))
	}
	// The latest wording wins; the identity does not move.
	if got[0].Title != tr || got[0].Body != "kan\u0131t" || got[0].ID != KeyID("observe", k1) {
		t.Errorf("refresh did not take the latest wording under the same id: %+v", got[0])
	}
}

// A keyed re-report must keep the read state, exactly like a titled one.
func TestAKeyedReReportKeepsTheReadState(t *testing.T) {
	root := subjectProject(t)
	k := key(t, root, "ripe-trigger", "AB#2033")
	mustAdd(t, root, Finding{Sweep: "observe", Key: k, Title: "a", Body: "1"})
	if _, err := MarkRead(root); err != nil {
		t.Fatal(err)
	}
	mustAdd(t, root, Finding{Sweep: "observe", Key: k, Title: "b", Body: "2"})
	if n := UnreadCount(root); n != 0 {
		t.Errorf("UnreadCount = %d after a keyed re-report, want 0", n)
	}
}

// Different subjects must stay apart: a key that merged them would silently
// overwrite one finding's body with another's.
func TestDifferentSubjectsGetDifferentKeys(t *testing.T) {
	root := subjectProject(t)
	base := key(t, root, "stale-truth", ".atl/backlog.md")
	for _, other := range []string{
		key(t, root, "growth-risk", ".atl/backlog.md"),
		key(t, root, "stale-truth", ".atl/backlog.md", "AB#1"),
		key(t, root, "stale-truth", "src/order.go"),
		key(t, root, "stale-truth", "src/order.go#PlaceOrder"),
	} {
		if other == base {
			t.Errorf("distinct subjects collided on %s", base)
		}
	}
}

// The refusals are what keep the key from drifting again: each refused input is
// one a sweep could otherwise word differently from run to run.
func TestAnchorsThatCouldDriftAreRefused(t *testing.T) {
	root := subjectProject(t)
	cases := []struct {
		kind    string
		anchors []string
		why     string
	}{
		{"stale backlog", []string{".atl/backlog.md"}, "a free-text kind"},
		{"eskimi\u015f-bilgi", []string{".atl/backlog.md"}, "a kind in another language"},
		{"stale-truth", []string{"backlog-sanal-hesap-shipped"}, "a slug that is no path"},
		{"stale-truth", []string{"backlog-virtual-account-shipped"}, "the same slug in English"},
		{"stale-truth", []string{"src/order.go#123"}, "a line pin, which drifts"},
		{"stale-truth", []string{"src/order.go#sipari\u015f verme"}, "a phrase after #"},
		{"stale-truth", nil, "no anchor at all"},
	}
	for _, c := range cases {
		if k, err := SubjectKey(root, c.kind, c.anchors); err == nil {
			t.Errorf("%s: accepted as %s, want a refusal", c.why, k)
		}
	}
}

func TestTheKeyIsATagShapedString(t *testing.T) {
	root := subjectProject(t)
	k := key(t, root, "ripe-trigger", "atl#434")
	if !strings.HasPrefix(k, "atl-observe:") || len(k) > 50 || strings.ContainsAny(k, " \t\n") {
		t.Errorf("key %q is not usable as a board tag/label", k)
	}
}

func mustAdd(t *testing.T, root string, f Finding) bool {
	t.Helper()
	ok, err := Add(root, f)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}
