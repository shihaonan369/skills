package dashboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/gofrs/flock"
)

// Scope is a private, caller-owned classification mailbox, not an LLM client.
// The initiating agent or its native subagent supplies decisions. The dashboard
// emits bounded metadata only; it never executes a classifier or reads prompts.
type Scope struct {
	Version   int                   `json:"version"`
	Criterion string                `json:"criterion"`
	Closed    bool                  `json:"closed,omitempty"`
	Entries   map[string]ScopeEntry `json:"entries"`
}
type ScopeRequest struct {
	Key         string  `json:"key"`
	Fingerprint string  `json:"fingerprint"`
	Bind        Binding `json:"bind"`
	Context     string  `json:"context"`
	Label       string  `json:"label"`
	Role        string  `json:"reported_role,omitempty"`
	Metadata    string  `json:"metadata,omitempty"`
}
type ScopeEntry struct {
	Request  ScopeRequest `json:"request"`
	Decision string       `json:"decision,omitempty"`
	Origin   string       `json:"origin,omitempty"`
	Evidence string       `json:"evidence,omitempty"`
}
type ScopeBatch struct {
	PolicyHash string         `json:"policy_hash"`
	Criterion  string         `json:"criterion"`
	Closed     bool           `json:"closed"`
	Requests   []ScopeRequest `json:"requests"`
}
type ScopeDecision struct {
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
	Decision    string `json:"decision"`
	Origin      string `json:"origin"`
	Evidence    string `json:"evidence,omitempty"`
}
type ScopeDecisions struct {
	PolicyHash string          `json:"policy_hash"`
	Decisions  []ScopeDecision `json:"decisions"`
}

