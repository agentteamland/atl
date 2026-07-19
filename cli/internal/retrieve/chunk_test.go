package retrieve

import (
	"strings"
	"testing"
)

func TestChunkTextShortIsOnePiece(t *testing.T) {
	if got := chunkText("a short page"); len(got) != 1 || got[0] != "a short page" {
		t.Fatalf("short text: want 1 chunk verbatim, got %v", got)
	}
	if got := chunkText("   "); got != nil {
		t.Fatalf("blank text: want nil, got %v", got)
	}
}

func TestChunkTextLongSplitsTokenSafeOnWordBoundaries(t *testing.T) {
	// ~5000 chars of space-separated words -> several chunks.
	text := strings.TrimSpace(strings.Repeat("alpha beta gamma delta ", 260)) // ~6000 chars
	chunks := chunkText(text)
	if len(chunks) < 2 {
		t.Fatalf("long text should split, got %d chunk(s)", len(chunks))
	}
	for i, c := range chunks {
		if len([]rune(c)) > chunkChars {
			t.Fatalf("chunk %d exceeds chunkChars (%d > %d)", i, len([]rune(c)), chunkChars)
		}
		if strings.TrimSpace(c) == "" {
			t.Fatalf("chunk %d is blank", i)
		}
		// Word-boundary: no chunk starts or ends mid-word fragment (our words are
		// whole tokens, so every chunk should begin and end with a full word).
		if strings.HasPrefix(c, "lpha") || strings.HasSuffix(c, "alph") {
			t.Fatalf("chunk %d cut mid-word: %q...", i, c[:20])
		}
	}
}

func TestChunkTextCapsAtMaxChunks(t *testing.T) {
	// A pathologically large document must not produce unbounded chunks.
	huge := strings.Repeat("word ", 200000) // ~1,000,000 chars
	chunks := chunkText(huge)
	if len(chunks) > maxChunks {
		t.Fatalf("want at most %d chunks, got %d", maxChunks, len(chunks))
	}
}
