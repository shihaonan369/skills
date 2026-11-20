package dashboard_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	d "agent-dashboard/internal/dashboard"
	rx "agent-dashboard/internal/rexwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func programFixture() rx.Snapshot {
	zero := 0
	return rx.Snapshot{Revision: 1, Started: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Sources: []rx.SourceState{{Source: rx.Source{Name: "synthetic-host", HostID: "fixture-host", Endpoint: "unix:///synthetic"}, InstanceID: "fixture-instance", Health: "online", Panes: 3, Unreported: 1, Programs: []rx.Program{
		{Key: rx.ProgramID("fixture-host", "fixture-session", "fixture-block", ""), Source: "synthetic-host", HostID: "fixture-host", SessionID: "fixture-session", SessionLabel: "Synthetic isolation", BlockID: "fixture-block", BlockLabel: "Same label", Record: rx.Record{State: "done", App: "pi", UpdatedAt: "2020-01-01T00:00:00Z"}},
		{Key: rx.ProgramID("fixture-host", "fixture-session", "other-block", "worker"), Source: "synthetic-host", HostID: "fixture-host", SessionID: "fixture-session", SessionLabel: "Synthetic isolation", BlockID: "other-block", BlockLabel: "Same label", Record: rx.Record{ID: "worker", State: "blocked", Kind: "permission", EffectiveApp: "claude-code", Progress: &zero, Message: "Approval is required"}},
	}}}}
}
func parsePanels(t *testing.T, raw string) *d.Layout {
	t.Helper()
	l, err := d.ParseLayout([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestProgramPanelsNeverInventClaudeLifecycleOrIssueCompletion(t *testing.T) {
	l := parsePanels(t, `{"version":1,"name":"Synthetic observer test","root":{"kind":"column","children":[{"id":"source","kind":"sources","size":8},{"id":"lifecycle","kind":"session","size":7},{"id":"reports","kind":"programs"}]}}`)
	state := d.ProgramState(programFixture(), l)
	if len(state.Agents) != 0 || len(state.Tasks) != 0 || len(state.Tools) != 0 || len(state.Events) != 0 || state.Finished || state.Session != "" || state.HookCount != 0 {
		t.Fatal("program status fabricated lifecycle telemetry")
	}
	frame := (d.Viewer{State: state, Width: 120, Height: 38}).View()
	for _, want := range []string{"REX PROGRAM REPORTS", "NOT FULL AGENT TELEMETRY", "No Claude lifecycle session bound", "[done] pi", "[blocked] claude-code", "progress unknown", "progress 0%", "Approval is required", "Same label#re-block", "Same label#er-block"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q in frame:\n%s", want, frame)
		}
	}
	for _, unwanted := range []string{"explicitly finished", "hooks ·", "no hooks observed"} {
		if strings.Contains(frame, unwanted) {
			t.Fatal("program observer implied Claude state", unwanted)
		}
	}
	if state.Layout != l {
		t.Fatal("caller composition was replaced")
	}
}

func TestProgramBindingsAreSourceQualifiedAndUnknownBindingsStayUnknown(t *testing.T) {
	snapshot := programFixture()
	key := snapshot.Sources[0].Programs[1].Key
	l := parsePanels(t, `{"version":1,"root":{"id":"reports","kind":"programs","sources":["synthetic-host"],"program_ids":["`+key+`"],"compact":true}}`)
	frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 120, Height: 24}).View()
	if !strings.Contains(frame, "[blocked] claude-code 0%") || strings.Contains(frame, "[done] pi") {
		t.Fatal("exact source-qualified binding was ignored", frame)
	}
	l.Root.ProgramIDs = []string{rx.ProgramID("different-host", "fixture-session", "other-block", "worker")}
	frame = (d.Viewer{State: d.ProgramState(snapshot, l), Width: 120, Height: 24}).View()
	if !strings.Contains(frame, "state/progress unknown") || strings.Contains(frame, "[blocked]") {
		t.Fatal("an unknown key borrowed another host's report")
	}
	l.Root.Sources = []string{"not-selected"}
	frame = (d.Viewer{State: d.ProgramState(snapshot, l), Width: 120, Height: 24}).View()
	if !strings.Contains(frame, "No matching Rex source") {
		t.Fatal("unknown source alias inferred another host")
	}
	for _, raw := range []string{
		`{"version":1,"root":{"id":"x","kind":"text","compact":true}}`,
		`{"version":1,"root":{"id":"x","kind":"agent","actor":"lead","sources":["host"]}}`,
		`{"version":1,"root":{"id":"x","kind":"sources","program_ids":["` + key + `"]}}`,
		`{"version":1,"root":{"id":"x","kind":"programs","program_ids":["worker"]}}`,
		`{"version":1,"root":{"kind":"row","sources":["host"],"children":[{"id":"x","kind":"programs"}]}}`,
	} {
		if _, err := d.ParseLayout([]byte(raw)); err == nil {
			t.Fatal("invalid program/source binding accepted", raw)
		}
	}
}

