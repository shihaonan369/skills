package dashboard

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const linkFixture = `{"version":1,"root":{"kind":"column","children":[{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"lead","bind":{"actor":"lead"}},{"id":"a","bind":{"actor":"absent"}},{"id":"b"},{"id":"end","bind":{"task":"t"}},{"id":"note"}],"edges":[{"from":"lead","to":"a"},{"from":"lead","to":"b"},{"from":"a","to":"end"},{"from":"b","to":"end"}]}},{"id":"log","kind":"events","size":5}]}}`

func viewerKey(v Viewer, key string) Viewer {
	m, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return m.(Viewer)
}
func receive(v Viewer, s *State, at time.Time) Viewer {
	m, _ := v.Update(snapshotMsg{state: s, at: at, epoch: v.readEpoch})
	return m.(Viewer)
}
func linkViewer(t *testing.T) (Viewer, string) {
	t.Helper()
	root := setup(t)
	if err := SetLayout(root, "s", []byte(linkFixture)); err != nil {
		t.Fatal(err)
	}
	return Viewer{State: snap(t, root), Width: 120, Height: 42}, root
}
func TestLinkFocusNeighborDirectionsAndAcyclicReachability(t *testing.T) {
	v, _ := linkViewer(t)
	graph := v.State.Layout.Root.Children[0].DAG
	focus := linkFocus{Node: "a"}
	relations := declaredRelations(graph, focus)
	if !reflect.DeepEqual(relations.incoming, map[int]bool{0: true}) || !reflect.DeepEqual(relations.outgoing, map[int]bool{2: true}) {
		t.Fatal("direct neighbor directions lost", relations)
	}
	if relations.edgeMark(0) != "<" || relations.edgeMark(2) != ">" || relations.nodeMark("a", "a") != "*" || relations.nodeMark("note", "a") != "." {
		t.Fatal("direction markers missing", relations)
	}
	focus = linkFocus{Node: "end", Scope: 3}
	relations = declaredRelations(graph, focus)
	if len(relations.incoming) != 4 || len(relations.outgoing) != 0 || !relations.up["lead"] || relations.up["note"] {
		t.Fatal("transitive ancestors incorrect", relations)
	}
	focus = linkFocus{Node: "lead", Scope: 2}
	relations = declaredRelations(graph, focus)
	if len(relations.outgoing) != 4 || len(relations.incoming) != 0 || !relations.down["end"] {
		t.Fatal("downstream scope missed descendants", relations)
	}
	focus.Scope = 1
	if relations = declaredRelations(graph, focus); len(relations.outgoing) != 0 {
		t.Fatal("upstream scope traversed downstream")
	}
}
func TestLinkFocusKeysKeepViewportAndTelemetryIndependent(t *testing.T) {
	v, root := linkViewer(t)
	before, _ := json.Marshal(v.State)
	v.Offsets = map[string]int{"g": 3}
	v.XOffsets = map[string]int{"g": 4}
	for _, key := range []string{"]", "]", "f", "i"} {
		v = viewerKey(v, key)
	}
	f := v.Links["g"]
	if f.Node != "a" || f.Scope != 1 || !f.Inspect || v.Offsets["g"] != 3 || v.XOffsets["g"] != 4 || v.Selected != 0 {
		t.Fatal("focus keys changed viewport/focus or wrong local selection", v)
	}
	frame := ansi.Strip(v.Render(time.Unix(100, 0)))
	for _, want := range []string{"DETAILS", "unobserved · unknown", "a:upstream"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	v = viewerKey(v, "c")
	if v.Links["g"].Node != "" || !v.Links["g"].Inspect {
		t.Fatal("clear should only clear selection")
	}
	v = viewerKey(v, "[")
	if v.Links["g"].Node != "note" {
		t.Fatal("previous from no selection must select last declared node")
	}
	after, _ := json.Marshal(snap(t, root))
	if string(before) != string(after) {
		t.Fatal("link UI wrote telemetry")
	}
	v.Page = 2
	v = viewerKey(v, "]")
	if v.Links["g"].Node != "note" {
		t.Fatal("diagnostic page changed hidden DAG selection")
	}
	v.Page, v.Help = 0, true
	v = viewerKey(v, "]")
	if v.Links["g"].Node != "note" {
		t.Fatal("help page changed hidden DAG selection")
	}
}
func TestLinkFocusSidebarGeometryAndLayoutReplacement(t *testing.T) {
	v, root := linkViewer(t)
	v = viewerKey(v, "]")
	v = viewerKey(v, "i")
	for _, width := range []int{28, 45, 81, 82, 100, 160} {
		for _, height := range []int{8, 14, 30, 60} {
			v.Width, v.Height = width, height
			frame := ansi.Strip(v.Render(time.Unix(100, 0)))
			assertDAGFrame(t, frame, width, height)
			if width == 81 && height >= 14 && !strings.Contains(frame, "needs 82 cols") {
				t.Fatal("narrow inspector silently changed caller layout", frame)
			}
		}
	}
	v.Width, v.Height = 120, 42
	for i := 0; i < 100; i++ {
		m, _ := v.Update(tea.KeyMsg{Type: tea.KeyRight})
		v = m.(Viewer)
	}
	xmax, _ := v.dagScrollLimits("g")
	if v.XOffsets["g"] != xmax {
		t.Fatal("inspector pan bounds disagree with viewport", v.XOffsets["g"], xmax)
	}
	// Preserve the focused leaf's identity when the caller reorders its tree.
	v.Selected = 1
	if err := SetLayout(root, "s", []byte(`{"version":1,"root":{"kind":"column","children":[{"id":"log","kind":"events"},{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"replacement"}]}}]}}`)); err != nil {
		t.Fatal(err)
	}
	v = receive(v, snap(t, root), time.Unix(100, 0))
	if v.focus() != "log" || v.Links["g"].Node != "" || !v.Links["g"].Inspect {
		t.Fatal("hot swap stole focus or retained removed node", v.Links, v.focus())
	}
	if err := SetLayout(root, "s", []byte(`{"version":1,"root":{"id":"g","kind":"text","text":["not a graph"]}}`)); err != nil {
		t.Fatal(err)
	}
	v = receive(v, snap(t, root), time.Unix(101, 0))
	if len(v.Links) != 0 {
		t.Fatal("removed DAG kept stale local state")
	}
}
func TestLinkFocusLegendKeepsForwardEdgesAndIsolatedNodeHonest(t *testing.T) {
	v, _ := linkViewer(t)
	g := v.State.Layout.Root.Children[0].DAG
	before, _ := json.Marshal(g)
	v.Links = map[string]linkFocus{"g": {Node: "note", Inspect: true}}
	v.Height = 80
	frame := ansi.Strip(v.Render(time.Unix(100, 0)))
	for _, want := range []string{"* note", "declared only", "No matching declared edges.", "[.] lead → a"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing isolated node or forward declaration %q\n%s", want, frame)
		}
	}
	after, _ := json.Marshal(g)
	if string(before) != string(after) || len(g.Nodes) != 5 || v.State.Finished || len(v.State.Agents) != 1 {
		t.Fatal("focus reordered/invented graph or lifecycle")
	}
}
