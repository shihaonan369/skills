package rexwatch_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	rx "agent-dashboard/internal/rexwatch"
)

func TestObservationHistoryDistinguishesBaselinesChangesRemovalAndReconnectGaps(t *testing.T) {
	m, f := monitor(t, config("host-a"))
	c := nextConnection(t, f)
	mustSend(t, c, hello("host-a"))
	mustSend(t, c, baseline("host-a"))
	snapshot := m.Snapshot()
	if len(snapshot.Changes) != 1 || snapshot.Changes[0].Kind != "baseline" || !strings.Contains(snapshot.Changes[0].Text, "unknown") || snapshot.Changes[0].ProgramKey != "" {
		t.Fatal("initial snapshot fabricated past activity", snapshot.Changes)
	}
	// Ages and contact clocks are not observation changes.
	ageOnly := strings.Replace(workingPane, "2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z", 1)
	mustSend(t, c, envelope("host-a", "instance-old", 3, "pane", `"pane":`+ageOnly))
	mustSend(t, c, envelope("host-a", "instance-old", 4, "heartbeat", ""))
	if len(m.Snapshot().Changes) != 1 {
		t.Fatal("clock-only changes became events")
	}
	mustSend(t, c, envelope("host-a", "instance-old", 5, "pane", `"pane":`+donePane))
	changes := m.Snapshot().Changes
	if len(changes) != 3 {
		t.Fatal("whole-record change and child removal not recorded", changes)
	}
	found := map[string]rx.Change{}
	for _, e := range changes[1:] {
		found[e.Kind] = e
		if e.Time.IsZero() || e.Source != "host-a" {
			t.Fatal("missing local time/provenance", e)
		}
	}
	if found["changed"].ProgramKey != rx.ProgramID("host-a", "session-one", "block-one", "") || !strings.Contains(found["changed"].Text, "[done]") || found["removed"].ProgramKey != rx.ProgramID("host-a", "session-one", "block-one", "build/test") || !strings.Contains(found["removed"].Text, "unknown") {
		t.Fatal("wrong event identity/outcome", changes)
	}
	// Returned snapshots are detached, including history strings and entries.
	detached := m.Snapshot()
	detached.Changes[0].Text = "mutated"
	if m.Snapshot().Changes[0].Text == "mutated" {
		t.Fatal("history aliases monitor memory")
	}
	c.input <- input{end: errors.New("synthetic stream interrupted")}
	await(t, func() bool { return m.Snapshot().Sources[0].Health == "reconnecting" })
	reconnect := nextConnection(t, f)
	mustSend(t, reconnect, hello("host-a"))
	// The old root was done. Reconnection contains working again, but the gap
	// cannot be reconstructed as a task restart or fabricated report change.
	before := len(m.Snapshot().Changes)
	mustSend(t, reconnect, baseline("host-a"))
	changes = m.Snapshot().Changes
	if len(changes) != before+1 || changes[len(changes)-1].Kind != "baseline" || !strings.Contains(changes[len(changes)-1].Text, "gap unknown") {
		t.Fatal("reconnect replayed a baseline as history", changes)
	}
	before = len(changes)
	mustSend(t, reconnect, envelope("host-a", "instance-old", 3, "snapshot", `"panes":[`+workingPane+`]`))
	if len(m.Snapshot().Changes) != before {
		t.Fatal("identical reconciliation invented events")
	}
}

func TestObservationHistoryIsBoundedAndReportsDiscardedEntries(t *testing.T) {
	m, f := monitor(t, config("host-a"))
	c := nextConnection(t, f)
	mustSend(t, c, hello("host-a"))
	mustSend(t, c, baseline("host-a"))
	for i := 0; i < rx.MaxChanges+3; i++ {
		pane := fmt.Sprintf(`{"session_id":"session-one","block_id":"block-one","records":[{"state":"working","msg":"Synthetic observation %d"}]}`, i)
		mustSend(t, c, envelope("host-a", "instance-old", i+3, "pane", `"pane":`+pane))
	}
	snapshot := m.Snapshot()
	// First update also removes the child report: one baseline + 504 report
	// differences leave exactly 500 observations and five discarded entries.
	if len(snapshot.Changes) != 500 || snapshot.DroppedChanges != 5 {
		t.Fatal("wrong bounded history accounting", len(snapshot.Changes), snapshot.DroppedChanges)
	}
	if !strings.Contains(snapshot.Changes[len(snapshot.Changes)-1].Text, "Synthetic observation 502") {
		t.Fatal("newest event not retained")
	}
}
