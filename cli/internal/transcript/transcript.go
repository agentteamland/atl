// Package transcript discovers and reads Claude Code session transcripts.
//
// Transcripts are JSONL files under ~/.claude/projects/<slug>/, one JSON
// record per line. The drain reads the assistant's text content (where capture
// markers are emitted) and feeds it to the queue. The coarse modtime cursor is
// only a performance filter (skip transcripts untouched since the last tick) —
// exactly-once correctness comes from the queue's marker-hash dedup, which
// spans processed items via a tombstone (a still-growing session file can be
// re-scanned whole after its markers were acked, so the dedup must outlive
// deletion). v1's separate per-marker processed-hash state file is gone; the
// same guarantee now lives inside the durable queue.
package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// SlugForPath converts an absolute project path to the Claude Code transcript
// slug. Claude Code replaces every NON-alphanumeric character with a hyphen — not
// just the path separator — so a path component with a dot (`.claude`, a version
// dir) or any other punctuation slugs the same way it does on disk. Getting this
// wrong silently breaks capture for every project whose path contains a dot,
// including all `.claude/worktrees/...` sessions (verified against real slugs on
// disk: `/Users/x/p/.claude/worktrees/y` -> `-Users-x-p--claude-worktrees-y`).
func SlugForPath(absPath string) string {
	s := filepath.ToSlash(absPath)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
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

// transcriptEvent is the subset of a JSONL record the drain reads. IsMeta marks
// harness-injected records (skill-content injections, command caveats) that carry
// a user role but are not the user speaking.
type transcriptEvent struct {
	IsMeta  bool `json:"isMeta"`
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

// Turn is one conversational turn — user or assistant prose — for the /drain
// correction-mining step. Tool calls and tool results never become a Turn.
type Turn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// ExtractFlow reads a transcript and returns the ordered user + assistant prose
// turns (only type:"text" content contributes, so tool calls and tool results
// are dropped — they're noise for correction-mining). Where ExtractText sees
// only the assistant surface (where markers live), this sees the back-and-forth
// the drain mines for user corrections, reverts, and repeated mistakes the agent
// never marked. Empty turns are skipped; malformed lines are skipped rather than
// failing the whole read.
func ExtractFlow(path string) ([]Turn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var turns []Turn
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
		if ev.IsMeta {
			continue // harness-injected (a skill's SKILL.md dump, a command caveat) — user-role but not the user speaking; noise for mining and for the watchdog alike
		}
		role := ev.Message.Role
		if role != "user" && role != "assistant" {
			continue
		}
		text := strings.TrimSpace(contentText(ev.Message.Content))
		if text == "" {
			continue
		}
		turns = append(turns, Turn{Role: role, Text: text})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return turns, nil
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

// Stretch measures the live session's capture "dry stretch": how much
// conversation has accumulated since the last assistant turn that carried a
// capture marker. The capture watchdog fires on it — the deterministic
// detector for the pipeline's one non-deterministic link (a marker the agent
// never wrote is invisible to everything downstream).
type Stretch struct {
	Turns int    // logical assistant turns since the last marker-bearing turn
	Chars int    // user-authored chars since the last marker-bearing turn
	Key   string // fire-once latch key: "<file-base>:<marker-turn-ordinal>"
}

// DryStretch reads a session transcript and measures its dry stretch. It is a
// pure function of the file — no incremental counters to drift across re-scans
// (tick re-reads whole files from a modtime cursor, so persisted counters would
// double-count). Consecutive assistant messages (tool-use interleaving splits
// one logical reply into several records) collapse into one logical turn. The
// Key changes whenever a new marker-bearing turn appears or the session file
// changes, which is exactly when a fired watchdog should re-arm.
func DryStretch(path string, isMarker func(string) bool) (Stretch, error) {
	turns, err := ExtractFlow(path)
	if err != nil {
		return Stretch{}, err
	}
	// Collapse consecutive same-role turns into logical turns.
	var logical []Turn
	for _, t := range turns {
		if n := len(logical); n > 0 && logical[n-1].Role == t.Role {
			logical[n-1].Text += "\n" + t.Text
			continue
		}
		logical = append(logical, t)
	}
	markerTurns := 0 // ordinal count of marker-bearing assistant turns seen
	st := Stretch{}
	for _, t := range logical {
		if t.Role == "assistant" {
			if isMarker(t.Text) {
				markerTurns++
				st.Turns, st.Chars = 0, 0 // reset: the stretch restarts after a marker
				continue
			}
			st.Turns++
			continue
		}
		// Only count what the user actually authored. Machine-injected user-role
		// records that survive the isMeta skip — task notifications, command
		// wrappers, hook envelopes — all open with a tag; a person's typed message
		// essentially never does. Without this, a couple of background-task
		// notifications (~10k chars each) would satisfy the chars threshold on
		// their own and the gate would degrade to turns-only. Runes, not bytes,
		// so multi-byte text (Turkish input) doesn't inflate the count.
		if strings.HasPrefix(t.Text, "<") {
			continue
		}
		st.Chars += utf8.RuneCountInString(t.Text)
	}
	st.Key = filepath.Base(path) + ":" + strconv.Itoa(markerTurns)
	return st, nil
}
