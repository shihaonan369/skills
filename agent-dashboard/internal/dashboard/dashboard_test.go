package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func setup(t *testing.T) string {
	t.Helper()
	r := t.TempDir()
	if e := Start(r, "s", "Test", false); e != nil {
		t.Fatal(e)
	}
	return r
}
func snap(t *testing.T, r string) *State {
	t.Helper()
	s, e := Snapshot(r, "s")
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func send(t *testing.T, r string, p Payload) {
	t.Helper()
	p.Session = "s"
	if e := Ingest(r, p); e != nil {
		t.Fatal(e)
	}
}
func rawMap(s string) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	_ = json.Unmarshal([]byte(s), &m)
	return m
}
func TestLifecycleParallelAndPermission(t *testing.T) {
	r := setup(t)
	for _, id := range []string{"a", "b"} {
		send(t, r, Payload{Kind: "PreToolUse", ToolID: id, ToolName: "Read"})
	}
	send(t, r, Payload{Kind: "PermissionRequest", ToolID: "b", ToolName: "Read"})
	send(t, r, Payload{Kind: "PostToolUseFailure", ToolID: "a", ToolName: "Read"})
	s := snap(t, r)
	if len(s.Tools) != 1 || s.Agents["lead"].Status != "permission" || s.Agents["lead"].Errors != 1 {
		t.Fatalf("bad parallel state: %+v", s)
	}
	send(t, r, Payload{Kind: "PostToolUse", ToolID: "b", ToolName: "Read"})
	send(t, r, Payload{Kind: "PostToolUse", ToolID: "b", ToolName: "Read"}) // deduplicate
	if a := snap(t, r).Agents["lead"]; a.Tools != 2 || a.Status != "running" {
		t.Fatalf("tool failure must not fail whole agent: %+v", a)
	}
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "worker", AgentType: "Explore"})
	send(t, r, Payload{Kind: "Stop"})
	s = snap(t, r)
	if s.Finished || s.Agents["lead"].Status != "waiting" || s.Agents["worker"].Status != "running" {
		t.Fatal("Stop incorrectly completed work or hid background agent")
	}
	if e := Finish(r, "s", "Checked"); e != nil {
		t.Fatal(e)
	}
	send(t, r, Payload{Kind: "PostToolUse", ToolID: "finish-bash", ToolName: "Bash"})
	send(t, r, Payload{Kind: "Stop"})
	if s = snap(t, r); !s.Finished || s.Agents["lead"].Status != "done" {
		t.Fatal("completion shell's PostToolUse reset finish")
	}
	send(t, r, Payload{Kind: "SubagentStop", AgentID: "worker"})
	if snap(t, r).Agents["worker"].Status != "returned" {
		t.Fatal("worker return lost")
	}
	send(t, r, Payload{Kind: "UserPromptSubmit"})
	if snap(t, r).Finished {
		t.Fatal("new turn retained finish")
	}
}
func TestPermissionWithoutIDAndActorIsolation(t *testing.T) {
	r := setup(t)
	send(t, r, Payload{Kind: "PreToolUse", ToolID: "same", ToolName: "Read"})
	send(t, r, Payload{Kind: "PermissionRequest", ToolName: "Read"})
	send(t, r, Payload{Kind: "PreToolUse", ToolID: "other", ToolName: "Bash"})
	if snap(t, r).Agents["lead"].Status != "permission" {
		t.Fatal("parallel start cleared other wait")
	}
	send(t, r, Payload{Kind: "PreToolUse", AgentID: "w", ToolID: "same", ToolName: "Read"})
	send(t, r, Payload{Kind: "PostToolUse", AgentID: "w", ToolID: "same", ToolName: "Read"})
	if len(snap(t, r).Tools) != 2 {
		t.Fatal("worker completion cleared lead tool")
	}
	send(t, r, Payload{Kind: "PostToolUse", ToolName: "Read"})
	if len(snap(t, r).Tools) != 2 {
		t.Fatal("uncorrelated completion cleared parallel tools")
	}
}
func TestSessionEndAndAPIFailure(t *testing.T) {
	r := setup(t)
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "w"})
	send(t, r, Payload{Kind: "StopFailure"})
	if snap(t, r).Finished {
		t.Fatal("API error completed issue")
	}
	send(t, r, Payload{Kind: "SessionEnd"})
	s := snap(t, r)
	if s.Agents["lead"].Status != "ended" || s.Agents["w"].Status != "unconfirmed" {
		t.Fatal("termination fabricated")
	}
}
func TestOptInIsolationAndDeactivate(t *testing.T) {
	r := t.TempDir()
	if e := Ingest(r, Payload{Session: "unobserved", Kind: "PreToolUse"}); e != ErrInactive {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(r)
	if len(entries) != 0 {
		t.Fatal("unobserved session stored data")
	}
	if e := Start(r, "s", "title", false); e != nil {
		t.Fatal(e)
	}
	if e := Start(r, "other", "other", false); e != nil {
		t.Fatal(e)
	}
	send(t, r, Payload{Kind: "PreToolUse", ToolName: "Read", ToolID: "1"})
	o, e := Snapshot(r, "other")
	if e != nil || o.HookCount != 0 {
		t.Fatal("session leak")
	}
	if e := Deactivate(r, "s"); e != nil {
		t.Fatal(e)
	}
	if e := Ingest(r, Payload{Session: "s", Kind: "Stop"}); e != ErrInactive {
		t.Fatal(e)
	}
	if snap(t, r).HookCount != 1 {
		t.Fatal("deactivated session collected")
	}
	for _, id := range []string{"${CLAUDE_SESSION_ID}", "bad\x1b[2Jid", "bad\nid"} {
		if e := Start(r, id, "bad", false); e == nil {
			t.Fatal("unsafe/literal session accepted")
		}
	}
	if e := SeedDemo(r, "demo"); e != nil {
		t.Fatal(e)
	}
	if e := Start(r, "demo", "real", false); e == nil {
		t.Fatal("demo reused as real")
	}
	if e := Ingest(r, Payload{Session: "demo", Kind: "Stop"}); e != ErrInactive {
		t.Fatal("real hooks modified demo")
	}
}
func TestNativeTasksAndPlanRollback(t *testing.T) {
	r := setup(t)
	send(t, r, Payload{Kind: "PostToolUse", ToolName: "TaskCreate", ToolID: "1", Input: rawMap(`{"subject":"Build","description":"SECRET DESCRIPTION"}`), Response: json.RawMessage(`{"task":{"id":"1","subject":"Build","status":"pending"}}`)})
	send(t, r, Payload{Kind: "PostToolUse", ToolName: "TaskUpdate", ToolID: "2", Input: rawMap(`{"taskId":"1","status":"in_progress","owner":"w","addBlockedBy":["2","2"]}`)})
	s := snap(t, r)
	task := s.Tasks["1"]
	if task.Status != "in_progress" || task.Owner != "w" || len(task.BlockedBy) != 1 {
		t.Fatalf("bad task %+v", task)
	}
	if s.Progress("w") != "0/1 steps" || s.Progress("missing") != "unknown" {
		t.Fatal("fabricated progress")
	}
	send(t, r, Payload{Kind: "PostToolUseFailure", ToolName: "TaskUpdate", ToolID: "3", Input: rawMap(`{"taskId":"1","status":"completed"}`)})
	send(t, r, Payload{Kind: "PostToolUse", ToolName: "TaskUpdate", ToolID: "4", Input: rawMap(`{"taskId":"1","status":"completed"}`), Response: json.RawMessage(`{"success":false}`)})
	if snap(t, r).Tasks["1"].Status != "in_progress" {
		t.Fatal("failed task mutation applied")
	}
	if e := Plan(r, "s", []byte(`[{"id":"x","subject":"Good"},{"id":"y","status":"invented"}]`)); e == nil {
		t.Fatal("invalid plan accepted")
	}
	if !reflect.DeepEqual(snap(t, r).Tasks, s.Tasks) {
		t.Fatal("failed plan partially committed")
	}
	send(t, r, Payload{Kind: "PostToolUse", ToolName: "TodoWrite", ToolID: "5", Input: rawMap(`{"todos":[{"content":"Test","status":"completed"}]}`)})
	if s = snap(t, r); len(s.Tasks) != 1 || s.Tasks["1"].Subject != "Test" || s.Progress("") != "1/1 steps" {
		t.Fatal("TodoWrite normalization")
	}
	send(t, r, Payload{Kind: "PostToolUse", ToolName: "TaskList", ToolID: "6", Response: json.RawMessage(`{"tasks":[]}`)})
	if len(snap(t, r).Tasks) != 0 {
		t.Fatal("authoritative list left stale tasks")
	}
	b, e := os.ReadFile(filepath.Join(sessionDir(r, "s"), "state.json"))
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), "SECRET DESCRIPTION") {
		t.Fatal("stored arbitrary input")
	}
}
func TestModelsScopedAndNeverRequested(t *testing.T) {
	r := setup(t)
	lead := filepath.Join(t.TempDir(), "main.jsonl")
	worker := filepath.Join(t.TempDir(), "worker.jsonl")
	os.WriteFile(lead, []byte("garbage\n"+`{"type":"assistant","message":{"model":"lead-actual"}}`+"\n"+`{"type":"assistant","agentId":"w","message":{"model":"other-agent"}}`+"\n"), 0600)
	os.WriteFile(worker, []byte(`{"type":"assistant","agentId":"w","message":{"model":"worker-actual"}}`+"\n"), 0600)
	send(t, r, Payload{Kind: "PreToolUse", ToolName: "Agent", ToolID: "1", Transcript: lead, Input: rawMap(`{"model":"requested-not-actual"}`)})
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "w", Transcript: lead})
	s := snap(t, r)
	if s.Agents["lead"].Model != "lead-actual" || s.Agents["w"].Model != "" {
		t.Fatal("borrowed lead/requested model")
	}
	send(t, r, Payload{Kind: "SubagentStop", AgentID: "w", AgentTranscript: worker})
	if snap(t, r).Agents["w"].Model != "worker-actual" {
		t.Fatal("own transcript not used")
	}
	send(t, r, Payload{Kind: "PostModelSwitch", ToModel: "new-model"})
	send(t, r, Payload{Kind: "PostToolUse", ToolID: "1", ToolName: "Agent", Transcript: lead})
	if snap(t, r).Agents["lead"].Model != "new-model" {
		t.Fatal("stale transcript overrode observed switch")
	}
}
func TestBoundsAndSanitization(t *testing.T) {
	r := setup(t)
	if e := Transaction(r, "s", false, func(sp **State) error {
		for i := 0; i < MaxEvents+10; i++ {
			(*sp).event("lead", "note", "\x1b]52;c;secret\a\n\r\u202ehidden")
		}
		for i := 0; i < MaxAgents-1; i++ {
			id := fmt.Sprint(i)
			(*sp).Agents[id] = &Agent{ID: id}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	send(t, r, Payload{Kind: "SubagentStart", AgentID: "overflow"})
	s := snap(t, r)
	if len(s.Events) != MaxEvents || len(s.Agents) != MaxAgents || s.Dropped != 1 {
		t.Fatal("unbounded store")
	}
	if strings.ContainsAny(s.Events[0].Text, "\x1b\a\n\r\u202e") {
		t.Fatal("terminal controls retained")
	}
	tasks := []map[string]string{}
	for i := 0; i < MaxTasks+5; i++ {
		tasks = append(tasks, map[string]string{"id": fmt.Sprint(i), "subject": "s"})
	}
	b, _ := json.Marshal(tasks)
	if e := Plan(r, "s", b); e == nil {
		t.Fatal("unbounded explicit plan")
	}
	b, _ = json.Marshal(map[string]any{"todos": tasks})
	send(t, r, Payload{Kind: "PostToolUse", ToolName: "TodoWrite", Input: rawMap(string(b))})
	if s = snap(t, r); len(s.Tasks) != MaxTasks || s.Dropped != 6 {
		t.Fatal("native plan unbounded")
	}
	if runtime.GOOS == "windows" {
		return
	} // Windows privacy relies on profile ACLs, not POSIX mode bits.
	dir := sessionDir(r, "s")
	if st, e := os.Stat(dir); e != nil || st.Mode().Perm() != 0700 {
		t.Fatal("session permissions")
	}
	if st, e := os.Stat(filepath.Join(dir, "state.json")); e != nil || st.Mode().Perm() != 0600 {
		t.Fatal("state permissions")
	}
}
func TestProcessWriter(t *testing.T) {
	if os.Getenv("DASH_TEST_WORKER") != "1" {
		return
	}
	r := os.Getenv("DASH_TEST_ROOT")
	for i := 0; i < 8; i++ {
		if e := Ingest(r, Payload{Session: "s", Kind: "UserPromptSubmit"}); e != nil {
			t.Fatal(e)
		}
	}
}
func TestConcurrentProcessesAndReaders(t *testing.T) {
	r := setup(t)
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	errch := make(chan error, 16)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := exec.Command(exe, "-test.run=^TestProcessWriter$")
			c.Env = append(os.Environ(), "DASH_TEST_WORKER=1", "DASH_TEST_ROOT="+r)
			if b, e := c.CombinedOutput(); e != nil {
				errch <- fmt.Errorf("%w %s", e, b)
			}
		}()
	}
	for i := 0; i < 15; i++ {
		if _, e := Snapshot(r, "s"); e != nil {
			t.Fatal(e)
		}
	}
	wg.Wait()
	close(errch)
	for e := range errch {
		t.Error(e)
	}
	if s := snap(t, r); s.HookCount != 96 {
		t.Fatalf("lost concurrent writes: %d", s.HookCount)
	}
}
func TestCorruptStateNotOverwritten(t *testing.T) {
	r := setup(t)
	p := filepath.Join(sessionDir(r, "s"), "state.json")
	os.WriteFile(p, []byte("broken"), 0600)
	if e := Start(r, "s", "overwrite?", false); e == nil {
		t.Fatal("corruption overwritten")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "broken" {
		t.Fatal("corruption destroyed")
	}
	os.WriteFile(p, []byte(`{"version":1,"session":"s","agents":{"lead":{},"invalid":null},"tasks":{},"tools":{}}`), 0600)
	if _, e := Snapshot(r, "s"); e == nil {
		t.Fatal("null agent corruption accepted")
	}
}
func TestResponsiveRendererAndKeys(t *testing.T) {
	r := t.TempDir()
	if e := SeedDemo(r, "demo"); e != nil {
		t.Fatal(e)
	}
	s, _ := Snapshot(r, "demo")
	s.Title = "Malicious\x1b[2Jtitle 👩‍💻 é"
	s.Tasks["evil"] = Task{ID: "evil", Subject: "Start 👩‍💻\x1b]52;c;secret\a end", Status: "blocked"}
	for _, w := range []int{20, 28, 40, 68, 80, 100, 160} {
		for _, h := range []int{8, 12, 24, 40} {
			for page := 0; page < 3; page++ {
				v := Viewer{State: s, Width: w, Height: h, Page: page, Selected: 999, Offset: 0}
				frame := v.Render(time.Now())
				lines := strings.Split(frame, "\n")
				if len(lines) > h {
					t.Errorf("height %d/%d", len(lines), h)
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > w {
						t.Errorf("width %d/%d: %q", ansi.StringWidth(line), w, line)
					}
					if strings.ContainsAny(line, "\x1b\a\r\u202e") {
						t.Errorf("control injection %q", line)
					}
				}
			}
		}
	}
	v := Viewer{State: s, Width: 100, Height: 30}
	if !strings.Contains(v.View(), "SYNTHETIC") {
		t.Fatal("demo unlabelled")
	}
	for _, key := range []string{"2", "3", "1", "p", "?", "j", "k"} {
		m, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		v = m.(Viewer)
	}
	if !v.Paused || !v.Help {
		t.Fatal("navigation keys")
	}
	before := s.Active
	_, quit := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quit == nil || s.Active != before {
		t.Fatal("quit affected collection")
	}
}
