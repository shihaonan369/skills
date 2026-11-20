package rexwatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

type observed struct {
	state    SourceState
	panes    map[string]pane
	epoch    uint64
	seq      uint64
	instance string
	baseline bool
	settled  bool
}

// Monitor owns one cancellable CLI child per explicitly selected source. Source
// failures preserve last-known observations but never change a program's state.
type Monitor struct {
	mu        sync.RWMutex
	cancel    context.CancelFunc
	workers   sync.WaitGroup
	sources   []observed
	revision  uint64
	started   time.Time
	ready     chan struct{}
	readyOnce sync.Once
}

func NewMonitor(ctx context.Context, client Client, config Config) (*Monitor, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if client.Binary == "" {
		return nil, errors.New("Rex CLI is not installed")
	}
	ctx, cancel := context.WithCancel(ctx)
	m := &Monitor{cancel: cancel, started: time.Now(), ready: make(chan struct{})}
	for _, source := range config.Sources {
		source.Sessions = append([]string(nil), source.Sessions...)
		m.sources = append(m.sources, observed{state: SourceState{Source: source, Health: "connecting", Programs: []Program{}}, panes: map[string]pane{}})
	}
	for i := range m.sources {
		m.workers.Add(1)
		go m.run(ctx, client, i)
	}
	return m, nil
}

func (m *Monitor) Close() { m.cancel(); m.workers.Wait() }

// Ready waits until every source has either produced a baseline or reported a
// connection/protocol failure. It does not mistake partial data for a full view.
func (m *Monitor) Ready(ctx context.Context) error {
	select {
	case <-m.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Monitor) settle(i int) {
	m.sources[i].settled = true
	for _, s := range m.sources {
		if !s.settled {
			return
		}
	}
	m.readyOnce.Do(func() { close(m.ready) })
}

func (m *Monitor) run(ctx context.Context, client Client, i int) {
	defer m.workers.Done()
	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		s := &m.sources[i]
		s.epoch++ // reject any late callback after the owning observer is stopped
		s.state.Health, s.state.Error = "stopped", ""
		m.settle(i)
		m.revision++
	}()
	retry := time.Second
	for ctx.Err() == nil {
		m.mu.Lock()
		s := &m.sources[i]
		s.epoch++
		epoch := s.epoch
		s.seq, s.instance, s.baseline = 0, "", false
		s.state.Health, s.state.Error = "connecting", ""
		source := s.state.Source
		m.revision++
		m.mu.Unlock()
		attempt, cancel := context.WithCancel(ctx)
		watchdog := time.AfterFunc(45*time.Second, cancel)
		err := client.watch(attempt, source, func(raw []byte) error {
			if err := m.accept(i, epoch, raw); err != nil {
				return err
			}
			watchdog.Reset(45 * time.Second)
			return nil
		})
		watchdog.Stop()
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			err = errors.New("Rex observer stopped")
		}
		m.mu.Lock()
		s = &m.sources[i]
		if s.baseline {
			retry = time.Second
		}
		s.state.Health, s.state.Error = "reconnecting", cleanError(err)
		m.settle(i)
		m.revision++
		m.mu.Unlock()
		timer := time.NewTimer(retry)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		retry = min(30*time.Second, retry*2)
	}
}

func (m *Monitor) accept(i int, epoch uint64, raw []byte) error {
	if len(raw) > MaxMessage {
		return errors.New("Rex message exceeds safety limit")
	}
	var msg message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("invalid Rex observer message: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &m.sources[i]
	if s.epoch != epoch {
		return errors.New("late message from an old Rex connection")
	}
	if msg.Version != 1 || msg.HostID != s.state.Source.HostID || !printable(msg.InstanceID, 128, false) {
		return errors.New("Rex observer identity or protocol mismatch")
	}
	if msg.Seq != s.seq+1 {
		return errors.New("Rex observer sequence gap; rebuilding snapshot")
	}
	if s.seq == 0 {
		if msg.Type != "hello" {
			return errors.New("Rex observer requires an initial identity handshake")
		}
		s.instance, s.state.Health = msg.InstanceID, "syncing"
	} else {
		if msg.InstanceID != s.instance || msg.Type == "hello" {
			return errors.New("Rex instance changed within an observation stream")
		}
		switch msg.Type {
		case "snapshot":
			if len(msg.Panes) > MaxPanes || msg.ExcludedPanes < 0 {
				return errors.New("Rex pane limit exceeded")
			}
			panes := map[string]pane{}
			count := 0
			for _, p := range msg.Panes {
				if err := validatePane(p, s.state.Source); err != nil {
					return err
				}
				key := paneKey(p)
				if _, exists := panes[key]; exists {
					return errors.New("duplicate Rex pane in snapshot")
				}
				panes[key] = p
				count += len(p.Records)
			}
			if count > MaxRecords {
				return errors.New("Rex record limit exceeded")
			}
			s.panes = panes
			s.baseline = true
			s.state.Health, s.state.Error = "online", ""
			s.state.InstanceID = msg.InstanceID
			s.state.ExcludedPanes = msg.ExcludedPanes
			s.state.LastSnapshot = time.Now()
			m.settle(i)
		case "pane":
			if !s.baseline || msg.Pane == nil {
				return errors.New("Rex pane update arrived without a baseline")
			}
			p := *msg.Pane
			if err := validatePane(p, s.state.Source); err != nil {
				return err
			}
			key := paneKey(p)
			old, exists := s.panes[key]
			if !exists {
				return errors.New("Rex pane update lacks source-qualified topology")
			}
			count := len(p.Records) - len(old.Records)
			for _, p := range s.panes {
				count += len(p.Records)
			}
			if count > MaxRecords {
				return errors.New("Rex record limit exceeded")
			}
			s.panes[key] = p
		case "heartbeat":
			// Discovery progress proves CLI contact, not a completed baseline.
			// In particular, it must not promote syncing to online or settle Ready.
		default:
			return errors.New("unknown Rex observer message")
		}
	}
	s.seq = msg.Seq
	s.state.LastContact = time.Now()
	m.revision++
	return nil
}

func (m *Monitor) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := Snapshot{Revision: m.revision, Started: m.started, Sources: make([]SourceState, 0, len(m.sources))}
	for _, s := range m.sources {
		copy := s.state
		copy.Source.Sessions = append([]string(nil), s.state.Source.Sessions...)
		copy.Programs = []Program{}
		copy.Panes, copy.Unreported = len(s.panes), 0
		for _, p := range s.panes {
			if len(p.Records) == 0 {
				copy.Unreported++
			}
			for _, r := range p.Records {
				if r.Progress != nil {
					value := *r.Progress
					r.Progress = &value
				}
				copy.Programs = append(copy.Programs, Program{Key: ProgramID(copy.Source.HostID, p.SessionID, p.BlockID, r.ID), Source: copy.Source.Name, HostID: copy.Source.HostID, SessionID: p.SessionID, SessionLabel: p.SessionLabel, BlockID: p.BlockID, BlockLabel: p.BlockLabel, Record: r})
			}
		}
		sort.Slice(copy.Programs, func(i, j int) bool {
			a, b := copy.Programs[i], copy.Programs[j]
			if a.SessionLabel != b.SessionLabel {
				return a.SessionLabel < b.SessionLabel
			}
			if a.SessionID != b.SessionID {
				return a.SessionID < b.SessionID
			}
			if a.BlockID != b.BlockID {
				return a.BlockID < b.BlockID
			}
			return a.Record.ID < b.Record.ID
		})
		result.Sources = append(result.Sources, copy)
	}
	return result
}
