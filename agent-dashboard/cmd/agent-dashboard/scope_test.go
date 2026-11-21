package main

import (
	d "agent-dashboard/internal/dashboard"
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopeCLIFromObservationToAgentReplyAndLiveSelection(t *testing.T) {
	t.Setenv("AGENT_DASHBOARD_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "scope.json")
	call := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err != nil {
			t.Fatal(args, err)
		}
		return out.String()
	}
	call("scope-init", "--file", path, "--criterion", "Show implementation agents only")
	call("start", "--session", "cli-scope")
	panel := map[string]any{"version": 1, "root": map[string]any{"id": "dynamic", "kind": "agents", "select": map[string]any{"kind": "actors", "scope_file": path}}}
	raw, _ := json.Marshal(panel)
	call("layout", "--session", "cli-scope", "--json", string(raw))
	if err := d.Ingest(mustRoot(t), d.Payload{Session: "cli-scope", Kind: "SubagentStart", AgentID: "actual-worker", AgentType: "general-purpose"}); err != nil {
		t.Fatal(err)
	}
	frame := call("tui", "--session", "cli-scope", "--plain")
	if !strings.Contains(frame, "1 pending") || strings.Contains(frame, "actual-worker") {
		t.Fatal(frame)
	}
	watch := call("scope-watch", "--file", path, "--duration", "1")
	var initial d.ScopeBatch
	if err := json.Unmarshal([]byte(watch), &initial); err != nil || len(initial.Requests) != 1 || initial.Requests[0].Bind.Actor != "actual-worker" {
		t.Fatal("initial pending work must be one JSON-line event without idle repeats", err, watch)
	}
	var batch d.ScopeBatch
	if err := json.Unmarshal([]byte(call("scope-poll", "--file", path)), &batch); err != nil || len(batch.Requests) != 1 {
		t.Fatal(err, batch)
	}
	r := batch.Requests[0]
	answer, _ := json.Marshal(d.ScopeDecisions{PolicyHash: batch.PolicyHash, Decisions: []d.ScopeDecision{{Key: r.Key, Fingerprint: r.Fingerprint, Decision: "include", Origin: "main-agent", Evidence: "Synthetic CLI fixture, not an actual model judgement"}}})
	call("scope-decide", "--file", path, "--json", string(answer))
	if frame = call("tui", "--session", "cli-scope", "--plain"); !strings.Contains(frame, "actual-worker") {
		t.Fatal(frame)
	}
	if status := call("scope-status", "--file", path); !strings.Contains(status, `"origin":"main-agent"`) {
		t.Fatal(status)
	}
	call("scope-close", "--file", path)
	if poll := call("scope-poll", "--file", path, "--wait", "60"); !strings.Contains(poll, `"closed":true`) {
		t.Fatal(poll)
	}
	if watch := call("scope-watch", "--file", path, "--duration", "1800"); !strings.Contains(watch, `"closed":true`) || !strings.Contains(watch, `"requests":[]`) {
		t.Fatal("closed watcher emitted work or failed to exit", watch)
	}
	for _, args := range [][]string{{"scope-watch", "--file", path, "--duration", "0"}, {"scope-watch", "--file", path, "--duration", "1801"}, {"scope-poll", "--file", path, "--wait", "61"}, {"scope-init"}, {"scope-invented", "--file", path}, {"scope-decide", "--file", path, "--json", "{}"}} {
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err == nil {
			t.Fatal("invalid scope CLI accepted", args)
		}
	}
}
func mustRoot(t *testing.T) string {
	t.Helper()
	root, err := d.Root()
	if err != nil {
		t.Fatal(err)
	}
	return root
}
