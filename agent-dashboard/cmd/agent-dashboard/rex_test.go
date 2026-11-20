package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rx "agent-dashboard/internal/rexwatch"
)

// A real native subprocess impersonates only the public Rex CLI protocol. No
// shell or installed Rex is needed by the portable CLI integration tests.
func TestMain(m *testing.M) {
	if os.Getenv("AGENT_DASHBOARD_TEST_FAKE_REX") == "1" {
		fakeRexCLI()
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func fakeRexCLI() {
	args := os.Args[1:]
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--server=unix:///unreachable") {
		fmt.Fprintln(os.Stderr, "synthetic endpoint unavailable")
		os.Exit(3)
	}
	if strings.Contains(joined, "hosts check") {
		fmt.Println(`{"hosts":[{"label":"selected","status":"reachable","endpoint":"unix:///fixture-other"}]}`)
		return
	}
	if strings.Contains(joined, "whoami") {
		host := "fixture-host"
		if strings.Contains(joined, "--server=unix:///fixture-other") {
			host = "fixture-other"
		}
		fmt.Printf(`{"host_id":%q,"instance_id":"fixture-instance","endpoint":"unix:///fixture","session_id":"fixture-session","principal":{"pid":%d,"transport":"unix"}}`, host, os.Getpid())
		return
	}
	for i, arg := range args {
		if arg == "--args" && i+1 < len(args) {
			var opts struct {
				Host     string   `json:"host_id"`
				Sessions []string `json:"sessions"`
			}
			_ = json.Unmarshal([]byte(args[i+1]), &opts)
			session := "fixture-session"
			if len(opts.Sessions) > 0 {
				session = opts.Sessions[0]
			}
			fmt.Printf("{\"agent_dashboard_rex\":1,\"type\":\"hello\",\"host_id\":%q,\"instance_id\":\"fixture-instance\",\"seq\":1}\n", opts.Host)
			fmt.Printf("{\"agent_dashboard_rex\":1,\"type\":\"snapshot\",\"host_id\":%q,\"instance_id\":\"fixture-instance\",\"seq\":2,\"panes\":[{\"session_id\":%q,\"block_id\":\"fixture-block\",\"records\":[{\"state\":\"done\",\"app\":\"claude-code\",\"msg\":\"Synthetic CLI contract only\"}]}]}\n", opts.Host, session)
			time.Sleep(5 * time.Minute)
			return
		}
	}
	fmt.Fprintln(os.Stderr, "unexpected synthetic Rex CLI request")
	os.Exit(4)
}

func TestRexCLICompositionAndStatusWithoutClaudeHooksOrStoredReports(t *testing.T) {
	t.Setenv("REX_SESSION", "fixture-session")
	root := t.TempDir()
	t.Setenv("AGENT_DASHBOARD_HOME", root)
	t.Setenv("AGENT_DASHBOARD_TEST_FAKE_REX", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	call := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err != nil {
			t.Fatal(args, err)
		}
		return out.String()
	}
	var cfg rx.Config
	if err := json.Unmarshal([]byte(call("rex-sources", "--host", "selected", "--rex-bin", exe)), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Sources) != 2 || cfg.Sources[0].Sessions[0] != "fixture-session" || cfg.Sources[1].Endpoint != "unix:///fixture-other" {
		t.Fatalf("unexpected source resolution: %+v", cfg)
	}
	cfgPath := filepath.Join(t.TempDir(), "sources.json")
	raw, _ := json.Marshal(cfg)
	if err := os.WriteFile(cfgPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	layoutPath := filepath.Join(t.TempDir(), "layout.json")
	if err := os.WriteFile(layoutPath, []byte(`{"version":1,"name":"Synthetic contract layout","root":{"id":"reports","kind":"programs","compact":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	frame := call("rex-watch", "--sources", "@"+cfgPath, "--layout", "@"+layoutPath, "--rex-bin", exe, "--plain", "--width", "100", "--height", "24")
	for _, want := range []string{"Synthetic contract layout", "REX PROGRAM REPORTS", "[done] claude-code ?", "current", "selected"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q in frame:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "explicitly finished") || strings.Contains(frame, "hooks ·") {
		t.Fatal("program status was presented as Claude lifecycle telemetry")
	}
	var snapshot rx.Snapshot
	if err := json.Unmarshal([]byte(call("rex-watch", "--sources", "@"+cfgPath, "--rex-bin", exe, "--status")), &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Sources) != 2 || len(snapshot.Sources[0].Programs) != 1 || snapshot.Sources[0].Programs[0].Record.Progress != nil {
		t.Fatal("unknown program progress was not preserved")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("read-only program observations wrote local session state or settings")
	}
}

func TestRexCLIValidationCapabilitiesAndTerminalFallback(t *testing.T) {
	t.Setenv("AGENT_DASHBOARD_HOME", t.TempDir())
	t.Setenv("REX_SESSION", "")
	t.Setenv("REX_BLOCK", "")
	var out bytes.Buffer
	if err := run([]string{"capabilities"}, strings.NewReader(""), &out); err != nil || !strings.Contains(out.String(), "rex-observer-v1") || !strings.Contains(out.String(), "rex-local-open-v1") || !strings.Contains(out.String(), "observation-bindings-v1") {
		t.Fatal("optional feature discovery failed", err)
	}
	for _, args := range [][]string{{"rex-watch"}, {"rex-watch", "--plain", "--status"}, {"rex-sources", "--sources", "@ignored"}, {"open"}, {"open", "--plain"}, {"open", "--session", "s", "--sources", "@ignored"}, {"rex-watch", "--sources", "{}"}, {"rex-watch", "--rex-bin", "relative"}} {
		out.Reset()
		if err := run(args, strings.NewReader(""), &out); err == nil {
			t.Fatal("invalid adapter arguments accepted", args)
		}
	}
	out.Reset()
	if err := run([]string{"start", "--session", "real-claude-session"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"open", "--session", "real-claude-session"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var opened struct {
		Mode   string   `json:"mode"`
		Args   []string `json:"args"`
		Reason string   `json:"reason"`
	}
	if json.Unmarshal(out.Bytes(), &opened) != nil || opened.Mode != "terminal" || opened.Reason == "" || !strings.Contains(strings.Join(opened.Args, " "), "--state-dir") {
		t.Fatal("non-Rex terminal-independent fallback was lost", out.String())
	}
	out.Reset()
	if err := run([]string{"tui", "--session", "real-claude-session", "--state-dir", os.Getenv("AGENT_DASHBOARD_HOME"), "--plain"}, strings.NewReader(""), &out); err != nil {
		t.Fatal("explicit viewer state directory was not supported", err)
	}
}

func TestRexCLIUnavailableSourcePrintsExplicitPartialStatusAndFails(t *testing.T) {
	t.Setenv("AGENT_DASHBOARD_TEST_FAKE_REX", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sources.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"sources":[{"name":"Synthetic unavailable host","endpoint":"unix:///unreachable","host_id":"fixture-host"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"rex-watch", "--sources", "@" + path, "--rex-bin", exe, "--status"}, strings.NewReader(""), &out); err == nil {
		t.Fatal("unavailable sources were reported as a complete snapshot")
	}
	var snapshot rx.Snapshot
	if json.Unmarshal(out.Bytes(), &snapshot) != nil || len(snapshot.Sources) != 1 || snapshot.Sources[0].Health != "reconnecting" || len(snapshot.Sources[0].Programs) != 0 || snapshot.Sources[0].Error == "" {
		t.Fatal("partial status did not explicitly retain unknown observations", out.String())
	}
}
