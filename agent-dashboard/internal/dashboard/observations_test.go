package dashboard_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	d "agent-dashboard/internal/dashboard"
	rx "agent-dashboard/internal/rexwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func recordPanel(t *testing.T, kind string, key string) *d.Layout {
	t.Helper()
	binding := `"bind":{"program":"` + key + `"}`
	if kind == "agents" || kind == "tasks" {
		binding = `"bindings":[{"program":"` + key + `"}]`
	}
	return parsePanels(t, `{"version":1,"name":"Synthetic typed observation","root":{"id":"record","kind":"`+kind+`",`+binding+`}}`)
}

func TestOneTypedProgramBindingWorksAcrossRecordPanelsAndDAG(t *testing.T) {
	snapshot := programFixture()
	key := snapshot.Sources[0].Programs[1].Key
	for _, kind := range []string{"session", "agent", "agents", "tasks"} {
		t.Run(kind, func(t *testing.T) {
			l := recordPanel(t, kind, key)
			state := d.ProgramState(snapshot, l)
			frame := (d.Viewer{State: state, Width: 160, Height: 24}).View()
			for _, want := range []string{"Rex program report (not native agent/task telemetry)", "[blocked] claude-code", "progress 0%", "Approval is required"} {
				if !strings.Contains(frame, want) {
					t.Fatalf("missing %q in %s:\n%s", want, kind, frame)
				}
			}
			for _, unwanted := range []string{"[done] pi", "tools 0", "errors 0", "explicitly finished", "✓"} {
				if strings.Contains(frame, unwanted) {
					t.Fatalf("invented/unselected observation %q in %s", unwanted, frame)
				}
			}
			if len(state.Agents) != 0 || len(state.Tasks) != 0 || len(state.Tools) != 0 || len(state.Events) != 0 || state.Finished {
				t.Fatal("projection manufactured Claude lifecycle")
			}
		})
	}
	for _, kind := range []string{"agents", "tasks"} {
		l := parsePanels(t, `{"version":1,"root":{"id":"list","kind":"`+kind+`","program_ids":["`+key+`"],"sources":["synthetic-host"]}}`)
		frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 160, Height: 24}).View()
		if !strings.Contains(frame, "Approval is required") || strings.Contains(frame, "[done] pi") {
			t.Fatal("program list filter ignored", frame)
		}
	}
	graph := parsePanels(t, `{"version":1,"root":{"id":"graph","kind":"dag","dag":{"version":1,"nodes":[{"id":"selected","bind":{"program":"`+key+`"}}]}}}`)
	frame := (d.Viewer{State: d.ProgramState(snapshot, graph), Width: 120, Height: 24}).View()
	if !strings.Contains(frame, "! blocked") || strings.Contains(frame, "✓") {
		t.Fatal("DAG lost literal report state", frame)
	}
	// The relationship inspector resolves exactly the same entity, not a second
	// widget-specific Rex implementation.
	v := d.Viewer{State: d.ProgramState(snapshot, graph), Width: 160, Height: 24}
	for _, key := range []rune{']', 'i'} {
		m, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		v = m.(d.Viewer)
	}
	if frame := v.View(); !strings.Contains(frame, "DETAILS") || !strings.Contains(frame, "Rex program") {
		t.Fatal("inspector did not resolve the typed binding", frame)
	}
}

func TestProgramProjectionRequiresExplicitSelectionAndNeverAliasesNativeIDs(t *testing.T) {
	snapshot := programFixture()
	key := snapshot.Sources[0].Programs[0].Key
	for _, kind := range []string{"agents", "tasks"} {
		l := parsePanels(t, `{"version":1,"root":{"id":"x","kind":"`+kind+`"}}`)
		frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 160, Height: 24}).View()
		if !strings.Contains(frame, "No native "+kind+" supplied by Rex") || strings.Contains(frame, "[done] pi") {
			t.Fatal("implicit native telemetry substitution", frame)
		}
	}
	l := parsePanels(t, `{"version":1,"root":{"id":"x","kind":"agent","actor":"`+key+`"}}`)
	frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 160, Height: 24}).View()
	if !strings.Contains(frame, "Not observed") || strings.Contains(frame, "[done] pi") {
		t.Fatal("actor namespace aliased a program", frame)
	}
	// Filter shorthand remains set-like even if a caller repeats a key.
	repeated := parsePanels(t, `{"version":1,"root":{"id":"x","kind":"tasks","program_ids":["`+key+`","`+key+`"]}}`)
	if frame := (d.Viewer{State: d.ProgramState(snapshot, repeated), Width: 160, Height: 24}).View(); strings.Count(frame, "Rex program report (not native agent/task telemetry)") != 1 {
		t.Fatal("repeated filter duplicated an entity", frame)
	}
	// An unknown host's identical record identity must not borrow the live one.
	missing := rx.ProgramID("other-host", "fixture-session", "fixture-block", "")
	for _, kind := range []string{"session", "agent", "agents", "tasks"} {
		frame := (d.Viewer{State: d.ProgramState(snapshot, recordPanel(t, kind, missing)), Width: 160, Height: 24}).View()
		if !strings.Contains(frame, "Not observed") || strings.Contains(frame, "[done] pi") {
			t.Fatal("missing binding borrowed another identity", frame)
		}
	}
	// Successful removal must also remove projected fields, not retain last-known
	// data as if it were a disconnection.
	l = recordPanel(t, "agent", key)
	snapshot.Sources[0].Programs = nil
	if frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 160, Height: 24}).View(); !strings.Contains(frame, "Not observed") || strings.Contains(frame, "[done]") {
		t.Fatal("deleted report remained in a card", frame)
	}
}

