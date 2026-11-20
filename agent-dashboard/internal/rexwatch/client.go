package rexwatch

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:embed observe.lua
var observer string

type Result struct {
	Stdout []byte
	PID    int
}

// Executor is the official CLI process boundary. Stream must stop and reap its
// child on context cancellation or a callback error. Neither method uses a shell.
type Executor interface {
	Output(context.Context, []string) (Result, error)
	Stream(context.Context, []string, func([]byte) error) error
}

type ProcessExecutor struct{}

// Do not embed bytes.Buffer: its promoted ReadFrom would let io.Copy bypass
// Write and therefore bypass the limit when os/exec drains a child pipe.
type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
}

func (b *boundedBuffer) String() string { return b.buffer.String() }
func (b *boundedBuffer) Bytes() []byte  { return b.buffer.Bytes() }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := max(0, b.limit-b.buffer.Len())
	if n > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	_, _ = b.buffer.Write(p)
	return n, nil
}

func (ProcessExecutor) Output(ctx context.Context, args []string) (Result, error) {
	if len(args) == 0 {
		return Result{}, errors.New("missing executable")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdout, stderr := &boundedBuffer{limit: MaxMessage}, &boundedBuffer{limit: 4096}
	cmd.Stdout, cmd.Stderr, cmd.WaitDelay = stdout, stderr, 2*time.Second
	if err := cmd.Start(); err != nil {
		return Result{}, err
	}
	pid := cmd.Process.Pid
	err := cmd.Wait()
	if err != nil {
		return Result{}, fmt.Errorf("Rex CLI: %w: %s", err, stderr.String())
	}
	if stdout.overflow {
		return Result{}, errors.New("Rex response exceeds safety limit")
	}
	return Result{Stdout: stdout.Bytes(), PID: pid}, nil
}

func (ProcessExecutor) Stream(ctx context.Context, args []string, receive func([]byte) error) error {
	if len(args) == 0 {
		return errors.New("missing executable")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stderr := &boundedBuffer{limit: 4096}
	cmd.Stderr, cmd.WaitDelay = stderr, 2*time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64<<10), MaxMessage)
	var readErr error
	for scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		if readErr = receive(scanner.Bytes()); readErr != nil {
			break
		}
	}
	if readErr == nil {
		readErr = scanner.Err()
	}
	if readErr != nil {
		cancel()
	}
	err = cmd.Wait()
	if readErr != nil {
		return readErr
	}
	if err != nil {
		return fmt.Errorf("Rex observer: %w: %s", err, stderr.String())
	}
	return io.EOF
}

type Client struct {
	Binary string
	Exec   Executor
}

func FindClient() (Client, error) {
	path, err := exec.LookPath("rex")
	return Client{Binary: path, Exec: ProcessExecutor{}}, err
}

func (c Client) executor() Executor {
	if c.Exec != nil {
		return c.Exec
	}
	return ProcessExecutor{}
}
func (c Client) args(endpoint string, rest ...string) []string {
	args := []string{c.Binary, "--autostart=false", "--timeout", "5s"}
	if endpoint != "" {
		args = append(args, "--server="+endpoint)
	}
	return append(args, rest...)
}

func (c Client) output(ctx context.Context, endpoint string, args ...string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if c.Binary == "" {
		return Result{}, errors.New("Rex CLI is not installed")
	}
	return c.executor().Output(ctx, c.args(endpoint, args...))
}

func (c Client) Command(ctx context.Context, endpoint string, args ...string) error {
	_, err := c.output(ctx, endpoint, args...)
	return err
}

func (c Client) JSON(ctx context.Context, endpoint string, result any, args ...string) error {
	r, err := c.output(ctx, endpoint, args...)
	if err != nil {
		return err
	}
	return json.Unmarshal(r.Stdout, result)
}

