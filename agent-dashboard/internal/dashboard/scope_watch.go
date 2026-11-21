package dashboard

import (
	"context"
	"errors"
	"os"
	"sort"
	"time"
)

func pendingScopeBatch(s Scope) ScopeBatch {
	b := ScopeBatch{PolicyHash: policyHash(s), Criterion: s.Criterion, Closed: s.Closed, Requests: []ScopeRequest{}}
	if !s.Closed {
		for _, entry := range s.Entries {
			if entry.Decision == "" {
				b.Requests = append(b.Requests, entry.Request)
			}
		}
		sort.Slice(b.Requests, func(i, j int) bool { return b.Requests[i].Key < b.Requests[j].Key })
	}
	return b
}

// WatchScope supplies a bounded, deduplicated event source for a caller-owned
// native monitor. It performs no classification or harness dispatch. Each watch
// replays pending requests on startup so restarting cannot lose unacknowledged work.
func WatchScope(ctx context.Context, path string, lifetime time.Duration, emit func(ScopeBatch) error) error {
	if lifetime <= 0 || lifetime > 30*time.Minute {
		return errors.New("scope watch duration must be positive and at most 1800 seconds")
	}
	deadline := time.NewTimer(lifetime)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	seen := map[string]string{}
	var previous os.FileInfo
	first := true
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		// Transactions replace the file atomically. Stat-only idle polling avoids
		// repeatedly decoding a mailbox or waking the model without new work.
		changed := previous == nil || !os.SameFile(previous, info) || previous.Size() != info.Size() || !previous.ModTime().Equal(info.ModTime())
		if changed {
			s, err := readScope(path)
			if err != nil {
				return err
			}
			previous = info
			batch := pendingScopeBatch(s)
			pending := map[string]string{}
			fresh := []ScopeRequest{}
			for _, r := range batch.Requests {
				pending[r.Key] = r.Fingerprint
				if seen[r.Key] != r.Fingerprint {
					fresh = append(fresh, r)
				}
			}
			seen = pending
			if batch.Closed || (first && len(fresh) == 0) {
				batch.Requests = []ScopeRequest{}
				if err := emit(batch); err != nil {
					return err
				}
			}
			first = false
			if batch.Closed {
				return nil
			}
			for len(fresh) > 0 {
				n := min(64, len(fresh))
				batch.Requests = fresh[:n]
				if err := emit(batch); err != nil {
					return err
				}
				fresh = fresh[n:]
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return nil
		case <-tick.C:
		}
	}
}
