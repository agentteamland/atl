package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkeletonFillsName(t *testing.T) {
	s, err := Skeleton(Project, "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "# my-app") {
		t.Fatalf("project skeleton did not fill {{NAME}}: %q", firstLine(s))
	}
	if strings.Contains(s, "{{NAME}}") {
		t.Errorf("placeholder left unfilled")
	}
	// Project tier references the managed-block skills + the owned fact sections.
	for _, want := range []string{"## Stack", "## Commands", "## Conventions", "/brainstorm", "/drain"} {
		if !strings.Contains(s, want) {
			t.Errorf("project skeleton missing %q", want)
		}
	}
}

func TestSkeletonGlobalIsPersona(t *testing.T) {
	s, err := Skeleton(Global, "")
	if err != nil {
		t.Fatal(err)
	}
	// Global is pure persona — ATL manages nothing, so no managed marker blocks.
	if strings.Contains(s, ":start -->") {
		t.Errorf("global persona must carry no managed marker blocks")
	}
	if !strings.Contains(s, "## Working style") {
		t.Errorf("global skeleton missing persona sections")
	}
}

func TestPathTiers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	gp, err := Path(Global, "/ignored")
	if err != nil {
		t.Fatal(err)
	}
	if gp != filepath.Join(home, ".claude", "CLAUDE.md") {
		t.Errorf("global path = %q", gp)
	}

	pp, err := Path(Project, "/proj")
	if err != nil {
		t.Fatal(err)
	}
	if pp != filepath.Join("/proj", "CLAUDE.md") {
		t.Errorf("project path = %q (must be the project root, not under .claude)", pp)
	}
}

func TestWriteIfAbsent(t *testing.T) {
	root := t.TempDir()

	// First call creates it.
	path, created, err := WriteIfAbsent(Project, root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first WriteIfAbsent should create the file")
	}
	if path != filepath.Join(root, "CLAUDE.md") {
		t.Fatalf("path = %q", path)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "# demo") {
		t.Errorf("written content missing filled name")
	}

	// Second call must NOT overwrite — the file is user-owned now.
	if err := os.WriteFile(path, []byte("USER EDITED"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, created, err = WriteIfAbsent(Project, root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("second WriteIfAbsent must not recreate an existing file")
	}
	body, _ = os.ReadFile(path)
	if string(body) != "USER EDITED" {
		t.Errorf("existing file was clobbered: %q", string(body))
	}
}

func TestWriteIfAbsentGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path, created, err := WriteIfAbsent(Global, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("global WriteIfAbsent should create ~/.claude/CLAUDE.md")
	}
	if path != filepath.Join(home, ".claude", "CLAUDE.md") {
		t.Errorf("global path = %q", path)
	}
}

func TestWriteStateFilesIfAbsent(t *testing.T) {
	root := t.TempDir()
	backlog := filepath.Join(root, ".atl", "backlog.md")
	tasks := filepath.Join(root, ".atl", "tasks.md")

	// First call creates both under .atl/.
	created, err := WriteStateFilesIfAbsent(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("first call should create 2 files, got %d: %v", len(created), created)
	}
	for _, p := range []string{backlog, tasks} {
		if _, serr := os.Stat(p); serr != nil {
			t.Fatalf("expected %s to exist: %v", p, serr)
		}
	}
	b, _ := os.ReadFile(backlog)
	if !strings.Contains(string(b), "# Backlog") || !strings.Contains(string(b), "Trigger:") {
		t.Errorf("backlog skeleton missing expected content: %q", firstLine(string(b)))
	}
	tb, _ := os.ReadFile(tasks)
	if !strings.Contains(string(tb), "# Tasks") {
		t.Errorf("tasks skeleton missing expected content: %q", firstLine(string(tb)))
	}

	// Second call must NOT overwrite a user-edited file.
	if err := os.WriteFile(backlog, []byte("USER EDITED"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err = WriteStateFilesIfAbsent(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range created {
		if p == backlog {
			t.Errorf("second call must not recreate an existing backlog.md")
		}
	}
	b, _ = os.ReadFile(backlog)
	if string(b) != "USER EDITED" {
		t.Errorf("existing backlog.md was clobbered: %q", string(b))
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// A board-backed project must get NEITHER decision-state file: the board is the
// authoritative deferral surface there, and scaffolding these produces exactly
// the second surface the brainstorm rule exists to prevent (#477).
func TestWriteStateFilesSkipsBoardBackedProject(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := []byte(`{"owner":"o","repo":"r","projectNumber":1,"backend":"github"}`)
	if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := WriteStateFilesIfAbsent(root)
	if err != nil {
		t.Fatalf("WriteStateFilesIfAbsent: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("created %v in a board-backed project — the board is the only deferral surface there", created)
	}
	for _, name := range stateFiles {
		if _, err := os.Stat(filepath.Join(root, ".atl", name)); !os.IsNotExist(err) {
			t.Fatalf(".atl/%s exists in a board-backed project", name)
		}
	}
}

// A .delivery/ with no backend field is not a board-backed project, and the check
// fails toward writing: silently scaffolding nothing would leave a project with
// nowhere to record a deferral and nothing reporting the absence.
func TestWriteStateFilesWritesWhenBackendIsAbsentOrUnreadable(t *testing.T) {
	for _, tc := range []struct {
		name, body string
	}{
		{"no backend field", `{"owner":"o","repo":"r"}`},
		{"empty backend", `{"backend":""}`},
		{"malformed json", `{oops`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			created, err := WriteStateFilesIfAbsent(root)
			if err != nil {
				t.Fatalf("WriteStateFilesIfAbsent: %v", err)
			}
			if len(created) != len(stateFiles) {
				t.Fatalf("created %d file(s), want %d — the check must fail toward writing", len(created), len(stateFiles))
			}
		})
	}
}
