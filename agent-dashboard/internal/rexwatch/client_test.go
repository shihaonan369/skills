package rexwatch_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	rx "agent-dashboard/internal/rexwatch"
)

type outputCLI struct {
	calls   [][]string
	outputs func([]string) (rx.Result, error)
}

func (f *outputCLI) Output(_ context.Context, args []string) (rx.Result, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	return f.outputs(args)
}
func (*outputCLI) Stream(context.Context, []string, func([]byte) error) error {
	return errors.New("unexpected stream")
}
func result(raw string) (rx.Result, error) { return rx.Result{Stdout: []byte(raw), PID: 1234}, nil }

func TestResolvePinsOnlyRequestedReachableEndpoints(t *testing.T) {
	t.Setenv("REX_SESSION", "session-current")
	f := &outputCLI{outputs: func(args []string) (rx.Result, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "hosts check devbox"):
			return result(`{"hosts":[{"label":"devbox","status":"reachable","endpoint":"unix:///remote-proxy.sock"}]}`)
		case strings.Contains(joined, "--server=unix:///remote-proxy.sock"):
			return result(`{"host_id":"host-remote","instance_id":"server-b","endpoint":"unix:///server-private-path.sock"}`)
		case strings.Contains(joined, "whoami"):
			return result(`{"host_id":"host-current","instance_id":"server-a","endpoint":"unix:///current.sock","session_id":"session-current","principal":{"pid":1234,"transport":"unix"}}`)
		default:
			return rx.Result{}, errors.New("unselected host queried")
		}
	}}
	c := rx.Client{Binary: "rex", Exec: f}
	cfg, err := c.Resolve(context.Background(), true, []string{"devbox"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Sources) != 2 || cfg.Sources[0].Sessions[0] != "session-current" || cfg.Sources[1].Endpoint != "unix:///remote-proxy.sock" || cfg.Sources[1].HostID != "host-remote" || len(cfg.Sources[1].Sessions) != 0 {
		t.Fatalf("incorrect explicit source resolution: %+v", cfg)
	}
	if len(f.calls) != 3 {
		t.Fatal("unexpected discovery outside the requested source set")
	}
	for _, args := range f.calls {
		if !strings.Contains(strings.Join(args, " "), "--autostart=false") {
			t.Fatal("discovery could autostart a server")
		}
	}
	id, err := c.Identity(context.Background(), "")
	if err != nil || !id.Local {
		t.Fatal("local IPC peer was not verified")
	}
}

func TestIdentityRequiresActualLocalChildPeerCredentials(t *testing.T) {
	for _, tc := range []struct {
		transport string
		pid       int
		local     bool
	}{{"unix", 1234, true}, {"pipe", 1234, true}, {"unix", 9999, false}, {"tcp", 1234, false}, {"unix", 0, false}} {
		f := &outputCLI{outputs: func([]string) (rx.Result, error) {
			return result(fmt.Sprintf(`{"host_id":"h","instance_id":"i","endpoint":"unix:///s","principal":{"transport":%q,"pid":%d}}`, tc.transport, tc.pid))
		}}
		id, err := (rx.Client{Binary: "rex", Exec: f}).Identity(context.Background(), "")
		if err != nil || id.Local != tc.local {
			t.Fatalf("incorrect local proof for %+v: %+v %v", tc, id, err)
		}
	}
}

func TestUnreachableSourcesDoNotFallBackOrAcquireAnUnpinnedIdentity(t *testing.T) {
	f := &outputCLI{outputs: func([]string) (rx.Result, error) {
		return result(`{"hosts":[{"label":"devbox","status":"not reachable from here","endpoint":""}]}`)
	}}
	if _, err := (rx.Client{Binary: "rex", Exec: f}).Resolve(context.Background(), false, []string{"devbox"}); err == nil {
		t.Fatal("unreachable selected host accepted")
	}
	if len(f.calls) != 1 {
		t.Fatal("a failed remote selection fell back to another server")
	}
	for _, raw := range []string{`{}`, `{"host_id":"h","endpoint":"unix:///s"}`, `{"host_id":"h","instance_id":"i"}`} {
		f := &outputCLI{outputs: func([]string) (rx.Result, error) { return result(raw) }}
		if _, err := (rx.Client{Binary: "rex", Exec: f}).Identity(context.Background(), ""); err == nil {
			t.Fatal("incomplete identity accepted")
		}
	}
}

