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
// one infrastructure, many per-channel processors. It deliberately knows no
// channel but the platform's own — every other channel is named by an installed
// team's declaration and arrives as a value, so this package never learns a team
// name and never needs a new constant to carry one.
type Channel string

// ChannelLearning is the platform's own capture channel: core ships the
// learning-capture rule and the /drain skill, so it exists on every machine.
const ChannelLearning Channel = "learning"

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

// Projects returns every project key that currently holds pending items, with
// its per-channel counts. It is the ONE read in this package that is not scoped
// to a single project, and it exists for exactly that reason: every other
// surface can only see the bucket you are standing in, so a bucket whose key is
// a directory that no longer exists is invisible everywhere — indistinguishable
// from no bucket at all.
//
// That is not hypothetical. `atl work dispatch` deletes each unit's worktree
// when it completes, and until the key became the repository root (see the CLI's
// projectKey) an autonomous worker's markers were queued under the worktree
// path. Measured 2026-08-08: 13 items stranded across 6 vanished buckets. The
// payloads were intact; nothing could name them.
//
// Reserved buckets are skipped — they are keyed by their own scheme, not by a
// project path.
func (s *Store) Projects() (map[string]map[Channel]int, error) {
	out := map[string]map[Channel]int{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			key := string(name)
			if key == cursorBucket || key == processedBucket || key == watchdogBucket {
				return nil
			}
			counts := map[Channel]int{}
			if err := b.ForEach(func(_, v []byte) error {
				if v == nil {
					return nil
				}
				var it Item
				if json.Unmarshal(v, &it) == nil && it.ID != "" {
					counts[it.Channel]++
				}
				return nil
			}); err != nil {
				return err
			}
			if len(counts) > 0 {
				out[key] = counts
			}
			return nil
		})
	})
	return out, err
}

// Recover moves every pending item out of the named source buckets and into
// target, then removes the emptied source buckets. It is the deliberate
// re-enqueue a stranded bucket needs: keying by the repository root stops NEW
// losses, but nothing looks for the old addresses afterwards, so items already
// stranded need moving before they are written off.
//
// One transaction, so a crash cannot leave an item in neither bucket. No
// tombstone is written for the source — a tombstone means "processed", and these
// were never processed; writing one would make the payload un-re-enqueueable if
// the move were ever repeated from a transcript.
//
// target is skipped if it appears in from, so a caller cannot empty a bucket
// into itself.
func (s *Store) Recover(target string, from []string) (moved int, err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		tb, err := tx.CreateBucketIfNotExists([]byte(target))
		if err != nil {
			return err
		}
		for _, src := range from {
			if src == target {
				continue
			}
			sb := tx.Bucket([]byte(src))
			if sb == nil {
				continue
			}
			var keys [][]byte
			if err := sb.ForEach(func(k, v []byte) error {
				if v == nil {
					return nil
				}
				// Re-stamp nothing: the payload, id and original EnqueuedAt are
				// what make this a rescue rather than a re-capture. An item that
				// already exists in the target is left as the target's copy.
				if tb.Get(k) == nil {
					if err := tb.Put(k, v); err != nil {
						return err
					}
					moved++
				}
				keys = append(keys, append([]byte(nil), k...))
				return nil
			}); err != nil {
				return err
			}
			for _, k := range keys {
				if err := sb.Delete(k); err != nil {
					return err
				}
			}
			if err := tx.DeleteBucket([]byte(src)); err != nil && err != bolt.ErrBucketNotFound {
				return err
			}
		}
		return nil
	})
	return moved, err
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

// SetCursor records the transcript high-water mark for project.
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

// lastTickBucket holds the wall-clock time the maintenance pass (tick /
// session-start) last RAN for a project — distinct from cursorBucket, which
// holds the newest transcript modtime. Conflating the two makes doctor read a
// transcript's age as "time since last tick"; they are different clocks.
const lastTickBucket = "__lasttick__"

// LastTick returns when the maintenance pass last ran for project (zero if
// never). Use this — not Cursor — to judge whether ticks are actually firing.
func (s *Store) LastTick(project string) (time.Time, error) {
	var t time.Time
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(lastTickBucket))
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
		return time.Time{}, fmt.Errorf("last-tick: %w", err)
	}
	return t, nil
}

// SetLastTick records that the maintenance pass ran for project at ts.
func (s *Store) SetLastTick(project string, ts time.Time) error {
	buf, err := ts.MarshalBinary()
	if err != nil {
		return fmt.Errorf("set last-tick: %w", err)
	}
	err = s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(lastTickBucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(project), buf)
	})
	if err != nil {
		return fmt.Errorf("set last-tick: %w", err)
	}
	return nil
}

// watchdogBucket holds, per (project, session transcript, channel), the
// dry-stretch key the capture watchdog last fired for — the fire-once latch.
// The key encodes the marker ordinal, so a new marker naturally re-arms the
// watchdog without any counter state. Keyed per session file — not per project
// — because two concurrent sessions in one project alternate as the "newest"
// transcript; a single project-wide slot would ping-pong and re-fire an
// already-fired stretch on every alternation. Keyed per channel for the same
// reason one level down: the watchdog measures a separate stretch per capture
// channel, and a shared slot would let the two overwrite each other's latch and
// nag on every alternation.
const watchdogBucket = "__watchdog__"

// watchdogKey is the composite (project, session, channel) latch key; NUL can't
// appear in a path, a transcript basename, or a channel name, so keys never
// alias (same scheme as processedKey).
func watchdogKey(project, session, channel string) []byte {
	return []byte(project + "\x00" + session + "\x00" + channel)
}

// WatchdogLatch returns the dry-stretch key the capture watchdog last fired
// for in (project, session, channel) — "" if it never fired for that session's
// channel.
func (s *Store) WatchdogLatch(project, session, channel string) (string, error) {
	var key string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(watchdogBucket))
		if b == nil {
			return nil
		}
		if v := b.Get(watchdogKey(project, session, channel)); v != nil {
			key = string(v)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("watchdog latch: %w", err)
	}
	return key, nil
}

// SetWatchdogLatch records that the capture watchdog fired for stretch key in
// (project, session, channel).
func (s *Store) SetWatchdogLatch(project, session, channel, key string) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(watchdogBucket))
		if err != nil {
			return err
		}
		return b.Put(watchdogKey(project, session, channel), []byte(key))
	})
	if err != nil {
		return fmt.Errorf("set watchdog latch: %w", err)
	}
	return nil
}