func scopeHash(value any) string {
	raw, _ := json.Marshal(value)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func policyHash(s Scope) string { return scopeHash([]any{s.Version, s.Criterion}) }
func scopeRequest(b Binding, e observationEntity, policy string) ScopeRequest {
	r := ScopeRequest{Bind: b, Context: e.Context, Label: e.Label, Role: e.Role, Metadata: e.IdentityMetadata}
	r.Key = scopeHash([]any{b, e.Context})
	r.Fingerprint = scopeHash([]any{policy, r})
	return r
}
func strictJSON(raw []byte, target any) error {
	if len(raw) > 8<<20 {
		return errors.New("scope data exceeds 8 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("exactly one scope object required")
	}
	return nil
}
func readScope(path string) (Scope, error) {
	var s Scope
	if !filepath.IsAbs(path) || len(path) > 4096 {
		return s, errors.New("scope file must be an absolute bounded path")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return s, err
	}
	if !info.Mode().IsRegular() || info.Size() > 8<<20 {
		return s, errors.New("scope must be a bounded regular file, not a link")
	}
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (8<<20)+1))
	if err != nil {
		return s, err
	}
	if err = strictJSON(raw, &s); err != nil {
		return s, err
	}
	if s.Version != 1 || s.Criterion == "" || len(s.Criterion) > 4096 || Clean(s.Criterion, 4096) != s.Criterion || len(s.Entries) > 8192 {
		return s, errors.New("invalid scope version, criterion or entry count")
	}
	if s.Entries == nil {
		s.Entries = map[string]ScopeEntry{}
	}
	for key, e := range s.Entries {
		r := e.Request
		if key != r.Key || len(key) != 64 || len(r.Fingerprint) != 64 || len(r.Context) > 1024 || len(r.Label) > 1024 || len(r.Role) > 400 || len(r.Metadata) > 4096 || len(e.Evidence) > 240 {
			return s, errors.New("invalid or oversized scope entry")
		}
		if r.Bind.Item || r.Bind.validate() != nil || r.Key != scopeHash([]any{r.Bind, r.Context}) {
			return s, errors.New("invalid scope identity")
		}
		if e.Decision != "" && e.Decision != "include" && e.Decision != "exclude" && e.Decision != "unknown" {
			return s, errors.New("invalid stored scope decision")
		}
		if e.Decision != "" && e.Origin != "subagent" && e.Origin != "main-agent" && e.Origin != "caller-declared" {
			return s, errors.New("missing scope decision provenance")
		}
	}
	return s, nil
}
func scopeTransaction(path string, create bool, fn func(*Scope) error) error {
	if !filepath.IsAbs(path) || len(path) > 4096 {
		return errors.New("scope file must be an absolute bounded path")
	}
	if create {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	}
	lockPath := path + ".lock"
	if info, err := os.Lstat(lockPath); err == nil && !info.Mode().IsRegular() {
		return errors.New("scope lock must be a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	lock := flock.New(lockPath, flock.SetPermissions(0600))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ok, err := lock.TryLockContext(ctx, 10*time.Millisecond)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("scope mailbox busy; retry later")
	}
	defer lock.Close()
	s, err := readScope(path)
	if err != nil && !(create && os.IsNotExist(err)) {
		return err
	}
	before, _ := json.Marshal(s)
	if err := fn(&s); err != nil {
		return err
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if len(raw) > 8<<20 {
		return errors.New("scope mailbox exceeds 8 MiB")
	}
	if bytes.Equal(before, raw) {
		return nil
	}
	return atomicWrite(path, raw, 0600)
}
func InitScope(path, criterion string) error {
	if criterion == "" || len(criterion) > 4096 || Clean(criterion, 4096) != criterion {
		return errors.New("scope criterion must be printable and 1..4096 bytes")
	}
	return scopeTransaction(path, true, func(s *Scope) error {
		if s.Version != 0 && s.Criterion != criterion {
			return errors.New("scope already has another criterion; use a new private file")
		}
		s.Version, s.Criterion, s.Closed = 1, criterion, false
		if s.Entries == nil {
			s.Entries = map[string]ScopeEntry{}
		}
		return nil
	})
}
func CloseScope(path string) error {
	return scopeTransaction(path, false, func(s *Scope) error { s.Closed = true; return nil })
}
func PollScope(path string, wait time.Duration) (ScopeBatch, error) {
	if wait < 0 || wait > 60*time.Second {
		return ScopeBatch{}, errors.New("scope wait must be 0..60 seconds")
	}
	deadline := time.Now().Add(wait)
	for {
		s, err := readScope(path)
		if err != nil {
			return ScopeBatch{}, err
		}
		b := pendingScopeBatch(s)
		if len(b.Requests) > 64 {
			b.Requests = b.Requests[:64]
		}
		if b.Closed || len(b.Requests) > 0 || !time.Now().Before(deadline) {
			return b, nil
		}
		time.Sleep(min(100*time.Millisecond, time.Until(deadline)))
	}
}
func DecideScope(path string, raw []byte) error {
	var batch ScopeDecisions
	if err := strictJSON(raw, &batch); err != nil {
		return err
	}
	if len(batch.Decisions) == 0 || len(batch.Decisions) > 64 {
		return errors.New("scope decision batch must contain 1..64 entries")
	}
	return scopeTransaction(path, false, func(s *Scope) error {
		if s.Closed || batch.PolicyHash != policyHash(*s) {
			return errors.New("scope closed or policy changed; poll again")
		}
		seen := map[string]bool{}
		for _, d := range batch.Decisions {
			e, ok := s.Entries[d.Key]
			if !ok || d.Fingerprint != e.Request.Fingerprint || seen[d.Key] {
				return errors.New("stale or duplicate scope decision; poll again")
			}
			if d.Decision != "include" && d.Decision != "exclude" && d.Decision != "unknown" {
				return errors.New("decision must be include, exclude or unknown")
			}
			if d.Origin != "subagent" && d.Origin != "main-agent" && d.Origin != "caller-declared" {
				return errors.New("decision origin must be subagent, main-agent or caller-declared")
			}
			if len(d.Evidence) > 240 || Clean(d.Evidence, 240) != d.Evidence {
				return errors.New("scope evidence must be a short printable factual summary")
			}
			seen[d.Key] = true
		}
		for _, d := range batch.Decisions {
			e := s.Entries[d.Key]
			e.Decision, e.Origin, e.Evidence = d.Decision, d.Origin, d.Evidence
			s.Entries[d.Key] = e
		}
		return nil
	})
}

// PreparePresentation runs only at observation refresh, never during rendering.
// Only explicitly referenced, preinitialized scope files can be touched.
func PreparePresentation(s *State) (*State, error) {
	if s == nil || s.Layout == nil {
		return s, nil
	}
	if err := validateDynamicSyntax(&s.Layout.Root, ""); err != nil {
		return nil, err
	}
	v := observations(s, time.Time{})
	groups := map[string][]Selection{}
	var collect func(Panel)
	add := func(q *Selection) {
		if q != nil && q.ScopeFile != "" {
			groups[q.ScopeFile] = append(groups[q.ScopeFile], *q)
		}
	}
	collect = func(p Panel) {
		add(p.Select)
		if p.Repeat != nil {
			add(&p.Repeat.Select)
		}
		if p.DAG != nil {
			add(p.DAG.Select)
		}
		for _, child := range p.Children {
			collect(child)
		}
	}
	collect(s.Layout.Root)
	if len(groups) == 0 {
		return s, nil
	}
	clone := *s
	clone.scopeResults = map[string]Scope{}
	for path, selections := range groups {
		var saved Scope
		err := scopeTransaction(path, false, func(scope *Scope) error {
			if !scope.Closed {
				entries := map[string]ScopeEntry{}
				for _, q := range selections {
					for _, b := range v.candidates(q) {
						r := scopeRequest(b, v.Entities[b], policyHash(*scope))
						e := scope.Entries[r.Key]
						if !reflect.DeepEqual(e.Request, r) {
							e = ScopeEntry{Request: r}
						}
						entries[r.Key] = e
					}
				}
				if len(entries) > 8192 {
					return errors.New("scope exceeds 8192 observed entities")
				}
				scope.Entries = entries
			}
			saved = *scope
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scope mailbox: %w", err)
		}
		clone.scopeResults[path] = saved
	}
	clone.observation = nil
	return &clone, nil
}

func (v *observationView) scopeDecision(q Selection, b Binding) string {
	if q.ScopeFile == "" {
		return "include"
	}
	scope, ok := v.ScopeResults[q.ScopeFile]
	if !ok {
		return "pending"
	}
	r := scopeRequest(b, v.Entities[b], policyHash(scope))
	e, ok := scope.Entries[r.Key]
	if !ok || e.Request.Fingerprint != r.Fingerprint || e.Decision == "" {
		return "pending"
	}
	return e.Decision
}
func (v *observationView) scopeNotice(q Selection) string {
	if q.ScopeFile == "" {
		return ""
	}
	pending, unknown := 0, 0
	for _, b := range v.candidates(q) {
		switch v.scopeDecision(q, b) {
		case "pending":
			pending++
		case "unknown":
			unknown++
		}
	}
	prefix := "Agent-classified scope"
	if scope, ok := v.ScopeResults[q.ScopeFile]; ok && scope.Closed {
		prefix += " (worker stop requested)"
	}
	return fmt.Sprintf("%s: %d pending, %d unknown; only explicit include decisions are shown", prefix, pending, unknown)
}

// InspectScope exposes classifier provenance without changing the mailbox.
func InspectScope(path string) (Scope, error) { return readScope(path) }
