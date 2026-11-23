package dashboard

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Binding names an observed entity, never a role inferred from a title or a
// program's record path. The same binding is used by cards, lists and DAGs.
type Binding struct {
	Actor   string `json:"actor,omitempty"`
	Task    string `json:"task,omitempty"`
	Program string `json:"program,omitempty"`
	Source  string `json:"source,omitempty"`
	Item    bool   `json:"item,omitempty"`
}

func validProgramKey(id string) bool {
	return len(id) == 68 && strings.HasPrefix(id, "rex:") && strings.IndexFunc(id[4:], func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) < 0
}
func (b Binding) validate() error {
	count := 0
	if b.Item {
		count++
	}
	for _, id := range []string{b.Actor, b.Task, b.Program, b.Source} {
		if id != "" {
			count++
		}
		if len(id) > 120 || Clean(id, 120) != id {
			return errors.New("binding ID must be printable and at most 120 bytes")
		}
	}
	if count != 1 {
		return errors.New("bind requires exactly one actor, task, program, source or template item")
	}
	if b.Source != "" && len(b.Source) > 80 {
		return errors.New("source binding exceeds 80 bytes")
	}
	if b.Program != "" && !validProgramKey(b.Program) {
		return errors.New("program binding requires a full source-qualified Rex key")
	}
	return nil
}
func (b Binding) label() string {
	if b.Actor != "" {
		return "Actor " + b.Actor
	}
	if b.Task != "" {
		return "Task " + b.Task
	}
	if b.Source != "" {
		return "Rex source " + b.Source
	}
	return "Rex program " + b.Program
}

// observationEntity is a presentation-neutral observed identity plus ordered
// factual rows. Adapters own semantics, unknown values and state classification;
// panels own geometry only. No adapter writes into another source's collections.
type observationEntity struct {
	Source, Label                      string
	Role, Context, IdentityMetadata    string
	Rows, CompactRows                  []reportRow
	Status, Prefix                     string
	Running, Success, Alert, LastKnown bool
	Keys                               []echoKey
}
type observationEvent struct {
	Event   Event
	Binding Binding
	Source  string
}
type observationView struct {
	Entities                map[Binding]observationEntity
	ScopeResults            map[string]Scope
	Actors, Tasks, Programs []Binding
	SourceBindings          []Binding
	Session                 []reportRow
	Summaries               []observationEntity
	EmptyActors, EmptyTasks string
	EventNotice             string
	Events                  []observationEvent
	Values                  map[echoKey]string
}

