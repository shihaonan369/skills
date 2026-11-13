package dashboard

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const echoDuration = 1200 * time.Millisecond

type echoKey struct{ kind, id, field string }
type changeEcho struct {
	values  map[echoKey]string
	changed map[echoKey]bool
	until   time.Time
}

func actorKey(id, field string) echoKey { return echoKey{"actor", id, field} }
func taskKey(id, field string) echoKey  { return echoKey{"task", id, field} }
func sessionKey(field string) echoKey   { return echoKey{"session", "", field} }

// Compare only observed fields, never wall-clock age, spinners, layout edits or
// event-buffer indices. Active calls retain their IDs, including parallel calls
// with the same tool name. No effects or acknowledgement are persisted.
func echoValues(s *State) map[echoKey]string {
	out := map[echoKey]string{}
	calls := map[string][]string{}
	for id, t := range s.Tools {
		calls[t.Actor] = append(calls[t.Actor], fmt.Sprintf("%q:%q:%t:%d", id, t.Name, t.Waiting, t.Started))
	}
	for id, a := range s.Agents {
		if a == nil {
			continue
		}
		sort.Strings(calls[id])
		for field, value := range map[string]string{
			"status": a.Status, "role": a.Role, "model": a.Model,
			"tools": strconv.Itoa(a.Tools), "errors": strconv.Itoa(a.Errors),
			"active": strings.Join(calls[id], "\n"), "progress": s.Progress(id),
		} {
			out[actorKey(id, field)] = value
		}
	}
	for id, t := range s.Tasks {
		out[taskKey(id, "status")] = t.Status
		out[taskKey(id, "subject")] = t.Subject
		out[taskKey(id, "owner")] = t.Owner
		out[taskKey(id, "blocked")] = strings.Join(t.BlockedBy, "\n")
	}
	out[sessionKey("title")] = s.Title
	out[sessionKey("finished")] = strconv.FormatBool(s.Finished)
	out[sessionKey("progress")] = s.Progress("")
	return out
}
func (e *changeEcho) observe(s *State, at time.Time, baseline, quiet bool) {
	values := echoValues(s)
	if baseline || e.values == nil {
		e.values, e.changed, e.until = values, nil, time.Time{}
		return
	}
	changed := map[echoKey]bool{}
	for k, value := range values {
		old, existed := e.values[k]
		// Newly observed unknown/empty fields do not pretend to be fresh facts.
		if (!existed && value != "" && value != "unknown") || (existed && value != old) {
			changed[k] = true
		}
	}
	for k := range e.values {
		if _, exists := values[k]; !exists {
			changed[k] = true
		}
	}
	e.values = values
	if len(changed) == 0 {
		return // duplicate snapshots cannot restart or extend an echo
	}
	e.changed, e.until = changed, time.Time{}
	if !quiet {
		e.until = at.Add(echoDuration)
	}
}
func (e changeEcho) marks(keys []echoKey, now time.Time, motion bool) (changed, hot bool) {
	for _, k := range keys {
		changed = changed || e.changed[k]
	}
	return changed, changed && motion && now.Before(e.until)
}
func echoPrefix(changed, hot bool) string {
	if hot {
		return "▸ "
	}
	if changed {
		return "· "
	}
	return "  "
}
func echoLine(text string, keys []echoKey, echo changeEcho, now time.Time, motion bool, width int) string {
	changed, hot := echo.marks(keys, now, motion)
	text = echoPrefix(changed, hot) + clip(Clean(text, 2000), max(0, width-2))
	if hot {
		color := palette["purple"]
		for _, k := range keys {
			if k.field == "errors" && echo.changed[k] {
				color = palette["amber"] // an error count, not an agent-failed state
			}
		}
		text = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Background(lipgloss.Color("#292338")).Render(text)
	}
	return text
}

