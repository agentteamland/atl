package commands

import "testing"

func TestFirstLine(t *testing.T) {
	if got := firstLine("one line"); got != "one line" {
		t.Errorf("single line = %q", got)
	}
	if got := firstLine("first\nsecond\nthird"); got != "first …" {
		t.Errorf("multiline = %q, want \"first …\"", got)
	}
	if got := firstLine(""); got != "" {
		t.Errorf("empty = %q", got)
	}
}
