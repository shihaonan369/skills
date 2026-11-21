package dashboard_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	d "agent-dashboard/internal/dashboard"
)

func TestScopeWatchHandsInitialAndLaterCandidatesToTheSameWorker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	if err := d.InitScope(path, "Implementation only"); err != nil {
		t.Fatal(err)
	}
	_, root := dynamicNative(t, scopeLayout(path))
	prepareScope(t, spawnDynamic(t, root, "first", "general-purpose"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	calls := 0
	err := d.WatchScope(ctx, path, 2*time.Second, func(batch d.ScopeBatch) error {
		calls++
		if batch.Closed {
			if calls != 3 || len(batch.Requests) != 0 {
				t.Fatalf("unexpected close: %#v, calls=%d", batch, calls)
			}
			return nil
		}
		if len(batch.Requests) != 1 {
			t.Fatalf("unexpected batch: %#v", batch)
		}
		want := "first"
		if calls == 2 {
			want = "later"
		}
		if calls > 2 || batch.Requests[0].Bind.Actor != want {
			t.Fatalf("unexpected candidate: %#v", batch)
		}
		if err := d.DecideScope(path, scopeAnswer(batch, "include")); err != nil {
			return err
		}
		if calls == 1 {
			prepareScope(t, spawnDynamic(t, root, "later", "general-purpose"))
		} else {
			frame := dynamicFrame(prepareScope(t, nativeSnapshot(t, root)))
			if strings.Count(frame, "first") < 3 || strings.Count(frame, "later") < 3 {
				t.Fatal(frame)
			}
			return d.CloseScope(path)
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatal("stream failed to deliver initial, later and close events", err, calls)
	}
}

func TestScopeWatchDeduplicatesPendingAndRequeuesChangedUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	if err := d.InitScope(path, "Implementation only"); err != nil {
		t.Fatal(err)
	}
	_, root := dynamicNative(t, scopeLayout(path))
	prepareScope(t, spawnDynamic(t, root, "same-worker", "general-purpose"))
	calls := 0
	original := ""
	err := d.WatchScope(context.Background(), path, 2*time.Second, func(batch d.ScopeBatch) error {
		calls++
		if batch.Closed {
			if calls != 4 {
				t.Fatal("lost or duplicated event", calls)
			}
			return nil
		}
		if len(batch.Requests) != 1 {
			t.Fatal("pending request repeated after another mailbox write", batch)
		}
		r := batch.Requests[0]
		switch calls {
		case 1:
			original = r.Fingerprint
			if err := d.DecideScope(path, scopeAnswer(batch, "unknown")); err != nil {
				return err
			}
			prepareScope(t, spawnDynamic(t, root, "same-worker", "implement"))
		case 2:
			if r.Bind.Actor != "same-worker" || r.Fingerprint == original {
				t.Fatal("changed identity not requeued", batch)
			}
			// Leave this request unacknowledged while another atomic write adds work.
			prepareScope(t, spawnDynamic(t, root, "new-worker", "general-purpose"))
		case 3:
			if r.Bind.Actor != "new-worker" {
				t.Fatal("unchanged pending request woke the worker twice", batch)
			}
			return d.CloseScope(path)
		default:
			t.Fatal("unexpected event", batch)
		}
		return nil
	})
	if err != nil || calls != 4 {
		t.Fatal(err, calls)
	}
	// A fresh watch must replay unacknowledged requests, not lose them to dedup.
	if err := d.InitScope(path, "Implementation only"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	count := 0
	err = d.WatchScope(ctx, path, time.Second, func(batch d.ScopeBatch) error { count = len(batch.Requests); cancel(); return nil })
	if !errors.Is(err, context.Canceled) || count != 2 {
		t.Fatal("restart lost pending work", err, count)
	}
}

func TestScopeWatchInitialFanoutIsBoundedButNotTruncated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	if err := d.InitScope(path, "Synthetic fanout"); err != nil {
		t.Fatal(err)
	}
	_, root := dynamicNative(t, scopeLayout(path))
	for i := 0; i < 71; i++ {
		spawnDynamic(t, root, fmt.Sprintf("worker-%02d", i), "general-purpose")
	}
	prepareScope(t, nativeSnapshot(t, root))
	counts := []int{}
	seen := map[string]bool{}
	err := d.WatchScope(context.Background(), path, 2*time.Second, func(batch d.ScopeBatch) error {
		if batch.Closed {
			return nil
		}
		counts = append(counts, len(batch.Requests))
		for _, r := range batch.Requests {
			if seen[r.Key] {
				t.Fatal("duplicate request", r.Key)
			}
			seen[r.Key] = true
		}
		if len(seen) == 71 {
			return d.CloseScope(path)
		}
		return nil
	})
	if err != nil || len(counts) != 2 || counts[0] != 64 || counts[1] != 7 {
		t.Fatal(err, counts)
	}
}

func TestScopeWatchIdleDeadlineAndFailureBoundaries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	if err := d.InitScope(path, "Synthetic idle worker"); err != nil {
		t.Fatal(err)
	}
	calls := 0
	start := time.Now()
	err := d.WatchScope(context.Background(), path, 250*time.Millisecond, func(batch d.ScopeBatch) error {
		calls++
		if batch.Closed || len(batch.Requests) != 0 {
			t.Fatal("idle/deadline invented work or close", batch)
		}
		return nil
	})
	if err != nil || calls != 1 || time.Since(start) > 2*time.Second {
		t.Fatal("idle watch spun or exceeded deadline", err, calls)
	}
	outputErr := errors.New("synthetic broken output")
	if err := d.WatchScope(context.Background(), path, time.Second, func(d.ScopeBatch) error { return outputErr }); !errors.Is(err, outputErr) {
		t.Fatal(err)
	}
	for _, duration := range []time.Duration{0, -time.Second, 30*time.Minute + time.Nanosecond} {
		if err := d.WatchScope(context.Background(), path, duration, func(d.ScopeBatch) error { t.Fatal("invalid watch emitted"); return nil }); err == nil {
			t.Fatal("invalid duration accepted", duration)
		}
	}
	if err := d.WatchScope(context.Background(), path, time.Second, func(d.ScopeBatch) error { return os.WriteFile(path, []byte("invalid mailbox"), 0600) }); err == nil {
		t.Fatal("corrupt mailbox silently ignored")
	}
	if err := d.WatchScope(context.Background(), filepath.Join(t.TempDir(), "missing.json"), time.Second, func(d.ScopeBatch) error { t.Fatal("missing scope emitted work"); return nil }); err == nil {
		t.Fatal("missing mailbox ignored")
	}
}
