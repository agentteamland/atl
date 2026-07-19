package retrieve

import "strings"

// Chunking bounds. The embedding model has a hard 512-token limit and does NOT
// truncate — it errors on longer input — and the vast majority of real knowledge
// pages exceed it, so every document is split into token-safe chunks before
// embedding. chunkChars keeps a chunk well under 512 tokens even for token-dense
// markdown (English WordPiece is ~0.25-0.35 tokens/char, so ~1000 chars is
// ~250-350 tokens). chunkOverlap carries context across a boundary so a concept
// split between two chunks is captured whole in at least one. maxChunks caps the
// work (and the mean-pool dilution) on a pathologically large file; its full text
// is still covered by BM25.
const (
	chunkChars   = 1000
	chunkOverlap = 150
	maxChunks    = 24
)

// chunkText splits text into overlapping, token-safe pieces on word boundaries.
// A text at or under chunkChars returns a single chunk. The split prefers the
// last whitespace before the limit so a chunk never cuts mid-word.
func chunkText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	if len(runes) <= chunkChars {
		return []string{text}
	}
	var chunks []string
	for start := 0; start < len(runes) && len(chunks) < maxChunks; {
		end := start + chunkChars
		if end >= len(runes) {
			chunks = append(chunks, strings.TrimSpace(string(runes[start:])))
			break
		}
		// Break at the last whitespace before the limit, so we don't cut a word.
		cut := end
		for i := end; i > start+chunkChars/2; i-- {
			if isSpace(runes[i]) {
				cut = i
				break
			}
		}
		chunks = append(chunks, strings.TrimSpace(string(runes[start:cut])))
		next := cut - chunkOverlap
		if next <= start { // guarantee forward progress
			next = cut
		}
		start = next
	}
	return chunks
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r'
}