// This child is a real native process, not a shell, Python, or the Rex server.
func TestRexCLIHelperProcess(t *testing.T) {
	index := -1
	for i, arg := range os.Args {
		if arg == "--" {
			index = i
			break
		}
	}
	if index < 0 || index+1 >= len(os.Args) {
		t.Skip("native subprocess helper entry point, exercised by process contract tests")
	}
	switch os.Args[index+1] {
	case "output":
		fmt.Printf(`{"pid":%d}`, os.Getpid())
	case "oversize":
		for range 130 {
			fmt.Print(strings.Repeat("x", 64<<10))
		}
	case "stream":
		fmt.Println(`{"ready":true}`)
		time.Sleep(5 * time.Minute)
	case "fail":
		fmt.Fprint(os.Stderr, strings.Repeat("failure", 1000))
		os.Exit(7)
	default:
		os.Exit(8)
	}
	os.Exit(0)
}
func helper(t *testing.T, mode string) []string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return []string{exe, "-test.run=^TestRexCLIHelperProcess$", "--", mode}
}

func TestNativeCLIProcessExecutionIsBoundedCancellableAndReaped(t *testing.T) {
	executor := rx.ProcessExecutor{}
	r, err := executor.Output(context.Background(), helper(t, "output"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		PID int `json:"pid"`
	}
	if json.Unmarshal(r.Stdout, &decoded) != nil || decoded.PID != r.PID || r.PID <= 0 {
		t.Fatal("CLI process identity was not captured")
	}
	if _, err = executor.Output(context.Background(), helper(t, "oversize")); err == nil {
		t.Fatal("oversize CLI output accepted")
	}
	_, err = executor.Output(context.Background(), helper(t, "fail"))
	if err == nil || len(err.Error()) > 4400 {
		t.Fatal("stderr was not bounded")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	received := false
	err = executor.Stream(ctx, helper(t, "stream"), func(raw []byte) error { received = string(raw) == `{"ready":true}`; cancel(); return nil })
	if !received || err == nil || time.Since(start) > 4*time.Second {
		t.Fatal("stream process was not cancelled and reaped promptly")
	}
	if err = executor.Stream(context.Background(), helper(t, "oversize"), func([]byte) error { t.Fatal("oversize wire line reached the consumer"); return nil }); err == nil {
		t.Fatal("oversize stream line accepted")
	}
	callbackErr := errors.New("consumer rejected protocol")
	if err = executor.Stream(context.Background(), helper(t, "stream"), func([]byte) error { return callbackErr }); !errors.Is(err, callbackErr) {
		t.Fatal("consumer error did not cancel/reap the child")
	}
}

func TestStaleCurrentContextCannotSilentlyWidenObservationScope(t *testing.T) {
	for _, kind := range []string{"session", "block"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("REX_SESSION", "")
			t.Setenv("REX_BLOCK", "")
			if kind == "session" {
				t.Setenv("REX_SESSION", "stale-session")
			} else {
				t.Setenv("REX_BLOCK", "stale-block")
			}
			f := &outputCLI{outputs: func([]string) (rx.Result, error) {
				return result(`{"host_id":"h","instance_id":"i","endpoint":"unix:///current.sock"}`)
			}}
			if _, err := (rx.Client{Binary: "rex", Exec: f}).Resolve(context.Background(), true, nil); err == nil {
				t.Fatal("stale context became permission to observe all sessions")
			}
			if len(f.calls) != 1 {
				t.Fatal("stale context queried additional sources")
			}
		})
	}
}
