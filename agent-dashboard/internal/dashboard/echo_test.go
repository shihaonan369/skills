package dashboard

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func copyState(t *testing.T, s *State) *State {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var next State
	if err := json.Unmarshal(raw, &next); err != nil {
		t.Fatal(err)
	}
	return &next
}
func echoViewer(t *testing.T) (Viewer, string, time.Time) {
	t.Helper()
	root := setup(t)
	if err := SetLayout(root, "s", []byte(`{"version":1,"root":{"kind":"column","children":[{"id":"a","kind":"agent","actor":"lead","size":7},{"id":"t","kind":"tasks","size":6},{"id":"log","kind":"events"}]}}`)); err != nil {
		t.Fatal(err)
	}
	at := time.Unix(100, 0)
	v := receive(Viewer{Width: 100, Height: 30}, snap(t, root), at)
	return v, root, at
}
func TestEchoFirstLoadDuplicateSnapshotAndExpiry(t *testing.T) {
	v, root, at := echoViewer(t)
	if len(v.Echo.changed) != 0 {
		t.Fatal("initial observation flashed")
	}
	send(t, root, Payload{Kind: "PostModelSwitch", ToModel: "observed-model"})
	v = receive(v, snap(t, root), at)
	if !reflect.DeepEqual(v.Echo.changed, map[echoKey]bool{actorKey("lead", "model"): true}) || v.Echo.until != at.Add(echoDuration) {
		t.Fatal("not an exact observed model change", v.Echo)
	}
	frame := ansi.Strip(v.Render(at))
	if !strings.Contains(frame, "▸ model observed-model") || strings.Contains(frame, "▸ lead · lead · running") {
		t.Fatal("echo leaked to unchanged rows", frame)
	}
	if !strings.Contains(ansi.Strip(v.Render(at.Add(echoDuration))), "· model observed-model") {
		t.Fatal("expired echo lost static change marker")
	}
	until := v.Echo.until
	v = receive(v, snap(t, root), at.Add(900*time.Millisecond))
	if v.Echo.until != until {
		t.Fatal("duplicate snapshot extended echo")
	}
	// Notes, revisions, layout title and elapsed time are not field changes.
	if err := Note(root, "s", "lead", "new event, same field values"); err != nil {
		t.Fatal(err)
	}
	v = receive(v, snap(t, root), at.Add(time.Second))
	if v.Echo.until != until || len(v.Echo.changed) != 1 {
		t.Fatal("event/revision fabricated a field change")
	}
	v = viewerKey(v, "x")
	v = receive(v, snap(t, root), at.Add(2*time.Second))
	if len(v.Echo.changed) != 0 {
		t.Fatal("acknowledged change replayed on duplicate")
	}
	fresh := receive(Viewer{}, snap(t, root), at)
	if len(fresh.Echo.changed) != 0 {
		t.Fatal("reopening flashed existing values")
	}
}
func TestEchoToolFailureKeepsOtherCallsAndAgentLifecycle(t *testing.T) {
	v, root, at := echoViewer(t)
	for _, id := range []string{"first", "second"} {
		send(t, root, Payload{Kind: "PreToolUse", ToolID: id, ToolName: "Read"})
	}
	v = receive(v, snap(t, root), at)
	if !v.Echo.changed[actorKey("lead", "active")] || v.Echo.changed[actorKey("lead", "status")] {
		t.Fatal("tool activity not independent from lifecycle")
	}
	send(t, root, Payload{Kind: "PostToolUseFailure", ToolID: "first", ToolName: "Read"})
	v = receive(v, snap(t, root), at.Add(time.Second))
	if len(v.State.Tools) != 1 || !v.Echo.changed[actorKey("lead", "errors")] || v.Echo.changed[actorKey("lead", "status")] || v.State.Agents["lead"].Status != "running" || v.State.Finished {
		t.Fatal("tool error cleared parallel call or failed/completed agent", v.Echo.changed)
	}
	frame := ansi.Strip(v.Render(at.Add(time.Second)))
	if !strings.Contains(frame, "▸ tools 1 · errors 1 | Read") || strings.Contains(frame, "▸ lead · lead · running") {
		t.Fatal("tool failure highlighted entire agent", frame)
	}
	// A different same-named call ID is a real activity change even when the
	// human-readable tool-name list is identical.
	next := copyState(t, v.State)
	for id, tool := range next.Tools {
		delete(next.Tools, id)
		next.Tools["different-correlated-call"] = tool
		break
	}
	next.Revision++
	v = receive(v, next, at.Add(2*time.Second))
	if !reflect.DeepEqual(v.Echo.changed, map[echoKey]bool{actorKey("lead", "active"): true}) {
		t.Fatal("call identity conflated with name", v.Echo.changed)
	}
	if err := Plan(root, "s", []byte(`[{"id":"1","subject":"Reported item","status":"completed"}]`)); err != nil {
		t.Fatal(err)
	}
	// The synthetic state revision above was one ahead of disk; make the next
	// real public transaction newer without changing lifecycle.
	if err := Note(root, "s", "lead", "plan evidence"); err != nil {
		t.Fatal(err)
	}
	v = receive(v, snap(t, root), at.Add(3*time.Second))
	if !v.Echo.changed[taskKey("1", "status")] || v.State.Finished || v.State.Agents["lead"].Status != "running" {
		t.Fatal("task completion became agent completion")
	}
	send(t, root, Payload{Kind: "Stop"})
	v = receive(v, snap(t, root), at.Add(4*time.Second))
	if v.State.Agents["lead"].Status != "waiting" || v.State.Finished || !v.Echo.changed[actorKey("lead", "status")] {
		t.Fatal("ordinary Stop incorrectly successful")
	}
}
func TestEchoPauseGenerationBaselineAndReadErrors(t *testing.T) {
	v, root, at := echoViewer(t)
	oldEpoch := v.readEpoch
	send(t, root, Payload{Kind: "PostModelSwitch", ToModel: "during-pause"})
	pending := snapshotMsg{state: snap(t, root), epoch: oldEpoch, at: at}
	v = viewerKey(v, "p")
	m, _ := v.Update(pending)
	v = m.(Viewer)
	if v.State.Agents["lead"].Model != "" {
		t.Fatal("in-flight update changed paused snapshot")
	}
	m, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	v = m.(Viewer)
	if cmd == nil || !v.baselineOnLoad {
		t.Fatal("resume must immediately schedule a fresh baseline read")
	}
	m, _ = v.Update(pending)
	v = m.(Viewer)
	if v.State.Agents["lead"].Model != "" || !v.baselineOnLoad {
		t.Fatal("pre-pause read was accepted after resume")
	}
	m, _ = v.Update(snapshotMsg{err: errors.New("private snapshot temporarily busy"), epoch: v.readEpoch})
	v = m.(Viewer)
	if v.Error == "" || !v.baselineOnLoad {
		t.Fatal("read failure lost retained view or pending baseline")
	}
	v = receive(v, snap(t, root), at.Add(time.Second))
	if v.State.Agents["lead"].Model != "during-pause" || len(v.Echo.changed) != 0 || v.baselineOnLoad || v.Error != "" {
		t.Fatal("resume replayed accumulated changes instead of baseline")
	}
	send(t, root, Payload{Kind: "PostModelSwitch", ToModel: "after-resume"})
	v = receive(v, snap(t, root), at.Add(2*time.Second))
	if !v.Echo.changed[actorKey("lead", "model")] {
		t.Fatal("baseline suppressed future fresh changes")
	}
	stale := copyState(t, v.State)
	stale.Revision--
	stale.Agents["lead"].Model = "old"
	v = receive(v, stale, at.Add(3*time.Second))
	if v.State.Agents["lead"].Model != "after-resume" {
		t.Fatal("old snapshot regressed observation")
	}
}
func TestEchoMotionHistoryAndMultiChangeGeometry(t *testing.T) {
	v, root, at := echoViewer(t)
	v = viewerKey(v, "a")
	send(t, root, Payload{Kind: "PostModelSwitch", ToModel: "quiet-model"})
	v = receive(v, snap(t, root), at)
	frame := ansi.Strip(v.Render(at))
	if !strings.Contains(frame, "· model quiet-model") || strings.Contains(frame, "▸ model quiet-model") || len(v.Echo.changed) == 0 {
		t.Fatal("motion-off lost readable change information", frame)
	}
	v = viewerKey(v, "a")
	if strings.Contains(ansi.Strip(v.Render(at)), "▸ model quiet-model") {
		t.Fatal("enabling animation replayed old change")
	}
	next := copyState(t, v.State)
	next.Revision++
	next.Agents["lead"].Model = "batch-model"
	next.Agents["lead"].Errors++
	next.Tasks["t"] = Task{ID: "t", Status: "completed", Subject: "合成任务 👩‍💻", Owner: "lead"}
	v = receive(v, next, at.Add(time.Second))
	if !v.Echo.changed[actorKey("lead", "errors")] || !v.Echo.changed[actorKey("lead", "model")] || !v.Echo.changed[taskKey("t", "status")] || v.Echo.changed[actorKey("lead", "status")] {
		t.Fatal("batch lost exact field changes", v.Echo.changed)
	}
	for _, w := range []int{28, 45, 80, 140} {
		for _, h := range []int{8, 14, 30, 56} {
			v.Width, v.Height = w, h
			assertDAGFrame(t, ansi.Strip(v.Render(at.Add(time.Second))), w, h)
			v.MotionOff = true
			assertDAGFrame(t, ansi.Strip(v.Render(at.Add(time.Second))), w, h)
			v.MotionOff = false
		}
	}
	// Entering historical/diagnostic/help views cancels transient emphasis;
	// returning must not replay it, even before the 1.2s deadline.
	v = viewerKey(v, "3")
	v = viewerKey(v, "1")
	if !v.Echo.until.IsZero() {
		t.Fatal("diagnostic navigation replays old echo")
	}
	v = viewerKey(v, "?")
	next = copyState(t, v.State)
	next.Revision++
	next.Agents["lead"].Model = "while-help"
	v = receive(v, next, at.Add(2*time.Second))
	v = viewerKey(v, "?")
	if !v.Echo.until.IsZero() || !v.Echo.changed[actorKey("lead", "model")] {
		t.Fatal("help should retain static summary, not animate")
	}
	v.Offsets = map[string]int{"a": 1}
	next = copyState(t, v.State)
	next.Revision++
	next.Agents["lead"].Model = "while-scrolled"
	v = receive(v, next, at.Add(3*time.Second))
	if !v.Echo.until.IsZero() || v.Offsets["a"] != 1 {
		t.Fatal("historical scroll animated or stole scroll position")
	}
}
func TestEchoEventHistoryAnchorsFilteredAndRotatingBuffers(t *testing.T) {
	v, _, at := echoViewer(t)
	old := copyState(t, v.State)
	old.Events = []Event{{Time: 1, Actor: "w", Text: "one"}, {Time: 2, Actor: "lead", Text: "two"}, {Time: 3, Actor: "w", Text: "three"}, {Time: 4, Actor: "lead", Text: "four"}}
	old.Layout.Root.Children[2].Actors = []string{"w"}
	v = receive(v, old, at)
	v.Selected, v.Page, v.Offset = 2, 2, 2
	v.Offsets = map[string]int{"log": 1}
	next := copyState(t, old)
	next.Revision++
	next.Events = append(next.Events[1:], Event{Time: 5, Actor: "w", Text: "five"}, Event{Time: 6, Actor: "lead", Text: "six"})
	v = receive(v, next, at.Add(time.Second))
	if v.Offsets["log"] != 1 { // oldest filtered anchor was evicted: clamp, not jump
		t.Fatal("filtered rotated buffer did not clamp evicted anchor", v.Offsets)
	}
	if v.Offset != 4 {
		t.Fatal("diagnostic history shifted instead of anchoring", v.Offset)
	}
	next = copyState(t, next)
	next.Revision++
	next.Events = append(next.Events, Event{Time: 7, Actor: "w", Text: "seven"})
	v = receive(v, next, at.Add(2*time.Second))
	if v.Offsets["log"] != 2 || v.Offset != 5 {
		t.Fatal("new matching event did not preserve reading anchor", v.Offsets, v.Offset)
	}
	// A duplicate read must not drift either offset.
	v = receive(v, next, at.Add(3*time.Second))
	if v.Offsets["log"] != 2 || v.Offset != 5 || !v.Echo.until.IsZero() {
		t.Fatal("duplicate/history read moved scroll or generated an effect")
	}
}

