// Package sprintsignal answers one question about a board-backed project: does it
// have open work and no active sprint?
//
// It is the deterministic half of the session-start signal the ceremony-adoption
// decision names as "what makes it stick" — a board that has drifted out of a
// sprint says nothing about itself, so nobody notices until someone thinks to
// look. The half that must be deterministic is the NOTICING; opening the next
// sprint stays a human act.
//
// The predicate comes from the delivery-team's own sprint resolution
// (backends/github/adapter.md §5, sprint-plan/SKILL.md): under mode "flow" the
// sprint carrier is a sprint:<n> LABEL, the current sprint is the highest ordinal
// COMPARED AS AN INTEGER, and it stays current until it is reviewed — reviewed
// meaning its docs/sprints/sprint-<n>-review.md page exists. So "no active sprint"
// is: no item carries a sprint: label at all, or the highest one is already
// reviewed.
//
// Two halves, split by where the answer lives. Which sprints are REVIEWED is a
// local filesystem read (the GitHub backend's durable-knowledge store is in-repo
// /docs), so it is free and always current. Which ordinal is HIGHEST, and whether
// any work is open, is a board read — network, slow, and failable — so it runs
// out of band and leaves a cached Verdict behind for session-start to print from.
package sprintsignal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ScanLimit caps the issue list one scan reads.
//
// The interface's "list means all" rule makes a partial read a bug, not a smaller
// answer — so the cap is a TRUNCATION DETECTOR, not a page size: a result that
// fills it means this scan cannot know it saw the highest sprint label, and Scan
// fails rather than caching a verdict it can't stand behind. That resolves to
// silence, which is the correct direction for a signal whose false positive
// ("no active sprint" during one) costs trust and whose false negative costs
// nothing but the status quo.
const ScanLimit = 1000

// candidateLabel marks a /request item awaiting the product owner's accept
// (backend-interface concept #14). A candidate is excluded from the ready
// frontier, so it is not open WORK — counting it would let a board with nothing
// but unaccepted requests report that a sprint is overdue.
const candidateLabel = "candidate"

// sprintLabelPrefix is the flow-mode sprint carrier's prefix (adapter §5).
const sprintLabelPrefix = "sprint:"

// Runner runs an external command and returns its stdout. The seam that keeps the
// board read testable: the real scan shells out to gh, tests inject a fake and
// assert the exact argv and call count. Same shape as promotiongate.Runner —
// deliberately, so the two provider reads in this binary look alike.
type Runner func(name string, args ...string) ([]byte, error)

// Verdict is one board scan's result: the two facts that cannot be answered
// locally. It is cached and read back by a later session, so it carries the time
// it was taken — a verdict has an age, and an old one must be discardable.
type Verdict struct {
	// OpenItems counts open work items excluding unaccepted candidates.
	OpenItems int `json:"openItems"`
	// HighestSprint is the greatest ordinal any item CARRIES, 0 when none does.
	// Read off the items rather than the repo's label definitions, which outlive
	// the issues that carried them and are a strict superset (adapter §5).
	HighestSprint int `json:"highestSprint"`
	// ScannedAt is when the board was read, in UTC.
	ScannedAt time.Time `json:"scannedAt"`
}

// ScanArgs is the exact gh argv one scan issues. Exported so a test can assert
// the call byte-for-byte against non-default coordinates: a broken board READ
// degrades into permanent silence that is indistinguishable from "no signal
// needed", so the read is the half that most needs pinning.
func ScanArgs(owner, repo string) []string {
	return []string{
		"issue", "list",
		"--repo", owner + "/" + repo,
		// --state all, not open: a sprint whose items are ALL closed is still the
		// highest ordinal, and it is still unreviewed until its page exists. Note
		// `gh search issues` rejects "all" — this is `gh issue list`, which takes it.
		"--state", "all",
		"--limit", strconv.Itoa(ScanLimit),
		"--json", "number,state,labels",
	}
}

