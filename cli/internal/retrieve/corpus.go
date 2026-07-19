package retrieve

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Doc is one indexable knowledge page — a wiki page, a journal entry, or an
// agent knowledge-base child. Path is its absolute location (read live at query
// time so injected content is never stale); Title is its display heading; Text
// is the body used for lexical (BM25) and semantic (embedding) matching.
type Doc struct {
	Path  string
	Title string
	Text  string
}

// WalkCorpus reads every Markdown file under each directory (recursively) into a
// Doc, sorted by path for a stable index. A directory that does not exist is
// skipped (a project may have no journal yet); an unreadable file is skipped
// rather than failing the whole walk (retrieval is best-effort). Temp files
// (name ending in .tmp) and empty files are ignored.
func WalkCorpus(dirs []string) ([]Doc, error) {
	var docs []Doc
	seen := map[string]bool{}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil // this corpus root doesn't exist at this layer
				}
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				abs = path
			}
			if seen[abs] {
				return nil // the same file reachable from two overlapping roots
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil // skip an unreadable file, don't fail the walk
			}
			text := strings.TrimSpace(string(b))
			if text == "" {
				return nil
			}
			seen[abs] = true
			docs = append(docs, Doc{Path: abs, Title: titleOf(text, path), Text: text})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	return docs, nil
}

// titleOf returns the first Markdown H1 (`# ...`) in text, or the file's base
// name without extension when there is no heading.
func titleOf(text, path string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return strings.TrimSuffix(filepath.Base(path), ".md")
}
