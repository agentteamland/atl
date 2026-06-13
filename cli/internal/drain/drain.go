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
	Found    int // markers parsed from the text
	Enqueued int // newly enqueued (already-present markers are not counted)
}

// Drain parses markers from text and transfers them into project's queue.
// Idempotent: re-draining the same text enqueues nothing new.
func Drain(text, project string, store *queue.Store) (Result, error) {
	var r Result
	for _, mk := range marker.Parse(text) {
		r.Found++
		ch := queue.Channel(mk.Channel)
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
