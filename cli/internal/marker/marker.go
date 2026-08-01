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

// Scan extracts the markers whose channel is in known, and separately reports
// channels it DROPPED that are within one edit of a known name — a near-miss,
// which is almost always a mistyped marker prefix.
//
// The allowlist is injected rather than held here: which channels exist is a
// question about what teams are installed, and this package stays pure text —
// it reads no manifests and no filesystem.
//
// Dropping an unknown channel silently is deliberate and not laziness: a
// transcript is full of comments that match the marker SHAPE without being
// markers. ATL's own always-loaded blocks — <!-- wiki:index -->,
// <!-- brainstorm:active:start -->, <!-- atl:managed --> — all parse as
// `channel: body`, so reporting every unknown channel would produce a
// guaranteed false-positive on every prompt in every ATL project. The near-miss
// filter keeps the one case that matters (a typo in a marker the agent meant to
// write) and none of the noise.
//
// Boundaries are scanned by hand so a marker whose `-->` is missing (a truncated
// or mistyped marker) is discarded on its own rather than consuming everything up
// to the NEXT marker's close — which would garble one learning and silently drop
// the following one. When a fresh `<!--` appears before the current marker's
// `-->`, the current marker is treated as unclosed and skipped, and scanning
// resumes at that inner open.
func Scan(text string, known []string) (found []Marker, nearMiss map[string]int) {
	allowed := make(map[string]bool, len(known))
	for _, k := range known {
		allowed[k] = true
	}
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
		body := strings.TrimSpace(m[2])
		if !allowed[channel] {
			// Only a body-carrying near-miss is worth reporting: an empty-bodied
			// comment is chrome, not a marker someone meant to write.
			if body != "" && nearKnown(channel, known) {
				if nearMiss == nil {
					nearMiss = map[string]int{}
				}
				nearMiss[channel]++
			}
			continue
		}
		if body == "" {
			continue
		}
		out = append(out, Marker{Channel: channel, Body: body})
	}
	return out, nearMiss
}

// Parse is Scan's first return — every recognized marker, in order.
func Parse(text string, known []string) []Marker {
	found, _ := Scan(text, known)
	return found
}

// Has reports whether text carries at least one marker on the given channel.
//
// It exists because a channel-agnostic "any marker at all" test is the wrong
// question for anything that watches ONE channel: a learning marker would
// satisfy it and mask a missed profile-fact. The capture watchdog measures its
// dry stretch per channel through this.
func Has(text, channel string) bool {
	return len(Parse(text, []string{channel})) > 0
}

// nearKnown reports whether channel is within one edit of a known channel name —
// the typo filter that separates "a marker prefix someone fat-fingered" from
// "an unrelated HTML comment". Case-sensitive: marker channels are lowercase by
// construction (see innerRe), so a case difference is already an edit.
func nearKnown(channel string, known []string) bool {
	for _, k := range known {
		if NearlyEqual(channel, k) {
			return true
		}
	}
	return false
}

// NearlyEqual reports whether a and b differ by at most one insertion,
// deletion, or substitution. Bounded at 1, so it is a single linear scan rather
// than a full Levenshtein matrix. Exported so a caller reporting a near-miss can
// name the channel that was probably meant, using the same predicate that
// classified it as a near-miss in the first place.
func NearlyEqual(a, b string) bool {
	if len(a)-len(b) > 1 || len(b)-len(a) > 1 {
		return false
	}
	if len(a) > len(b) {
		a, b = b, a // a is now the shorter (or equal-length) one
	}
	i, j, edits := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			i++
			j++
			continue
		}
		edits++
		if edits > 1 {
			return false
		}
		if len(a) == len(b) {
			i++ // substitution
		}
		j++ // insertion into b (or the substituted byte)
	}
	return edits+(len(b)-j) <= 1
}
