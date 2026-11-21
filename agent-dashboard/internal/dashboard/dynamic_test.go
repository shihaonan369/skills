package dashboard_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	d "agent-dashboard/internal/dashboard"
	rx "agent-dashboard/internal/rexwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func dynamicNative(t *testing.T, raw string) (*d.State, string) {
	t.Helper()
	root := t.TempDir()
	if err := d.Start(root, "dynamic-session", "Synthetic dynamic contract", false); err != nil {
		t.Fatal(err)
	}
	if err := d.SetLayout(root, "dynamic-session", []byte(raw)); err != nil {
		t.Fatal(err)
	}
	return nativeSnapshot(t, root), root
}
func nativeSnapshot(t *testing.T, root string) *d.State {
	t.Helper()
	s, err := d.Snapshot(root, "dynamic-session")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func spawnDynamic(t *testing.T, root, id, role string) *d.State {
	t.Helper()
	if err := d.Ingest(root, d.Payload{Session: "dynamic-session", Kind: "SubagentStart", AgentID: id, AgentType: role}); err != nil {
		t.Fatal(err)
	}
	return nativeSnapshot(t, root)
}
func dynamicFrame(s *d.State) string {
	return ansi.Strip((d.Viewer{State: s, Width: 200, Height: 80}).View())
}

func TestDynamicNativeMembershipAcrossGraphCardsListsAndAnnotations(t *testing.T) {
	q := `{"kind":"actors","roles":["implement"]}`
	raw := `{"version":1,"root":{"kind":"column","children":[
 {"id":"graph","kind":"dag","size":18,"dag":{"version":1,"select":` + q + `,"nodes":[],"edges":[]}},
 {"id":"cards","kind":"agent","size":18,"bind":{"item":true},"repeat":{"select":` + q + `,"direction":"row"}},
 {"id":"rows","kind":"agents","size":15,"select":` + q + `},
 {"id":"notes","kind":"text","text":["Caller note"],"repeat":{"select":` + q + `,"direction":"row"}}
 ]}}`
	s, root := dynamicNative(t, raw)
	before, _ := json.Marshal(s.Layout)
	if f := dynamicFrame(s); !strings.Contains(f, "No matching observations") {
		t.Fatal(f)
	}
	s = spawnDynamic(t, root, "excluded-review", "review")
	s = spawnDynamic(t, root, "included-worker", "implement")
	f := dynamicFrame(s)
	if strings.Contains(f, "excluded-review") || strings.Count(f, "included-worker") < 4 || !strings.Contains(f, "Caller note") {
		t.Fatal("shared dynamic membership not applied to every projection", f)
	}
	if strings.Contains(f, "caller-declared grouping") {
		t.Fatal("invented parent edge", f)
	}
	after, _ := json.Marshal(s.Layout)
	if string(before) != string(after) {
		t.Fatal("rendering rewrote caller declaration")
	}
	s = spawnDynamic(t, root, "second-worker", "implement")
	f = dynamicFrame(s)
	if strings.Count(f, "second-worker") < 4 || strings.Count(f, "included-worker") < 4 {
		t.Fatal("new worker did not join every projection", f)
	}
	if len(s.Agents) != 4 || len(s.Tasks) != 0 || s.Finished {
		t.Fatal("presentation manufactured work")
	}
}

func TestRepeatWorksForEveryRecordShapeAndSplitTree(t *testing.T) {
	for _, kind := range []string{"session", "agent", "agents", "tasks", "events", "text", "dag", "programs", "sources", "row", "column"} {
		t.Run(kind, func(t *testing.T) {
			selected := "programs"
			if kind == "sources" {
				selected = "sources"
			}
			fields := `,"bind":{"item":true}`
			if kind == "text" {
				fields = `,"text":["Repeated caller note"]`
			}
			if kind == "dag" {
				fields = `,"dag":{"version":1,"nodes":[{"id":"item","label":"Repeated record","bind":{"item":true}}]}`
			}
			if kind == "row" || kind == "column" {
				fields = `,"children":[{"id":"nested-card","kind":"agent","bind":{"item":true}},{"id":"nested-note","kind":"text","text":["Nested caller note"]}]`
			}
			layout := parsePanels(t, `{"version":1,"root":{"id":"items","kind":"`+kind+`","repeat":{"select":{"kind":"`+selected+`"}},`+strings.TrimPrefix(fields, ",")+`}}`)
			snapshot := programFixture()
			state := d.ProgramState(snapshot, layout)
			original, _ := json.Marshal(layout)
			frame := dynamicFrame(state)
			if strings.Contains(frame, "Not observed") || strings.Contains(frame, "Invalid DAG") || strings.Contains(frame, "requires an enclosing") {
				t.Fatal(frame)
			}
			if kind == "sources" {
				if !strings.Contains(frame, "synthetic-host") {
					t.Fatal(frame)
				}
			} else if kind == "text" {
				if strings.Count(frame, "Repeated caller note") != 2 {
					t.Fatal(frame)
				}
			} else if kind == "events" {
				if !strings.Contains(frame, "Local Rex observation history") {
					t.Fatal(frame)
				}
			} else if kind == "dag" {
				if strings.Count(frame, "Repeated record") != 2 {
					t.Fatal(frame)
				}
			} else if !strings.Contains(frame, "Rex program report") {
				t.Fatal(frame)
			}
			snapshot.Sources[0].Programs = nil
			frame = dynamicFrame(d.ProgramState(snapshot, layout))
			if selected != "sources" && !strings.Contains(frame, "No matching observations") {
				t.Fatal("removed record remained live", frame)
			}
			after, _ := json.Marshal(layout)
			if string(original) != string(after) {
				t.Fatal("template mutated")
			}
		})
	}
}

func TestDynamicSelectionsRespectSourceStateAndTypedIdentity(t *testing.T) {
	for _, kind := range []string{"agents", "tasks", "events", "programs", "sources"} {
		t.Run(kind, func(t *testing.T) {
			q := `{"kind":"programs","sources":["synthetic-host"],"states":["blocked"]}`
			if kind == "sources" {
				q = `{"kind":"sources","sources":["absent"]}`
			}
			layout := parsePanels(t, `{"version":1,"root":{"id":"selected","kind":"`+kind+`","select":`+q+`}}`)
			s := programFixture()
			frame := dynamicFrame(d.ProgramState(s, layout))
			if strings.Contains(frame, "[done] pi") {
				t.Fatal("selection widened", frame)
			}
			if kind != "events" && kind != "sources" && !strings.Contains(frame, "Approval is required") {
				t.Fatal(frame)
			}
			if kind == "sources" && strings.Contains(frame, "fixture-host") {
				t.Fatal("empty selection fell back to all sources", frame)
			}
		})
	}
}

func TestDynamicGraphGroupingIsExplicitForwardAndBounded(t *testing.T) {
	layout := parsePanels(t, `{"version":1,"root":{"id":"graph","kind":"dag","dag":{"version":1,"select":{"kind":"actors","limit":1},"attach_from":"group","nodes":[{"id":"group","label":"Caller group"}]}}}`)
	s := &d.State{Layout: layout, Active: true, Agents: map[string]*d.Agent{"a": {ID: "a", Role: "implement", Status: "running"}, "b": {ID: "b", Role: "review", Status: "running"}}}
	f := dynamicFrame(s)
	if !strings.Contains(f, "caller-declared grouping") || !strings.Contains(f, "1 additional observations") || strings.Contains(f, "review · b") {
		t.Fatal(f)
	}
	// Dynamic overflow is an explicit diagnostic, never unbounded expansion.
	layout = parsePanels(t, `{"version":1,"root":{"id":"outer","kind":"row","repeat":{"select":{"kind":"actors","limit":256}},"children":[{"id":"inner","kind":"text","text":["bounded"],"repeat":{"select":{"kind":"actors","limit":256}}}]}}`)
	s.Layout = layout
	for i := 0; i < 30; i++ {
		id := fmt.Sprintf("worker-%02d", i)
		s.Agents[id] = &d.Agent{ID: id, Role: "implement"}
	}
	if f = dynamicFrame(s); !strings.Contains(f, "exceeds 256 expanded panels") {
		t.Fatal(f)
	}
}

func TestDynamicLayoutValidationRejectsAmbiguousOrUnscopedTemplates(t *testing.T) {
	bad := []string{
		`{"id":"a","kind":"agent","bind":{"item":true}}`,
		`{"id":"a","kind":"agents","select":{"kind":"guess"}}`,
		`{"id":"a","kind":"agents","select":{"kind":"actors","roles":["x","x"]}}`,
		`{"id":"a","kind":"agents","select":{"kind":"actors","scope_file":"relative.json"}}`,
		`{"id":"a","kind":"agents","actors":["x"],"select":{"kind":"actors"}}`,
		`{"id":"a","kind":"agent","bind":{"item":true},"repeat":{"select":{"kind":"actors"},"direction":"grid"}}`,
		`{"id":"@dynamic/reserved","kind":"text"}`,
		`{"id":"a","kind":"dag","dag":{"version":1,"select":{"kind":"actors"},"attach_from":"missing","nodes":[]}}`,
		`{"id":"a","kind":"dag","dag":{"version":1,"select":{"kind":"actors"},"nodes":[],"edges":[{"from":"a","to":"b","style":"return"}]}}`,
	}
	for _, panel := range bad {
		if _, err := d.ParseLayout([]byte(`{"version":1,"root":` + panel + `}`)); err == nil {
			t.Fatal("accepted", panel)
		}
	}
}

func TestDynamicViewerFocusSurvivesMembershipChangesAndCleansRemovedState(t *testing.T) {
	layout := parsePanels(t, `{"version":1,"root":{"id":"cards","kind":"agent","bind":{"item":true},"repeat":{"select":{"kind":"programs"}}}}`)
	snapshot := programFixture()
	v := d.Viewer{State: d.ProgramState(snapshot, layout), Width: 180, Height: 50, Offsets: map[string]int{}, XOffsets: map[string]int{}}
	m, _ := v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = m.(d.Viewer)
	before := v.View()
	var heading string
	for _, line := range strings.Split(ansi.Strip(before), "\n") {
		if strings.Contains(line, "▶") {
			heading = line
		}
	}
	if heading == "" {
		t.Fatal(before)
	}
	next := snapshot
	next.Revision++
	next.Sources = append([]rx.SourceState(nil), snapshot.Sources...)
	p := snapshot.Sources[0].Programs[0]
	p.Key = rx.ProgramID("new-host", "new-session", "new-block", "new-record")
	p.Record.Title = "Additional synthetic record"
	next.Sources[0].Programs = append(append([]rx.Program(nil), snapshot.Sources[0].Programs...), p)
	v = updateObservationViewer(t, v, d.ProgramState(next, layout))
	if after := ansi.Strip(v.View()); !strings.Contains(after, strings.TrimSpace(heading)) {
		t.Fatal("focus moved to another identity", after)
	}
	v.Offsets["removed"] = 4
	v.XOffsets["removed"] = 2
	next.Revision++
	next.Sources[0].Programs = nil
	v = updateObservationViewer(t, v, d.ProgramState(next, layout))
	if len(v.Offsets) > 0 || len(v.XOffsets) > 0 {
		t.Fatal("removed identities leaked navigation state")
	}
}
