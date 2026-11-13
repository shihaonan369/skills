package dashboard

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"context"
	"github.com/gofrs/flock"
)

const MaxEvents = 500
const MaxAgents = 256
const MaxTasks = 1024
const MaxTools = 1024
const MaxInput = 4 << 20

var ErrInactive = errors.New("session not active; run start first")

type Agent struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Model      string `json:"model,omitempty"`
	Transcript string `json:"transcript,omitempty"`
	Started    int64  `json:"started"`
	Ended      int64  `json:"ended,omitempty"`
	Tools      int    `json:"tools"`
	Errors     int    `json:"errors"`
}
type Task struct {
	ID        string   `json:"id"`
	Subject   string   `json:"subject"`
	Status    string   `json:"status"`
	Owner     string   `json:"owner,omitempty"`
	BlockedBy []string `json:"blockedBy,omitempty"`
}
type Tool struct {
	Actor   string `json:"actor"`
	Name    string `json:"name"`
	Started int64  `json:"started"`
	Waiting bool   `json:"waiting,omitempty"`
}
type Event struct {
	Time  int64  `json:"time"`
	Actor string `json:"actor"`
	Kind  string `json:"kind"`
	Text  string `json:"text"`
}
type State struct {
	Layout    *Layout           `json:"layout,omitempty"`
	Revision  uint64            `json:"revision"`
	Version   int               `json:"version"`
	Session   string            `json:"session"`
	Title     string            `json:"title"`
	Demo      bool              `json:"demo"`
	Active    bool              `json:"active"`
	Finished  bool              `json:"finished"`
	Started   int64             `json:"started"`
	LastHook  int64             `json:"last_hook"`
	HookCount int               `json:"hook_count"`
	Dropped   int               `json:"dropped"`
	Agents    map[string]*Agent `json:"agents"`
	Tasks     map[string]Task   `json:"tasks"`
	Tools     map[string]Tool   `json:"tools"`
	Events    []Event           `json:"events"`
	Seen      []string          `json:"seen"`
}

