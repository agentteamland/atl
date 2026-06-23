package docsstate

import (
	"testing"
	"time"
)

func TestLoadMissingIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.LastAuditSHA != "" {
		t.Errorf("fresh state should have no SHA, got %q", s.LastAuditSHA)
	}
	if s.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", s.SchemaVersion, SchemaVersion)
	}
}

func TestRecordRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	when := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	if err := Record("abc123", when); err != nil {
		t.Fatalf("Record: %v", err)
	}
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.LastAuditSHA != "abc123" {
		t.Errorf("LastAuditSHA = %q, want abc123", s.LastAuditSHA)
	}
	if s.LastAuditAt != "2026-06-23T10:00:00Z" {
		t.Errorf("LastAuditAt = %q, want 2026-06-23T10:00:00Z", s.LastAuditAt)
	}
}
