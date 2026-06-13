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

// markerRe matches `<!-- channel: body -->`. The (?s) flag lets the body span
// multiple lines (multi-line YAML markers); the non-greedy body stops at the
// first `-->` so adjacent markers don't merge into one.
var markerRe = regexp.MustCompile(`(?s)<!--\s*([a-z][a-z0-9-]*)\s*:\s*(.*?)\s*-->`)

// known is the set of channels the drain recognizes. A comment that merely
// looks marker-shaped (some other `<!-- x: y -->`) is not necessarily ours, so
// unknown channels are ignored rather than enqueued as junk.
var known = map[string]bool{
	"learning":     true,
	"profile-fact": true,
}

// Parse extracts all recognized markers from text, in order of appearance.
// Unknown-channel and empty-body markers are skipped.
func Parse(text string) []Marker {
	matches := markerRe.FindAllStringSubmatch(text, -1)
	out := make([]Marker, 0, len(matches))
	for _, m := range matches {
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
