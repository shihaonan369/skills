// Package rexwatch observes explicitly selected Rex hosts through the official CLI.
// Program reports are not agent, task, tool, or issue lifecycle telemetry.
package rexwatch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
)

const (
	MaxSources = 16
	MaxPanes   = 256
	MaxRecords = 4096
	MaxMessage = 8 << 20
)

type Source struct {
	Name     string   `json:"name"`
	Endpoint string   `json:"endpoint"`
	HostID   string   `json:"host_id"`
	Sessions []string `json:"sessions,omitempty"`
}

type Config struct {
	Version int      `json:"version"`
	Sources []Source `json:"sources"`
}

func printable(s string, limit int, empty bool) bool {
	if (!empty && s == "") || len(s) > limit {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return false
		}
	}
	return true
}

func ParseConfig(raw []byte) (Config, error) {
	var c Config
	if len(raw) > 256<<10 {
		return c, errors.New("Rex sources exceed 256 KiB")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return c, errors.New("exactly one Rex sources object required")
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	if c.Version != 1 || len(c.Sources) == 0 || len(c.Sources) > MaxSources {
		return errors.New("Rex sources require version 1 and 1..16 explicit hosts")
	}
	names, hosts := map[string]bool{}, map[string]bool{}
	for _, s := range c.Sources {
		if !printable(s.Name, 80, false) || !printable(s.Endpoint, 2048, false) || !printable(s.HostID, 128, false) {
			return errors.New("Rex source name, endpoint and pinned host_id must be printable and bounded")
		}
		if names[s.Name] || hosts[s.HostID] {
			return errors.New("Rex source names and pinned host IDs must be unique; combine session filters for one host")
		}
		names[s.Name], hosts[s.HostID] = true, true
		if len(s.Sessions) > 64 {
			return errors.New("Rex source limited to 64 session filters")
		}
		seen := map[string]bool{}
		for _, id := range s.Sessions {
			if !printable(id, 256, false) || seen[id] {
				return errors.New("Rex session filters must be unique, concrete IDs")
			}
			seen[id] = true
		}
	}
	return nil
}

type Record struct {
	ID           string `json:"id"`
	App          string `json:"app,omitempty"`
	EffectiveApp string `json:"effective_app,omitempty"`
	Title        string `json:"title,omitempty"`
	State        string `json:"state,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Message      string `json:"msg,omitempty"`
	Progress     *int   `json:"progress,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// ProgramID qualifies even an empty root record ID by its actual source.
func ProgramID(host, session, block, record string) string {
	raw, _ := json.Marshal([]string{host, session, block, record})
	h := sha256.Sum256(raw)
	return "rex:" + hex.EncodeToString(h[:])
}

type Program struct {
	Key          string `json:"key"`
	Source       string `json:"source"`
	HostID       string `json:"host_id"`
	SessionID    string `json:"session_id"`
	SessionLabel string `json:"session_label"`
	BlockID      string `json:"block_id"`
	BlockLabel   string `json:"block_label"`
	Record       Record `json:"record"`
}

type SourceState struct {
	Source        Source    `json:"source"`
	InstanceID    string    `json:"instance_id,omitempty"`
	Health        string    `json:"health"`
	Error         string    `json:"error,omitempty"`
	LastContact   time.Time `json:"last_contact,omitzero"`
	LastSnapshot  time.Time `json:"last_snapshot,omitzero"`
	Panes         int       `json:"panes"`
	Unreported    int       `json:"unreported"`
	ExcludedPanes int       `json:"excluded_panes"`
	Programs      []Program `json:"programs"`
}

type Snapshot struct {
	Revision uint64        `json:"revision"`
	Started  time.Time     `json:"started"`
	Sources  []SourceState `json:"sources"`
}

// Rex Lua encodes empty tables as {}, including empty API arrays. Only that
// exact empty-object case is accepted as an empty list at this wire boundary.
type list[T any] []T

func (l *list[T]) UnmarshalJSON(raw []byte) error {
	if string(bytes.TrimSpace(raw)) == "{}" {
		*l = nil
		return nil
	}
	return json.Unmarshal(raw, (*[]T)(l))
}

type pane struct {
	SessionID    string       `json:"session_id"`
	SessionLabel string       `json:"session_label"`
	BlockID      string       `json:"block_id"`
	BlockLabel   string       `json:"block_label"`
	Records      list[Record] `json:"records"`
}

type message struct {
	Version       int        `json:"agent_dashboard_rex"`
	Type          string     `json:"type"`
	HostID        string     `json:"host_id"`
	InstanceID    string     `json:"instance_id"`
	Seq           uint64     `json:"seq"`
	Panes         list[pane] `json:"panes,omitempty"`
	ExcludedPanes int        `json:"excluded_panes,omitempty"`
	Pane          *pane      `json:"pane,omitempty"`
}

func validatePane(p pane, source Source) error {
	if !printable(p.SessionID, 256, false) || !printable(p.BlockID, 256, false) || !printable(p.SessionLabel, 1024, true) || !printable(p.BlockLabel, 1024, true) {
		return errors.New("invalid Rex pane identity or label")
	}
	if len(source.Sessions) > 0 {
		allowed := false
		for _, id := range source.Sessions {
			allowed = allowed || id == p.SessionID
		}
		if !allowed {
			return errors.New("Rex report escaped the selected session scope")
		}
	}
	if len(p.Records) > MaxRecords {
		return errors.New("too many Rex program records")
	}
	ids := map[string]bool{}
	for _, r := range p.Records {
		if !printable(r.ID, 128, true) || ids[r.ID] {
			return errors.New("duplicate or invalid Rex program record identity")
		}
		ids[r.ID] = true
		if len(r.App) > 1024 || len(r.EffectiveApp) > 1024 || len(r.Title) > 4096 || len(r.Message) > 8192 || len(r.State) > 256 || len(r.Kind) > 256 || len(r.UpdatedAt) > 128 {
			return errors.New("Rex program field exceeds safety limit")
		}
		if r.Progress != nil && (*r.Progress < 0 || *r.Progress > 100) {
			return errors.New("Rex progress must be 0..100 when known")
		}
	}
	return nil
}

func paneKey(p pane) string { b, _ := json.Marshal([]string{p.SessionID, p.BlockID}); return string(b) }
func cleanError(err error) string {
	s := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, err.Error())
	r := []rune(s)
	if len(r) > 400 {
		s = string(r[:400])
	}
	return fmt.Sprintf("%s", s)
}
