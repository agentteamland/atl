package skillsstate

import (
	"testing"
	"time"
)

func TestRecordAndLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	when := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	if err := Record("abc123", when); err != nil {
		t.Fatal(err)
	}
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.LastStocktakeSHA != "abc123" {
		t.Errorf("SHA: got %q want abc123", s.LastStocktakeSHA)
	}
	if s.LastStocktakeAt == "" {
		t.Error("timestamp was not recorded")
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s, err := Load()
	if err != nil {
		t.Fatalf("missing state should not error: %v", err)
	}
	if s.LastStocktakeSHA != "" || s.LastStocktakeAt != "" {
		t.Errorf("missing state should be empty, got %+v", s)
	}
}