// Identity contains only the metadata needed to pin an observation source or to
// verify an optional local split. Local is verified using IPC peer credentials:
// the Rex server must identify the actual local CLI child process as its peer.
type Identity struct {
	HostID     string `json:"host_id"`
	InstanceID string `json:"instance_id"`
	Endpoint   string `json:"endpoint"`
	SessionID  string `json:"session_id"`
	BlockID    string `json:"block_id"`
	Principal  struct {
		PID       int    `json:"pid"`
		Transport string `json:"transport"`
	} `json:"principal"`
	Local bool `json:"-"`
}

func (c Client) Identity(ctx context.Context, endpoint string) (Identity, error) {
	var id Identity
	r, err := c.output(ctx, endpoint, "whoami", "--json")
	if err != nil {
		return id, err
	}
	if err = json.Unmarshal(r.Stdout, &id); err != nil {
		return id, err
	}
	if !printable(id.HostID, 128, false) || !printable(id.InstanceID, 128, false) || !printable(id.Endpoint, 2048, false) {
		return id, errors.New("Rex returned an incomplete server identity")
	}
	id.Local = r.PID > 0 && id.Principal.PID == r.PID && (id.Principal.Transport == "unix" || id.Principal.Transport == "pipe")
	return id, nil
}

// Resolve reads only requested saved hosts. It pins their reachable explicit
// endpoints, never silently falls back to a local server, and never autostarts.
// Current selects the current host; its concrete session is retained if present.
func (c Client) Resolve(ctx context.Context, current bool, hosts []string) (Config, error) {
	cfg := Config{Version: 1}
	if current {
		id, err := c.Identity(ctx, "")
		if err != nil {
			return cfg, err
		}
		expectedSession, expectedBlock := os.Getenv("REX_SESSION"), os.Getenv("REX_BLOCK")
		if expectedSession != "" && id.SessionID != expectedSession || expectedSession == "" && expectedBlock != "" && id.BlockID != expectedBlock {
			return cfg, errors.New("current Rex context is unavailable; refusing to widen observation to all host sessions")
		}
		s := Source{Name: "current", Endpoint: id.Endpoint, HostID: id.HostID}
		if id.SessionID != "" {
			s.Sessions = []string{id.SessionID}
		}
		cfg.Sources = append(cfg.Sources, s)
	}
	if len(hosts) > MaxSources {
		return cfg, errors.New("too many selected Rex hosts")
	}
	for _, label := range hosts {
		if !printable(label, 80, false) || strings.HasPrefix(label, "-") {
			return cfg, errors.New("invalid saved Rex host label")
		}
		var checks struct {
			Hosts []struct {
				Label    string `json:"label"`
				Endpoint string `json:"endpoint"`
				Status   string `json:"status"`
				Error    string `json:"error"`
			} `json:"hosts"`
		}
		if err := c.JSON(ctx, "", &checks, "hosts", "check", label, "--json"); err != nil {
			return cfg, err
		}
		if len(checks.Hosts) != 1 || checks.Hosts[0].Status != "reachable" || checks.Hosts[0].Endpoint == "" {
			return cfg, fmt.Errorf("selected Rex host %q has no reachable endpoint", label)
		}
		h := checks.Hosts[0]
		id, err := c.Identity(ctx, h.Endpoint)
		if err != nil {
			return cfg, err
		}
		cfg.Sources = append(cfg.Sources, Source{Name: label, Endpoint: h.Endpoint, HostID: id.HostID})
	}
	return cfg, cfg.Validate()
}

func (c Client) watch(ctx context.Context, source Source, receive func([]byte) error) error {
	options, _ := json.Marshal(struct {
		HostID   string   `json:"host_id"`
		Sessions []string `json:"sessions"`
	}{source.HostID, source.Sessions})
	args := []string{"do", "--fail-fast", "-e", observer, "--args", string(options)}
	if len(source.Sessions) == 0 {
		args = append(args, "--all")
	} else {
		args = append(args, "-s", source.Sessions[0])
	}
	return c.executor().Stream(ctx, c.args(source.Endpoint, args...), receive)
}