// Scan reads the board once and returns the verdict. One call answers both
// questions, so there is a single failure point and a single argv to pin.
func Scan(run Runner, owner, repo string) (Verdict, error) {
	if owner == "" || repo == "" {
		return Verdict{}, fmt.Errorf("sprintsignal: config carries no owner/repo")
	}
	out, err := run("gh", ScanArgs(owner, repo)...)
	if err != nil {
		return Verdict{}, fmt.Errorf("sprintsignal: gh issue list: %w", err)
	}
	var items []struct {
		Number int    `json:"number"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(out, &items); err != nil {
		return Verdict{}, fmt.Errorf("sprintsignal: parse gh output: %w", err)
	}
	if len(items) >= ScanLimit {
		return Verdict{}, fmt.Errorf("sprintsignal: board read filled the %d-item cap — "+
			"a partial read cannot rule out a higher sprint ordinal", ScanLimit)
	}

	v := Verdict{ScannedAt: time.Now().UTC()}
	for _, it := range items {
		candidate := false
		for _, l := range it.Labels {
			if l.Name == candidateLabel {
				candidate = true
			}
			// Every item's labels feed the ordinal, open or closed.
			if n, ok := SprintOrdinal(l.Name); ok && n > v.HighestSprint {
				v.HighestSprint = n
			}
		}
		if strings.EqualFold(it.State, "OPEN") && !candidate {
			v.OpenItems++
		}
	}
	return v, nil
}

// SprintOrdinal parses a sprint:<n> carrier label, returning ok=false for
// anything else.
//
// Digits only, and the comparison in Scan is on the returned INT: "sprint:10"
// outranks "sprint:9", where a lexical maximum hands back a stale ordinal. That
// is the adapter's own warning, and it is why this returns a number rather than
// letting a caller compare strings.
func SprintOrdinal(label string) (int, bool) {
	rest, ok := strings.CutPrefix(label, sprintLabelPrefix)
	if !ok || rest == "" {
		return 0, false
	}
	for _, r := range rest {
		if r < '0' || r > '9' {
			return 0, false // sprint:<slug>, sprint:+3 — not the ordinal carrier
		}
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// ReviewPagePath is where the GitHub backend keeps a sprint's review page:
// in-repo docs/sprints/sprint-<n>-review.md (adapter §9). Its EXISTENCE is what
// marks the sprint reviewed.
func ReviewPagePath(projectRoot string, n int) string {
	return filepath.Join(projectRoot, "docs", "sprints", fmt.Sprintf("sprint-%d-review.md", n))
}

// NoActiveSprint reports whether the board has drifted out of a sprint.
//
// Checked against the working tree at PRINT time rather than baked into the
// cached verdict, because a review page written an hour ago should silence this
// signal today — not after the next board scan.
func (v Verdict) NoActiveSprint(projectRoot string) bool {
	if v.HighestSprint <= 0 {
		return true // no sprint has ever been opened
	}
	_, err := os.Stat(ReviewPagePath(projectRoot, v.HighestSprint))
	return err == nil // the highest sprint is reviewed → it is no longer current
}

// Save writes the verdict to path, creating its directory.
func Save(path string, v Verdict) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Load reads a saved verdict. Missing, unreadable, malformed, or older than
// maxAge all return ok=false — one shape for "this machine has nothing it can
// stand behind", so every one of them resolves to silence.
//
// The age bound is the part that is easy to leave out. Without it a board read
// that stops succeeding (gh removed, a token expired, the repo renamed) leaves
// the last good verdict asserting a board state that may be months stale, and
// nothing about the printed line would say so.
func Load(path string, maxAge time.Duration, now time.Time) (Verdict, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Verdict{}, false
	}
	var v Verdict
	if err := json.Unmarshal(b, &v); err != nil {
		return Verdict{}, false
	}
	if v.ScannedAt.IsZero() || now.Sub(v.ScannedAt) > maxAge {
		return Verdict{}, false
	}
	return v, true
}
