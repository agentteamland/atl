// Package queue is the durable, multi-channel work queue at the heart of the
// v2 learning loop.
//
// Markers captured in conversation are transferred here exactly once
// (idempotent by item ID), processed by per-channel handlers, then deleted.
// Deletion tombstones the item ID in a processed-set, so a re-scanned transcript
// can never re-enqueue an already-drained marker. This keeps the v1 re-report
// bug class (H-3) dead: reports come from the queue, and the coarse modtime
// cursor is only a performance filter — exactly-once holds because Enqueue
// dedups against BOTH the pending items and the processed tombstones, not just
// the (deleted-on-ack) pending set.
//
// Backed by bbolt: a single embedded file (~/.atl/queue.db), no server, no
// CGo — which keeps cross-platform builds trivial (the v2 script-only
// distribution). One file holds every project's queue, isolated into
// per-project buckets.
package queue

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Channel identifies the kind of work an item carries. The queue is generic:
// one infrastructure, many per-channel processors.
type Channel string

const (
	ChannelLearning    Channel = "learning"
	ChannelProfileFact Channel = "profile-fact"
)

// Item is a single unit of queued work.
type Item struct {
	ID         string    `json:"id"`          // dedup key — same marker ⇒ same ID
	Channel    Channel   `json:"channel"`     // which processor handles it
	Payload    string    `json:"payload"`     // the marker body / work content
	EnqueuedAt time.Time `json:"enqueued_at"` // for stable ordering
}

// NewID derives a stable dedup ID from a channel + payload. The same marker
// transferred twice produces the same ID and dedups on enqueue — the
// marker-hash-dedup pattern that makes transfer exactly-once.
func NewID(channel Channel, payload string) string {
	sum := sha256.Sum256([]byte(string(channel) + "\x00" + payload))
	return hex.EncodeToString(sum[:])
}

// Store is a bbolt-backed queue.
type Store struct {
	db *bolt.DB
}

// DefaultPath returns the standard queue location (~/.atl/queue.db), creating
// the ~/.atl directory if needed.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".atl")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "queue.db"), nil
}

// Open opens (creating if needed) the queue database at path.
func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open queue db: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Enqueue adds it to project's queue, idempotently by it.ID. It reports
// whether the item was newly added (false = a same-ID item already existed, so
// this call is a no-op — the dedup that makes marker transfer exactly-once).
func (s *Store) Enqueue(project string, it Item) (added bool, err error) {
	if it.ID == "" {
		return false, fmt.Errorf("enqueue: empty item ID")
	}
	if it.Channel == "" {
		return false, fmt.Errorf("enqueue: empty channel")
	}
	if it.EnqueuedAt.IsZero() {
		it.EnqueuedAt = time.Now().UTC()
	}
	err = s.db.Update(func(tx *bolt.Tx) error {
		// Tombstone check: a marker already processed (acked, so its queue item
		// was deleted) must not re-enqueue when a transcript is re-scanned. This
		// is what makes transfer exactly-once across a re-scan — the modtime
		// cursor is coarse (a still-growing session file re-reads whole).
		if pb := tx.Bucket([]byte(processedBucket)); pb != nil {
			if pb.Get(processedKey(project, it.ID)) != nil {
				return nil // already processed — dedup no-op
			}
		}
		b, err := tx.CreateBucketIfNotExists([]byte(project))
		if err != nil {
			return err
		}
		if b.Get([]byte(it.ID)) != nil {
			return nil // already pending — dedup no-op
		}
		buf, err := json.Marshal(it)
		if err != nil {
			return err
		}
		added = true
		return b.Put([]byte(it.ID), buf)
	})
	if err != nil {
		return false, fmt.Errorf("enqueue: %w", err)
	}
	return added, nil
}

// Pending returns all queued items for project, sorted by EnqueuedAt then ID.
// If channel is non-empty, only items on that channel are returned.
func (s *Store) Pending(project string, channel Channel) ([]Item, error) {
	var items []Item
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(project))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			var it Item
			if err := json.Unmarshal(v, &it); err != nil {
				return err
			}
			if channel == "" || it.Channel == channel {
				items = append(items, it)
			}
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("pending: %w", err)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].EnqueuedAt.Equal(items[j].EnqueuedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].EnqueuedAt.Before(items[j].EnqueuedAt)
	})
	return items, nil
}

// Delete removes a processed item (processed-then-deleted) and tombstones its
// ID in the processed-set, so a later transcript re-scan can never re-enqueue
// this already-drained marker (the re-report bug: ack deleted the item, so the
// pending-dedup forgot it). Only the ID hash is retained; the payload is freed.
// Idempotent: deleting a missing item still records the tombstone (Delete is the
// ack path, so the ID is always one we've decided is processed).
func (s *Store) Delete(project, id string) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte(project)); b != nil {
			if err := b.Delete([]byte(id)); err != nil {
				return err
			}
		}
		pb, err := tx.CreateBucketIfNotExists([]byte(processedBucket))
		if err != nil {
			return err
		}
		ts, err := time.Now().UTC().MarshalBinary()
		if err != nil {
			return err
		}
		return pb.Put(processedKey(project, id), ts)
	})
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// Counts returns the pending item count per channel for project. This is what
// `atl learnings status` and the SessionStart count read from — correct by
// construction, never inferred from re-scanning.
func (s *Store) Counts(project string) (map[Channel]int, error) {
	items, err := s.Pending(project, "")
	if err != nil {
		return nil, err
	}
	counts := map[Channel]int{}
	for _, it := range items {
		counts[it.Channel]++
	}
	return counts, nil
}

// cursorBucket holds per-project last-tick timestamps. It is a reserved bucket
// name a project key (an absolute path) can never collide with.
const cursorBucket = "__cursors__"

// processedBucket holds tombstones for acked (processed-then-deleted) item IDs
// so a re-scanned transcript can't re-enqueue an already-drained marker. Keyed
// by project + "\x00" + id (one bucket across all projects, like __cursors__);
// the value is the processed timestamp (informational). A reserved name a
// project key (an absolute path) can never collide with.
const processedBucket = "__processed__"

// processedKey is the composite (project, id) tombstone key. The NUL separator
// can't appear in an absolute path or a hex id, so keys never alias.
func processedKey(project, id string) []byte {
	return []byte(project + "\x00" + id)
}

// Cursor returns the last-tick time for project (zero if never ticked). It is
// the coarse modtime filter for transcript scanning — only a performance
// optimization to avoid re-parsing old transcripts. Exactly-once correctness
// comes from Enqueue's dedup, not from this cursor.
func (s *Store) Cursor(project string) (time.Time, error) {
	var t time.Time
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(cursorBucket))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(project))
		if v == nil {
			return nil
		}
		return t.UnmarshalBinary(v)
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("cursor: %w", err)
	}
	return t, nil
}

// SetCursor records the last-tick time for project.
func (s *Store) SetCursor(project string, ts time.Time) error {
	buf, err := ts.MarshalBinary()
	if err != nil {
		return fmt.Errorf("set cursor: %w", err)
	}
	err = s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(cursorBucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(project), buf)
	})
	if err != nil {
		return fmt.Errorf("set cursor: %w", err)
	}
	return nil
}
