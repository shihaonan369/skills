package rexwatch_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	rx "agent-dashboard/internal/rexwatch"
)

// This fake implements the public CLI process boundary, not monitor internals.
// Every callback is acknowledged so tests exercise ordered real consumption.
type input struct {
	raw string
	end error
	ack chan error
}
type connection struct {
	args    []string
	input   chan input
	done    chan struct{}
	receive func([]byte) error
}
type fakeCLI struct{ connections chan *connection }

func newCLI() *fakeCLI { return &fakeCLI{connections: make(chan *connection, 16)} }
func (*fakeCLI) Output(context.Context, []string) (rx.Result, error) {
	return rx.Result{}, errors.New("unexpected nonstreaming call")
}
func (f *fakeCLI) Stream(ctx context.Context, args []string, receive func([]byte) error) error {
	c := &connection{args: append([]string(nil), args...), input: make(chan input), done: make(chan struct{}), receive: receive}
	defer close(c.done)
	select {
	case f.connections <- c:
	case <-ctx.Done():
		return ctx.Err()
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case in := <-c.input:
			if in.end != nil {
				return in.end
			}
			err := receive([]byte(in.raw))
			in.ack <- err
			if err != nil {
				return err
			}
		}
	}
}
func nextConnection(t *testing.T, f *fakeCLI) *connection {
	t.Helper()
	select {
	case c := <-f.connections:
		return c
	case <-time.After(4 * time.Second):
		t.Fatal("no CLI connection")
		return nil
	}
}
func send(t *testing.T, c *connection, raw string) error {
	t.Helper()
	ack := make(chan error, 1)
	select {
	case c.input <- input{raw: raw, ack: ack}:
	case <-c.done:
		t.Fatal("connection already stopped")
	case <-time.After(time.Second):
		t.Fatal("CLI input blocked")
	}
	select {
	case err := <-ack:
		return err
	case <-time.After(time.Second):
		t.Fatal("CLI callback blocked")
		return nil
	}
}
func mustSend(t *testing.T, c *connection, raw string) {
	t.Helper()
	if err := send(t, c, raw); err != nil {
		t.Fatal(err)
	}
}
func await(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("observation did not reach the expected public state")
}
func config(hosts ...string) rx.Config {
	c := rx.Config{Version: 1}
	for _, host := range hosts {
		c.Sources = append(c.Sources, rx.Source{Name: host, HostID: host, Endpoint: "unix:///" + host})
	}
	return c
}
func monitor(t *testing.T, cfg rx.Config) (*rx.Monitor, *fakeCLI) {
	t.Helper()
	f := newCLI()
	m, err := rx.NewMonitor(context.Background(), rx.Client{Binary: "rex", Exec: f}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	return m, f
}
func envelope(host, instance string, seq int, kind, fields string) string {
	if fields != "" {
		fields = "," + fields
	}
	return fmt.Sprintf(`{"agent_dashboard_rex":1,"host_id":%q,"instance_id":%q,"seq":%d,"type":%q%s}`, host, instance, seq, kind, fields)
}
func hello(host string) string { return envelope(host, "instance-old", 1, "hello", "") }

const workingPane = `{"session_id":"session-one","session_label":"Work","block_id":"block-one","block_label":"Terminal","records":[{"state":"working","app":"pi","progress":42,"msg":"Compiling","updated_at":"2026-01-01T00:00:00Z"},{"id":"build/test","state":"blocked","kind":"permission","effective_app":"pi","msg":"Awaiting approval"}]}`
const donePane = `{"session_id":"session-one","session_label":"Work","block_id":"block-one","block_label":"Terminal","records":[{"state":"done","app":"pi","updated_at":"2026-01-01T00:00:01Z"}]}`

func baseline(host string) string {
	return envelope(host, "instance-old", 2, "snapshot", `"panes":[`+workingPane+`]`)
}

func TestProgramReportsReplaceWholePaneWithoutCreatingLifecycleData(t *testing.T) {
	m, f := monitor(t, config("host-a"))
	c := nextConnection(t, f)
	mustSend(t, c, hello("host-a"))
	mustSend(t, c, baseline("host-a"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	s := m.Snapshot().Sources[0]
	if s.Health != "online" || s.Panes != 1 || len(s.Programs) != 2 || s.Programs[0].Record.Progress == nil || *s.Programs[0].Record.Progress != 42 {
		t.Fatalf("unexpected baseline: %+v", s)
	}
	if s.Programs[1].Record.EffectiveApp != "pi" || s.Programs[0].Key == s.Programs[1].Key {
		t.Fatal("root and child reports were conflated")
	}
	mustSend(t, c, envelope("host-a", "instance-old", 3, "pane", `"pane":`+donePane))
	s = m.Snapshot().Sources[0]
	if len(s.Programs) != 1 || s.Programs[0].Record.State != "done" || s.Programs[0].Record.Progress != nil || s.Programs[0].Record.Message != "" || s.Programs[0].Record.Kind != "" {
		t.Fatalf("removed reports/fields survived a replacement: %+v", s)
	}
	empty := strings.Replace(donePane, `[{"state":"done","app":"pi","updated_at":"2026-01-01T00:00:01Z"}]`, `{}`, 1)
	mustSend(t, c, envelope("host-a", "instance-old", 4, "pane", `"pane":`+empty))
	s = m.Snapshot().Sources[0]
	if len(s.Programs) != 0 || s.Unreported != 1 {
		t.Fatal("an empty Lua table was not treated as an unreported pane")
	}
}

func TestSelectedHostsCannotOverwriteEachOthersRecords(t *testing.T) {
	m, f := monitor(t, config("host-a", "host-b"))
	connections := map[string]*connection{}
	for range 2 {
		c := nextConnection(t, f)
		for _, a := range c.args {
			if a == "--server=unix:///host-a" {
				connections["host-a"] = c
			}
			if a == "--server=unix:///host-b" {
				connections["host-b"] = c
			}
		}
	}
	for host, c := range connections {
		mustSend(t, c, hello(host))
		mustSend(t, c, baseline(host))
	}
	s := m.Snapshot()
	if len(s.Sources[0].Programs) != 2 || len(s.Sources[1].Programs) != 2 || s.Sources[0].Programs[0].Key == s.Sources[1].Programs[0].Key {
		t.Fatal("same pane/record IDs collided across hosts")
	}
	if err := send(t, connections["host-a"], envelope("host-b", "instance-old", 3, "pane", `"pane":`+donePane)); err == nil {
		t.Fatal("a substituted host was accepted")
	}
	await(t, func() bool { return m.Snapshot().Sources[0].Health == "reconnecting" })
	s = m.Snapshot()
	if s.Sources[0].Programs[0].Record.State != "working" || s.Sources[1].Health != "online" || s.Sources[1].Programs[0].Record.State != "working" {
		t.Fatal("a source failure corrupted another source or its own last-known data")
	}
}

func TestReconnectKeepsLastKnownDataUntilAnIndependentNewBaseline(t *testing.T) {
	m, f := monitor(t, config("host-a"))
	first := nextConnection(t, f)
	mustSend(t, first, hello("host-a"))
	mustSend(t, first, baseline("host-a"))
	first.input <- input{end: errors.New("connection lost")}
	await(t, func() bool { return m.Snapshot().Sources[0].Health == "reconnecting" })
	before := m.Snapshot().Sources[0]
	if before.Programs[0].Record.State != "working" || before.Programs[0].Record.UpdatedAt != "2026-01-01T00:00:00Z" {
		t.Fatal("disconnection changed program facts")
	}
	second := nextConnection(t, f)
	mustSend(t, second, envelope("host-a", "instance-new", 1, "hello", ""))
	pending := m.Snapshot().Sources[0]
	if pending.Health != "syncing" || pending.InstanceID != "instance-old" || len(pending.Programs) != 2 {
		t.Fatal("old data was relabelled as belonging to the new instance")
	}
	mustSend(t, second, envelope("host-a", "instance-new", 2, "snapshot", `"panes":[`+donePane+`]`))
	if err := first.receive([]byte(envelope("host-a", "instance-old", 3, "pane", `"pane":`+workingPane))); err == nil {
		t.Fatal("late old-connection message was not rejected")
	}
	after := m.Snapshot().Sources[0]
	if after.Health != "online" || after.InstanceID != "instance-new" || len(after.Programs) != 1 || after.Programs[0].Record.State != "done" {
		t.Fatalf("new baseline was corrupted: %+v", after)
	}
}

func TestSourceScopeAndWireFailuresNeverPublishUnverifiedReports(t *testing.T) {
	cases := map[string]string{
		"gap":       envelope("host-a", "instance-old", 4, "pane", `"pane":`+donePane),
		"instance":  envelope("host-a", "wrong-instance", 3, "pane", `"pane":`+donePane),
		"scope":     envelope("host-a", "instance-old", 3, "pane", `"pane":`+strings.Replace(donePane, "session-one", "outside-scope", 1)),
		"topology":  envelope("host-a", "instance-old", 3, "pane", `"pane":`+strings.Replace(donePane, "block-one", "unknown-block", 1)),
		"duplicate": envelope("host-a", "instance-old", 3, "snapshot", `"panes":[`+donePane+`,`+donePane+`]`),
		"progress":  envelope("host-a", "instance-old", 3, "pane", `"pane":`+strings.Replace(donePane, `"state":"done"`, `"state":"working","progress":101`, 1)),
		"malformed": `{"type":`,
		"banner":    `listening: a handler`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := config("host-a")
			cfg.Sources[0].Sessions = []string{"session-one"}
			m, f := monitor(t, cfg)
			c := nextConnection(t, f)
			mustSend(t, c, hello("host-a"))
			mustSend(t, c, baseline("host-a"))
			if err := send(t, c, raw); err == nil {
				t.Fatal("unverified data was accepted")
			}
			await(t, func() bool { return m.Snapshot().Sources[0].Health == "reconnecting" })
			s := m.Snapshot().Sources[0]
			if len(s.Programs) != 2 || s.Programs[0].Record.State != "working" || s.Error == "" {
				t.Fatalf("last-known state was not retained: %+v", s)
			}
		})
	}
}

func TestSnapshotsAreDetachedAndCloseReapsAllObservers(t *testing.T) {
	cfg := config("host-a")
	cfg.Sources[0].Sessions = []string{"session-one"}
	m, f := monitor(t, cfg)
	c := nextConnection(t, f)
	cfg.Sources[0].Sessions[0] = "mutated-by-caller"
	mustSend(t, c, hello("host-a"))
	mustSend(t, c, baseline("host-a"))
	s := m.Snapshot()
	*s.Sources[0].Programs[0].Record.Progress = 99
	s.Sources[0].Programs[0].Record.Message = "mutated"
	s.Sources[0].Source.Sessions[0] = "mutated"
	fresh := m.Snapshot().Sources[0]
	if *fresh.Programs[0].Record.Progress != 42 || fresh.Programs[0].Record.Message != "Compiling" || fresh.Source.Sessions[0] != "session-one" {
		t.Fatal("snapshot/config mutation crossed the public boundary")
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				_ = m.Snapshot()
			}
		}()
	}
	wg.Wait()
	m.Close()
	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("observer outlived its owner")
	}
	if m.Snapshot().Sources[0].Health != "stopped" {
		t.Fatal("closed observers still claimed a live connection")
	}
	if err := c.receive([]byte(envelope("host-a", "instance-old", 3, "pane", `"pane":`+donePane))); err == nil {
		t.Fatal("a late callback revived a stopped observer")
	}
	m.Close() // idempotent cleanup
}

