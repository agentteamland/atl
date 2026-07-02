package source

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func makeTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractRoot(t *testing.T) {
	data := makeTarGz(t, map[string]string{
		"repo-1.0/team.json":         `{"name":"x"}`,
		"repo-1.0/agents/a/agent.md": "A",
		"repo-1.0/sub/deep.txt":      "D",
	})
	dest := t.TempDir()
	if err := Extract(bytes.NewReader(data), "", dest); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for _, rel := range []string{"team.json", "agents/a/agent.md", "sub/deep.txt"} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestExtractSubpath(t *testing.T) {
	data := makeTarGz(t, map[string]string{
		"repo-1.0/team.json":         `{}`,
		"repo-1.0/agents/a/agent.md": "A",
		"repo-1.0/agents/b/agent.md": "B",
	})
	dest := t.TempDir()
	if err := Extract(bytes.NewReader(data), "agents", dest); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// subpath becomes the root: agents/a/agent.md -> a/agent.md
	if _, err := os.Stat(filepath.Join(dest, "a/agent.md")); err != nil {
		t.Errorf("missing a/agent.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "team.json")); err == nil {
		t.Error("team.json should be excluded by the subpath filter")
	}
}

func TestExtractNoMatch(t *testing.T) {
	data := makeTarGz(t, map[string]string{"repo-1.0/readme": "x"})
	if err := Extract(bytes.NewReader(data), "nonexistent", t.TempDir()); err == nil {
		t.Error("expected error when subpath matches nothing")
	}
}

func TestExtractTraversal(t *testing.T) {
	data := makeTarGz(t, map[string]string{"repo-1.0/../evil.txt": "pwned"})
	if err := Extract(bytes.NewReader(data), "", t.TempDir()); err == nil {
		t.Error("expected error on path traversal")
	}
}

func TestTarballURL(t *testing.T) {
	got := TarballURL("acme/example-team", "v1.2.1")
	want := "https://github.com/acme/example-team/archive/v1.2.1.tar.gz"
	if got != want {
		t.Errorf("TarballURL = %q, want %q", got, want)
	}
}