func TestSessionScopeFiltersAndLastKnownStateAreSharedAcrossPanels(t *testing.T) {
	snapshot := programFixture()
	source := snapshot.Sources[0]
	source.Source.Name = "other-source"
	source.Source.HostID = "other-host"
	source.Programs = nil
	snapshot.Sources = append(snapshot.Sources, source)
	l := parsePanels(t, `{"version":1,"root":{"id":"scope","kind":"session","sources":["synthetic-host"]}}`)
	frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 180, Height: 24}).View()
	for _, want := range []string{"Rex observation scope", "fixture-host", "fixture-instance", "Reported Rex session", "fixture-session"} {
		if !strings.Contains(frame, want) {
			t.Fatal("missing scope fact", want, frame)
		}
	}
	if strings.Contains(frame, "other-source") || strings.Contains(frame, "tools 0") {
		t.Fatal("scope widened or unknown became zero", frame)
	}
	l.Root.Sources = []string{"missing"}
	if frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 180, Height: 24}).View(); !strings.Contains(frame, "No matching source scope") || strings.Contains(frame, "fixture-host") {
		t.Fatal("missing source widened scope", frame)
	}
	key := snapshot.Sources[0].Programs[0].Key
	snapshot.Sources[0].Programs[0].Record.State = "working"
	snapshot.Sources[0].Health = "reconnecting"
	for _, kind := range []string{"session", "agent", "agents", "tasks"} {
		frame := (d.Viewer{State: d.ProgramState(snapshot, recordPanel(t, kind, key)), Width: 180, Height: 24}).View()
		if !strings.Contains(frame, "LAST KNOWN") || !strings.Contains(frame, "[working] pi") {
			t.Fatal("disconnection changed reported state", kind, frame)
		}
	}
	graph := parsePanels(t, `{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"p","bind":{"program":"`+key+`"}}]}}}`)
	v := d.Viewer{State: d.ProgramState(snapshot, graph), Width: 120, Height: 24}
	a, b := ansi.Strip(v.Render(time.Unix(100, 0))), ansi.Strip(v.Render(time.Unix(101, 0)))
	if !strings.Contains(a, "LAST KNOWN · working") || a != b {
		t.Fatal("stale DAG node still animates or changes state", a, b)
	}
	snapshot.Sources[0].Health = "online"
	snapshot.Sources[0].Programs[0].Record.State = "done"
	frame = (d.Viewer{State: d.ProgramState(snapshot, graph), Width: 120, Height: 24}).View()
	if !strings.Contains(frame, "done") || strings.Contains(frame, "✓") || strings.Contains(frame, "LAST KNOWN") {
		t.Fatal("program done was presented as native completion", frame)
	}
}