func TestSourceFailureRetainsReportedStateAndNeverInfersActivityFromAge(t *testing.T) {
	snapshot := programFixture()
	snapshot.Sources[0].Health = "reconnecting"
	snapshot.Sources[0].Error = "synthetic connection lost"
	l := parsePanels(t, `{"version":1,"root":{"kind":"column","children":[{"id":"source","kind":"sources","size":10},{"id":"reports","kind":"programs","compact":true}]}}`)
	frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: 120, Height: 30}).View()
	for _, want := range []string{"LAST KNOWN", "reconnecting", "Source error (not program failure)", "[done] pi ?", "[blocked] claude-code 0%"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "[error] pi") || strings.Contains(frame, "[idle] pi") {
		t.Fatal("age or source health changed a program outcome")
	}
}

func TestProgramTextIsInertAndFramesStayWithinCallerGeometry(t *testing.T) {
	snapshot := programFixture()
	snapshot.Sources[0].Programs[0].Record.App = "pi\x1b]52;c;unsafe\a\u202e"
	snapshot.Sources[0].Programs[0].Record.Message = "line one\nline two\x1b[2J"
	l := parsePanels(t, `{"version":1,"root":{"id":"reports","kind":"programs"}}`)
	for _, width := range []int{28, 45, 80, 120} {
		for _, height := range []int{8, 24, 38} {
			frame := (d.Viewer{State: d.ProgramState(snapshot, l), Width: width, Height: height}).View()
			if strings.Contains(frame, "\x1b]52") || strings.Contains(frame, "\x1b[2J") || strings.ContainsRune(frame, '\u202e') || strings.ContainsRune(frame, '\a') {
				t.Fatal("untrusted program text injected terminal controls")
			}
			lines := strings.Split(frame, "\n")
			if len(lines) > height {
				t.Fatal("frame exceeded caller height")
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > width {
					t.Fatalf("frame exceeded width %d: %q", width, line)
				}
			}
		}
	}
	state := d.ProgramState(rx.Snapshot{}, nil)
	if state.Layout == nil || !strings.Contains(state.Layout.Name, "inspector") {
		t.Fatal("missing layout did not remain an explicit diagnostic inspector")
	}
}

// Drive the real public Bubble Tea model/runtime headlessly. No private snapshot
// messages or echo fields are accessed. Updating the source buffer must not use
// the Claude event-buffer anchoring rule or turn report ages into change echoes.
func TestProgramViewerSourceScrollSurvivesLiveReadsWithoutClockEchoes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	initial := programFixture()
	next := programFixture()
	for _, name := range []string{"two", "three", "four", "five"} {
		source := rx.SourceState{Source: rx.Source{Name: "synthetic-" + name, HostID: "fixture-" + name, Endpoint: "unix:///" + name}, InstanceID: "fixture-" + name, Health: "online"}
		initial.Sources = append(initial.Sources, source)
		next.Sources = append(next.Sources, source)
	}
	next.Revision = 2
	next.Sources[0].LastContact = time.Now()
	next.Sources[0].LastSnapshot = time.Now()
	next.Sources[0].Programs[0].Record.UpdatedAt = "2025-01-01T00:00:00Z"
	loads := make(chan *d.State, 2)
	loads <- d.ProgramState(initial, nil)
	secondRead := make(chan struct{}, 1)
	v := d.Viewer{State: d.ProgramState(initial, nil), Width: 100, Height: 24, Page: 2, Offset: 3, LoadSnapshot: func() (*d.State, error) {
		select {
		case state := <-loads:
			return state, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
	program := tea.NewProgram(v, tea.WithoutRenderer(), tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithFilter(func(model tea.Model, msg tea.Msg) tea.Msg {
		if viewer, ok := model.(d.Viewer); ok && viewer.State != nil && viewer.State.Revision == 2 {
			select {
			case secondRead <- struct{}{}:
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
	loads <- d.ProgramState(next, nil)
	select {
	case <-secondRead:
	case <-ctx.Done():
		t.Fatal("viewer did not consume a live public snapshot", ctx.Err())
	}
	program.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	select {
	case result := <-done:
		if result.err != nil {
			t.Fatal(result.err)
		}
		final := result.model.(d.Viewer)
		if final.Offset != 3 {
			t.Fatal("source diagnostic scroll reset as though it were a Claude event buffer")
		}
		if strings.Contains(final.View(), "Δ ") {
			t.Fatal("wall-clock/source-contact/report-age updates created a change echo")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