// One key list per existing row: the row count and caller geometry do not depend
// on whether an effect is on, off, expired or acknowledged.
func panelEchoKeys(p Panel, s *State) [][]echoKey {
	agent := func(id string) [][]echoKey {
		return [][]echoKey{
			{actorKey(id, "role"), actorKey(id, "status")},
			{actorKey(id, "model"), actorKey(id, "progress")},
			{actorKey(id, "tools"), actorKey(id, "errors"), actorKey(id, "active")},
		}
	}
	switch p.Kind {
	case "session":
		return [][]echoKey{{sessionKey("title")}, nil, {actorKey("lead", "status"), sessionKey("finished")}, {actorKey("lead", "model")}, {sessionKey("progress")}, {actorKey("lead", "active")}}
	case "agent":
		if s.Agents[p.Actor] != nil {
			return agent(p.Actor)
		}
	case "agents":
		out := [][]echoKey{}
		for _, a := range sortedAgents(s) {
			if contains(p.Actors, a.ID) {
				out = append(out, agent(a.ID)...)
				out = append(out, nil)
			}
		}
		return out
	case "tasks":
		out := [][]echoKey{}
		for _, t := range sortedTasks(s) {
			if contains(p.TaskIDs, t.ID) {
				out = append(out, []echoKey{taskKey(t.ID, "status"), taskKey(t.ID, "subject")}, []echoKey{taskKey(t.ID, "owner"), taskKey(t.ID, "blocked")})
			}
		}
		return out
	}
	return nil
}
func nodeEchoKeys(n DAGNode, s *State) []echoKey {
	if n.Bind == nil {
		return nil
	}
	if n.Bind.Actor != "" {
		keys := []echoKey{actorKey(n.Bind.Actor, "status")}
		if a := s.Agents[n.Bind.Actor]; a != nil && a.Status == "running" {
			keys = append(keys, actorKey(n.Bind.Actor, "tools"))
		}
		return keys
	}
	return []echoKey{taskKey(n.Bind.Task, "status")}
}
func (v Viewer) quietEcho() bool {
	return v.MotionOff || v.Help || v.Page != 0 || v.Offsets[v.focus()] > 0
}

// Keep the historical top event anchored when bounded newest-first buffers gain
// entries. Matching the suffix/prefix handles rotation without using timestamps
// as unique IDs; repeated identical events remain legitimate buffer entries.
func appendedEvents(old, next []Event) []Event {
	for overlap := min(len(old), len(next)); overlap > 0; overlap-- {
		equal := true
		for i := 0; i < overlap; i++ {
			if old[len(old)-overlap+i] != next[i] {
				equal = false
				break
			}
		}
		if equal {
			return next[overlap:]
		}
	}
	return nil // wholesale replacement/empty baseline: do not invent continuity
}
func (v *Viewer) anchorEventHistory(next *State) {
	if v.State == nil {
		return
	}
	added := appendedEvents(v.State.Events, next.Events)
	if v.Page == 2 && v.Offset > 0 {
		v.Offset = min(v.Offset+len(added), max(0, len(next.Events)-1))
	}
	if v.State.Layout == nil || next.Layout == nil {
		return
	}
	oldPanels := map[string]Panel{}
	for _, p := range leaves(v.State.Layout.Root) {
		oldPanels[p.ID] = p
	}
	for _, p := range leaves(next.Layout.Root) {
		old := oldPanels[p.ID]
		if p.Kind != "events" || old.Kind != "events" || strings.Join(old.Actors, "\x00") != strings.Join(p.Actors, "\x00") || v.Offsets[p.ID] <= 0 {
			continue
		}
		n, total := 0, 0
		for _, e := range added {
			if contains(p.Actors, e.Actor) {
				n++
			}
		}
		for _, e := range next.Events {
			if contains(p.Actors, e.Actor) {
				total++
			}
		}
		v.Offsets[p.ID] = min(v.Offsets[p.ID]+n, max(0, total-1))
	}
}
