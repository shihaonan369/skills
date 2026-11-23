package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalDataCLIRefreshesWithoutRewritingLayoutOrTelemetry(t *testing.T) {
	t.Setenv("AGENT_DASHBOARD_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "value.json")
	call := func(input string, args ...string) string {
		t.Helper()
		var out bytes.Buffer
		if err := run(args, strings.NewReader(input), &out); err != nil {
			t.Fatal(args, err)
		}
		return out.String()
	}
	call("", "start", "--session", "data-cli")
	layout := map[string]any{"version": 1, "data_sources": map[string]any{"timer": map[string]any{"type": "json-file", "path": path}}, "root": map[string]any{"id": "timer", "kind": "text", "value": map[string]any{"source": "timer", "path": "/seconds", "format": "mm:ss"}}}
	raw, _ := json.Marshal(layout)
	call("", "layout", "--session", "data-cli", "--json", string(raw))
	before := call("", "status", "--session", "data-cli")
	call(`{"seconds":92}`, "data-write", "--file", path, "--json", "-")
	if frame := call("", "tui", "--session", "data-cli", "--plain"); !strings.Contains(frame, "01:32") || !strings.Contains(frame, "External data") {
		t.Fatal(frame)
	}
	input := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(input, []byte(`{"seconds":7}`), 0600); err != nil {
		t.Fatal(err)
	}
	call("", "data-write", "--file", path, "--json", "@"+input)
	if frame := call("", "tui", "--session", "data-cli", "--plain"); !strings.Contains(frame, "00:07") || strings.Contains(frame, "01:32") {
		t.Fatal(frame)
	}
	if after := call("", "status", "--session", "data-cli"); before != after {
		t.Fatal("data publication rewrote native state")
	}
	for _, args := range [][]string{{"data-write"}, {"data-write", "--file", path, "--json", "{"}, {"data-write", "--file", path, "--json", "@" + input + "-missing"}, {"data-write", "--file", path, "--json", "{}", "unexpected"}} {
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err == nil {
			t.Fatal("invalid data-write accepted", args)
		}
	}
}
