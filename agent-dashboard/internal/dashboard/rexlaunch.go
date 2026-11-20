package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	rx "agent-dashboard/internal/rexwatch"
	"github.com/gofrs/flock"
)

type OpenRequest struct {
	Root         string
	Executable   string
	Args         []string
	SessionID    string
	BlockID      string
	Presentation string // auto, rex, or terminal
	Fingerprint  string // caller configuration content, not a transcript
}

type OpenResult struct {
	Mode    string   `json:"mode"`
	BlockID string   `json:"block_id,omitempty"`
	Reused  bool     `json:"reused"`
	Args    []string `json:"args"`
	Reason  string   `json:"reason,omitempty"`
}

type launchRecord struct {
	HostID     string `json:"host_id"`
	SessionID  string `json:"session_id"`
	BlockID    string `json:"block_id"`
	Label      string `json:"label"`
	Signature  string `json:"signature"`
	InstanceID string `json:"instance_id,omitempty"`
	Pending    bool   `json:"pending,omitempty"`
}

type rexBlock struct {
	ID        string `json:"block_id"`
	Label     string `json:"label"`
	HostID    string `json:"host_id"`
	WindowID  string `json:"window_id"`
	Creator   string `json:"creator_name"`
	Placement string `json:"placement"`
}

func launchSignature(request OpenRequest, host string) (string, error) {
	f, err := os.Open(request.Executable)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	// A replaced binary is a different launch even when its path is unchanged.
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	raw, _ := json.Marshal([]any{request.Executable, request.Args, request.SessionID, request.BlockID, host, filepath.Clean(request.Root), request.Fingerprint})
	_, _ = h.Write(raw)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// OpenViewer optionally creates or reuses one owned right-hand Rex pane. A
// verified local IPC peer, concrete live anchor and same-host placement are
// prerequisites. No terminal content is read and no unrelated pane is changed.
func OpenViewer(ctx context.Context, client rx.Client, request OpenRequest) (OpenResult, error) {
	result := OpenResult{Mode: "terminal", Args: append([]string{request.Executable}, request.Args...)}
	fallback := func(err error) (OpenResult, error) {
		if request.Presentation == "rex" {
			return result, err
		}
		result.Reason = Clean(err.Error(), 400)
		return result, nil
	}
	if request.Presentation != "auto" && request.Presentation != "rex" && request.Presentation != "terminal" {
		return result, errors.New("presentation must be auto, rex or terminal")
	}
	if !filepath.IsAbs(request.Executable) {
		return result, errors.New("viewer executable must be an absolute local path")
	}
	if request.Presentation == "terminal" {
		return result, nil
	}
	if request.SessionID == "" || request.BlockID == "" {
		return fallback(errors.New("no concrete Rex session/block context; use the terminal-independent viewer"))
	}
	identity, err := client.Identity(ctx, "")
	if err != nil {
		return fallback(err)
	}
	if !identity.Local {
		return fallback(errors.New("cannot verify that the Rex server shares this machine and account; refusing to execute a local path remotely"))
	}
	if identity.SessionID != request.SessionID || identity.BlockID != request.BlockID {
		return fallback(errors.New("Rex context is stale or belongs to a different endpoint"))
	}
	if !filepath.IsAbs(request.Root) {
		return result, errors.New("Rex launch receipts require an absolute private state root")
	}
	signature, err := launchSignature(request, identity.HostID)
	if err != nil {
		return result, err
	}
	label := "agent-dashboard-" + signature[:24]
	dir := filepath.Join(request.Root, "rex-launch")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return result, err
	}
	lock := flock.New(filepath.Join(dir, signature+".lock"), flock.SetPermissions(0600))
	lockContext, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ok, err := lock.TryLockContext(lockContext, 20*time.Millisecond)
	if err != nil {
		return result, err
	}
	if !ok {
		return result, errors.New("another dashboard launch is in progress")
	}
	defer lock.Close()
	path := filepath.Join(dir, signature+".json")
	var owned launchRecord
	if raw, readErr := os.ReadFile(path); readErr == nil {
		if len(raw) > 4096 || json.Unmarshal(raw, &owned) != nil {
			return result, errors.New("invalid local Rex launch receipt")
		}
	} else if !os.IsNotExist(readErr) {
		return result, readErr
	}
	list := func() ([]rexBlock, error) {
		var blocks struct {
			Blocks []rexBlock `json:"blocks"`
		}
		err := client.JSON(ctx, identity.Endpoint, &blocks, "block", "ls", "-s", request.SessionID, "--json")
		return blocks.Blocks, err
	}
	verify := func(blocks []rexBlock) (string, error) {
		var anchor *rexBlock
		for i := range blocks {
			if blocks[i].ID == request.BlockID {
				anchor = &blocks[i]
			}
		}
		if anchor == nil || anchor.HostID != identity.HostID || anchor.Placement != "placed" || anchor.WindowID == "" {
			return "", errors.New("Rex anchor is missing, detached, or on an unverified host")
		}
		matches := []rexBlock{}
		for _, b := range blocks {
			if b.Label == label {
				matches = append(matches, b)
			}
		}
		if len(matches) > 1 {
			return "", errors.New("ambiguous dashboard label; refusing to create another pane")
		}
		if len(matches) == 0 {
			return "", nil
		}
		b := matches[0]
		if b.ID == request.BlockID || b.HostID != identity.HostID || b.WindowID != anchor.WindowID || b.Placement != "placed" || b.Creator != "com.superlogical.terminal" {
			return "", errors.New("dashboard label belongs to a different placement; refusing to reuse or replace it")
		}
		var process struct {
			Child *struct {
				Path string `json:"path"`
			} `json:"child"`
			LastExit json.RawMessage `json:"last_exit"`
		}
		if err := client.JSON(ctx, identity.Endpoint, &process, "block", "call", "-s", request.SessionID, "-b", b.ID, "process"); err != nil {
			return "", err
		}
		if process.Child != nil && (len(process.LastExit) == 0 || string(process.LastExit) == "null") {
			actual, e1 := os.Stat(process.Child.Path)
			expected, e2 := os.Stat(request.Executable)
			if e1 == nil && e2 == nil && os.SameFile(actual, expected) {
				return b.ID, nil
			}
			return "", errors.New("dashboard label is occupied by another program; no input was sent")
		}
		// Only a local, exact launch receipt authorizes removing an exited pane.
		if owned.HostID == identity.HostID && owned.SessionID == request.SessionID && owned.BlockID == b.ID && owned.Label == label && owned.Signature == signature && len(process.LastExit) > 0 && string(process.LastExit) != "null" {
			if err := client.Command(ctx, identity.Endpoint, "block", "close", "-s", request.SessionID, b.ID); err != nil {
				return "", err
			}
			owned.Pending = false
			raw, _ := json.Marshal(owned)
			if err := atomicWrite(path, raw, 0600); err != nil {
				return "", err
			}
			return "", nil
		}
		return "", errors.New("dashboard pane is not a verified running viewer; inspect it before retrying")
	}
	blocks, err := list()
	if err != nil {
		return fallback(err)
	}
	block, err := verify(blocks)
	if err != nil {
		return result, err
	}
	reused := block != ""
	if block == "" {
		if owned.Pending && owned.InstanceID == identity.InstanceID {
			return result, errors.New("a previous Rex split has an unconfirmed outcome; no duplicate was created. Inspect the server and reconcile this launch receipt before authorizing another split")
		}
		owned = launchRecord{HostID: identity.HostID, SessionID: request.SessionID, Label: label, Signature: signature, InstanceID: identity.InstanceID, Pending: true}
		pending, _ := json.Marshal(owned)
		if err := atomicWrite(path, pending, 0600); err != nil {
			return result, err
		}
		var created struct {
			BlockIDs []string `json:"block_ids"`
		}
		args := []string{"split", "-s", request.SessionID, "-b", request.BlockID, "--split=right", "--ratio", "42", "--focus=false", "--shell=none", "--label", label, "--json", "--", request.Executable}
		args = append(args, request.Args...)
		err = client.JSON(ctx, identity.Endpoint, &created, args...)
		// Always reconcile a timeout/ambiguous response before any future retry.
		if err != nil || len(created.BlockIDs) != 1 {
			if err == nil {
				err = errors.New("Rex split response did not contain exactly one block ID")
			}
			blocks, readErr := list()
			if readErr != nil {
				return result, fmt.Errorf("dashboard split outcome unknown; inspect before retrying: %w", errors.Join(err, fmt.Errorf("reconcile: %w", readErr)))
			}
			block, readErr = verify(blocks)
			if readErr != nil {
				return result, readErr
			}
			if block == "" {
				return result, fmt.Errorf("dashboard split did not produce a verified viewer: %w", err)
			}
		} else {
			block = created.BlockIDs[0]
			owned.BlockID = block
			pending, _ := json.Marshal(owned)
			if err := atomicWrite(path, pending, 0600); err != nil {
				return result, err
			}
			// Confirm placement and program, rather than treating an RPC success as
			// proof that the requested viewer is actually running.
			blocks, readErr := list()
			if readErr != nil {
				return result, readErr
			}
			verified, readErr := verify(blocks)
			if readErr != nil {
				return result, readErr
			}
			if verified != block {
				return result, errors.New("new dashboard pane could not be verified; inspect before retrying")
			}
		}
	}
	receipt := launchRecord{HostID: identity.HostID, SessionID: request.SessionID, BlockID: block, Label: label, Signature: signature, InstanceID: identity.InstanceID}
	raw, _ := json.Marshal(receipt)
	if err = atomicWrite(path, raw, 0600); err != nil {
		return result, err
	}
	result.Mode, result.BlockID, result.Reused = "rex", block, reused
	return result, nil
}