func TestTypedBindingValidationRejectsAmbiguityBeforeReplacingLayout(t *testing.T) {
	key := programFixture().Sources[0].Programs[0].Key
	for _, root := range []string{
		`{"id":"x","kind":"agent","bind":{}}`,
		`{"id":"x","kind":"agent","actor":"lead","bind":{"program":"` + key + `"}}`,
		`{"id":"x","kind":"agent","bind":{"actor":"lead","program":"` + key + `"}}`,
		`{"id":"x","kind":"agent","bind":{"program":"root"}}`,
		`{"id":"x","kind":"agent","bind":{"program":"rex:` + strings.Repeat("A", 64) + `"}}`,
		`{"id":"x","kind":"tasks","bindings":[{"program":"` + key + `"},{"program":"` + key + `"}]}`,
		`{"id":"x","kind":"tasks","task_ids":["1"],"bindings":[{"program":"` + key + `"}]}`,
		`{"id":"x","kind":"events","bind":{"program":"` + key + `"},"select":{"kind":"programs"}}`,
		`{"id":"x","kind":"text","bindings":[{"program":"` + key + `"}]}`,
		`{"kind":"column","bind":{"program":"` + key + `"},"children":[{"id":"x","kind":"session"}]}`,
		`{"id":"x","kind":"dag","dag":{"version":1,"nodes":[{"id":"p","bind":{"task":"1","program":"` + key + `"}}]}}`,
	} {
		if _, err := d.ParseLayout([]byte(`{"version":1,"root":` + root + `}`)); err == nil {
			t.Fatal("accepted ambiguous binding", root)
		}
	}
	// The same resolver accepts the original actor/task namespace on all record
	// shapes, independent of whether a Rex installation exists.
	for _, root := range []string{
		`{"id":"x","kind":"agent","bind":{"actor":"lead"}}`,
		`{"id":"x","kind":"session","bind":{"task":"1"}}`,
		`{"id":"x","kind":"agents","bindings":[{"actor":"lead"},{"task":"1"}]}`,
	} {
		if _, err := d.ParseLayout([]byte(`{"version":1,"root":` + root + `}`)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRexEventsAreFilteredLocalObservationsNotClaudeLifecycle(t *testing.T) {
	snapshot := programFixture()
	key := snapshot.Sources[0].Programs[1].Key
	snapshot.Changes = []rx.Change{
		{Time: time.Unix(100, 0), Source: "synthetic-host", Kind: "baseline", Text: "Earlier activity unknown"},
		{Time: time.Unix(101, 0), Source: "synthetic-host", ProgramKey: key, Kind: "changed", Text: "Selected observation"},
		{Time: time.Unix(102, 0), Source: "other-source", ProgramKey: rx.ProgramID("other-host", "s", "b", ""), Kind: "changed", Text: "Excluded observation"},
	}
	l := parsePanels(t, `{"version":1,"root":{"id":"history","kind":"events","sources":["synthetic-host"],"bindings":[{"program":"`+key+`"}]}}`)
	frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 180, Height: 24}).View()
	if !strings.Contains(frame, "Local Rex observation history · not agent lifecycle") || !strings.Contains(frame, "Rex observation changed · Selected observation") || strings.Contains(frame, "Excluded observation") || strings.Contains(frame, "Earlier activity unknown") {
		t.Fatal("event filter/origin failed", frame)
	}
	l.Root.Bindings = nil
	frame = (d.Viewer{State: d.ProgramState(snapshot, l), Width: 180, Height: 24}).View()
	if !strings.Contains(frame, "Rex observation baseline · Earlier activity unknown") {
		t.Fatal("baseline boundary hidden", frame)
	}
	snapshot.Changes = nil
	if frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 180, Height: 24}).View(); !strings.Contains(frame, "earlier activity is unknown") {
		t.Fatal("empty history asserted no activity", frame)
	}
}

func TestNormalizedRexPanelsKeepUntrustedTextInertAndGeometryBounded(t *testing.T) {
	snapshot := programFixture()
	key := snapshot.Sources[0].Programs[1].Key
	snapshot.Sources[0].Programs[1].Record.Message = "Synthetic\x1b]52;c;unsafe\a\u202e\nreport"
	for _, kind := range []string{"session", "agent", "agents", "tasks"} {
		for _, width := range []int{28, 45, 100, 180} {
			for _, height := range []int{8, 24, 38} {
				frame := (d.Viewer{State: d.ProgramState(snapshot, recordPanel(t, kind, key)), Width: width, Height: height}).View()
				if strings.Contains(frame, "\x1b]52") || strings.ContainsRune(frame, '\u202e') || strings.ContainsRune(frame, '\a') {
					t.Fatal("terminal injection", kind)
				}
				if len(strings.Split(frame, "\n")) > height {
					t.Fatal("height overflow")
				}
				for _, line := range strings.Split(frame, "\n") {
					if ansi.StringWidth(line) > width {
						t.Fatal("width overflow")
					}
				}
			}
		}
	}
	// Round-trip public layout serialization keeps the single binding contract.
	l := recordPanel(t, "tasks", key)
	raw, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ParseLayout(raw); err != nil {
		t.Fatal(err)
	}
}

