// Package drain transfers captured markers into the durable queue.
//
// This is the first half of the learning loop: markers parsed from
// conversation text are enqueued exactly once. "Exactly once" is guaranteed by
// the queue's marker-hash dedup, not by a transcript cursor — so draining the
// same text twice is safe and enqueues nothing the second pass. (A per-source
// cursor is a later performance optimization to avoid re-parsing, not a
// correctness requirement.)
package drain

import (
	"github.com/agentteamland/atl/cli/internal/marker"
	"github.com/agentteamland/atl/cli/internal/queue"
)

// Result reports what a drain pass did.
type Result struct {
	Found    int            // markers parsed from the text
	Enqueued int            // newly enqueued (already-present markers are not counted)
	NearMiss map[string]int // dropped channel -> marker count; a likely typo
}

// Drain parses markers from text and transfers them into project's queue,
// accepting only the channels in known — the active set, core's own plus the
// ones installed teams declare. Idempotent: re-draining the same text enqueues
// nothing new.
//
// The allowlist is applied here, at the boundary where team-declared input
// enters the queue, rather than inside the store: the queue is generic
// infrastructure that must stay able to carry any future channel, so it is the
// two write seams (this one and `atl learnings _enqueue`) that decide what a
// valid channel is. An undeclared channel is never enqueued — otherwise a typo
// leaves pending items in the project's bucket that no drain will ever claim,
// inflating `learnings status` and doctor's backlog check forever.
func Drain(text, project string, store *queue.Store, known []string) (Result, error) {
	var r Result
	found, nearMiss := marker.Scan(text, known)
	r.NearMiss = nearMiss
	for _, mk := range found {
		ch := queue.Channel(mk.Channel)
		// Structurally guaranteed by Scan, asserted here so the guarantee survives
		// a future change to either side rather than holding by coincidence.
		if !allowed(known, mk.Channel) {
			continue
		}
		r.Found++
		added, err := store.Enqueue(project, queue.Item{
			ID:      queue.NewID(ch, mk.Body),
			Channel: ch,
			Payload: mk.Body,
		})
		if err != nil {
			return r, err
		}
		if added {
			r.Enqueued++
		}
	}
	return r, nil
}

func allowed(known []string, channel string) bool {
	for _, k := range known {
		if k == channel {
			return true
		}
	}
	return false
}
