package dashboard

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type Viewer struct {
	Root, Session          string
	State                  *State
	Width, Height          int
	Page, Selected, Offset int
	Offsets                map[string]int
	XOffsets               map[string]int
	Paused, Help           bool
	MotionOff              bool
	MotionTime             time.Time
	Error                  string
}
type tickMsg time.Time
type motionMsg time.Time

// Drawing and observation run on independent clocks. A motion message never
// reads a snapshot, acquires a state lock or creates telemetry.
func motionTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return motionMsg(t) })
}

type snapshotMsg struct {
	state *State
	err   error
}

func (v Viewer) load() tea.Msg { s, e := Snapshot(v.Root, v.Session); return snapshotMsg{s, e} }
func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}
func (v Viewer) Init() tea.Cmd { return tea.Batch(v.load, tick(), motionTick()) }
func (v Viewer) focus() string {
	if v.State == nil || v.State.Layout == nil {
		return ""
	}
	panels := leaves(v.State.Layout.Root)
	if len(panels) == 0 {
		return ""
	}
	return panels[v.Selected%len(panels)].ID
}
func (v *Viewer) scroll(delta int, home bool) {
	if v.Page != 0 {
		v.Offset = max(0, v.Offset+delta)
		if home {
			v.Offset = 0
		}
		return
	}
	if v.Offsets == nil {
		v.Offsets = map[string]int{}
	}
	id := v.focus()
	_, ymax := v.dagScrollLimits(id)
	v.Offsets[id] = max(0, min(v.Offsets[id], ymax)+delta)
	v.Offsets[id] = min(v.Offsets[id], ymax)
	if home {
		v.Offsets[id] = 0
	}
}

