package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSlugForPath(t *testing.T) {
	got := SlugForPath("/Users/foo/projects/myapp")
	want := "-Users-foo-projects-myapp"
	if got != want {
		t.Fatalf("slug: got %q want %q", got, want)
	}
}

func TestExtractText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	lines := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Decided. <!-- learning: A -->"},{"type":"tool_use","name":"x"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<!-- learning: from-user-ignored -->"}]}}
{"type":"assistant","message":{"role":"assistant","content":"string shape <!-- profile-fact: B -->"}}
not even json
`
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	text, err := ExtractText(path)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !strings.Contains(text, "<!-- learning: A -->") {
		t.Error("missing assistant array-content marker")
	}
	if !strings.Contains(text, "<!-- profile-fact: B -->") {
		t.Error("missing assistant string-content marker")
	}
	if strings.Contains(text, "from-user-ignored") {
		t.Error("user-message text must be ignored")
	}
}

func TestFindModTimeFilterAndOrder(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.jsonl")
	recent := filepath.Join(dir, "recent.jsonl")
	_ = os.WriteFile(old, []byte("{}"), 0o600)
	_ = os.WriteFile(recent, []byte("{}"), 0o600)
	t0 := time.Now().Add(-2 * time.Hour)
	t1 := time.Now().Add(-1 * time.Minute)
	_ = os.Chtimes(old, t0, t0)
	_ = os.Chtimes(recent, t1, t1)

	// since 1h ago → only recent
	files, err := Find(dir, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0].Path) != "recent.jsonl" {
		t.Fatalf("modtime filter: got %+v", files)
	}

	// zero since → both, oldest first
	all, _ := Find(dir, time.Time{})
	if len(all) != 2 || filepath.Base(all[0].Path) != "old.jsonl" {
		t.Fatalf("ordering: got %+v", all)
	}
}

func TestFindMissingDir(t *testing.T) {
	files, err := Find(filepath.Join(t.TempDir(), "nope"), time.Time{})
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if files != nil {
		t.Fatalf("want nil, got %+v", files)
	}
}

func TestFindIgnoresNonJsonl(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o600)
	files, _ := Find(dir, time.Time{})
	if len(files) != 1 {
		t.Fatalf("want 1 jsonl, got %d", len(files))
	}
}
