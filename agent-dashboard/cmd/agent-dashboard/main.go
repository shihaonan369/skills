package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"agent-dashboard/internal/buildinfo"
	d "agent-dashboard/internal/dashboard"
)

func main() {
	if e := run(os.Args[1:], os.Stdin, os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, "agent-dashboard:", e)
		os.Exit(1)
	}
}
func help(w io.Writer) {
	fmt.Fprintln(w, `Agent Dashboard · composable local observer
Commands:
  install / uninstall   --skill-dir PATH [--settings PATH] [--skills-dir PATH]
  doctor                [--settings PATH] (configuration only; not hook liveness)
  start                 --session ID --title TEXT
  layout                --session ID --json @layout.json (caller-defined panels)
  plan                  --session ID --json '[{...}]' (or @file)
  note                  --session ID --text TEXT [--actor ID]
  finish                --session ID --summary TEXT
  deactivate            --session ID (collection off, keeps snapshot)
  status                --session ID (JSON)
  tui                   --session ID [--plain] [--width 100 --height 32]
  open                  --session ID | --sources @file [--layout @file]
                        [--presentation auto|rex|terminal]
  rex-sources           [--current=false] [--host LABEL (repeat)]
  rex-watch             --sources @file [--layout @file] [--plain | --status]
  capabilities          (machine-readable optional feature discovery)
  demo                  [--json @layout.json] (synthetic isolated preview)
  hook                  (internal, JSON stdin; always fail-open)
  version
Environment: AGENT_DASHBOARD_HOME overrides the private local state directory.
Viewer: q/Ctrl-C close only the viewer; 1/2/3 buffers, Tab panels, arrows scroll.
No extra runtime, daemon or network listener required. Rex is optional.`)
}
func locateSkill(exe string) string {
	p := filepath.Dir(exe)
	for i := 0; i < 5; i++ {
		if _, e := os.Stat(filepath.Join(p, "SKILL.md")); e == nil {
			return p
		}
		p = filepath.Dir(p)
	}
	return ""
}
func run(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		help(out)
		return nil
	}
	command := args[0]
	if command == "hook" {
		// Never produce hook decisions, context, permission overrides or a nonzero exit.
		raw, e := io.ReadAll(io.LimitReader(in, d.MaxInput+1))
		if e != nil || len(raw) > d.MaxInput {
			return nil
		}
		var p d.Payload
		if json.Unmarshal(raw, &p) != nil {
			return nil
		}
		root, e := d.Root()
		if e == nil {
			_ = d.Ingest(root, p)
		}
		return nil
	}
	if command == "version" {
		fmt.Fprintf(out, "agent-dashboard %s · %s/%s\n", buildinfo.Version, runtime.GOOS, runtime.GOARCH)
		return nil
	}
	if command == "capabilities" {
		return json.NewEncoder(out).Encode(map[string]any{"version": buildinfo.Version, "features": []string{"rex-observer-v1", "rex-local-open-v1", "observation-bindings-v1"}})
	}
	if command == "rex-sources" || command == "rex-watch" || command == "open" {
		return runRex(command, args[1:], out)
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(out)
	session := f.String("session", "", "concrete Claude session ID")
	stateDir := f.String("state-dir", "", "explicit private state directory (optional)")
	title := f.String("title", "Claude task", "short issue title")
	text := f.String("text", "", "operational note")
	summary := f.String("summary", "Work explicitly marked finished", "completion summary")
	actor := f.String("actor", "lead", "observed actor ID")
	plan := f.String("json", "", "layout/plan JSON or @file")
	plain := f.Bool("plain", false, "print one clipped frame")
	width := f.Int("width", 100, "plain frame width")
	height := f.Int("height", 32, "plain frame height")
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	exe, e = filepath.EvalSymlinks(exe)
	if e != nil {
		return e
	}
	home, e := os.UserHomeDir()
	if e != nil {
		return e
	}
	skillDir := f.String("skill-dir", locateSkill(exe), "skill source directory")
	settings := f.String("settings", filepath.Join(home, ".claude", "settings.json"), "Claude settings file")
	skillsDir := f.String("skills-dir", filepath.Join(home, ".claude", "skills"), "Claude skills directory")
	if e = f.Parse(args[1:]); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() > 0 {
		return errors.New("unexpected positional arguments; use --text or --title")
	}
	root, e := d.Root()
	if e != nil {
		return e
	}
	if *stateDir != "" {
		root, e = filepath.Abs(*stateDir)
		if e != nil {
			return e
		}
	}
	switch command {
	case "install", "uninstall":
		if *skillDir == "" {
			return errors.New("--skill-dir is required")
		}
		*skillDir, e = filepath.Abs(*skillDir)
		if e != nil {
			return e
		}
		if _, e = os.Stat(filepath.Join(*skillDir, "SKILL.md")); e != nil {
			return e
		}
		if command == "install" {
			if e = d.LinkSkill(*skillDir, *skillsDir); e != nil {
				return fmt.Errorf("skill link (on Windows enable Developer Mode or create a directory junction): %w", e)
			}
		}
		backup, e := d.Configure(*settings, exe, command == "uninstall")
		if e != nil {
			return e
		}
		if command == "uninstall" {
			if e = d.UnlinkSkill(*skillDir, *skillsDir); e != nil {
				return e
			}
		}
		fmt.Fprintln(out, command+" complete; other skills/hooks preserved.")
		if backup != "" {
			fmt.Fprintln(out, "Settings backup:", backup)
		}
		fmt.Fprintln(out, "Start a fresh Claude session before use; existing sessions may retain hooks.")
	case "doctor":
		info, e := d.Inspect(*settings, exe)
		if e != nil {
			return e
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	case "start":
		if e = d.Start(root, *session, *title, false); e != nil {
			return e
		}
		fmt.Fprintln(out, "Collection enabled for this session. In another terminal on this host run:")
		printLaunch(out, exe, *session)
	case "status":
		s, e := d.Snapshot(root, *session)
		if e != nil {
			return e
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	case "note":
		if *text == "" {
			return errors.New("--text is required")
		}
		return d.Note(root, *session, *actor, *text)
	case "finish":
		return d.Finish(root, *session, *summary)
	case "deactivate":
		return d.Deactivate(root, *session)
	case "plan", "layout":
		raw := []byte(*plan)
		if strings.HasPrefix(*plan, "@") {
			raw, e = os.ReadFile((*plan)[1:])
			if e != nil {
				return e
			}
		}
		if len(raw) > d.MaxInput {
			return errors.New("plan too large")
		}
		if command == "layout" {
			return d.SetLayout(root, *session, raw)
		}
		return d.Plan(root, *session, raw)
	case "tui":
		if *plain {
			s, e := d.Snapshot(root, *session)
			if e != nil {
				return e
			}
			fmt.Fprintln(out, (d.Viewer{State: s, Width: *width, Height: *height}).View())
			return nil
		}
		return d.RunViewer(root, *session)
	case "demo":
		id := "DEMO-layout"
		if e = d.SeedDemo(root, id); e != nil {
			return e
		}
		if *plan != "" {
			raw := []byte(*plan)
			if strings.HasPrefix(*plan, "@") {
				raw, e = os.ReadFile((*plan)[1:])
				if e != nil {
					return e
				}
			}
			if e = d.SetLayout(root, id, raw); e != nil {
				return e
			}
		}
		fmt.Fprintln(out, "DEMO: synthetic layout, not a Claude session.")
		printLaunch(out, exe, id)
	default:
		return fmt.Errorf("unknown command %q; use --help", command)
	}
	return nil
}
func printLaunch(out io.Writer, exe, session string) {
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	if runtime.GOOS == "windows" {
		quote = func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
		fmt.Fprint(out, "& ")
	}
	fmt.Fprintf(out, "%s tui --session %s\n", quote(exe), quote(session))
}
