package gc

import (
	"os"
	"os/exec"
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
	writeFile(t, filepath.Join(claudeDir, "knowledge/stale.md"), "stale")         // unowned knowledge asset → orphan (gc must walk knowledge/)

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

// TestScanRespectsUserRules: a rule the user authored (present in .atl/rules) is
// reflected into .claude/rules and must be treated as owned, while a .claude/rules
// file with no .atl/rules source is a genuine orphan.
func TestScanRespectsUserRules(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()

	// A user rule authored at the project layer: source in .atl/rules, reflected
	// copy in .claude/rules.
	writeFile(t, filepath.Join(proj, ".atl", "rules", "house-style.md"), "source")
	writeFile(t, filepath.Join(proj, ".claude", "rules", "house-style.md"), "reflected")
	// A .claude/rules file with no source → a real orphan (source was deleted, or
	// it was hand-dropped).
	writeFile(t, filepath.Join(proj, ".claude", "rules", "stale.md"), "no source")

	orphans, err := Scan(proj, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	byRel := map[string]Orphan{}
	for _, o := range orphans {
		byRel[o.Rel] = o
	}
	if _, ok := byRel["rules/house-style.md"]; ok {
		t.Error("a reflected user rule with a live .atl/rules source must never be an orphan")
	}
	if _, ok := byRel["rules/stale.md"]; !ok {
		t.Error("a .claude/rules file with no .atl/rules source must be reported as an orphan")
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

// gitRepo turns dir into a repo and commits the given paths (relative to dir).
func gitRepo(t *testing.T, dir string, commit ...string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	for _, p := range commit {
		run("add", "-f", p)
	}
	run("commit", "-q", "-m", "seed")
}

// The live case this was built for: `atl gc` listed two HAND-WRITTEN skills — each
// with its own commit history — as reclaimable, under a message naming the very
// case it could not distinguish. Nothing reaches a commit by accident.
func TestACommittedFileIsNotReportedAsDisposable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	claude := filepath.Join(root, ".claude")
	writeFile(t, filepath.Join(claude, "skills/mine/SKILL.md"), "hand-written")
	writeFile(t, filepath.Join(claude, "skills/scratch/SKILL.md"), "not committed")
	gitRepo(t, claude, "skills/mine/SKILL.md")

	got, err := Scan(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var mine, scratch *Orphan
	for i := range got {
		switch got[i].Rel {
		case "skills/mine/SKILL.md":
			mine = &got[i]
		case "skills/scratch/SKILL.md":
			scratch = &got[i]
		}
	}
	if mine == nil || scratch == nil {
		t.Fatalf("both files should be scanned; got %+v", got)
	}
	if !mine.Tracked {
		t.Error("a committed file must be marked Tracked — it is the whole signal")
	}
	if scratch.Tracked {
		t.Error("an untracked file must not be marked Tracked")
	}
	if mine.Origin() == scratch.Origin() {
		t.Errorf("committed and uncommitted must not read identically: both say %q", mine.Origin())
	}
}

// No git, not a repo, git missing from PATH — all must degrade to the previous
// behaviour rather than failing the scan. This decides whether to WARN, so being
// wrong must cost nothing.
func TestScanWorksWhereThereIsNoGit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	writeFile(t, filepath.Join(root, ".claude", "skills/loose/SKILL.md"), "x")

	got, err := Scan(root, time.Now())
	if err != nil {
		t.Fatalf("a non-repo must still scan: %v", err)
	}
	for _, o := range got {
		if o.Tracked {
			t.Error("nothing can be Tracked outside a repo")
		}
	}
	if len(got) == 0 {
		t.Error("the orphan should still be found")
	}
}

// The guard's per-session state grows forever otherwise: measured 2026-08-07 at
// 69 session dirs and 2,372 EMPTY marker files, invisible to any size-based check.
// A session's state is worthless once the session ends, so gc reclaims the dirs
// past GuardStateMaxAge — and must leave a recent one, which is how the current
// session's own state survives its own gc run.
func TestScanGuardStatePrunesOnlyOldSessions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, ".atl", "cache", "guard")

	now := time.Now()
	mk := func(name string, age time.Duration) string {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		// An empty marker, exactly as the guard writes them.
		if err := os.WriteFile(filepath.Join(dir, "marker"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		when := now.Add(-age)
		if err := os.Chtimes(dir, when, when); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	old := mk("old-session", GuardStateMaxAge+24*time.Hour)
	mk("fresh-session", time.Hour)

	got, err := scanGuardState(now)
	if err != nil {
		t.Fatalf("scanGuardState: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d orphan(s), want exactly the one past GuardStateMaxAge: %+v", len(got), got)
	}
	if got[0].Abs != old {
		t.Fatalf("reclaimed %q, want %q — a fresh session's state must survive", got[0].Abs, old)
	}
	if got[0].Scope != "guard-state" {
		t.Fatalf("scope %q, want guard-state so the report says what it is", got[0].Scope)
	}
}

// No guard dir at all is the common case (a machine that has never run the hook)
// and must be silent, not an error — gc runs on every session.
func TestScanGuardStateAbsentIsNotAnError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := scanGuardState(time.Now())
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v / %d orphan(s), want no error and nothing to reclaim", err, len(got))
	}
}