func TestSourceConfigurationIsExplicitBoundedAndUnambiguous(t *testing.T) {
	good := `{"version":1,"sources":[{"name":"dev","endpoint":"unix:///test.sock","host_id":"host-a","sessions":["session-one"]}]}`
	if _, err := rx.ParseConfig([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`null`, `{}`, `{"version":1,"sources":[]}`, good + ` {}`, strings.Replace(good, `"version":1`, `"version":1,"unknown":true`, 1), strings.Replace(good, `"host_id":"host-a"`, `"host_id":""`, 1), strings.Replace(good, `"name":"dev"`, `"name":"dev\u001b"`, 1)} {
		if _, err := rx.ParseConfig([]byte(raw)); err == nil {
			t.Fatalf("accepted config %s", raw)
		}
	}
	c := config("host-a", "host-a")
	if c.Validate() == nil {
		t.Fatal("duplicate pinned hosts accepted")
	}
	c = config("host-a", "host-b")
	c.Sources[1].Name = "host-a"
	if c.Validate() == nil {
		t.Fatal("duplicate aliases accepted")
	}
	c = config("host-a")
	c.Sources[0].Sessions = []string{"same", "same"}
	if c.Validate() == nil {
		t.Fatal("duplicate scopes accepted")
	}
	if rx.ProgramID("a/b", "c", "d", "") == rx.ProgramID("a", "b/c", "d", "") {
		t.Fatal("identity tuple boundaries were lost")
	}
	if rx.ProgramID("a", "s", "b", "") != rx.ProgramID("a", "s", "b", "") {
		t.Fatal("source-qualified key is unstable")
	}
}

func TestCLIUsesExplicitEndpointAndSelectedSessionSubscriptions(t *testing.T) {
	cfg := config("host-a")
	cfg.Sources[0].Sessions = []string{"session-one", "session-two"}
	_, f := monitor(t, cfg)
	c := nextConnection(t, f)
	joined := strings.Join(c.args, "\n")
	if !strings.Contains(joined, "--autostart=false") || !strings.Contains(joined, "--server=unix:///host-a") || strings.Contains(joined, "\n--all") {
		t.Fatal("filtered source subscribed beyond its scope")
	}
	foundSession, foundOptions := false, false
	for i, arg := range c.args {
		if arg == "-s" && i+1 < len(c.args) && c.args[i+1] == "session-one" {
			foundSession = true
		}
		if arg == "--args" {
			var opts struct {
				Host     string   `json:"host_id"`
				Sessions []string `json:"sessions"`
			}
			if err := json.Unmarshal([]byte(c.args[i+1]), &opts); err != nil {
				t.Fatal(err)
			}
			foundOptions = opts.Host == "host-a" && len(opts.Sessions) == 2
		}
	}
	if !foundSession || !foundOptions {
		t.Fatal("pinned identity/session arguments were not passed as data")
	}
}

func TestDiscoveryProgressDoesNotPretendThatABaselineIsComplete(t *testing.T) {
	m, f := monitor(t, config("host-a"))
	c := nextConnection(t, f)
	mustSend(t, c, hello("host-a"))
	mustSend(t, c, envelope("host-a", "instance-old", 2, "heartbeat", ""))
	source := m.Snapshot().Sources[0]
	if source.Health != "syncing" || len(source.Programs) != 0 || source.LastContact.IsZero() || !source.LastSnapshot.IsZero() || source.InstanceID != "" {
		t.Fatal("discovery progress fabricated a complete observation baseline")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if !errors.Is(m.Ready(ctx), context.DeadlineExceeded) {
		t.Fatal("progress settled an incomplete source")
	}
	mustSend(t, c, envelope("host-a", "instance-old", 3, "snapshot", `"panes":[`+donePane+`]`))
	if m.Snapshot().Sources[0].Health != "online" {
		t.Fatal("completed baseline was not accepted after discovery progress")
	}
}