func TestEchoWorkerListAndDAGUseOnlyDisplayedFacts(t *testing.T) {
	root := setup(t)
	if err := SetLayout(root, "s", []byte(`{"version":1,"root":{"kind":"column","children":[{"id":"g","kind":"dag","size":22,"dag":{"version":1,"nodes":[{"id":"worker","bind":{"actor":"w"}},{"id":"task","bind":{"task":"1"}},{"id":"note"}]}},{"id":"workers","kind":"agents"}]}}`)); err != nil {
		t.Fatal(err)
	}
	at := time.Unix(100, 0)
	v := receive(Viewer{Width: 120, Height: 35}, snap(t, root), at)
	send(t, root, Payload{Kind: "SubagentStart", AgentID: "w", AgentType: "Observed role"})
	v = receive(v, snap(t, root), at)
	frame := ansi.Strip(v.Render(at))
	if !strings.Contains(frame, "▸ Observed role · w · running") || !strings.Contains(frame, "unobserved · unknown") || strings.Contains(frame, "✓ running") {
		t.Fatal("new worker list or absent task binding misrepresented", frame)
	}
	send(t, root, Payload{Kind: "PreToolUse", AgentID: "w", ToolID: "a", ToolName: "Read"})
	v = receive(v, snap(t, root), at.Add(time.Second))
	n := v.State.Layout.Root.Children[0].DAG.Nodes[0]
	if changed, _ := v.Echo.marks(nodeEchoKeys(n, v.State), at.Add(time.Second), true); changed {
		t.Fatal("active-call detail echoed a DAG status/count that did not change")
	}
	if err := Plan(root, "s", []byte(`[{"id":"1","subject":"First title","status":"in_progress"}]`)); err != nil {
		t.Fatal(err)
	}
	v = receive(v, snap(t, root), at.Add(2*time.Second))
	n = v.State.Layout.Root.Children[0].DAG.Nodes[1]
	if changed, _ := v.Echo.marks(nodeEchoKeys(n, v.State), at.Add(2*time.Second), true); !changed {
		t.Fatal("new observed task state did not echo its bound DAG row")
	}
	if err := Plan(root, "s", []byte(`[{"id":"1","subject":"Changed title","status":"in_progress"}]`)); err != nil {
		t.Fatal(err)
	}
	v = receive(v, snap(t, root), at.Add(3*time.Second))
	if changed, _ := v.Echo.marks(nodeEchoKeys(n, v.State), at.Add(3*time.Second), true); changed {
		t.Fatal("subject edit echoed a DAG state row that does not display subject")
	}
	send(t, root, Payload{Kind: "SubagentStop", AgentID: "w"})
	v = receive(v, snap(t, root), at.Add(4*time.Second))
	frame = ansi.Strip(v.Render(at.Add(4 * time.Second)))
	if !strings.Contains(frame, "▸ returned") || strings.Contains(frame, "✓ returned") || !strings.Contains(frame, "declared only") {
		t.Fatal("return or unbound annotation became completion", frame)
	}
}