// Feed only public snapshots through the real Bubble Tea lifecycle. The test
// never manufactures private snapshot messages or inspects the echo internals.
func updateObservationViewer(t *testing.T, viewer d.Viewer, next *d.State) d.Viewer {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	viewer.LoadSnapshot = func() (*d.State, error) { return next, nil }
	consumed := make(chan struct{}, 1)
	program := tea.NewProgram(viewer, tea.WithoutRenderer(), tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithFilter(func(model tea.Model, msg tea.Msg) tea.Msg {
		if v, ok := model.(d.Viewer); ok && v.State != nil && v.State.Revision == next.Revision {
			select {
			case consumed <- struct{}{}:
			default:
			}
		}
		return msg
	}))
	type result struct {
		model tea.Model
		err   error
	}
	done := make(chan result, 1)
	go func() { model, err := program.Run(); done <- result{model, err} }()
	select {
	case <-consumed:
	case <-ctx.Done():
		t.Fatal("snapshot was not consumed", ctx.Err())
	}
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatal(r.err)
		}
		return r.model.(d.Viewer)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		return d.Viewer{}
	}
}

func TestProgramChangeEchoIsSharedByEveryRecordShape(t *testing.T) {
	initial, next := programFixture(), programFixture()
	key := initial.Sources[0].Programs[1].Key
	next.Revision = 2
	next.Sources[0].Programs[1].Record.Message = "New synthetic observation"
	for _, kind := range []string{"session", "agent", "agents", "tasks"} {
		t.Run(kind, func(t *testing.T) {
			layout := recordPanel(t, kind, key)
			viewer := updateObservationViewer(t, d.Viewer{State: d.ProgramState(initial, layout), Width: 180, Height: 24}, d.ProgramState(next, layout))
			frame := ansi.Strip(viewer.View())
			if !strings.Contains(frame, "▸ progress 0%") || !strings.Contains(frame, "Δ 1 observations") {
				t.Fatal("record view lost the shared change echo", frame)
			}
			// Rendering time passing is not another observation and cannot replay an echo.
			if future := ansi.Strip(viewer.Render(time.Now().Add(2 * time.Second))); strings.Contains(future, "▸ progress") || !strings.Contains(future, "· progress") {
				t.Fatal("echo did not expire without losing the observed marker", future)
			}
		})
	}
}

func TestRemovedProgramEchoAndUnknownStateUseTheSameResolver(t *testing.T) {
	initial, next := programFixture(), programFixture()
	key := initial.Sources[0].Programs[1].Key
	next.Revision = 2
	next.Sources[0].Programs = next.Sources[0].Programs[:1]
	for _, kind := range []string{"agent", "dag"} {
		layout := recordPanel(t, "agent", key)
		expected := "▸ Not observed; state/model/progress unknown"
		if kind == "dag" {
			layout = parsePanels(t, `{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"gone","bind":{"program":"`+key+`"}}]}}}`)
			expected = "▸ unobserved · unknown"
		}
		viewer := updateObservationViewer(t, d.Viewer{State: d.ProgramState(initial, layout), Width: 180, Height: 24}, d.ProgramState(next, layout))
		if frame := ansi.Strip(viewer.View()); !strings.Contains(frame, expected) || strings.Contains(frame, "blocked") || strings.Contains(frame, "✓") {
			t.Fatal("removal lost identity or invented completion", frame)
		}
	}
}

func TestRexEventPanelAnchorsHistoryWithoutTreatingSourceDiagnosticsAsEvents(t *testing.T) {
	initial := programFixture()
	key := initial.Sources[0].Programs[0].Key
	for i := 0; i < 5; i++ {
		initial.Changes = append(initial.Changes, rx.Change{Time: time.Unix(int64(100+i), 0), Source: "synthetic-host", ProgramKey: key, Kind: "changed", Text: "Synthetic history"})
	}
	next := programFixture()
	next.Revision = 2
	next.Changes = append(append([]rx.Change(nil), initial.Changes...), rx.Change{Time: time.Unix(200, 0), Source: "synthetic-host", ProgramKey: key, Kind: "changed", Text: "New observation"})
	layout := parsePanels(t, `{"version":1,"root":{"id":"history","kind":"events","program_ids":["`+key+`"]}}`)
	viewer := updateObservationViewer(t, d.Viewer{State: d.ProgramState(initial, layout), Width: 120, Height: 24, Offsets: map[string]int{"history": 2}, Page: 2, Offset: 3}, d.ProgramState(next, layout))
	if viewer.Offsets["history"] != 3 {
		t.Fatal("scrolled local history lost its reading anchor", viewer.Offsets)
	}
	if viewer.Offset != 3 {
		t.Fatal("source diagnostic scroll was treated as an event log", viewer.Offset)
	}
}