func observations(s *State, now time.Time) *observationView {
	if s.observation != nil {
		return s.observation
	}
	var v *observationView
	if s.ProgramReports != nil {
		v = rexObservations(s, now)
	} else {
		v = claudeObservations(s, now)
	}
	v.ScopeResults = s.scopeResults
	return v
}
func stateEntity(rows []reportRow, status, prefix string, keys []echoKey) observationEntity {
	return observationEntity{Rows: rows, Status: status, Prefix: prefix, Keys: keys,
		Running: status == "running" || status == "in_progress", Success: status == "done" || status == "completed",
		Alert: status == "failed" || status == "blocked" || status == "api_error" || status == "API error"}
}
func claudeObservations(s *State, now time.Time) *observationView {
	v := &observationView{Entities: map[Binding]observationEntity{}, Values: map[echoKey]string{}}
	calls := map[string][]string{}
	for id, t := range s.Tools {
		calls[t.Actor] = append(calls[t.Actor], fmt.Sprintf("%q:%q:%t:%d", id, t.Name, t.Waiting, t.Started))
	}
	for id, a := range s.Agents {
		if a == nil {
			continue
		}
		b := Binding{Actor: id}
		rows := []reportRow{
			{text: Clean(a.Role, 100) + " · " + Clean(a.ID, 120) + " · " + Clean(a.Status, 80), keys: []echoKey{actorKey(id, "role"), actorKey(id, "status")}},
			{text: "model " + unknown(a.Model) + " | " + s.Progress(id), keys: []echoKey{actorKey(id, "model"), actorKey(id, "progress")}},
			{text: fmt.Sprintf("tools %d · errors %d | %s", a.Tools, a.Errors, active(s, id)), keys: []echoKey{actorKey(id, "tools"), actorKey(id, "errors"), actorKey(id, "active")}},
		}
		prefix := ""
		if a.Status == "running" {
			prefix = fmt.Sprintf(" · tools %d", a.Tools)
		}
		keys := []echoKey{actorKey(id, "status")}
		if a.Status == "running" {
			keys = append(keys, actorKey(id, "tools"))
		}
		e := stateEntity(rows, a.Status, prefix, keys)
		e.Label = Clean(a.Role, 80) + " · " + Clean(a.ID, 80)
		e.Role, e.Context = a.Role, fmt.Sprintf("Claude:%s:%d:%d", s.Session, s.Started, a.Started)
		v.Entities[b] = e
		sort.Strings(calls[id])
		for field, value := range map[string]string{"status": a.Status, "role": a.Role, "model": a.Model, "tools": strconv.Itoa(a.Tools), "errors": strconv.Itoa(a.Errors), "active": strings.Join(calls[id], "\n"), "progress": s.Progress(id)} {
			v.Values[actorKey(id, field)] = value
		}
	}
	for _, a := range sortedAgents(s) {
		v.Actors = append(v.Actors, Binding{Actor: a.ID})
	}
	for id, t := range s.Tasks {
		b := Binding{Task: id}
		rows := []reportRow{{text: fmt.Sprintf("#%s [%s] %s", Clean(t.ID, 120), Clean(t.Status, 80), Clean(t.Subject, 240)), keys: []echoKey{taskKey(id, "status"), taskKey(id, "subject")}}, {text: "owner " + unknown(t.Owner) + " | blocked by " + Clean(strings.Join(t.BlockedBy, ", "), 400), keys: []echoKey{taskKey(id, "owner"), taskKey(id, "blocked")}}}
		e := stateEntity(rows, t.Status, "", []echoKey{taskKey(id, "status")})
		e.Label = "#" + Clean(id, 80) + " · " + Clean(t.Subject, 80)
		e.IdentityMetadata = "subject=" + Clean(t.Subject, 240) + "; owner=" + Clean(t.Owner, 120)
		e.Context = fmt.Sprintf("Claude:%s:%d", s.Session, s.Started)
		v.Entities[b] = e
		for field, value := range map[string]string{"status": t.Status, "subject": t.Subject, "owner": t.Owner, "blocked": strings.Join(t.BlockedBy, "\n")} {
			v.Values[taskKey(id, field)] = value
		}
	}
	for _, t := range sortedTasks(s) {
		v.Tasks = append(v.Tasks, Binding{Task: t.ID})
	}
	if a := s.Agents["lead"]; a != nil {
		completion := "not marked finished"
		if s.Finished {
			completion = "explicitly finished"
		}
		age := now.Sub(time.UnixMilli(s.Started)).Round(time.Second)
		if age < 0 {
			age = 0
		}
		v.Session = []reportRow{{text: Clean(s.Title, 160), keys: []echoKey{sessionKey("title")}}, {text: "Claude session · " + Clean(s.Session, 256)}, {text: "Status " + Clean(a.Status, 80) + " | " + completion, keys: []echoKey{actorKey("lead", "status"), sessionKey("finished")}}, {text: "Model " + unknown(a.Model), keys: []echoKey{actorKey("lead", "model")}}, {text: "Reported plan " + s.Progress("") + " | elapsed " + age.String(), keys: []echoKey{sessionKey("progress")}}, {text: active(s, "lead"), keys: []echoKey{actorKey("lead", "active")}}}
	}
	for _, e := range s.Events {
		v.Events = append(v.Events, observationEvent{Event: e, Binding: Binding{Actor: e.Actor}})
	}
	v.Values[sessionKey("title")] = s.Title
	v.Values[sessionKey("finished")] = strconv.FormatBool(s.Finished)
	v.Values[sessionKey("progress")] = s.Progress("")
	return v
}

