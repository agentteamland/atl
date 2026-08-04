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
	"bytes"
	"encoding/json"
	"io"
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

// File is a discovered transcript. Size comes from the same stat the walk
// already does, so a resuming mine can tell "nothing appended" from "unread
// bytes" without opening the file.
type File struct {
	Path    string
	ModTime time.Time
	Size    int64
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
		files = append(files, File{Path: filepath.Join(dir, e.Name()), ModTime: info.ModTime(), Size: info.Size()})
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
	IsMeta bool `json:"isMeta"`
	// IsCompactSummary marks the recap the harness injects as a user-role message
	// when it compacts a long session. It is machine-authored and enormous
	// (14-18k chars observed), so counting it as user input is what turned the
	// watchdog's char gate into a formality after any compaction.
	IsCompactSummary bool `json:"isCompactSummary"`
	Message          struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// interruptMarker is the literal user-role string the harness writes when the
// user cancels a turn. It carries no flag of its own — unlike a compact summary
// — so it is matched by text.
const interruptMarker = "[Request interrupted"

const (
	// maxRecordBytes caps one JSONL record. Records genuinely run large (a tool
	// result echoing a file — 1.2 MB observed in a real transcript), so this is a
	// memory bound on a pathological line (a pasted blob, a base64 payload), not
	// a limit anything normal approaches.
	maxRecordBytes = 8 * 1024 * 1024
	// readBufBytes is the read-ahead window; a record larger than it is assembled
	// across several reads.
	readBufBytes = 64 * 1024
)

// scanLines calls fn for each non-empty newline-delimited record in r, and
// returns how many records were skipped for exceeding maxRecordBytes plus the
// offset of the last complete record boundary (see below).
//
// Skipping is the point. An over-long record used to abort the whole file
// (bufio.Scanner cannot resume after a too-long token), which took out the
// mining input AND the capture watchdog for that entire session — and the
// watchdog swallows a read error, so the loss was silent: exactly the failure
// shape the watchdog exists to remove, one layer down. One pathological line
// costs that one record instead, and the count makes the loss reportable.
//
// start is the byte offset r begins at, so a caller that seeked can resume its
// accounting; each fn call receives the offset just past that record's newline.
// The returned offset is that of the last NEWLINE-TERMINATED record — a trailing
// record with no newline is still handed to fn (the file is being appended to, or
// a write was cut short) but never counted as consumed, because the writer may
// still be extending it. A resuming caller must re-read it rather than skip past
// a half-written line: re-reading is safe (the queue dedups by content hash),
// skipping is the silent loss this reader exists to remove.
//
// It reads with ReadSlice rather than ReadLine because ReadSlice keeps the
// delimiter, and only a byte count that includes every terminator can produce an
// offset a later run can seek to. ReadLine strips "\n" — and a preceding "\r" —
// leaving no way to tell how many bytes it consumed.
//
// The slice passed to fn is only valid until the next record (the same contract
// bufio.Scanner's Bytes had); callers unmarshal it immediately.
func scanLines(r io.Reader, start int64, fn func(record []byte, end int64)) (int, int64, error) {
	br := bufio.NewReaderSize(r, readBufBytes)
	var rec []byte
	over := false // this record is already past the cap: consume it, don't buffer it
	skipped := 0
	pos := start      // bytes consumed, terminators included
	complete := start // pos after the last newline-terminated record
	emit := func() {
		if over {
			skipped++
			over = false
		} else if r := trimEOL(rec); len(r) > 0 {
			fn(r, pos)
		}
		rec = rec[:0]
	}
	for {
		chunk, err := br.ReadSlice('\n')
		pos += int64(len(chunk))
		if !over && len(rec)+len(chunk) > maxRecordBytes {
			over, rec = true, nil
		}
		if !over {
			rec = append(rec, chunk...)
		}
		switch err {
		case nil:
			emit()
			complete = pos
		case bufio.ErrBufferFull:
			// the record continues past the read window
		case io.EOF:
			emit() // the trailing record, if any — deliberately not counted in complete
			return skipped, complete, nil
		default:
			return skipped, complete, err
		}
	}
}

// trimEOL drops a record's trailing newline (and a preceding carriage return),
// which ReadSlice keeps. A blank line trims to empty and is therefore skipped,
// matching the old ReadLine-based behavior.
func trimEOL(rec []byte) []byte {
	rec = bytes.TrimSuffix(rec, []byte("\n"))
	return bytes.TrimSuffix(rec, []byte("\r"))
}

// ExtractText reads a transcript file and returns the concatenated assistant
// text content (the surface markers are emitted into) plus the number of
// over-long records skipped. User messages, tool calls, and tool results are
// ignored. Malformed lines — and over-long ones — are skipped rather than
// failing the whole drain.
func ExtractText(path string) (string, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	var sb strings.Builder
	skipped, _, err := scanLines(f, 0, func(record []byte, _ int64) {
		var ev transcriptEvent
		if err := json.Unmarshal(record, &ev); err != nil {
			return
		}
		if ev.Message.Role != "assistant" {
			return
		}
		sb.WriteString(contentText(ev.Message.Content))
		sb.WriteByte('\n')
	})
	if err != nil {
		return "", skipped, err
	}
	return sb.String(), skipped, nil
}

// Turn is one conversational turn — user or assistant prose — for the /drain
// correction-mining step. Tool calls and tool results never become a Turn.
type Turn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// ExtractFlow reads a transcript and returns the ordered user + assistant prose
// turns (only type:"text" content contributes, so tool calls and tool results
// are dropped — they're noise for correction-mining) plus the number of
// over-long records skipped. Where ExtractText sees only the assistant surface
// (where markers live), this sees the back-and-forth the drain mines for user
// corrections, reverts, and repeated mistakes the agent never marked. Empty
// turns are skipped; malformed lines — and over-long ones — are skipped rather
// than failing the whole read.
func ExtractFlow(path string) ([]Turn, int, error) {
	fl, err := ExtractFlowFrom(path, 0, 0)
	return fl.Turns, fl.Skipped, err
}

// Flow is one resumable read of a transcript's prose: the turns extracted, the
// offset a following read should resume from, and whether the byte budget cut
// the read short.
type Flow struct {
	Turns   []Turn
	Next    int64 // resume offset — the boundary after the last turn returned
	Pending bool  // the budget ran out with records left in this file
	Skipped int   // over-long records dropped
}

// ExtractFlowFrom reads a transcript's prose forward from a byte offset, which
// is what lets a mine resume where the last one stopped instead of re-reading a
// tail. offset must be a boundary a previous call returned in Next (0 for the
// whole file); an offset past the end yields nothing.
//
// maxProseBytes budgets the prose returned (0 = unbudgeted). The budget is
// enforced against the turns collected, and Next tracks the collected turns
// exactly, so a caller that stops early resumes at the first turn it did NOT
// receive — the property that makes several bounded mines add up to full
// coverage of a session rather than repeated samples of its tail.
//
// The remaining semantics are ExtractFlow's: only type:"text" content
// contributes (tool calls and results are noise for correction-mining), empty
// turns are skipped, and malformed or over-long records are skipped rather than
// failing the read.
func ExtractFlowFrom(path string, offset int64, maxProseBytes int) (Flow, error) {
	f, err := os.Open(path)
	if err != nil {
		return Flow{Next: offset}, err
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return Flow{Next: offset}, err
		}
	}

	fl := Flow{Next: offset}
	prose := 0
	skipped, complete, err := scanLines(f, offset, func(record []byte, end int64) {
		if fl.Pending {
			return // budget already spent — consume the rest without collecting it
		}
		var ev transcriptEvent
		if err := json.Unmarshal(record, &ev); err != nil {
			return
		}
		if ev.IsMeta || ev.IsCompactSummary {
			return // harness-injected (a skill's SKILL.md dump, a command caveat, a compaction recap) — user-role but not the user speaking; noise for mining and for the watchdog alike
		}
		role := ev.Message.Role
		if role != "user" && role != "assistant" {
			return
		}
		text := strings.TrimSpace(contentText(ev.Message.Content))
		if text == "" {
			return
		}
		if role == "user" && strings.HasPrefix(text, interruptMarker) {
			return // the harness's own cancellation notice, not something the user typed
		}
		// The budget is checked before appending, but the FIRST turn is always
		// taken: one oversized turn must not return an empty read forever, which
		// would wedge the cursor at that record.
		if maxProseBytes > 0 && len(fl.Turns) > 0 && prose+len(text) > maxProseBytes {
			fl.Pending = true
			return
		}
		prose += len(text)
		fl.Turns = append(fl.Turns, Turn{Role: role, Text: text})
		fl.Next = end
	})
	fl.Skipped = skipped
	if err != nil {
		return fl, err
	}
	if !fl.Pending {
		// Nothing was cut short, so everything up to the last complete record has
		// been seen — including records that contributed no turn (tool traffic,
		// harness injections). Without this the cursor would stall on a stretch
		// that produced no prose at all.
		fl.Next = complete
	}
	return fl, nil
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
	// A skipped over-long record simply isn't measured: the stretch is a
	// heuristic, and reporting the skip belongs to the two read surfaces that
	// print (tick's drain pass and `atl learnings transcript`), not to a nudge.
	turns, _, err := ExtractFlow(path)
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
