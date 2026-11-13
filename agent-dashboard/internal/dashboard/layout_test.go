package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestCallerOwnsArrangementBindingsAndLiveReplacement(t *testing.T) {
	r := setup(t)
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "actual-id", AgentType: "arbitrary-caller-role"})
	raw := []byte(`{"version":1,"name":"Anything, not a preset","root":{"kind":"row","children":[{"id":"left","kind":"text","text":["LEFT"]},{"id":"right","kind":"agent","actor":"actual-id"}]}}`)
	if e := SetLayout(r, "s", raw); e != nil {
		t.Fatal(e)
	}
	v := Viewer{State: snap(t, r), Width: 100, Height: 24}
	frame := v.View()
	found := false
	for _, line := range strings.Split(frame, "\n") {
		a, b := strings.Index(line, "left · text"), strings.Index(line, "right · agent")
		if a >= 0 && b > a {
			found = true
		}
	}
	if !found {
		t.Fatal("caller row not honored", frame)
	}
	if strings.Contains(frame, "advisor") || strings.Contains(frame, "reproduce") || strings.Contains(frame, "ROUTING") {
		t.Fatal("workflow policy leaked")
	}
	raw = []byte(`{"version":1,"name":"Rearranged","root":{"kind":"column","children":[{"id":"right","kind":"agent","actor":"not-yet-observed"},{"id":"left","kind":"text","text":["BOTTOM"]}]}}`)
	if e := SetLayout(r, "s", raw); e != nil {
		t.Fatal(e)
	}
	v.State = snap(t, r)
	frame = v.View()
	if strings.Index(frame, "Not observed") > strings.Index(frame, "BOTTOM") || !strings.Contains(frame, "Not observed") {
		t.Fatal("caller column/binding not honored", frame)
	}
	if len(v.State.Agents) != 2 {
		t.Fatal("presentation spawned fictional agents")
	}
	if s := snap(t, r); len(s.Tasks) != 0 || s.Finished {
		t.Fatal("layout changed execution state")
	}
}
func TestLayoutValidationRollbackAndLimits(t *testing.T) {
	r := setup(t)
	good := []byte(`{"version":1,"name":"x","root":{"id":"s","kind":"session"}}`)
	if e := SetLayout(r, "s", good); e != nil {
		t.Fatal(e)
	}
	invalid := []string{`null`, `{}`, `{"version":1,"root":{"id":"x","kind":"invented"}}`, `{"version":1,"unknown":true,"root":{"id":"x","kind":"text"}}`, `{"version":1,"root":{"kind":"row","children":[]}}`, `{"version":1,"root":{"kind":"row","children":[{"id":"x","kind":"session"},{"id":"x","kind":"tasks"}]}}`, `{"version":1,"root":{"id":"x","kind":"agent"}}`, `{"version":1,"root":{"id":"x","kind":"session","size":-1}}`, `{"version":1,"root":{"id":"x","kind":"events","color":"\u001b[31m"}}`, `{"version":1,"root":{"id":"x","kind":"session","actor":"fake"}}`, `{"version":1,"root":{"id":"x","kind":"session","text":["fake"]}}`, string(good) + ` {}`}
	for _, raw := range invalid {
		if e := SetLayout(r, "s", []byte(raw)); e == nil {
			t.Errorf("invalid accepted %s", raw)
		}
	}
	if snap(t, r).Layout.Name != "x" {
		t.Fatal("invalid layout partially replaced state")
	}
	p := Panel{ID: "leaf", Kind: "text"}
	for i := 0; i < 9; i++ {
		p = Panel{Kind: "column", Children: []Panel{p}}
	}
	raw, _ := json.Marshal(Layout{Version: 1, Root: p})
	if _, e := ParseLayout(raw); e == nil {
		t.Fatal("unbounded recursion")
	}
	children := []Panel{}
	for i := 0; i < 65; i++ {
		children = append(children, Panel{ID: strings.Repeat("x", i+1), Kind: "text"})
	}
	raw, _ = json.Marshal(Layout{Version: 1, Root: Panel{Kind: "row", Children: children}})
	if _, e := ParseLayout(raw); e == nil {
		t.Fatal("unbounded nodes")
	}
}
func TestLayoutFiltersScrollAndFocus(t *testing.T) {
	r := setup(t)
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "show", AgentType: "role-show"})
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "hide", AgentType: "role-hide"})
	s := snap(t, r)
	p := Panel{Kind: "agents", Actors: []string{"show"}}
	data := strings.Join(panelData(p, s, time.Now()), "\n")
	if !strings.Contains(data, "role-show") || strings.Contains(data, "role-hide") {
		t.Fatal("actor filter ignored")
	}
	s.Layout = &Layout{Version: 1, Root: Panel{Kind: "column", Children: []Panel{{ID: "a", Kind: "events"}, {ID: "b", Kind: "tasks"}}}}
	v := Viewer{State: s}
	v.scroll(5, false)
	v.Selected = 1
	v.scroll(3, false)
	if v.Offsets["a"] != 5 || v.Offsets["b"] != 3 {
		t.Fatal("buffers share scroll position")
	}
	m, _ := v.Update(tea.KeyMsg{Type: tea.KeyHome})
	v = m.(Viewer)
	if v.Offsets["b"] != 0 || v.Offsets["a"] != 5 {
		t.Fatal("Home reset wrong buffer")
	}
}
func TestExamplesResponsiveGeometryAndSafety(t *testing.T) {
	r := t.TempDir()
	SeedDemo(r, "demo")
	s, _ := Snapshot(r, "demo")
	for _, name := range []string{"tdd", "review"} {
		raw, e := os.ReadFile(filepath.Join("..", "..", "examples", "ai-hero", name+".layout.json"))
		if e != nil {
			t.Fatal(e)
		}
		l, e := ParseLayout(raw)
		if e != nil {
			t.Fatal(e)
		}
		s.Layout = l
		for _, w := range []int{28, 40, 68, 80, 100, 160} {
			for _, h := range []int{8, 12, 24, 40} {
				frame := (Viewer{State: s, Width: w, Height: h}).View()
				lines := strings.Split(frame, "\n")
				if len(lines) != h {
					t.Errorf("height %d/%d", len(lines), h)
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > w {
						t.Error("width overflow")
					}
					if strings.ContainsAny(line, "\x1b\a\r\u202e") {
						t.Error("unsafe terminal sequence")
					}
				}
			}
		}
	}
	raw := []byte(`{"version":1,"root":{"id":"x","kind":"text","title":"\u001b[2J","text":["Text 👩‍💻 é\u001b]52;c;secret\u0007\n\u202e"]}}`)
	l, e := ParseLayout(raw)
	if e != nil {
		t.Fatal(e)
	}
	s.Layout = l
	frame := (Viewer{State: s, Width: 40, Height: 15}).View()
	if strings.ContainsAny(frame, "\x1b\a\r\u202e") {
		t.Fatal("caller text injection")
	}
}
func TestOptInDoesNotInventLayout(t *testing.T) {
	r := setup(t)
	s := snap(t, r)
	if s.Layout != nil {
		t.Fatal("start inserted workflow")
	}
	frame := (Viewer{State: s, Width: 100, Height: 24}).View()
	if !strings.Contains(frame, "Layout not supplied") {
		t.Fatal("no unknown-layout warning")
	}
}

func TestWorkerTurnDoesNotChangeLeadCompletion(t *testing.T) {
	r := setup(t)
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "w"})
	Finish(r, "s", "caller declaration")
	send(t, r, Payload{Kind: "UserPromptSubmit", AgentID: "w"})
	send(t, r, Payload{Kind: "Stop", AgentID: "w"})
	s := snap(t, r)
	if !s.Finished || s.Agents["lead"].Status != "done" || s.Agents["w"].Status != "waiting" {
		t.Fatal("worker turn changed lead completion")
	}
}

func TestSnapshotsCannotRegressCanvasRevision(t *testing.T) {
	v := Viewer{State: &State{Revision: 7, Title: "new"}}
	m, _ := v.Update(snapshotMsg{state: &State{Revision: 6, Title: "stale"}})
	if m.(Viewer).State.Title != "new" {
		t.Fatal("slow old snapshot overwrote current canvas")
	}
	if Clean("👩‍💻 é", 40) != "👩‍💻 é" {
		t.Fatal("Unicode grapheme shaping changed")
	}
}
