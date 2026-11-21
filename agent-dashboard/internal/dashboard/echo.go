package dashboard

import (
	"reflect"
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
	return observations(s, time.Time{}).Values
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

func nodeEchoKeys(n DAGNode, s *State) []echoKey {
	if n.Bind == nil {
		return nil
	}
	e, _ := observations(s, time.Time{}).resolve(*n.Bind)
	return e.Keys
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
	beforeView, afterView := observations(v.State, time.Time{}), observations(next, time.Time{})
	// Rex diagnostic page 3 is still source metadata, not the event panel.
	if v.Page == 2 && v.Offset > 0 && next.ProgramReports == nil {
		before, after := beforeView.matchingEvents(Panel{}), afterView.matchingEvents(Panel{})
		added := appendedEvents(before, after)
		v.Offset = min(v.Offset+len(added), max(0, len(after)-1))
	}
	beforeLayout, afterLayout := v.canvasLayout(), presentationLayout(next, time.Time{})
	if beforeLayout == nil || afterLayout == nil {
		return
	}
	oldPanels := map[string]Panel{}
	for _, p := range leaves(beforeLayout.Root) {
		oldPanels[p.ID] = p
	}
	for _, p := range leaves(afterLayout.Root) {
		old := oldPanels[p.ID]
		if p.Kind != "events" || old.Kind != "events" || !sameEventFilter(old, p) || v.Offsets[p.ID] <= 0 {
			continue
		}
		before, after := beforeView.matchingEvents(p), afterView.matchingEvents(p)
		added := appendedEvents(before, after)
		v.Offsets[p.ID] = min(v.Offsets[p.ID]+len(added), max(0, len(after)-1))
	}
}
func sameEventFilter(a, b Panel) bool {
	if !reflect.DeepEqual(a.Select, b.Select) || !reflect.DeepEqual(a.Bind, b.Bind) {
		return false
	}
	if strings.Join(a.Actors, "\x00") != strings.Join(b.Actors, "\x00") || strings.Join(a.Sources, "\x00") != strings.Join(b.Sources, "\x00") || strings.Join(a.ProgramIDs, "\x00") != strings.Join(b.ProgramIDs, "\x00") || len(a.Bindings) != len(b.Bindings) {
		return false
	}
	for i := range a.Bindings {
		if a.Bindings[i] != b.Bindings[i] {
			return false
		}
	}
	return true
}
