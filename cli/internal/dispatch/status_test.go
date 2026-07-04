package dispatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadStatus(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "phase": "self-test",
  "heartbeatTs": "2026-07-04T12:00:00Z",
  "blocker": "",
  "lastOutputSummary": "running widget tests"
}`
	if err := os.WriteFile(filepath.Join(dir, StatusFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := ReadStatus(dir)
	if err != nil {
		t.Fatalf("ReadStatus: %v", err)
	}
	if s.Phase != "self-test" {
		t.Errorf("Phase = %q, want self-test", s.Phase)
	}
	want := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	if !s.HeartbeatTS.Equal(want) {
		t.Errorf("HeartbeatTS = %v, want %v", s.HeartbeatTS, want)
	}
	if s.Blocker != "" {
		t.Errorf("Blocker = %q, want empty", s.Blocker)
	}
	if s.LastOutputSummary != "running widget tests" {
		t.Errorf("LastOutputSummary = %q", s.LastOutputSummary)
	}
}

func TestReadStatusMissingIsNotExist(t *testing.T) {
	// A missing status.json is the expected pre-first-heartbeat state — callers
	// distinguish it via os.IsNotExist, so the error must preserve that.
	_, err := ReadStatus(t.TempDir())
	if !os.IsNotExist(err) {
		t.Errorf("missing status should be os.IsNotExist, got %v", err)
	}
}

func TestReadStatusMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, StatusFileName), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStatus(dir); err == nil {
		t.Error("ReadStatus of malformed JSON should error")
	}
}
