package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	d "agent-dashboard/internal/dashboard"
	rx "agent-dashboard/internal/rexwatch"
)

type hostFlags []string

func (h *hostFlags) String() string { return strings.Join(*h, ",") }
func (h *hostFlags) Set(value string) error {
	if len(*h) >= rx.MaxSources {
		return errors.New("at most 16 explicit Rex hosts")
	}
	*h = append(*h, value)
	return nil
}

func readReference(ref string, limit int) ([]byte, string, error) {
	if !strings.HasPrefix(ref, "@") || len(ref) < 2 {
		return nil, "", errors.New("use @file for a private caller-owned configuration")
	}
	path, err := filepath.Abs(ref[1:])
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, int64(limit+1)))
	if err != nil {
		return nil, "", err
	}
	if len(raw) > limit {
		return nil, "", errors.New("configuration exceeds safety limit")
	}
	return raw, "@" + path, nil
}

func runRex(command string, args []string, out io.Writer) error {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(out)
	var opts struct {
		Sources, Layout, Session, Presentation, RexBinary string
		Plain, Status, Current                            bool
		Width, Height                                     int
		Hosts                                             hostFlags
	}
	f.StringVar(&opts.RexBinary, "rex-bin", "", "absolute local Rex CLI path (optional)")
	if command == "rex-sources" {
		f.BoolVar(&opts.Current, "current", true, "include the current Rex host/session")
		f.Var(&opts.Hosts, "host", "explicit saved Rex host label; repeat to select hosts")
	} else {
		f.StringVar(&opts.Sources, "sources", "", "explicit pinned Rex sources as @file")
		f.StringVar(&opts.Layout, "layout", "", "caller-owned panel composition as @file")
		if command == "open" {
			f.StringVar(&opts.Session, "session", "", "concrete Claude lifecycle session")
			f.StringVar(&opts.Presentation, "presentation", "auto", "auto, rex or terminal")
		} else {
			f.BoolVar(&opts.Plain, "plain", false, "print one clipped program-status frame")
			f.BoolVar(&opts.Status, "status", false, "print a program-status snapshot as JSON")
			f.IntVar(&opts.Width, "width", 100, "plain frame width")
			f.IntVar(&opts.Height, "height", 32, "plain frame height")
		}
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if opts.Plain && opts.Status {
		return errors.New("--plain and --status are mutually exclusive")
	}
	client, _ := rx.FindClient()
	if opts.RexBinary != "" {
		if !filepath.IsAbs(opts.RexBinary) {
			return errors.New("--rex-bin must be an absolute local path")
		}
		client = rx.Client{Binary: opts.RexBinary}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	encode := func(value any) error { enc := json.NewEncoder(out); enc.SetIndent("", "  "); return enc.Encode(value) }
	if command == "rex-sources" {
		cfg, err := client.Resolve(ctx, opts.Current, opts.Hosts)
		if err != nil {
			return err
		}
		return encode(cfg)
	}
	var cfg rx.Config
	var raw []byte
	var err error
	if opts.Sources != "" {
		raw, opts.Sources, err = readReference(opts.Sources, 256<<10)
		if err != nil {
			return err
		}
		cfg, err = rx.ParseConfig(raw)
		if err != nil {
			return err
		}
	}
	var composition *d.Layout
	var layoutRaw []byte
	if opts.Layout != "" {
		layoutRaw, opts.Layout, err = readReference(opts.Layout, 256<<10)
		if err != nil {
			return err
		}
		composition, err = d.ParseLayout(layoutRaw)
		if err != nil {
			return err
		}
	}
	if command == "open" {
		if (opts.Sources == "") == (opts.Session == "") {
			return errors.New("open requires exactly one of --session or --sources")
		}
		root, err := d.Root()
		if err != nil {
			return err
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return err
		}
		viewerArgs := []string{"tui", "--session", opts.Session, "--state-dir", root}
		if opts.Sources != "" {
			viewerArgs = []string{"rex-watch", "--sources", opts.Sources}
			if opts.Layout != "" {
				viewerArgs = append(viewerArgs, "--layout", opts.Layout)
			}
			if client.Binary != "" {
				viewerArgs = append(viewerArgs, "--rex-bin", client.Binary)
			}
		} else {
			if opts.Layout != "" {
				return errors.New("Claude layouts are stored with layout --session ID --json @file")
			}
			if _, err = d.Snapshot(root, opts.Session); err != nil {
				return err
			}
		}
		h := sha256.New()
		_, _ = h.Write(raw)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(layoutRaw)
		result, err := d.OpenViewer(ctx, client, d.OpenRequest{Root: root, Executable: exe, Args: viewerArgs, SessionID: os.Getenv("REX_SESSION"), BlockID: os.Getenv("REX_BLOCK"), Presentation: opts.Presentation, Fingerprint: hex.EncodeToString(h.Sum(nil))})
		if err != nil {
			return err
		}
		return encode(result)
	}
	if opts.Sources == "" {
		return errors.New("rex-watch requires --sources @file; hosts are never selected implicitly")
	}
	monitor, err := rx.NewMonitor(ctx, client, cfg)
	if err != nil {
		return err
	}
	defer monitor.Close()
	if opts.Plain || opts.Status {
		initial, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		readyErr := monitor.Ready(initial)
		snapshot := monitor.Snapshot()
		if opts.Status {
			err = encode(snapshot)
		} else {
			state, prepareErr := d.PreparePresentation(d.ProgramState(snapshot, composition))
			if prepareErr != nil {
				return prepareErr
			}
			_, err = fmt.Fprintln(out, (d.Viewer{State: state, Width: opts.Width, Height: opts.Height}).View())
		}
		if err != nil {
			return err
		}
		if readyErr != nil {
			return readyErr
		}
		for _, source := range snapshot.Sources {
			if source.Health != "online" {
				return errors.New("one or more Rex sources are unavailable; partial observations are explicitly marked")
			}
		}
		return nil
	}
	return d.RunProgramViewer(monitor, composition)
}