// Clean strips terminal controls, bidi overrides and line breaks before storage or display.
func Clean(s string, max int) string {
	var b strings.Builder
	n := 0
	for _, r := range strings.ToValidUTF8(s, "�") {
		if unicode.IsControl(r) || (unicode.In(r, unicode.Cf) && r != '\u200d' && r != '\u200c') {
			continue
		}
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
func Root() (string, error) {
	if s := os.Getenv("AGENT_DASHBOARD_HOME"); s != "" {
		return filepath.Abs(s)
	}
	s, e := os.UserConfigDir()
	return filepath.Join(s, "agent-dashboard", "state"), e
}
func sessionDir(root, session string) string {
	h := sha256.Sum256([]byte(session))
	return filepath.Join(root, hex.EncodeToString(h[:]))
}
func validSession(s string) error {
	if s == "" || len(s) > 256 || strings.Contains(s, "${") || Clean(s, 256) != s {
		return errors.New("a concrete Claude session ID (1..256 bytes) is required")
	}
	return nil
}
func newState(session, title string, demo bool) *State {
	now := time.Now().UnixMilli()
	return &State{Version: 1, Session: session, Title: Clean(title, 160), Demo: demo, Active: true, Started: now, Agents: map[string]*Agent{"lead": {ID: "lead", Role: "lead", Status: "running", Started: now}}, Tasks: map[string]Task{}, Tools: map[string]Tool{}, Events: []Event{}, Seen: []string{}}
}
func (s *State) event(actor, kind, msg string) {
	s.Events = append(s.Events, Event{time.Now().UnixMilli(), Clean(actor, 120), kind, Clean(msg, 400)})
	if len(s.Events) > MaxEvents {
		s.Events = s.Events[len(s.Events)-MaxEvents:]
	}
}

// Transaction is shared by all CLI processes. LockFileEx/flock locks are released by
// the OS on process death. All reads use the same lock, including on Windows.
func Transaction(root, session string, create bool, fn func(**State) error) error {
	if e := validSession(session); e != nil {
		return e
	}
	dir := sessionDir(root, session)
	if !create {
		if _, e := os.Stat(filepath.Join(dir, "state.json")); e != nil {
			return ErrInactive
		}
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	lock := flock.New(filepath.Join(dir, "state.lock"), flock.SetPermissions(0600))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ok, e := lock.TryLockContext(ctx, 10*time.Millisecond)
	if e != nil {
		return e
	}
	if !ok {
		return errors.New("state busy; try again")
	}
	defer lock.Close()
	var s *State
	raw, e := os.ReadFile(filepath.Join(dir, "state.json"))
	if e == nil {
		if len(raw) > 8<<20 {
			return errors.New("state exceeds safety limit")
		}
		if e = json.Unmarshal(raw, &s); e != nil {
			return e
		}
		if s == nil || s.Version != 1 || s.Session != session || s.Agents == nil || s.Agents["lead"] == nil || s.Tasks == nil || s.Tools == nil {
			return errors.New("incompatible state")
		}
	} else if !os.IsNotExist(e) || !create {
		return e
	}
	if s != nil {
		for _, a := range s.Agents {
			if a == nil {
				return errors.New("invalid null agent in state")
			}
		}
	}
	if e = fn(&s); e != nil {
		return e
	}
	if s == nil {
		return nil
	}
	s.Revision++
	raw, e = json.Marshal(s)
	if e != nil {
		return e
	}
	return atomicWrite(filepath.Join(dir, "state.json"), raw, 0600)
}
func atomicWrite(path string, raw []byte, mode os.FileMode) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".dashboard-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(raw)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(tmp, path)
}

// Snapshot does not rewrite the state. The transaction callback returns a sentinel
// to release the lock before any transcript IO or rendering.
func Snapshot(root, session string) (*State, error) {
	var out *State
	done := errors.New("read only")
	e := Transaction(root, session, false, func(s **State) error { out = *s; return done })
	if e != done {
		return nil, e
	}
	for _, a := range out.Agents {
		if a.Transcript != "" && a.Model == "" {
			if m := transcriptModel(a.Transcript, a.ID); m != "" {
				a.Model = m
			}
		}
	}
	return out, nil
}
func Start(root, session, title string, demo bool) error {
	return Transaction(root, session, true, func(s **State) error {
		if *s == nil {
			*s = newState(session, title, demo)
		} else {
			if (*s).Demo != demo {
				return errors.New("demo and real sessions must use different IDs")
			}
			(*s).Active = true
			(*s).Finished = false
			(*s).Title = Clean(title, 160)
			(*s).Agents["lead"].Status = "running"
		}
		(*s).event("lead", "attach", "Observation enabled; earlier activity is unknown")
		return nil
	})
}
func Note(root, session, actor, msg string) error {
	return Transaction(root, session, false, func(s **State) error {
		if _, ok := (*s).Agents[actor]; !ok {
			return errors.New("actor is not observed")
		}
		(*s).event(actor, "note", msg)
		return nil
	})
}
func Finish(root, session, summary string) error {
	return Transaction(root, session, false, func(s **State) error {
		(*s).Finished = true
		(*s).Agents["lead"].Status = "done"
		(*s).event("lead", "finish", summary)
		return nil
	})
}
func Deactivate(root, session string) error {
	return Transaction(root, session, false, func(s **State) error {
		(*s).Active = false
		(*s).event("lead", "detach", "Collection disabled; persisted snapshot retained")
		return nil
	})
}
func transcriptModel(path, actor string) string {
	if strings.HasPrefix(path, "~/") {
		if h, e := os.UserHomeDir(); e == nil {
			path = filepath.Join(h, path[2:])
		}
	}
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() {
		return ""
	}
	f, e := os.Open(path)
	if e != nil {
		return ""
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		return ""
	}
	off := st.Size() - 131072
	if off < 0 {
		off = 0
	}
	if _, e = f.Seek(off, io.SeekStart); e != nil {
		return ""
	}
	sc := bufio.NewScanner(io.LimitReader(f, 131072))
	sc.Buffer(make([]byte, 4096), 131072)
	model := ""
	for sc.Scan() {
		var row struct {
			Type    string `json:"type"`
			AgentID string `json:"agentId"`
			Message struct {
				Model string `json:"model"`
			} `json:"message"`
		}
		if json.Unmarshal(sc.Bytes(), &row) != nil {
			continue
		}
		if row.Type != "assistant" {
			continue
		}
		if row.AgentID != "" && row.AgentID != actor {
			continue
		}
		if m := row.Message.Model; m != "" && m != "<synthetic>" {
			model = Clean(m, 100)
		}
	}
	return model
}
func (s *State) Progress(owner string) string {
	done, total := 0, 0
	for _, t := range s.Tasks {
		if t.Status == "deleted" || (owner != "" && t.Owner != owner) {
			continue
		}
		total++
		if t.Status == "completed" {
			done++
		}
	}
	if total == 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d/%d steps", done, total)
}
