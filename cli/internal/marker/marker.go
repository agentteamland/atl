// Package marker extracts capture markers from conversation text.
//
// Markers are silent HTML comments the assistant emits inline during a
// session — the ergonomic, zero-noise capture surface that survives into v2.
// A marker looks like:
//
//	<!-- learning: the user prefers Node for APIs -->
//
//	<!-- profile-fact:
//	  entity: ahmet
//	  field: traits.fears
//	  value: ["confrontation"]
//	-->
//
// The drain step transfers parsed markers into the durable queue (exactly
// once, via the queue's marker-hash dedup).
package marker

import (
	"regexp"
	"strings"
)

// Marker is a single captured marker.
type Marker struct {
	Channel string // e.g. "learning", "profile-fact"
	Body    string // the trimmed marker body — the queue payload
}

// innerRe matches the `channel: body` content BETWEEN one marker's `<!--` and
// its `-->`. The (?s) flag lets the body span multiple lines (multi-line YAML
// markers). Scanning the open/close boundaries is done by hand (below) rather
// than in the regex, so an unclosed marker can't swallow the marker after it.
var innerRe = regexp.MustCompile(`(?s)^\s*([a-z][a-z0-9-]*)\s*:\s*(.*?)\s*$`)

// known is the set of channels the drain recognizes. A comment that merely
// looks marker-shaped (some other `<!-- x: y -->`) is not necessarily ours, so
// unknown channels are ignored rather than enqueued as junk.
var known = map[string]bool{
	"learning":     true,
	"profile-fact": true,
}

// Parse extracts all recognized markers from text, in order of appearance.
// Unknown-channel and empty-body markers are skipped.
//
// Boundaries are scanned by hand so a marker whose `-->` is missing (a truncated
// or mistyped marker) is discarded on its own rather than consuming everything up
// to the NEXT marker's close — which would garble one learning and silently drop
// the following one. When a fresh `<!--` appears before the current marker's
// `-->`, the current marker is treated as unclosed and skipped, and scanning
// resumes at that inner open.
func Parse(text string) []Marker {
	var out []Marker
	for i := 0; i < len(text); {
		rel := strings.Index(text[i:], "<!--")
		if rel < 0 {
			break
		}
		open := i + rel
		rest := text[open+4:]
		closeAt := strings.Index(rest, "-->")
		if closeAt < 0 {
			break // no close anywhere ahead → the remaining opens are all unclosed
		}
		if nextOpen := strings.Index(rest, "<!--"); nextOpen >= 0 && nextOpen < closeAt {
			i = open + 4 + nextOpen // unclosed marker → skip it, resume at the inner open
			continue
		}
		inner := rest[:closeAt]
		i = open + 4 + closeAt + 3
		m := innerRe.FindStringSubmatch(inner)
		if m == nil {
			continue
		}
		channel := m[1]
		if !known[channel] {
			continue
		}
		body := strings.TrimSpace(m[2])
		if body == "" {
			continue
		}
		out = append(out, Marker{Channel: channel, Body: body})
	}
	return out
}