// Resolve missing references into the same unknown view everywhere. Identity
// keys survive removal so a transition to unknown can still receive an echo.
func (v *observationView) resolve(b Binding) (observationEntity, bool) {
	if e, ok := v.Entities[b]; ok {
		return e, true
	}
	var keys []echoKey
	switch {
	case b.Actor != "":
		keys = []echoKey{actorKey(b.Actor, "status")}
	case b.Task != "":
		keys = []echoKey{taskKey(b.Task, "status")}
	case b.Program != "":
		keys = []echoKey{programKey(b.Program)}
	case b.Source != "":
		keys = []echoKey{sourceKey(b.Source)}
	}
	return observationEntity{Rows: []reportRow{{text: b.label()}, {text: "Not observed; state/model/progress unknown", keys: keys}}, Keys: keys}, false
}

func (v *observationView) matches(p Panel, b Binding) bool {
	e, ok := v.Entities[b]
	if !ok {
		return len(p.Sources) == 0 && (len(p.ProgramIDs) == 0 || b.Program != "" && contains(p.ProgramIDs, b.Program))
	}
	return contains(p.Sources, e.Source) && (len(p.ProgramIDs) == 0 || b.Program != "" && contains(p.ProgramIDs, b.Program))
}
func (v *observationView) selection(p Panel) []Binding {
	if p.Select != nil {
		selected, _ := v.selected(*p.Select)
		return selected
	}
	if p.Bind != nil {
		return []Binding{*p.Bind}
	}
	if len(p.Bindings) > 0 {
		return p.Bindings
	}
	if p.Kind == "agent" {
		return []Binding{{Actor: p.Actor}}
	}
	var selected []Binding
	switch p.Kind {
	case "agents":
		selected = v.Actors
		if len(p.ProgramIDs) > 0 {
			selected = programBindings(p.ProgramIDs)
		}
	case "tasks":
		selected = v.Tasks
		if len(p.ProgramIDs) > 0 {
			selected = programBindings(p.ProgramIDs)
		}
	case "programs":
		selected = v.Programs
	}
	out := []Binding{}
	for _, b := range selected {
		// Legacy filters remain typed. An actor ID never aliases a program key.
		if len(p.Actors) > 0 && (b.Actor == "" || !contains(p.Actors, b.Actor)) {
			continue
		}
		if len(p.TaskIDs) > 0 && (b.Task == "" || !contains(p.TaskIDs, b.Task)) {
			continue
		}
		out = append(out, b)
	}
	return out
}
func (v *observationView) rows(p Panel) []reportRow {
	if p.Kind == "session" && p.Bind == nil {
		if len(v.Summaries) == 0 && len(p.Sources) == 0 {
			return v.Session
		}
		out := []reportRow{}
		for _, e := range v.Summaries {
			if contains(p.Sources, e.Source) {
				out = append(out, e.Rows...)
			}
		}
		if len(out) == 0 {
			return []reportRow{{text: "No matching source scope; lifecycle/progress unknown."}}
		}
		return out
	}
	if p.Kind == "events" {
		out := []reportRow{}
		if v.EventNotice != "" {
			out = append(out, reportRow{text: v.EventNotice})
		}
		events := v.matchingEvents(p)
		for i := len(events) - 1; i >= 0; i-- {
			e := events[i]
			out = append(out, reportRow{text: time.UnixMilli(e.Time).Format("15:04:05") + " · " + Clean(e.Actor, 120) + " · " + Clean(e.Text, 400)})
		}
		if p.Select != nil {
			if scope := v.scopeNotice(*p.Select); scope != "" {
				out = append(out, reportRow{text: scope})
			}
			_, omitted := v.selected(*p.Select)
			if omitted > 0 {
				out = append(out, reportRow{text: selectionNotice(omitted)})
			}
		}
		return out
	}
	out := []reportRow{}
	for _, b := range v.selection(p) {
		if !v.matches(p, b) {
			continue
		}
		e, _ := v.resolve(b)
		if p.Kind == "programs" && p.Compact && len(e.CompactRows) > 0 {
			out = append(out, e.CompactRows...)
		} else {
			out = append(out, e.Rows...)
		}
		if p.Kind == "agents" {
			out = append(out, reportRow{})
		}
	}
	if len(out) == 0 {
		message := "No matching observations; state/progress unknown."
		if p.Kind == "tasks" {
			message = "No reported tasks; progress unknown."
			if v.EmptyTasks != "" && len(p.ProgramIDs) == 0 && len(p.Bindings) == 0 && p.Select == nil {
				message = v.EmptyTasks
			}
		}
		if p.Kind == "agents" {
			message = "No matching workers observed since opt-in. No parentage or model inferred."
			if v.EmptyActors != "" && len(p.ProgramIDs) == 0 && len(p.Bindings) == 0 && p.Select == nil {
				message = v.EmptyActors
			}
		}
		out = []reportRow{{text: message}}
	}
	if p.Select != nil {
		if scope := v.scopeNotice(*p.Select); scope != "" {
			out = append(out, reportRow{text: scope})
		}
		_, omitted := v.selected(*p.Select)
		if omitted > 0 {
			out = append(out, reportRow{text: selectionNotice(omitted)})
		}
	}
	return out
}

