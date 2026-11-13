package dashboard

import (
	"encoding/json"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercise the caller's existing publication/snapshot/viewer interface, not a
// second graph store or an orchestration engine.
func TestDAGPanelComposesAndBindsObservedState(t *testing.T) {
	root := setup(t)
	raw := []byte(`{"version":1,"name":"Caller graph","root":{"kind":"row","children":[{"id":"graph","kind":"dag","dag":{"version":1,"nodes":[{"id":"lead-node","label":"Lead","bind":{"actor":"lead"}},{"id":"work","label":"Reported work","bind":{"task":"1"}},{"id":"missing","label":"Not started","bind":{"actor":"never-seen"}},{"id":"note","label":"Caller annotation"}],"edges":[{"from":"lead-node","to":"work"},{"from":"lead-node","to":"missing"},{"from":"work","to":"note"},{"from":"note","to":"lead-node","style":"return"}]}},{"id":"events","kind":"events"}]}}`)
	if err := SetLayout(root, "s", raw); err != nil {
		t.Fatal(err)
	}
	if err := Plan(root, "s", []byte(`[{"id":"1","subject":"Actual reported work","status":"blocked"}]`)); err != nil {
		t.Fatal(err)
	}
	s := snap(t, root)
	frame := (Viewer{State: s, Width: 180, Height: 44}).Render(time.Unix(100, 0))
	for _, want := range []string{"graph · dag", "events · events", "Lead", "running", "blocked", "unobserved", "declared only", "Declared edges"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	if len(s.Agents) != 1 || len(s.Tasks) != 1 || s.Finished || s.HookCount != 0 {
		t.Fatal("graph changed execution state")
	}
}

func TestDAGRejectsInvalidGraphsWithoutReplacingLayout(t *testing.T) {
	root := setup(t)
	good := `{"version":1,"name":"Keep me","root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a"},{"id":"b"}],"edges":[{"from":"a","to":"b"}]}}}`
	if err := SetLayout(root, "s", []byte(good)); err != nil {
		t.Fatal(err)
	}
	invalid := []string{
		strings.Replace(good, `"dag":{"version":1`, `"dag":{"version":2`, 1),
		strings.Replace(good, `"nodes":[{"id":"a"},{"id":"b"}]`, `"nodes":[]`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":"a"}`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":""}`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":"\u001b"}`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":"b","bind":{}}`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":"b","bind":{"actor":"lead","task":"1"}}`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":"b","bind":{"actor":"\u001b"}}`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":"b","color":"fake"}`, 1),
		strings.Replace(good, `{"id":"b"}`, `{"id":"b","meta":["1","2","3","4","5"]}`, 1),
		strings.Replace(good, `"to":"b"`, `"to":"missing"`, 1),
		strings.Replace(good, `"to":"b"`, `"to":"b","style":"fake"`, 1),
		strings.Replace(good, `"to":"b"`, `"to":"b","execute":true`, 1),
		strings.Replace(good, `"edges":[{"from":"a","to":"b"}]`, `"edges":[{"from":"a","to":"b"},{"from":"a","to":"b"}]`, 1),
		strings.Replace(good, `"edges":[{"from":"a","to":"b"}]`, `"edges":[{"from":"a","to":"b"},{"from":"b","to":"a"}]`, 1),
		strings.Replace(good, `"to":"b"`, `"to":"a"`, 1),
		strings.Replace(good, `"kind":"dag"`, `"kind":"text"`, 1),
		strings.Replace(good, `"kind":"dag"`, `"kind":"dag","actor":"lead"`, 1),
		`{"version":1,"root":{"id":"g","kind":"dag"}}`,
	}
	for _, raw := range invalid {
		if err := SetLayout(root, "s", []byte(raw)); err == nil {
			t.Errorf("invalid accepted: %s", raw)
		}
	}
	after := snap(t, root)
	if after.Layout.Name != "Keep me" || after.Revision != 2 {
		t.Fatal("rejected graph changed persisted layout/revision")
	}
}

func TestDAGShowsLiteralLifecycleAndTaskStates(t *testing.T) {
	root := setup(t)
	raw := []byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a","bind":{"actor":"lead"}},{"id":"t","bind":{"task":"1"}}]}}}`)
	if err := SetLayout(root, "s", raw); err != nil {
		t.Fatal(err)
	}
	s := snap(t, root)
	for _, status := range []string{"running", "waiting", "returned", "stopped", "failed", "API error", "unconfirmed", "permission", "ended", "done", ""} {
		s.Agents["lead"].Status = status
		frame := (Viewer{State: s, Width: 120, Height: 24}).Render(time.Unix(100, 0))
		want := status
		if want == "" {
			want = "unknown"
		}
		if !strings.Contains(frame, want) {
			t.Fatalf("lost literal state %q\n%s", want, frame)
		}
		if status != "done" && strings.Contains(frame, "✓") {
			t.Fatalf("falsely marked %s complete", status)
		}
	}
	for _, status := range []string{"pending", "in_progress", "completed", "blocked", "deleted", ""} {
		s.Agents["lead"].Status = "waiting"
		s.Tasks["1"] = Task{ID: "1", Status: status}
		frame := (Viewer{State: s, Width: 120, Height: 24}).Render(time.Unix(100, 0))
		want := status
		if want == "" {
			want = "unknown"
		}
		if !strings.Contains(frame, want) {
			t.Fatalf("lost task state %q\n%s", want, frame)
		}
		if (status == "completed") != strings.Contains(frame, "✓") {
			t.Fatalf("wrong completion marker for %q", status)
		}
	}
}

func TestDAGViewportFocusPanScrollAndReset(t *testing.T) {
	root := setup(t)
	raw := []byte(`{"version":1,"root":{"kind":"row","children":[{"id":"g1","kind":"dag","dag":{"version":1,"nodes":[{"id":"a","label":"LEFTMOST"},{"id":"b","label":"MIDDLE"},{"id":"c","label":"RIGHTMOST"}]}},{"id":"g2","kind":"dag","dag":{"version":1,"nodes":[{"id":"a","label":"OTHER"}]}}]}}`)
	if err := SetLayout(root, "s", raw); err != nil {
		t.Fatal(err)
	}
	v := Viewer{State: snap(t, root), Width: 90, Height: 24, MotionTime: time.Unix(100, 0)}
	first := v.View()
	if strings.Contains(first, "RIGHTMOST") {
		t.Fatal("graph squeezed/reordered instead of clipping")
	}
	key := func(k tea.KeyType) { m, _ := v.Update(tea.KeyMsg{Type: k}); v = m.(Viewer) }
	for i := 0; i < 13; i++ {
		key(tea.KeyRight)
	}
	if frame := v.View(); !strings.Contains(frame, "RIGHTMOST") || !strings.Contains(frame, "OTHER") {
		t.Fatal("horizontal pan failed or changed another graph", frame)
	}
	for i := 0; i < 100; i++ {
		key(tea.KeyRight)
	}
	key(tea.KeyLeft)
	if !strings.Contains(v.View(), "x 54/58") {
		t.Fatal("pan accumulated invisible overscroll", v.View())
	}
	key(tea.KeyTab)
	key(tea.KeyHome)
	if !strings.Contains(v.View(), "RIGHTMOST") {
		t.Fatal("Home reset another panel")
	}
	key(tea.KeyShiftTab)
	key(tea.KeyHome)
	if v.View() != first {
		t.Fatal("Home did not reset graph viewport")
	}
	for i := 0; i < 20; i++ {
		key(tea.KeyDown)
	}
	key(tea.KeyHome)
	if v.View() != first {
		t.Fatal("vertical scroll reset failed")
	}
}

func TestDAGExamplesUnicodeGeometryAndDefensiveRendering(t *testing.T) {
	root := t.TempDir()
	if err := SeedDemo(root, "demo"); err != nil {
		t.Fatal(err)
	}
	s, err := Snapshot(root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob("../../examples/*/*.layout.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		l, err := ParseLayout(raw)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		s.Layout = l
		for _, w := range []int{28, 40, 68, 80, 120, 160} {
			for _, h := range []int{8, 12, 24, 50} {
				frame := (Viewer{State: s, Width: w, Height: h}).Render(time.Unix(100, 0))
				assertDAGFrame(t, frame, w, h)
			}
		}
	}
	raw := []byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a","label":"Text 👩‍💻 é\u001b]52;c;secret\u0007\n\u202e","meta":["DATA\u001b[2J"]},{"id":"b","label":"NEXT"},{"id":"c","label":"END"}],"edges":[{"from":"a","to":"b"},{"from":"b","to":"c"},{"from":"a","to":"c","label":"CROSS-RANK"},{"from":"c","to":"a","style":"return","label":"RETURN"}]}}}`)
	l, err := ParseLayout(raw)
	if err != nil {
		t.Fatal(err)
	}
	s.Layout = l
	v := Viewer{State: s, Width: 100, Height: 50, Links: map[string]linkFocus{"g": {Node: "a", Scope: 3, Inspect: true}}}
	frame := v.Render(time.Unix(100, 0))
	for _, want := range []string{"Text 👩‍💻 é", "NEXT", "END", "CROSS-RANK", "RETURN", "[return]", "DATA"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	if strings.Index(frame, "NEXT") < strings.Index(frame, "Text") || strings.Index(frame, "END") < strings.Index(frame, "NEXT") {
		t.Fatal("topological rank order lost")
	}
	for x := 0; x < 12; x++ {
		v.XOffsets = map[string]int{"g": x}
		v.Width = 28
		assertDAGFrame(t, v.View(), 28, 50)
	}
	// A corrupt on-disk graph must be diagnosed, not panic or execute anything.
	s.Layout.Root.DAG.Edges = append(s.Layout.Root.DAG.Edges, DAGEdge{From: "absent", To: "a"})
	if !strings.Contains(v.View(), "Invalid DAG") {
		t.Fatal("corrupt persisted graph not diagnosed")
	}
}
func assertDAGFrame(t *testing.T, frame string, w, h int) {
	t.Helper()
	lines := strings.Split(frame, "\n")
	if len(lines) != h {
		t.Fatalf("height %d, want %d", len(lines), h)
	}
	for _, line := range lines {
		if ansi.StringWidth(line) > w {
			t.Fatalf("width overflow %d > %d: %q", ansi.StringWidth(line), w, line)
		}
		if strings.ContainsAny(ansi.Strip(line), "\x1b\a\r\u202e") {
			t.Fatal("unsafe terminal sequence")
		}
	}
}
func TestDAGBoundsAndLargeGraphViewport(t *testing.T) {
	makeLayout := func(g *DAG) []byte {
		raw, err := json.Marshal(Layout{Version: 1, Root: Panel{ID: "g", Kind: "dag", DAG: g}})
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	cases := []*DAG{
		{Version: 1, Nodes: []DAGNode{{ID: strings.Repeat("i", 81)}}},
		{Version: 1, Name: strings.Repeat("n", 161), Nodes: []DAGNode{{ID: "a"}}},
		{Version: 1, Nodes: []DAGNode{{ID: "a", Label: strings.Repeat("l", 161)}}},
		{Version: 1, Nodes: []DAGNode{{ID: "a", Meta: []string{strings.Repeat("m", 2001)}}}},
		{Version: 1, Nodes: []DAGNode{{ID: "a", Bind: &DAGBinding{Actor: strings.Repeat("x", 121)}}}},
		{Version: 1, Nodes: []DAGNode{{ID: "a"}}, Edges: []DAGEdge{{From: "a", To: "a", Style: "return", Label: strings.Repeat("l", 161)}}},
	}
	for _, g := range cases {
		if _, err := ParseLayout(makeLayout(g)); err == nil {
			t.Fatal("unbounded dag field accepted")
		}
	}
	g := &DAG{Version: 1}
	for i := 0; i < 64; i++ {
		g.Nodes = append(g.Nodes, DAGNode{ID: fmt.Sprint(i), Label: fmt.Sprintf("NODE-%02d", i)})
	}
	for i := 0; i < 64; i++ {
		for j := i + 1; j < 64 && len(g.Edges) < 256; j++ {
			g.Edges = append(g.Edges, DAGEdge{From: fmt.Sprint(i), To: fmt.Sprint(j)})
		}
	}
	l, err := ParseLayout(makeLayout(g))
	if err != nil {
		t.Fatal(err)
	}
	s := newState("bounds", "bounds", true)
	s.Layout = l
	assertDAGFrame(t, (Viewer{State: s, Width: 160, Height: 40, XOffsets: map[string]int{"g": 100000}, Offsets: map[string]int{"g": 100000}}).View(), 160, 40)
	assertDAGFrame(t, (Viewer{State: s, Width: 160, Height: 40, Links: map[string]linkFocus{"g": {Node: "0", Scope: 3, Inspect: true}}, XOffsets: map[string]int{"g": 100000}, Offsets: map[string]int{"g": 100000}}).View(), 160, 40)
	g.Edges = append(g.Edges, DAGEdge{From: "63", To: "0", Style: "return"})
	if _, err := ParseLayout(makeLayout(g)); err == nil {
		t.Fatal("257 edges accepted")
	}
	g.Edges = nil
	g.Nodes = append(g.Nodes, DAGNode{ID: "64"})
	if _, err := ParseLayout(makeLayout(g)); err == nil {
		t.Fatal("65 nodes accepted")
	}
}

func FuzzDAGLayoutViewer(f *testing.F) {
	f.Add([]byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a"}]}}}`))
	f.Add([]byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a"},{"id":"b"}],"edges":[{"from":"a","to":"b"},{"from":"b","to":"a","style":"return"}]}}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		l, err := ParseLayout(raw)
		if err != nil {
			return
		}
		s := newState("fuzz", "synthetic", true)
		s.Layout = l
		assertDAGFrame(t, (Viewer{State: s, Width: 80, Height: 24}).View(), 80, 24)
		links := map[string]linkFocus{}
		for _, p := range leaves(l.Root) {
			if p.Kind == "dag" {
				links[p.ID] = linkFocus{Node: p.DAG.Nodes[0].ID, Scope: 3, Inspect: true}
			}
		}
		assertDAGFrame(t, (Viewer{State: s, Width: 100, Height: 30, Links: links}).View(), 100, 30)
	})
}

func TestDAGFanoutLinesRemainConnectedBelowSource(t *testing.T) {
	raw := []byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"root"},{"id":"a"},{"id":"b"}],"edges":[{"from":"root","to":"a"},{"from":"root","to":"b"}]}}}`)
	l, err := ParseLayout(raw)
	if err != nil {
		t.Fatal(err)
	}
	s := newState("lines", "synthetic", true)
	s.Layout = l
	frame := (Viewer{State: s, Width: 100, Height: 30, MotionOff: true}).View()
	// The first branch's source junction has up + down + left strokes. A top
	// corner here would disconnect the branch from its source box above it.
	if !strings.Contains(frame, "┤") {
		t.Fatal("fan-out connector lacks source stroke", frame)
	}
}
