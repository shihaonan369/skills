package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	d "agent-dashboard/internal/dashboard"
)

func TestCLIIntegration(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AGENT_DASHBOARD_HOME", root)
	call := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		if e := run(args, strings.NewReader(""), &out); e != nil {
			t.Fatal(args, e)
		}
		return out.String()
	}
	result := call("start", "--session", "cli", "--title", "Test issue")
	if !strings.Contains(result, "tui --session 'cli'") {
		t.Fatal(result)
	}
	var out bytes.Buffer
	payload := `{"session_id":"cli","hook_event_name":"PreToolUse","tool_use_id":"one","tool_name":"Bash","tool_input":{"command":"SECRET COMMAND"}}`
	if e := run([]string{"hook", "--source", "agent-dashboard-v1"}, strings.NewReader(payload), &out); e != nil || out.Len() != 0 {
		t.Fatal("hook not silent/failopen")
	}
	status := call("status", "--session", "cli")
	if strings.Contains(status, "SECRET COMMAND") {
		t.Fatal("tool input leaked")
	}
	var s d.State
	if e := json.Unmarshal([]byte(status), &s); e != nil || s.HookCount != 1 {
		t.Fatal(status)
	}
	call("plan", "--session", "cli", "--json", `[{"id":"1","subject":"Test","status":"completed"}]`)
	call("note", "--session", "cli", "--text", "Verified")
	call("finish", "--session", "cli", "--summary", "Tests passed")
	frame := call("tui", "--session", "cli", "--plain", "--width", "80", "--height", "24")
	if !strings.Contains(frame, "explicitly finished") {
		t.Fatal(frame)
	}
	call("deactivate", "--session", "cli")
	call("demo")
	demo := call("tui", "--session", "DEMO-layout", "--plain")
	if !strings.Contains(demo, "SYNTHETIC") {
		t.Fatal("unlabelled demo")
	}
}
func TestHooksFailOpenMalformedInactiveOversizeAndCorrupt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AGENT_DASHBOARD_HOME", root)
	for _, p := range []string{"", "{bad", `{"session_id":"inactive","hook_event_name":"Stop"}`, strings.Repeat("x", d.MaxInput+1)} {
		var out bytes.Buffer
		if e := run([]string{"hook"}, strings.NewReader(p), &out); e != nil || out.Len() != 0 {
			t.Fatal("hook blocks work")
		}
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("inactive hooks stored data")
	}
	if e := d.Start(root, "corrupt", "test", false); e != nil {
		t.Fatal(e)
	}
	paths, e := filepath.Glob(filepath.Join(root, "*", "state.json"))
	if e != nil || len(paths) != 1 {
		t.Fatal("state fixture missing")
	}
	os.WriteFile(paths[0], []byte("broken"), 0600)
	var out bytes.Buffer
	if e := run([]string{"hook"}, strings.NewReader(`{"session_id":"corrupt","hook_event_name":"Stop"}`), &out); e != nil || out.Len() != 0 {
		t.Fatal("corrupt hook blocked")
	}
	b, _ := os.ReadFile(paths[0])
	if string(b) != "broken" {
		t.Fatal("corrupt state overwritten")
	}
}
func TestCLIInvalidCommandsAndPlanFile(t *testing.T) {
	t.Setenv("AGENT_DASHBOARD_HOME", t.TempDir())
	var out bytes.Buffer
	for _, args := range [][]string{{"unknown"}, {"start"}, {"start", "--session", "${CLAUDE_SESSION_ID}"}, {"start", "--session", "s", "extra"}, {"plan", "--session", "s", "--json", "bad"}} {
		if e := run(args, strings.NewReader(""), &out); e == nil {
			t.Fatal("invalid args accepted", args)
		}
	}
	run([]string{"start", "--session", "s"}, strings.NewReader(""), &out)
	p := filepath.Join(t.TempDir(), "plan.json")
	os.WriteFile(p, []byte(`[{"id":"1","subject":"From file"}]`), 0600)
	if e := run([]string{"plan", "--session", "s", "--json", "@" + p}, strings.NewReader(""), &out); e != nil {
		t.Fatal(e)
	}
}

func TestCLINativeDAGFileLivePlanAndRollback(t *testing.T) {
	t.Setenv("AGENT_DASHBOARD_HOME", t.TempDir())
	call := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err != nil {
			t.Fatal(args, err)
		}
		return out.String()
	}
	call("demo", "--json", "@../../examples/dag/fanout.layout.json")
	frame := call("tui", "--session", "DEMO-layout", "--plain", "--width", "140", "--height", "48")
	for _, want := range []string{"SYNTHETIC", "Declared topology", "dag", "in_progress", "Forward flow"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	for _, obsolete := range []string{"[return]", "⇄", "▲"} {
		if strings.Contains(frame, obsolete) {
			t.Fatal("CLI rendered obsolete reverse flow", frame)
		}
	}
	call("plan", "--session", "DEMO-layout", "--json", `[{"id":"2","subject":"New observed value","status":"blocked"}]`)
	frame = call("tui", "--session", "DEMO-layout", "--plain", "--width", "140", "--height", "48")
	if !strings.Contains(frame, "blocked") || strings.Contains(frame, "in_progress") {
		t.Fatal("graph failed to refresh reported task state", frame)
	}
	var out bytes.Buffer
	if err := run([]string{"layout", "--session", "DEMO-layout", "--json", `{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a"}],"edges":[{"from":"a","to":"a"}]}}}`}, strings.NewReader(""), &out); err == nil {
		t.Fatal("CLI accepted cycle")
	}
	if !strings.Contains(call("status", "--session", "DEMO-layout"), "Synthetic DAG composition") {
		t.Fatal("invalid publication replaced layout")
	}
}