// All record widgets cross this one seam. Source-specific behavior belongs in
// the two adapters above/below, not in the renderer, DAG, echo or link inspector.
func observationRows(p Panel, s *State, now time.Time) []reportRow {
	if p.notice != "" {
		return []reportRow{{text: p.notice}}
	}
	if p.Kind == "programs" && p.Select == nil && p.Bind == nil && len(p.Bindings) == 0 {
		return programRows(p, s, now)
	}
	if p.Kind == "sources" && p.Select == nil && p.Bind == nil && len(p.Bindings) == 0 {
		return sourceRows(p, s, now)
	}
	if p.Kind == "text" {
		if p.Value != nil {
			rows := append([]reportRow{{text: "[External data · not native agent/task telemetry]"}}, textRows(p.Text)...)
			return append(rows, reportRow{text: renderExternalValue(s, p.Value, now)})
		}
		return append([]reportRow{{text: "[Caller-authored text · not hook telemetry]"}}, textRows(p.Text)...)
	}
	return observations(s, now).rows(p)
}
func textRows(lines []string) []reportRow {
	out := make([]reportRow, len(lines))
	for i, line := range lines {
		out[i].text = line
	}
	return out
}

func (v *observationView) matchingEvents(p Panel) []Event {
	selected := p.Bindings
	if p.Bind != nil || p.Select != nil {
		selected = v.selection(p)
	}
	filtered := p.Select != nil || p.Bind != nil || len(p.Bindings) > 0
	out := []Event{}
	for _, e := range v.Events {
		if !contains(p.Sources, e.Source) || len(p.Actors) > 0 && !contains(p.Actors, e.Binding.Actor) || len(p.ProgramIDs) > 0 && (e.Binding.Program == "" || !contains(p.ProgramIDs, e.Binding.Program)) {
			continue
		}
		if filtered {
			found := false
			for _, b := range selected {
				found = found || b == e.Binding || b.Source != "" && b.Source == e.Source
			}
			if !found {
				continue
			}
		}
		out = append(out, e.Event)
	}
	return out
}

func programBindings(ids []string) []Binding {
	out := make([]Binding, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if !seen[id] {
			out = append(out, Binding{Program: id})
			seen[id] = true
		}
	}
	return out
}
