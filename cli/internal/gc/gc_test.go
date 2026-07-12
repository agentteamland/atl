package gc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/coreassets"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/pin"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanOwnedVsUnowned(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()

	// A team installed at the project layer owns agents/api/agent.md.
	m := &manifest.Manifest{
		Handle: "acme", Name: "team",
		Files: map[string]string{"agents/api/agent.md": "sha"},
	}
	if err := m.Write(filepath.Join(proj, ".atl")); err != nil {
		t.Fatal(err)
	}
	claudeDir := filepath.Join(proj, ".claude")
	writeFile(t, filepath.Join(claudeDir, "agents/api/agent.md"), "owned")        // owned → not orphan
	writeFile(t, filepath.Join(claudeDir, "agents/api/children/gain.md"), "gain") // sibling of an owned unit → orphan, Owned
	writeFile(t, filepath.Join(claudeDir, "skills/rogue/SKILL.md"), "rogue")      // wholly unowned unit → orphan
	writeFile(t, filepath.Join(claudeDir, "knowledge/stale.md"), "stale")        // unowned knowledge asset → orphan (gc must walk knowledge/)

	orphans, err := Scan(proj, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	byRel := map[string]Orphan{}
	for _, o := range orphans {
		byRel[o.Rel] = o
	}
	if _, ok := byRel["agents/api/agent.md"]; ok {
		t.Error("a manifest-owned file must never be an orphan")
	}
	gain, ok := byRel["agents/api/children/gain.md"]
	if !ok || !gain.Owned {
		t.Errorf("a sibling gain should be an owned-unit orphan: %+v (ok=%v)", gain, ok)
	}
	rogue, ok := byRel["skills/rogue/SKILL.md"]
	if !ok || rogue.Owned {
		t.Errorf("a rogue file should be an unowned-unit orphan: %+v (ok=%v)", rogue, ok)
	}
	if _, ok := byRel["knowledge/stale.md"]; !ok {
		t.Error("an unowned knowledge/ asset must be reported as an orphan (gc must walk knowledge/)")
	}
}

// TestScanRespectsPins: a project-pinned path is treated as owned and never
// reported as an orphan, while an unpinned sibling gain still is.
func TestScanRespectsPins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()

	m := &manifest.Manifest{Handle: "acme", Name: "team",
		Files: map[string]string{"agents/api/agent.md": "sha"}}
	if err := m.Write(filepath.Join(proj, ".atl")); err != nil {
		t.Fatal(err)
	}
	claudeDir := filepath.Join(proj, ".claude")
	writeFile(t, filepath.Join(claudeDir, "agents/api/agent.md"), "owned")
	writeFile(t, filepath.Join(claudeDir, "agents/api/children/pinned.md"), "gain") // pinned → not an orphan
	writeFile(t, filepath.Join(claudeDir, "agents/api/children/free.md"), "gain2")  // unpinned gain → orphan

	pins := &pin.Set{}
	pins.Add("agents/api/children/pinned.md")
	if err := pins.Write(filepath.Join(proj, ".atl")); err != nil {
		t.Fatal(err)
	}

	orphans, err := Scan(proj, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	byRel := map[string]Orphan{}
	for _, o := range orphans {
		byRel[o.Rel] = o
	}
	if _, ok := byRel["agents/api/children/pinned.md"]; ok {
		t.Error("a pinned path must not be reported as an orphan")
	}
	if _, ok := byRel["agents/api/children/free.md"]; !ok {
		t.Error("an unpinned gain should still be reported as an orphan")
	}
}

func TestScanTreatsCoreAsOwned(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Reflect one real core file into ~/.claude — it carries no install manifest,
	// but as a platform asset from the binary it must never be flagged an orphan.
	paths, err := coreassets.Paths()
	if err != nil || len(paths) == 0 {
		t.Fatalf("core paths: %v (n=%d)", err, len(paths))
	}
	writeFile(t, filepath.Join(home, ".claude", filepath.FromSlash(paths[0])), "core")

	orphans, err := Scan(t.TempDir(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range orphans {
		if o.Scope == "global" {
			t.Errorf("a reflected core file must not be a global orphan: %+v", o)
		}
	}
}

func TestSoftDeleteAndUndoRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()
	orphanPath := filepath.Join(proj, ".claude", "skills/rogue/SKILL.md")
	writeFile(t, orphanPath, "rogue")

	trash := filepath.Join(home, ".atl", "gc-trash")
	orphans := []Orphan{{Scope: "project", Rel: "skills/rogue/SKILL.md", Abs: orphanPath}}

	if _, err := SoftDelete(orphans, trash, "20260701-000000"); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Error("soft-delete must remove the file from disk")
	}
	n, err := Undo(trash)
	if err != nil || n != 1 {
		t.Fatalf("undo: n=%d err=%v", n, err)
	}
	if b, err := os.ReadFile(orphanPath); err != nil || string(b) != "rogue" {
		t.Errorf("undo must restore the file with its content: %q err=%v", b, err)
	}
	// Trash batch is gone after undo.
	if _, err := os.Stat(filepath.Join(trash, "20260701-000000")); !os.IsNotExist(err) {
		t.Error("undo should remove the batch")
	}
}

func TestPurgeByAge(t *testing.T) {
	home := t.TempDir()
	trash := filepath.Join(home, ".atl", "gc-trash")
	oldBatch := filepath.Join(trash, "old")
	recentBatch := filepath.Join(trash, "recent")
	if err := os.MkdirAll(oldBatch, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(recentBatch, 0o755); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(oldBatch, past, past)

	n, err := Purge(trash, 24*time.Hour, time.Now())
	if err != nil || n != 1 {
		t.Fatalf("purge: n=%d err=%v", n, err)
	}
	if _, err := os.Stat(oldBatch); !os.IsNotExist(err) {
		t.Error("the old batch should be purged")
	}
	if _, err := os.Stat(recentBatch); err != nil {
		t.Error("the recent batch should survive")
	}
}

func TestScanHistoryExpiry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	histRoot := filepath.Join(home, ".atl", "history")
	oldSnap := filepath.Join(histRoot, "acme__team", "abc123")
	newSnap := filepath.Join(histRoot, "acme__team", "def456")
	writeFile(t, filepath.Join(oldSnap, "agents/x/agent.md"), "old")
	writeFile(t, filepath.Join(newSnap, "agents/x/agent.md"), "new")
	past := time.Now().Add(-40 * 24 * time.Hour)
	_ = os.Chtimes(oldSnap, past, past)

	orphans, err := Scan(t.TempDir(), time.Now()) // an empty project — only history matters
	if err != nil {
		t.Fatal(err)
	}
	var hist []string
	for _, o := range orphans {
		if o.Scope == "history" {
			hist = append(hist, o.Rel)
		}
	}
	if len(hist) != 1 || hist[0] != "acme__team/abc123" {
		t.Errorf("only the expired snapshot should be reclaimable, got %v", hist)
	}
}
