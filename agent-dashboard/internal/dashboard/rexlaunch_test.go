package dashboard_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	d "agent-dashboard/internal/dashboard"
	rx "agent-dashboard/internal/rexwatch"
)

// Fake only the documented Rex CLI boundary. Local receipts/OS locks/files and
// the public OpenViewer operation are real, including concurrent launch calls.
type launchCLI struct {
	mu             sync.Mutex
	executable     string
	blocks         []map[string]any
	created        int
	closed         []string
	command        []string
	focused        string
	peerPID        int
	contextBlock   string
	instance       string
	exited         map[string]bool
	foreign        map[string]bool
	timeout        bool
	delayed        bool
	pending        map[string]any
	listError      bool
	breakReconcile bool
}

func launchFixture(t *testing.T) (*launchCLI, rx.Client, d.OpenRequest) {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "viewer with 'quotes'.bin")
	if err := os.WriteFile(exe, []byte("synthetic executable identity fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	f := &launchCLI{executable: exe, focused: "unrelated-focus", peerPID: 777, contextBlock: "anchor", instance: "server-one", exited: map[string]bool{}, foreign: map[string]bool{}}
	f.blocks = []map[string]any{{"block_id": "anchor", "label": "Original work", "host_id": "host-local", "window_id": "window-one", "placement": "placed", "creator_name": "com.superlogical.terminal"}}
	return f, rx.Client{Binary: "rex", Exec: f}, d.OpenRequest{Root: t.TempDir(), Executable: exe, Args: []string{"tui", "--session", "actual session 'quoted'"}, SessionID: "rex-session", BlockID: "anchor", Presentation: "auto"}
}
func (*launchCLI) Stream(context.Context, []string, func([]byte) error) error {
	return errors.New("unexpected observer stream")
}
func (f *launchCLI) Output(ctx context.Context, args []string) (rx.Result, error) {
	if err := ctx.Err(); err != nil {
		return rx.Result{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	output := func(v any) (rx.Result, error) {
		raw, err := json.Marshal(v)
		return rx.Result{Stdout: raw, PID: 777}, err
	}
	get := func(flag string) string {
		for i, arg := range args {
			if arg == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	has := func(value string) bool {
		for _, arg := range args {
			if arg == value {
				return true
			}
		}
		return false
	}
	if has("whoami") {
		return output(map[string]any{"host_id": "host-local", "instance_id": f.instance, "endpoint": "unix:///local.sock", "session_id": "rex-session", "block_id": f.contextBlock, "principal": map[string]any{"pid": f.peerPID, "transport": "unix"}})
	}
	if has("split") {
		f.created++
		id := fmt.Sprintf("viewer-%d", f.created)
		block := map[string]any{"block_id": id, "label": get("--label"), "host_id": "host-local", "window_id": "window-one", "placement": "placed", "creator_name": "com.superlogical.terminal"}
		for i, arg := range args {
			if arg == "--" {
				f.command = append([]string(nil), args[i+1:]...)
				break
			}
		}
		if !has("--focus=false") {
			f.focused = id
		}
		if !has("--shell=none") || !has("--split=right") {
			return rx.Result{}, errors.New("unsafe viewer launch")
		}
		if f.delayed {
			f.pending = block
		} else {
			f.blocks = append(f.blocks, block)
		}
		if f.timeout || f.delayed {
			if f.breakReconcile {
				f.listError = true
			}
			return rx.Result{}, context.DeadlineExceeded
		}
		return output(map[string]any{"block_ids": []string{id}})
	}
	if has("ls") {
		if f.listError {
			return rx.Result{}, errors.New("synthetic discovery unavailable")
		}
		return output(map[string]any{"blocks": f.blocks})
	}
	if has("process") {
		id := get("-b")
		if f.exited[id] {
			return output(map[string]any{"child": nil, "last_exit": map[string]any{"exit_code": 0}})
		}
		path := f.executable
		if f.foreign[id] {
			path = filepath.Join(filepath.Dir(path), "not-the-viewer")
		}
		return output(map[string]any{"child": map[string]any{"path": path}, "last_exit": nil})
	}
	if has("close") {
		id := args[len(args)-1]
		for i, b := range f.blocks {
			if b["block_id"] == id {
				f.blocks = append(f.blocks[:i], f.blocks[i+1:]...)
				break
			}
		}
		f.closed = append(f.closed, id)
		return rx.Result{}, nil // block close intentionally has no JSON output
	}
	return rx.Result{}, fmt.Errorf("unexpected CLI request: %s", strings.Join(args, " "))
}

func TestOpenViewerCreatesRightPaneKeepsFocusAndReusesWithoutSendingInput(t *testing.T) {
	f, client, request := launchFixture(t)
	first, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Mode != "rex" || first.Reused || first.BlockID != "viewer-1" || f.focused != "unrelated-focus" {
		t.Fatalf("unexpected first launch: %+v focus %q", first, f.focused)
	}
	want := append([]string{request.Executable}, request.Args...)
	if !reflect.DeepEqual(f.command, want) {
		t.Fatalf("command arguments passed through a shell or lost quoting: %q", f.command)
	}
	second, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused || second.BlockID != first.BlockID || f.created != 1 || len(f.closed) != 0 {
		t.Fatal("idempotent open changed existing work")
	}
	if _, err := os.Stat(filepath.Join(request.Root, "rex-launch")); err != nil {
		t.Fatal("private launch receipts were not created")
	}
}

func TestConcurrentOpensCreateOnlyOneViewer(t *testing.T) {
	f, client, request := launchFixture(t)
	var workers sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := d.OpenViewer(context.Background(), client, request)
			if err == nil && result.BlockID != "viewer-1" {
				err = errors.New("different viewer returned")
			}
			results <- err
		}()
	}
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.created != 1 || len(f.blocks) != 2 {
		t.Fatal("concurrent launch duplicated the right pane")
	}
}

func TestOpenViewerReconcilesAmbiguousRPCSuccessBeforeRetry(t *testing.T) {
	f, client, request := launchFixture(t)
	f.timeout = true
	first, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.BlockID != second.BlockID || !second.Reused || f.created != 1 {
		t.Fatal("timeout caused duplicate creation")
	}
}

func TestUnconfirmedSplitReceiptPreventsDelayedDuplicateCreation(t *testing.T) {
	f, client, request := launchFixture(t)
	f.delayed = true
	if _, err := d.OpenViewer(context.Background(), client, request); err == nil {
		t.Fatal("unconfirmed creation reported as opened")
	}
	if _, err := d.OpenViewer(context.Background(), client, request); err == nil || !strings.Contains(err.Error(), "unconfirmed outcome") {
		t.Fatal("pending operation did not block an unsafe retry", err)
	}
	if f.created != 1 {
		t.Fatal("a still-pending server request was duplicated")
	}
	f.blocks = append(f.blocks, f.pending)
	f.pending = nil
	result, err := d.OpenViewer(context.Background(), client, request)
	if err != nil || result.BlockID != "viewer-1" || !result.Reused || f.created != 1 {
		t.Fatal("late completed split was not reconciled", result, err)
	}
}

func TestOnlyAnExactlyOwnedExitedViewerCanBeReplaced(t *testing.T) {
	f, client, request := launchFixture(t)
	first, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	f.exited[first.BlockID] = true
	second, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.BlockID != "viewer-2" || second.Reused || len(f.closed) != 1 || f.closed[0] != first.BlockID || len(f.blocks) != 2 {
		t.Fatal("exited own viewer replacement changed unrelated panes")
	}
	f.foreign[second.BlockID] = true
	if _, err := d.OpenViewer(context.Background(), client, request); err == nil || len(f.closed) != 1 || f.created != 2 {
		t.Fatal("a label was trusted instead of verifying its running program")
	}
}

func TestOpenViewerNeverExecutesLocalPathsOnUnverifiedOrStaleHosts(t *testing.T) {
	for _, kind := range []string{"no-context", "nonlocal-peer", "stale-context", "remote-anchor", "detached-anchor", "unavailable"} {
		t.Run(kind, func(t *testing.T) {
			f, client, request := launchFixture(t)
			switch kind {
			case "no-context":
				request.BlockID = ""
			case "nonlocal-peer":
				f.peerPID = 999
			case "stale-context":
				f.contextBlock = "not-the-anchor"
			case "remote-anchor":
				f.blocks[0]["host_id"] = "unapproved-remote-host"
			case "detached-anchor":
				f.blocks[0]["placement"] = "detached"
			case "unavailable":
				f.listError = true
			}
			result, err := d.OpenViewer(context.Background(), client, request)
			if f.created != 0 || len(f.closed) != 0 || result.Mode == "rex" {
				t.Fatal("an unverified target was changed", result, err)
			}
			if err == nil && result.Reason == "" {
				t.Fatal("fallback did not disclose why no pane was opened")
			}
		})
	}
}

func TestTerminalPresentationHasNoRexDependencyAndExplicitRexDoesNotPretendSuccess(t *testing.T) {
	f, client, request := launchFixture(t)
	request.Presentation = "terminal"
	result, err := d.OpenViewer(context.Background(), rx.Client{}, request)
	if err != nil || result.Mode != "terminal" || f.created != 0 {
		t.Fatal("terminal-independent mode tried to require Rex")
	}
	request.Presentation = "rex"
	f.peerPID = 999
	if _, err := d.OpenViewer(context.Background(), client, request); err == nil {
		t.Fatal("explicit Rex mode hid its verification failure")
	}
	request.Presentation = "unexpected"
	if _, err := d.OpenViewer(context.Background(), client, request); err == nil {
		t.Fatal("unsupported presentation accepted")
	}
}

func TestCallerConfigurationChangesDoNotReuseAnotherObservationScope(t *testing.T) {
	f, client, request := launchFixture(t)
	first, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	request.Fingerprint = "different-explicit-selected-hosts"
	second, err := d.OpenViewer(context.Background(), client, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.BlockID == second.BlockID || second.Reused || f.created != 2 || len(f.closed) != 0 {
		t.Fatal("different caller scope reused or destroyed the existing viewer")
	}
}

func TestRexLaunchRequiresAbsoluteReceiptRootAndPreservesAmbiguousErrorCauses(t *testing.T) {
	for _, root := range []string{"", "relative-receipts"} {
		f, client, request := launchFixture(t)
		request.Root = root
		if _, err := d.OpenViewer(context.Background(), client, request); err == nil || f.created != 0 {
			t.Fatal("relative receipt root was accepted")
		}
	}
	f, client, request := launchFixture(t)
	f.timeout, f.breakReconcile = true, true
	if _, err := d.OpenViewer(context.Background(), client, request); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("reconciliation discarded the original uncertain split error", err)
	}
	if f.created != 1 {
		t.Fatal("ambiguous split was retried")
	}
}
