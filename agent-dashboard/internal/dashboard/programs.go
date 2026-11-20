package dashboard

import (
	"fmt"
	"strings"
	"time"

	rx "agent-dashboard/internal/rexwatch"
	tea "github.com/charmbracelet/bubbletea"
)

func programInspector(observations *rx.Snapshot) *Layout {
	return &Layout{Version: 1, Name: "Rex observation inspector", Root: Panel{Kind: "column", Children: []Panel{{ID: "source-inspector", Kind: "sources", Size: min(15, 3+4*len(observations.Sources)), Title: "Source inspector"}, {ID: "program-inspector", Kind: "programs", Title: "Program inspector"}}}}
}

// ProgramState retains typed Rex observations without creating Claude telemetry.
func ProgramState(observations rx.Snapshot, layout *Layout) *State {
	if layout == nil {
		layout = programInspector(&observations)
	}
	return &State{Version: 1, Revision: observations.Revision, Started: observations.Started.UnixMilli(), Active: true, Title: "Rex program reports", Layout: layout, ProgramReports: &observations}
}

func ProgramViewer(monitor *rx.Monitor, layout *Layout) Viewer {
	load := func() (*State, error) { return ProgramState(monitor.Snapshot(), layout), nil }
	state, _ := load()
	return Viewer{State: state, LoadSnapshot: load, Width: 80, Height: 24}
}

func RunProgramViewer(monitor *rx.Monitor, layout *Layout) error {
	_, err := tea.NewProgram(ProgramViewer(monitor, layout), tea.WithAltScreen()).Run()
	return err
}

func reportAge(at string, now time.Time) string {
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return "unknown"
	}
	age := now.Sub(t)
	if age < 0 {
		return "future timestamp (clock skew)"
	}
	return age.Round(time.Second).String() + " ago"
}
func contactAge(at time.Time, now time.Time) string {
	if at.IsZero() {
		return "unknown"
	}
	age := now.Sub(at)
	if age < 0 {
		return "clock skew"
	}
	return age.Round(time.Second).String() + " ago"
}
func programKey(id string) echoKey  { return echoKey{"program", id, "report"} }
func sourceKey(name string) echoKey { return echoKey{"source", name, "health"} }

type reportRow struct {
	text string
	keys []echoKey
}

func programRows(p Panel, s *State, now time.Time) []reportRow {
	if s.ProgramReports == nil {
		return []reportRow{{text: "No Rex source bound; program state/progress unknown."}}
	}
	out := []reportRow{}
	for _, source := range s.ProgramReports.Sources {
		if !contains(p.Sources, source.Source.Name) {
			continue
		}
		freshness := "live transport"
		if source.Health != "online" {
			freshness = "LAST KNOWN / " + Clean(source.Health, 40)
		}
		if !p.Compact {
			out = append(out, reportRow{text: Clean(source.Source.Name, 80) + " · " + freshness, keys: []echoKey{sourceKey(source.Source.Name)}})
		}
		count := 0
		for _, program := range source.Programs {
			if !contains(p.ProgramIDs, program.Key) {
				continue
			}
			count++
			out = append(out, programReportRows(program, source, p.Compact, now)...)
		}
		if count == 0 {
			out = append(out, reportRow{text: Clean(source.Source.Name, 80) + " · " + freshness + " · no matching program reports; state/progress unknown."})
		}
	}
	if len(out) == 0 {
		return []reportRow{{text: "No matching Rex source; bindings remain unknown."}}
	}
	if p.Compact {
		out = append([]reportRow{{text: "? = progress not reported"}}, out...)
	}
	return out
}

func sourceRows(p Panel, s *State, now time.Time) []reportRow {
	if s.ProgramReports == nil {
		return []reportRow{{text: "No Rex source bound."}}
	}
	out := []reportRow{}
	for _, source := range s.ProgramReports.Sources {
		if !contains(p.Sources, source.Source.Name) {
			continue
		}
		keys := []echoKey{sourceKey(source.Source.Name)}
		out = append(out,
			reportRow{text: Clean(source.Source.Name, 80) + " · " + Clean(source.Health, 40), keys: keys},
			reportRow{text: "host " + Clean(source.Source.HostID, 128) + " · instance " + unknown(source.InstanceID)},
			reportRow{text: fmt.Sprintf("%d reports · %d panes without reports · %d foreign/unverified panes excluded", len(source.Programs), source.Unreported, source.ExcludedPanes)},
			reportRow{text: "contact " + contactAge(source.LastContact, now) + " · full snapshot " + contactAge(source.LastSnapshot, now)})
		if source.Error != "" {
			out = append(out, reportRow{text: "Source error (not program failure): " + Clean(source.Error, 400), keys: keys})
		}
	}
	if len(out) == 0 {
		return []reportRow{{text: "No matching Rex source; source health unknown."}}
	}
	return out
}
func rowText(rows []reportRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.text
	}
	return out
}
func rowKeys(rows []reportRow) [][]echoKey {
	out := make([][]echoKey, len(rows))
	for i, r := range rows {
		out[i] = r.keys
	}
	return out
}

func programReportRows(program rx.Program, source rx.SourceState, compact bool, now time.Time) []reportRow {
	out := []reportRow{}
	freshness := "live transport"
	if source.Health != "online" {
		freshness = "LAST KNOWN / " + Clean(source.Health, 40)
	}
	r := program.Record
	app := r.EffectiveApp
	if app == "" {
		app = r.App
	}
	location := program.SessionLabel
	if location == "" {
		location = program.SessionID
	}
	shortBlock := program.BlockID
	if len(shortBlock) > 8 {
		shortBlock = shortBlock[len(shortBlock)-8:]
	}
	block := program.BlockLabel
	if block == "" {
		block = program.BlockID
	} else {
		block += "#" + shortBlock
	}
	id := r.ID
	if id == "" {
		id = "root"
	}
	progress := "unknown"
	if r.Progress != nil {
		progress = fmt.Sprintf("%d%%", *r.Progress)
	}
	details := []string{"progress " + progress, "report " + reportAge(r.UpdatedAt, now)}
	if r.Kind != "" {
		details = append(details, r.Kind)
	}
	if r.Title != "" {
		details = append(details, r.Title)
	}
	if r.Message != "" {
		details = append(details, r.Message)
	}
	keys := []echoKey{programKey(program.Key)}
	if compact {
		if r.Progress == nil {
			progress = "?"
		}
		prefix := source.Source.Name + " "
		if source.Health != "online" {
			prefix += "[" + freshness + "] "
		}
		out = append(out, reportRow{text: Clean(prefix+"["+unknown(r.State)+"] "+unknown(app)+" "+progress+" · "+location+" / "+block+" ("+id+")", 800), keys: append(keys, sourceKey(source.Source.Name))})
	} else {
		out = append(out, reportRow{text: Clean("["+unknown(r.State)+"] "+unknown(app)+" · "+location+" / "+block+" ("+id+")", 800), keys: keys}, reportRow{text: Clean(strings.Join(details, " · "), 1000), keys: keys})
	}
	return out
}
