package dashboard

import (
	"fmt"
	"strings"
	"time"
)

// rexObservations is the only place where Rex reports become generic record
// views. A program remains a program even inside a card or work-list panel.
// In particular it never becomes a Claude actor/task or an Issue completion.
func rexObservations(s *State, now time.Time) *observationView {
	v := &observationView{Entities: map[Binding]observationEntity{}, Values: map[echoKey]string{}}
	for _, source := range s.ProgramReports.Sources {
		v.Values[sourceKey(source.Source.Name)] = source.Health + "\n" + source.Error
		for _, program := range source.Programs {
			b := Binding{Program: program.Key}
			keys := []echoKey{programKey(program.Key), sourceKey(source.Source.Name)}
			rows := []reportRow{{text: Clean(source.Source.Name, 80) + " · Rex program report (not native agent/task telemetry)", keys: keys}}
			rows = append(rows, programReportRows(program, source, false, now)...)
			if source.Health != "online" {
				rows = append(rows, reportRow{text: "LAST KNOWN / " + Clean(source.Health, 40) + " · " + Clean(source.Error, 400), keys: []echoKey{sourceKey(source.Source.Name)}})
			}
			e := observationEntity{Source: source.Source.Name, Rows: rows, Status: program.Record.State, Keys: keys, LastKnown: source.Health != "online", Running: program.Record.State == "working", Alert: program.Record.State == "error" || program.Record.State == "blocked"}
			// `done` means a program reported a result, not verified task/Issue success.
			// Keep it literal and do not add a green completion check.
			v.Entities[b] = e
			v.Programs = append(v.Programs, b)
			v.Values[programKey(program.Key)] = program.Record.ObservationValue()
		}
		rows := []reportRow{{text: "Rex observation scope · " + Clean(source.Source.Name, 80)}, {text: "No Claude lifecycle session bound. Model/tools/task lifecycle unknown."}}
		if source.Health != "online" {
			rows = append(rows, reportRow{text: "LAST KNOWN / " + Clean(source.Health, 40), keys: []echoKey{sourceKey(source.Source.Name)}})
		}
		rows = append(rows, sourceRows(Panel{Sources: []string{source.Source.Name}}, s, now)...)
		sessions := map[string]bool{}
		for _, program := range source.Programs {
			if !sessions[program.SessionID] {
				label := program.SessionLabel
				if label == "" {
					label = program.SessionID
				}
				rows = append(rows, reportRow{text: "Reported Rex session · " + Clean(label, 160) + " · " + Clean(program.SessionID, 256)})
				sessions[program.SessionID] = true
			}
		}
		if len(sessions) == 0 {
			rows = append(rows, reportRow{text: "No sessions with program reports observed; lifecycle/progress unknown."})
		}
		v.Summaries = append(v.Summaries, observationEntity{Source: source.Source.Name, Rows: rows})
	}
	// Only explicit typed bindings project reports into actor/task-shaped panels.
	// Native collections stay empty; legacy filters never alias program keys.
	v.EmptyActors = "No native agents supplied by Rex. Select program_ids or bindings for program-report cards."
	v.EmptyTasks = "No native tasks supplied by Rex. Select program_ids or bindings for work-report rows."
	for _, change := range s.ProgramReports.Changes {
		binding := Binding{}
		if change.ProgramKey != "" {
			binding.Program = change.ProgramKey
		}
		who := change.Source
		if change.ProgramKey != "" {
			who += " / " + change.ProgramKey
		}
		text := fmt.Sprintf("Rex observation %s · %s", change.Kind, change.Text)
		v.Events = append(v.Events, observationEvent{Event: Event{Time: change.Time.UnixMilli(), Actor: who, Kind: change.Kind, Text: text}, Binding: binding, Source: change.Source})
	}
	// Empty history is explicitly not an assertion that no activity occurred.
	if len(v.Events) == 0 {
		v.EventNotice = "No local observation changes recorded; earlier activity is unknown."
	}
	if s.ProgramReports.DroppedChanges > 0 {
		v.EventNotice += fmt.Sprintf(" %d older observations discarded.", s.ProgramReports.DroppedChanges)
	}
	v.EventNotice = strings.TrimSpace("Local Rex observation history · not agent lifecycle. " + v.EventNotice)
	return v
}