// Non-DAG buffers keep their existing scroll behavior. Only DAGs have a
// two-dimensional natural canvas with viewport-sized upper bounds.
func (v Viewer) dagScrollLimits(id string) (int, int) {
	unbounded := int(^uint(0) >> 1)
	if v.State == nil || v.State.Layout == nil {
		return unbounded, unbounded
	}
	for _, p := range leaves(v.State.Layout.Root) {
		if p.ID != id || p.Kind != "dag" || p.DAG == nil {
			continue
		}
		if len(p.DAG.Nodes) == 0 || len(p.DAG.Nodes) > 64 || len(p.DAG.Edges) > 256 {
			return 0, 0
		}
		geo, err := placeDAG(p.DAG)
		if err != nil {
			return 0, 0
		}
		w, h := v.Width, v.Height
		if w <= 0 {
			w = 80
		}
		if h <= 0 {
			h = 24
		}
		w = min(w, 1000)
		h = min(h, 500)
		cw, ch, ok := panelDimensions(v.State.Layout.Root, id, w, h-4)
		if !ok || cw < 8 || ch < 4 {
			return 0, 0
		}
		return max(0, geo.width-(cw-2)), max(0, geo.height-(ch-3))
	}
	return unbounded, unbounded
}
func (v Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		v.Width = m.Width
		v.Height = m.Height
	case motionMsg:
		if !v.Paused && !v.MotionOff {
			v.MotionTime = time.Time(m)
		}
		return v, motionTick()
	case snapshotMsg:
		if v.Paused {
			return v, nil
		}
		if m.err != nil {
			v.Error = Clean(m.err.Error(), 180)
		} else if v.State == nil || m.state.Revision >= v.State.Revision {
			v.State = m.state
			v.Error = ""
		}
	case tickMsg:
		if v.Paused {
			return v, tick()
		}
		return v, tea.Batch(v.load, tick())
	case tea.KeyMsg:
		switch m.String() {
		case "q", "ctrl+c":
			return v, tea.Quit
		case "p":
			v.Paused = !v.Paused
			if v.Paused && v.MotionTime.IsZero() {
				v.MotionTime = time.Now()
			}
		case "a":
			v.MotionOff = !v.MotionOff
		case "?":
			v.Help = !v.Help
		case "esc":
			v.Help = false
		case "1":
			v.Page = 0
			v.Offset = 0
		case "2":
			v.Page = 1
			v.Offset = 0
		case "3", "l":
			v.Page = 2
			v.Offset = 0
		case "tab", "j":
			v.Selected++
		case "shift+tab", "k":
			if v.Selected > 0 {
				v.Selected--
			}
		case "down":
			v.scroll(1, false)
		case "up":
			v.scroll(-1, false)
		case "pgdown":
			v.scroll(10, false)
		case "pgup":
			v.scroll(-10, false)
		case "left", "right":
			if v.Page == 0 && v.State != nil && v.State.Layout != nil {
				for _, p := range leaves(v.State.Layout.Root) {
					if p.ID == v.focus() && p.Kind == "dag" {
						if v.XOffsets == nil {
							v.XOffsets = map[string]int{}
						}
						delta := 4
						if m.String() == "left" {
							delta = -4
						}
						xmax, _ := v.dagScrollLimits(p.ID)
						v.XOffsets[p.ID] = max(0, min(xmax, min(v.XOffsets[p.ID], xmax)+delta))
					}
				}
			}
		case "home":
			v.scroll(0, true)
			if v.Page == 0 {
				delete(v.XOffsets, v.focus())
			}
		}
	}
	return v, nil
}
func shade(color, text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(Clean(text, 4000))
}
func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "")
}
func unknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return Clean(s, 240)
}
func sortedAgents(s *State) []*Agent {
	a := []*Agent{}
	for id, w := range s.Agents {
		if id != "lead" {
			a = append(a, w)
		}
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].Started == a[j].Started {
			return a[i].ID < a[j].ID
		}
		return a[i].Started < a[j].Started
	})
	return a
}
func sortedTasks(s *State) []Task {
	a := []Task{}
	for _, t := range s.Tasks {
		if t.Status != "deleted" {
			a = append(a, t)
		}
	}
	sort.Slice(a, func(i, j int) bool { return a[i].ID < a[j].ID })
	return a
}
func active(s *State, actor string) string {
	a := []string{}
	for _, t := range s.Tools {
		if t.Actor == actor {
			x := Clean(t.Name, 100)
			if t.Waiting {
				x += " (permission)"
			}
			a = append(a, x)
		}
	}
	sort.Strings(a)
	if len(a) == 0 {
		return "no observed active tools"
	}
	return strings.Join(a, ", ")
}
func (v Viewer) View() string {
	now := time.Now()
	if v.Paused && !v.MotionTime.IsZero() {
		now = v.MotionTime
	}
	return v.Render(now)
}
func (v Viewer) Render(now time.Time) string {
	w, h := v.Width, v.Height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	w = min(w, 1000)
	h = min(h, 500) // Bound allocation even for pathological plain-frame dimensions.
	if w < 28 || h < 8 {
		return clip("Resize to ≥28 columns ×8 rows; q closes viewer only", w)
	}
	s := v.State
	if s == nil {
		return clip("Agent Dashboard · waiting for snapshot: "+v.Error, w)
	}
	label := "OBSERVED TELEMETRY"
	if s.Demo {
		label = "DEMO · SYNTHETIC · NOT REAL AGENTS"
	}
	if !s.Active {
		label += " · collection OFF"
	}
	if v.Paused {
		label += " · viewer PAUSED"
	}
	name := "no caller layout"
	if s.Layout != nil {
		name = s.Layout.Name
	}
	lines := []string{shade("#F5C2A1", "AGENT DASHBOARD · "+Clean(name, 160)), shade("#A6E3A1", label+" | 1 Canvas · 2 Tasks · 3 Events")}
	motionNow := now
	if !v.MotionTime.IsZero() {
		motionNow = v.MotionTime
	}
	if v.MotionOff {
		motionNow = time.Unix(0, 0)
	}
	bodyH := h - 4
	hidden := 0
	if v.Help {
		lines = append(lines, fit([]string{"Caller defines layout; dashboard supplies panels only.", "Tab / Shift-Tab (j/k): focus next/previous panel", "↑↓ / PgUp/PgDn: scroll; ←→: pan DAG; Home: reset", "1: caller canvas; 2/3: task/event diagnostic buffers", "p: pause viewer + animation; a: toggle motion; ?: help", "Flow packets decorate declared edges, not measured traffic.", "q / Ctrl-C: close viewer; agents and hooks continue", "Layouts bind observed IDs; unobserved bindings stay unknown."}, w, bodyH)...)
	} else if v.Error != "" {
		lines = append(lines, fit([]string{"Snapshot read error · last view retained", v.Error}, w, bodyH)...)
	} else {
		var root Panel
		if v.Page == 1 {
			root = Panel{ID: "task-buffer", Kind: "tasks", Title: "Task diagnostic buffer"}
		} else if v.Page == 2 {
			root = Panel{ID: "event-buffer", Kind: "events", Title: "Event diagnostic buffer"}
		} else if s.Layout != nil {
			root = s.Layout.Root
		} else {
			// An inspector is not an implicit workflow or a fallback orchestration graph.
			root = Panel{Kind: "column", Children: []Panel{{ID: "unconfigured", Kind: "text", Size: 5, Title: "Layout not supplied", Text: []string{"Caller chooses panels, positions and data bindings.", "Set a layout using: layout --session ID --json @file"}}, {ID: "session-inspector", Kind: "session", Title: "Session inspector"}}}
		}
		offsets := v.Offsets
		if v.Page != 0 {
			offsets = map[string]int{root.ID: v.Offset}
		}
		focus := v.focus()
		if v.Page != 0 {
			focus = root.ID
		}
		lines = append(lines, renderPanel(root, s, w, bodyH, now, canvasOptions{Focus: focus, Offsets: offsets, XOffsets: v.XOffsets, Hidden: &hidden, Motion: !v.MotionOff, Clock: motionNow})...)
	}
	since := "no hooks observed"
	if s.LastHook > 0 {
		age := now.Sub(time.UnixMilli(s.LastHook)).Round(time.Second)
		if age < 0 {
			age = 0
		}
		since = fmt.Sprintf("last hook %s ago · quiet ≠ offline", age)
	}
	lines = append(lines, fmt.Sprintf("%s | %d hooks · %d dropped · %d panels too small", since, s.HookCount, s.Dropped, hidden), "q viewer only · Tab focus · ↑↓ scroll · ←→ DAG · a motion · p pause · ? help")
	for i := range lines {
		lines[i] = clip(lines[i], w)
	}
	return strings.Join(lines, "\n")
}
func RunViewer(root, session string) error {
	if _, e := Snapshot(root, session); e != nil {
		return e
	}
	_, e := tea.NewProgram(Viewer{Root: root, Session: session, Width: 80, Height: 24}, tea.WithAltScreen()).Run()
	return e
}
