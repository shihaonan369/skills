package rexwatch_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	rx "agent-dashboard/internal/rexwatch"
)

// Opt-in: execute the embedded observer in the installed OFFICIAL Lua runtime.
// All API functions below are replaced locally, so no synthetic report is ever
// submitted to Rex and no remote program, file, hook, or setting is changed.
// Ordinary CI uses the portable protocol/process tests without requiring Rex.
type luaFixtureCLI struct {
	messages chan []byte
	finished chan error
}

func (*luaFixtureCLI) Output(context.Context, []string) (rx.Result, error) {
	return rx.Result{}, errors.New("unexpected fixture command")
}
func (f *luaFixtureCLI) Stream(ctx context.Context, args []string, receive func([]byte) error) error {
	args = append([]string(nil), args...)
	for i, arg := range args {
		if arg == "-e" {
			args[i+1] = fixturePrefix + args[i+1] + fixtureSuffix
		}
	}
	err := (rx.ProcessExecutor{}).Stream(ctx, args, func(raw []byte) error {
		if err := receive(raw); err != nil {
			return err
		}
		f.messages <- append([]byte(nil), raw...)
		return nil
	})
	f.finished <- err
	return err
}

const fixturePrefix = `
local fixture = {handlers = {}, records = {{state="done", updated_at="2026-01-01T00:00:00Z", owner={name="excluded-owner",pid=9}}}, calls=0, attaches=0}
local allowed = rex.args.sessions[1]
local host = rex.args.host_id
rex.call = function(method, args)
  if method == "server.status" then return {host_id=host, instance_id="fixture-instance"} end
  if method == "session.list" then return {sessions={{session_id=allowed,label="Synthetic fixture"},{session_id="not-selected",label="Do not query"}}} end
  if method == "session.list_blocks" then
    assert(args.session_id == allowed, "queried an unselected session")
    return {blocks={{block_id="fixture-block",label="Synthetic",host_id=host,creator_name="com.superlogical.terminal"}, {block_id="foreign-block",host_id="unapproved-host",creator_name="com.superlogical.terminal"}}}
  end
  if method == "com.superlogical.terminal.program_status" then
    assert(args.session_id == allowed and args.block_id == "fixture-block", "queried a foreign or unverified pane")
    fixture.calls = fixture.calls+1
    return {records=fixture.records}
  end
  error("observer called an unexpected or non-read-only API: "..method,0)
end
rex.session.attach = function(args)
  assert(args.session_id==allowed, "attached outside the selected scope")
  fixture.attaches=fixture.attaches+1
  return {}
end
rex.on = function(name,fn) fixture.handlers[name]=fn end
rex.every = function() return {} end
`
const fixtureSuffix = `
local ctx = {session_id=allowed,block_id="fixture-block"}
-- A queued change older than the baseline MUST NOT roll done back to working.
fixture.handlers["terminal.program_status_changed"](ctx,{record={state="working"}})
fixture.records={}
fixture.handlers["terminal.program_status_removed"](ctx,{ids={""}})
fixture.records={{state="blocked",kind="permission",progress=0,msg="Approve compile",updated_at="2026-01-01T00:00:01Z"}}
fixture.handlers["terminal.program_status_changed"](ctx,{})
fixture.records={{state="done",updated_at="2026-01-01T00:00:02Z"}}
fixture.handlers["terminal.program_status_changed"](ctx,{})
fixture.handlers["terminal.program_status_changed"]({session_id="not-selected",block_id="foreign-block"},{})
assert(fixture.calls==5,"unexpected report read count")
assert(fixture.attaches==1,"repeated session attachment")
-- Server event contexts need not contain a session_id. Reconcile safely and
-- keep attachment bookkeeping bounded by the current selected-session set.
fixture.handlers["session_destroyed"]({}, {})
assert(fixture.calls==6 and fixture.attaches==1,"invalid server event context handling")
`

func TestOfficialLuaObserverHandlesSnapshotRacesRemovalAndPrivacy(t *testing.T) {
	binary := os.Getenv("AGENT_DASHBOARD_TEST_REX")
	if binary == "" {
		t.Skip("set AGENT_DASHBOARD_TEST_REX to an installed Rex CLI for the read-only Lua contract test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := rx.Client{Binary: binary}
	id, err := client.Identity(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if id.SessionID == "" {
		t.Skip("a concrete existing Rex session is required; this test never creates one")
	}
	f := &luaFixtureCLI{messages: make(chan []byte, 16), finished: make(chan error, 16)}
	cfg := rx.Config{Version: 1, Sources: []rx.Source{{Name: "Synthetic local Lua contract only", HostID: id.HostID, Endpoint: id.Endpoint, Sessions: []string{id.SessionID}}}}
	m, err := rx.NewMonitor(ctx, rx.Client{Binary: binary, Exec: f}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	messages := [][]byte{}
	finished := f.finished
	completed := false
	for len(messages) < 7 {
		select {
		case raw := <-f.messages:
			var header struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &header); err != nil {
				t.Fatal(err)
			}
			if header.Type != "heartbeat" {
				messages = append(messages, raw)
			}
		case err := <-finished:
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
			completed, finished = true, nil
			if len(messages)+len(f.messages) < 7 {
				t.Fatal("Lua fixture stopped before emitting its complete contract")
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if !completed {
		select {
		case err := <-finished:
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	var captured []struct {
		Type     string `json:"type"`
		Excluded int    `json:"excluded_panes"`
		Pane     struct {
			Records json.RawMessage `json:"records"`
		} `json:"pane"`
	}
	for _, raw := range messages {
		if strings.Contains(string(raw), "excluded-owner") || strings.Contains(string(raw), `"owner"`) {
			t.Fatal("process-owner metadata escaped the observer")
		}
		var message struct {
			Type     string `json:"type"`
			Excluded int    `json:"excluded_panes"`
			Pane     struct {
				Records json.RawMessage `json:"records"`
			} `json:"pane"`
		}
		if err := json.Unmarshal(raw, &message); err != nil {
			t.Fatal(err)
		}
		captured = append(captured, message)
	}
	if captured[1].Type != "snapshot" || captured[1].Excluded != 1 {
		t.Fatal("foreign panes were not explicitly excluded")
	}
	if !strings.Contains(string(captured[2].Pane.Records), `"state":"done"`) {
		t.Fatal("queued stale event rolled the baseline backward")
	}
	if string(captured[3].Pane.Records) != "{}" {
		t.Fatal("removed records were retained")
	}
	if !strings.Contains(string(captured[4].Pane.Records), `"msg":"Approve compile"`) || !strings.Contains(string(captured[4].Pane.Records), `"progress":0`) {
		t.Fatal("decoded API message/known zero progress was lost")
	}
	last := m.Snapshot().Sources[0]
	if len(last.Programs) != 1 || last.Programs[0].Record.State != "done" || last.Programs[0].Record.Progress != nil || last.Programs[0].Record.Message != "" || last.ExcludedPanes != 1 {
		t.Fatalf("native Lua adapter did not replace complete reports: %+v", last)
	}
}
