package digest

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// # Why a finding's key cannot come from its wording
//
// A sweep is an LLM writing prose, and the person it works for writes in a
// language other than English. Measured on 2026-11-09 in a board-backed project:
// one /observe sweep carded a stale backlog row with the tag
// `observe:backlog-virtual-account-shipped`, a later sweep carded the SAME row as
// `observe:backlog-sanal-hesap-shipped`, the check-first query looked for the
// second tag, found nothing, and opened a duplicate (AB#2033 / AB#2137). The
// digest's (sweep, title) key has the identical exposure, since the title is the
// sentence a sweep happens to write that day.
//
// Asking the sweep to "always write the key in English" narrows that and does not
// close it: two English sentences about one finding differ just as easily. So the
// key is derived from what a finding is ABOUT, never from what it SAYS — the lens
// that produced it plus the identifiers its evidence cites. Identifiers are copied
// verbatim rather than translated, which is what makes them language-independent:
// `.atl/backlog.md` and `AB#2033` are the same string in every sweep.

// SubjectKinds is the fixed lens vocabulary a subject key may name. Fixed, and in
// English, because a free-text kind would reintroduce exactly the drift the key
// exists to remove.
var SubjectKinds = []string{
	"ripe-trigger",
	"shipped-vs-designed",
	"growth-risk",
	"decided-unshipped",
	"stale-truth",
	"global-setup",
}

var (
	// A card reference: AB#2033, atl#434, agentteamland/atl#434.
	cardRef = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._/-]*#[0-9]+$`)
	// The optional fragment after a path: a symbol, never a line number. A line
	// pin drifts while the claim it supports stays true, so a key containing one
	// would split a single finding the first time the file above it is edited.
	symbol = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
)

// SubjectKey derives the language-independent key of a sweep finding from its
// lens and the identifiers its evidence cites. The same subject yields the same
// key whatever the sweep wrote about it, in whichever language, in whichever
// order it listed the anchors.
//
// An anchor is one of:
//   - a card reference (`AB#2033`, `atl#434`) — compared case-insensitively;
//   - a path that exists, relative to projectRoot (or absolute, or `~/…`),
//     optionally followed by `#Symbol`.
//
// Anything else is refused rather than hashed: a free phrase is precisely the
// input that varies from sweep to sweep, and accepting it would let the key
// drift again while looking stable.
func SubjectKey(projectRoot, kind string, anchors []string) (string, error) {
	kind = strings.TrimSpace(kind)
	known := false
	for _, k := range SubjectKinds {
		if k == kind {
			known = true
			break
		}
	}
	if !known {
		return "", fmt.Errorf("unknown kind %q — one of: %s", kind, strings.Join(SubjectKinds, ", "))
	}
	if len(anchors) == 0 {
		return "", fmt.Errorf("at least one anchor (a card reference or an existing path) is required")
	}
	seen := map[string]bool{}
	var norm []string
	for _, a := range anchors {
		n, err := normalizeAnchor(projectRoot, a)
		if err != nil {
			return "", err
		}
		if !seen[n] {
			seen[n] = true
			norm = append(norm, n)
		}
	}
	sort.Strings(norm)
	sum := sha1.Sum([]byte(kind + "\n" + strings.Join(norm, "\n")))
	return "atl-observe:" + hex.EncodeToString(sum[:])[:12], nil
}

func normalizeAnchor(projectRoot, raw string) (string, error) {
	a := strings.TrimSpace(raw)
	if a == "" {
		return "", fmt.Errorf("empty anchor")
	}
	// A card reference — unless what precedes the # is a path that exists, in
	// which case `file.go#123` is a line pin wearing a card's shape.
	if cardRef.MatchString(a) {
		if _, err := os.Stat(filepath.Join(projectRoot, a[:strings.LastIndex(a, "#")])); err != nil {
			return strings.ToLower(a), nil
		}
	}
	p, frag := a, ""
	if i := strings.LastIndex(a, "#"); i >= 0 {
		p, frag = a[:i], a[i+1:]
		if !symbol.MatchString(frag) {
			return "", fmt.Errorf("anchor %q: the part after # must be a symbol name, not a line number or a phrase", raw)
		}
	}
	home, _ := os.UserHomeDir()
	var abs string
	switch {
	case p == "~" || strings.HasPrefix(p, "~/"):
		if home == "" {
			return "", fmt.Errorf("anchor %q: no home directory to resolve ~ against", raw)
		}
		abs = filepath.Join(home, strings.TrimPrefix(p, "~"))
	case filepath.IsAbs(p):
		abs = filepath.Clean(p)
	default:
		abs = filepath.Join(projectRoot, p)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("anchor %q is neither a card reference (AB#123) nor an existing path", raw)
	}
	// One spelling per place: inside the project relative to it, under home as
	// ~/…, otherwise absolute — so `./x`, `x` and `<root>/x` hash identically.
	var out string
	if rel, err := filepath.Rel(filepath.Clean(projectRoot), abs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		out = filepath.ToSlash(rel)
	} else if home != "" {
		if rel, err := filepath.Rel(home, abs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			out = "~/" + filepath.ToSlash(rel)
		}
	}
	if out == "" {
		out = filepath.ToSlash(abs)
	}
	if frag != "" {
		out += "#" + frag
	}
	return out, nil
}
