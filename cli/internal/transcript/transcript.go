// Package transcript discovers and reads Claude Code session transcripts.
//
// Transcripts are JSONL files under ~/.claude/projects/<slug>/, one JSON
// record per line. The drain reads the assistant's text content (where capture
// markers are emitted) and feeds it to the queue. v2 keeps only the coarse
// modtime cursor for performance — correctness (exactly-once) comes from the
// queue's marker-hash dedup, so v1's per-marker processed-hash state file is
// gone.
package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SlugForPath converts an absolute project path to the Claude Code transcript
// slug: every path separator becomes a hyphen.
func SlugForPath(absPath string) string {
	return strings.ReplaceAll(filepath.ToSlash(absPath), "/", "-")
}

// ProjectDir returns the transcript directory for projectRoot:
// ~/.claude/projects/<slug>.
func ProjectDir(projectRoot string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects", SlugForPath(projectRoot)), nil
}

// File is a discovered transcript.
type File struct {
	Path    string
	ModTime time.Time
}

// Find returns .jsonl transcripts in dir modified strictly after since, oldest
// first. "Strictly after" means a static transcript already drained at the
// cursor time is not re-scanned; a still-growing one (modtime advancing) is.
// A missing dir yields no files (not an error). A zero since returns all.
func Find(dir string, since time.Time) ([]File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		if !since.IsZero() && !info.ModTime().After(since) {
			continue
		}
		files = append(files, File{Path: filepath.Join(dir, e.Name()), ModTime: info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Before(files[j].ModTime)
	})
	return files, nil
}

// transcriptEvent is the subset of a JSONL record the drain reads.
type transcriptEvent struct {
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// ExtractText reads a transcript file and returns the concatenated assistant
// text content (the surface markers are emitted into). User messages, tool
// calls, and tool results are ignored. Malformed lines are skipped rather than
// failing the whole drain.
func ExtractText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var sb strings.Builder
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // allow long transcript lines
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev transcriptEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.Message.Role != "assistant" {
			continue
		}
		sb.WriteString(contentText(ev.Message.Content))
		sb.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// contentText extracts text from a message content field, which may be a plain
// string (legacy) or an array of typed parts (only type:"text" contributes).
func contentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var sb strings.Builder
		for _, p := range parts {
			if p.Type == "text" {
				sb.WriteString(p.Text)
				sb.WriteByte('\n')
			}
		}
		return sb.String()
	}
	return ""
}
