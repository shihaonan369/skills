#!/usr/bin/env python3
"""Session-scoped Claude Code telemetry and Rex panel launcher (stdlib only)."""
from __future__ import annotations

import argparse
import datetime as dt
import fcntl
import hashlib
import json
import os
from pathlib import Path
import shlex
import shutil
import sqlite3
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent
EVENTS = ("UserPromptSubmit", "PreToolUse", "PostToolUse", "PostToolUseFailure",
          "SubagentStart", "SubagentStop", "PermissionRequest", "Stop", "SessionEnd")
STATES = {"pending", "in_progress", "completed", "blocked", "deleted"}


def data_root():
    return Path(os.environ.get("REX_DASHBOARD_HOME", str(Path.home() / ".local/state/rex-agent-dashboard"))).expanduser()


def session_dir(session):
    if not isinstance(session, str) or not session or len(session) > 512:
        raise ValueError("A nonempty Claude session ID is required")
    return data_root() / hashlib.sha256(session.encode()).hexdigest()[:32]


def connect(session, create=False):
    directory = session_dir(session)
    path = directory / "telemetry.sqlite3"
    if not create and not path.exists():
        return None
    if create:
        directory.mkdir(mode=0o700, parents=True, exist_ok=True)
        directory.chmod(0o700)
    db = sqlite3.connect(path, timeout=2)
    if create:
        path.chmod(0o600)
        db.execute("PRAGMA journal_mode=WAL")
        db.executescript("""
        CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS agents (id TEXT PRIMARY KEY, data TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS tasks (id TEXT PRIMARY KEY, data TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS events (
            seq INTEGER PRIMARY KEY, time REAL NOT NULL, actor TEXT NOT NULL,
            kind TEXT NOT NULL, text TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS tools (id TEXT PRIMARY KEY, actor TEXT NOT NULL,
            name TEXT NOT NULL, started REAL NOT NULL);
        """)
    db.row_factory = sqlite3.Row
    return db


def meta(db, key, default=None):
    row = db.execute("SELECT value FROM meta WHERE key=?", (key,)).fetchone()
    return json.loads(row[0]) if row else default


def set_meta(db, key, value):
    db.execute("INSERT OR REPLACE INTO meta VALUES (?,?)", (key, json.dumps(value, ensure_ascii=False)))


def get_item(db, table, key, default=None):
    row = db.execute(f"SELECT data FROM {table} WHERE id=?", (str(key),)).fetchone()
    return json.loads(row[0]) if row else default


def put_item(db, table, item):
    db.execute(f"INSERT INTO {table} VALUES (?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data",
               (str(item["id"]), json.dumps(item, ensure_ascii=False)))


def event(db, actor, kind, text, now=None):
    db.execute("INSERT INTO events(time,actor,kind,text) VALUES (?,?,?,?)",
               (now or time.time(), actor, kind, str(text)[:500]))
    # Bound history; agent/task state remains independent of the event window.
    db.execute("DELETE FROM events WHERE seq <= (SELECT MAX(seq)-1500 FROM events)")


def initialize(session, title, cwd=None, transcript=None):
    db = connect(session, create=True)
    with db:
        db.execute("BEGIN IMMEDIATE")
        if meta(db, "session") is None:
            set_meta(db, "session", session)
            set_meta(db, "started", time.time())
            put_item(db, "agents", {"id": "lead", "name": "Claude · lead", "role": "orchestrator",
                                     "status": "ready", "started": time.time(), "tools": 0})
            event(db, "lead", "dashboard", "Dashboard attached · awaiting live hooks")
        set_meta(db, "title", title)
        set_meta(db, "cwd", cwd or os.getcwd())
        if transcript:
            set_meta(db, "transcript", transcript)
    db.close()


def short_tool(name, args):
    # Store summaries, not tool responses, source code, command lines or prompts.
    for field in ("description", "subject", "file_path", "pattern", "query"):
        value = args.get(field)
        if value:
            return f"{name} · {str(value)[:150]}"
    return name or "tool"


def normalize_task(db, task_id, changes):
    task_id = str(task_id)
    old = get_item(db, "tasks", task_id, {"id": task_id, "subject": f"Task {task_id}", "status": "pending"})
    aliases = {"title": "subject", "content": "subject", "depends_on": "blockedBy"}
    for key, value in changes.items():
        key = aliases.get(key, key)
        if key in {"subject", "status", "owner", "blockedBy", "activeForm"}:
            if key == "status" and value not in STATES:
                raise ValueError(f"Invalid task status: {value}")
            old[key] = value
    put_item(db, "tasks", old)


def sync_tasks(db, name, args, response):
    if name == "TodoWrite":
        # TodoWrite is a full replacement, unlike TaskUpdate patches.
        db.execute("DELETE FROM tasks")
        for i, todo in enumerate(args.get("todos", []), 1):
            normalize_task(db, str(i), todo)
    elif name == "TaskCreate":
        task = response.get("task", {}) if isinstance(response, dict) else {}
        task_id = task.get("id") or (response.get("taskId") if isinstance(response, dict) else None)
        if task_id is not None:
            normalize_task(db, task_id, {**args, **task})
    elif name == "TaskUpdate" and args.get("taskId") is not None:
        task_id = args["taskId"]
        changes = dict(args)
        old = get_item(db, "tasks", task_id, {})
        if "addBlockedBy" in args:
            changes["blockedBy"] = list(dict.fromkeys(old.get("blockedBy", []) + args["addBlockedBy"]))
        normalize_task(db, task_id, changes)
    elif name == "TaskList":
        tasks = response.get("tasks", []) if isinstance(response, dict) else []
        for task in tasks:
            if isinstance(task, dict) and task.get("id") is not None:
                normalize_task(db, task["id"], task)


def ingest(payload):
    session = payload.get("session_id")
    if not session:
        return
    db = connect(session)
    if db is None:  # Global hooks are inert until this session opts in.
        return
    try:
        with db:
            db.execute("BEGIN IMMEDIATE")
            now = time.time()
            kind = payload.get("hook_event_name", "unknown")
            actor = str(payload.get("agent_id") or "lead")
            agent = get_item(db, "agents", actor, {
                "id": actor, "name": (payload.get("agent_type") or "agent") + " · " + actor[:6],
                "role": payload.get("agent_type") or "subagent", "status": "running",
                "started": now, "tools": 0, "parent": "lead"})
            set_meta(db, "last_hook", now)
            set_meta(db, "hook_count", meta(db, "hook_count", 0) + 1)
            if payload.get("transcript_path"):
                set_meta(db, "transcript", payload["transcript_path"])
            if payload.get("agent_transcript_path"):
                agent["transcript"] = payload["agent_transcript_path"]
            if kind == "UserPromptSubmit":
                set_meta(db, "finished", False)
                agent["status"] = "running"
                agent.pop("ended", None)
                event(db, actor, "turn", "New turn · planning / execution")
            elif kind == "SubagentStart":
                agent.update(status="running", name=(payload.get("agent_type") or "agent") + " · " + actor[:6], started=now)
                agent.pop("ended", None)
                event(db, actor, "spawn", f"{agent['name']} started")
            elif kind == "SubagentStop":
                agent.update(status="done", ended=now, current="")
                db.execute("DELETE FROM tools WHERE actor=?", (actor,))
                event(db, actor, "done", f"{agent['name']} returned to lead")
            elif kind in {"PreToolUse", "PostToolUse", "PostToolUseFailure"}:
                name = payload.get("tool_name", "tool")
                args = payload.get("tool_input") or {}
                tool_id = str(payload.get("tool_use_id") or f"{actor}:{name}")
                if kind == "PreToolUse":
                    if actor == "lead":
                        set_meta(db, "finished", False)
                    agent.update(status="running", current=short_tool(name, args))
                    agent.pop("ended", None)
                    db.execute("INSERT OR REPLACE INTO tools VALUES (?,?,?,?)", (tool_id, actor, name, now))
                    if name in {"Agent", "Task"}:
                        event(db, actor, "delegate", short_tool(name, args))
                    else:
                        event(db, actor, "tool", short_tool(name, args))
                else:
                    db.execute("DELETE FROM tools WHERE id=?", (tool_id,))
                    agent["tools"] = agent.get("tools", 0) + 1
                    # One parallel completion must not clear another tool's status.
                    pending = db.execute("SELECT name FROM tools WHERE actor=? ORDER BY started DESC LIMIT 1", (actor,)).fetchone()
                    agent["current"] = pending[0] if pending else ""
                    agent["status"] = "done" if actor == "lead" and meta(db, "finished", False) else "running"
                    if kind == "PostToolUseFailure":
                        agent["last_error"] = f"{name} failed"
                        event(db, actor, "error", f"{name} failed · inspect Claude for details")
                    else:
                        sync_tasks(db, name, args, payload.get("tool_response"))
                        event(db, actor, "result", f"{name} completed")
            elif kind == "PermissionRequest":
                agent["status"] = "blocked"
                event(db, actor, "blocked", f"Awaiting permission · {payload.get('tool_name', 'tool')}")
            elif kind == "Stop":
                finished = meta(db, "finished", False)
                agent.update(status="done" if finished else "waiting", current="")
                event(db, actor, "turn", "Lead turn ended · task marked complete" if finished else "Lead turn ended · waiting for next prompt")
            elif kind == "SessionEnd":
                agent.update(status="ended", ended=now, current="")
                for row in db.execute("SELECT data FROM agents").fetchall():
                    other = json.loads(row[0])
                    if other["id"] != actor and other["status"] in {"running", "blocked"}:
                        other.update(status="interrupted", ended=now, current="")
                        put_item(db, "agents", other)
                db.execute("DELETE FROM tools")
                event(db, actor, "session", "Claude session ended")
            put_item(db, "agents", agent)
    finally:
        db.close()


def transcript_metrics(path):
    """Read a bounded tail of an explicitly known local transcript; never infer cost."""
    try:
        with Path(path).expanduser().open("rb") as file:
            size = file.seek(0, 2)
            file.seek(max(0, size - 131072))
            raw = file.read()
        model = None
        for line in raw.splitlines():
            try:
                item = json.loads(line)
                message = item.get("message") or {}
                if item.get("type") == "assistant" and message.get("model") not in {None, "<synthetic>"}:
                    model = message["model"]
            except (ValueError, TypeError, AttributeError):
                continue
        return {"model": model} if model else {}
    except (OSError, ValueError):
        return {}


def snapshot(session):
    db = connect(session)
    if db is None:
        raise ValueError("Session not initialized; run start first")
    try:
        # A consistent read transaction across all tables.
        db.execute("BEGIN")
        result = {"meta": {r[0]: json.loads(r[1]) for r in db.execute("SELECT * FROM meta")},
                  "agents": [json.loads(r[0]) for r in db.execute("SELECT data FROM agents ORDER BY rowid")],
                  "tasks": [json.loads(r[0]) for r in db.execute("SELECT data FROM tasks ORDER BY rowid")],
                  "events": [dict(r) for r in db.execute("SELECT * FROM events ORDER BY seq DESC LIMIT 250")],
                  "tools": [dict(r) for r in db.execute("SELECT * FROM tools")]}
    finally:
        db.close()
    transcript = result["meta"].get("transcript")
    for agent in result["agents"]:
        path = agent.get("transcript")
        if agent["id"] == "lead":
            path = transcript
        elif not path and transcript:
            main = Path(transcript).expanduser()
            path = main.parent / main.stem / "subagents" / f"agent-{agent['id']}.jsonl"
        if path:
            agent.update(transcript_metrics(path))
    result["tasks"] = [t for t in result["tasks"] if t.get("status") != "deleted"]
    return result


def rex_json(args):
    args = list(args)
    # `block call` always emits JSON and has no --json flag. For split, flags
    # belong BEFORE --, never in the launched program's argument list.
    if args[:2] != ["block", "call"]:
        args.insert(args.index("--") if "--" in args else len(args), "--json")
    result = subprocess.run(["rex", *args], capture_output=True, text=True, timeout=20)
    if result.returncode:
        raise RuntimeError(result.stderr.strip() or result.stdout.strip())
    return json.loads(result.stdout)


def open_panel(session, ratio=48):
    if not shutil.which("rex"):
        raise RuntimeError("Rex CLI not found. Use `dashboard.py tui --session ID` in a terminal.")
    if not os.environ.get("REX_SESSION") or not os.environ.get("REX_BLOCK"):
        raise RuntimeError("Run start from a Claude terminal inside Rex (REX_SESSION and REX_BLOCK are required).")
    directory = session_dir(session)
    with (directory / "panel.lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        db = connect(session)
        try:
            previous = meta(db, "panel")
            if previous:
                try:
                    rex_json(["block", "inspect", previous["block_id"], "-s", previous["rex_session"]])
                except RuntimeError as exc:
                    if not any(s in str(exc).lower() for s in ("not found", "not_found", "no block", "does not exist")):
                        raise  # A connection failure is not proof that a panel was closed.
                else:
                    # An exited TUI is not a live dashboard.
                    process = rex_json(["block", "call", "-s", previous["rex_session"], "-b", previous["block_id"], "process"])
                    if process.get("last_exit") is None:
                        return {**previous, "reused": True}
                    # Keep history visible; a fresh panel gets its own process.
            # A timed-out split may have succeeded. Persist its unique label BEFORE
            # requesting it, then reconcile on retry instead of making duplicates.
            pending = meta(db, "panel_pending")
            if pending:
                listing = rex_json(["block", "ls", "-s", pending["rex_session"]])
                matches = [b for b in listing.get("blocks", []) if b.get("label") == pending["label"]]
                if len(matches) > 1:
                    raise RuntimeError("Multiple matching dashboard panes found; inspect them before retrying.")
                if matches:
                    process = rex_json(["block", "call", "-s", pending["rex_session"], "-b", matches[0]["block_id"], "process"])
                    if process.get("last_exit") is None:
                        panel = {"block_id": matches[0]["block_id"], "rex_session": pending["rex_session"]}
                        with db:
                            set_meta(db, "panel", panel)
                            set_meta(db, "panel_pending", None)
                        return {**panel, "reused": True}
            label = "Agent Dashboard · " + directory.name[:6]
            with db:
                set_meta(db, "panel_pending", {"label": label, "rex_session": os.environ["REX_SESSION"]})
            created = rex_json(["split", "-s", os.environ["REX_SESSION"], "-b", os.environ["REX_BLOCK"],
                                "--split=right", "--ratio", str(ratio), "--focus=false", "--shell=none",
                                "--label", label, "--", sys.executable, str(ROOT / "dashboard.py"),
                                "tui", "--session", session])
            panel = {"block_id": created["block_ids"][0], "rex_session": os.environ["REX_SESSION"]}
            with db:
                set_meta(db, "panel", panel)
                set_meta(db, "panel_pending", None)
            return panel
        finally:
            db.close()


def install_hooks(settings, uninstall=False):
    skill = Path.home() / ".claude/skills/rex-dashboard"
    target = ROOT / "skills/rex-dashboard"
    if not uninstall and (skill.exists() or skill.is_symlink()) and not (skill.is_symlink() and skill.resolve() == target):
        raise RuntimeError(f"Skill already exists, preserved: {skill}")
    settings = Path(settings).expanduser()
    settings.parent.mkdir(parents=True, exist_ok=True)
    with settings.with_suffix(".rex-dashboard.lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        original = settings.read_text() if settings.exists() else "{}\n"
        data = json.loads(original)
        hooks = data.setdefault("hooks", {})
        command = f"{shlex.quote(sys.executable)} {shlex.quote(str(ROOT / 'dashboard.py'))} hook"
        # Identify exactly our command rather than deleting similarly named hooks.
        for name in list(hooks):
            entries = []
            for entry in hooks[name]:
                copy = dict(entry)
                copy["hooks"] = [h for h in entry.get("hooks", []) if h.get("command") != command]
                if copy["hooks"]:
                    entries.append(copy)
            if entries:
                hooks[name] = entries
            else:
                del hooks[name]
        if not uninstall:
            for name in EVENTS:
                hooks.setdefault(name, []).append({"hooks": [{"type": "command", "command": command, "timeout": 4}]})
        updated = json.dumps(data, ensure_ascii=False, indent=2) + "\n"
        if updated != original:
            backup = settings.with_name(settings.name + ".rex-dashboard-" + dt.datetime.now().strftime("%Y%m%d-%H%M%S-%f") + ".bak")
            backup.write_text(original)
            backup.chmod(0o600)
            temp = settings.with_name(settings.name + f".{os.getpid()}.tmp")
            try:
                temp.write_text(updated)
                temp.chmod(0o600)
                os.replace(temp, settings)
            finally:
                temp.unlink(missing_ok=True)
            print(f"Settings updated; backup: {backup}")
        else:
            print("Hooks already configured")
    if not uninstall:
        skill.parent.mkdir(parents=True, exist_ok=True)
        if skill.is_symlink() and skill.resolve() == target:
            pass
        elif skill.exists() or skill.is_symlink():
            raise RuntimeError(f"Skill already exists, preserved: {skill}")
        else:
            skill.symlink_to(target, target_is_directory=True)
        print(f"Skill installed: {skill}")
    elif skill.is_symlink() and skill.resolve() == target:
        skill.unlink()
        print(f"Skill removed: {skill}")


def demo(session):
    initialize(session, "DEMO · Issue #128 · Reliable session reconnect", cwd="example/rex")
    ingest({"session_id": session, "hook_event_name": "UserPromptSubmit"})
    db = connect(session)
    with db:
        db.execute("BEGIN IMMEDIATE")
        for table in ("agents", "tasks", "tools", "events"):
            db.execute(f"DELETE FROM {table}")
        set_meta(db, "demo", True)
        set_meta(db, "transcript", None)
        set_meta(db, "finished", False)
        for key, subject, status, owner in [
            ("1", "Map reconnect flow & reproduce", "completed", "search"),
            ("2", "Fix transport retry state", "in_progress", "lead"),
            ("3", "Add regression tests", "in_progress", "tests"),
            ("4", "Review diff & verify", "pending", "audit")]:
            normalize_task(db, key, {"subject": subject, "status": status, "owner": owner,
                                     "blockedBy": ["2", "3"] if key == "4" else []})
        for key, name, model, state, current, count in [
            ("lead", "Claude · lead", "sonnet · DEMO", "running", "Edit · retry state machine", 18),
            ("search", "search", "haiku · DEMO", "done", "Mapped 12 call sites", 9),
            ("tests", "tests", "haiku · DEMO", "running", "Bash · regression suite", 7),
            ("audit", "audit", "sonnet · DEMO", "running", "Read · transport.ts", 4),
            ("advisor", "advisor", "opus · DEMO", "waiting", "Review retry invariants", 2)]:
            put_item(db, "agents", {"id": key, "name": name, "model": model, "status": state,
                                     "role": "advisor" if key == "advisor" else "worker", "tools": count,
                                     "current": current, "started": time.time() - 125,
                                     "ended": time.time() - 15 if state == "done" else None})
        for actor, kind, text in [("advisor", "note", "Keep retry ownership in one layer"),
                                  ("lead", "delegate", "search / tests / audit"),
                                  ("search", "done", "Mapped 12 call sites · returned to lead"),
                                  ("tests", "tool", "Regression suite running"),
                                  ("lead", "tool", "Editing retry state machine")]:
            event(db, actor, kind, text)
    db.close()


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    start = sub.add_parser("start", help="Opt in and open / reuse right Rex panel")
    start.add_argument("--session", required=True, help="Claude session ID, not Rex session ID")
    start.add_argument("--title", required=True)
    start.add_argument("--ratio", type=int, default=48, choices=range(20, 81), metavar="20..80")
    start.add_argument("--no-panel", action="store_true")
    start.add_argument("--transcript")
    sub.add_parser("hook", help="Consume one Claude hook JSON payload on stdin; always fail open")
    for name in ("tui", "status", "finish", "note", "plan", "demo"):
        child = sub.add_parser(name)
        child.add_argument("--session", required=True)
        if name == "note":
            child.add_argument("text")
            child.add_argument("--actor", default="lead")
        if name == "plan":
            child.add_argument("--json", required=True, help="JSON task array; use @path to read a file")
        if name == "finish":
            child.add_argument("--summary", default="Implementation and verification complete")
        if name == "tui":
            child.add_argument("--plain", action="store_true", help="Print one snapshot for diagnostics")
    for name in ("install", "uninstall"):
        child = sub.add_parser(name)
        child.add_argument("--settings", default=str(Path.home() / ".claude/settings.json"))
    return parser


def main():
    os.umask(0o077)
    args = build_parser().parse_args()
    if args.command == "hook":
        try:
            payload = json.loads(sys.stdin.read(4 * 1024 * 1024))
            ingest(payload)
        except Exception:
            # Observability must never block execution or write context/decisions.
            pass
        return
    if args.command in {"install", "uninstall"}:
        install_hooks(args.settings, args.command == "uninstall")
    elif args.command == "start":
        initialize(args.session, args.title, transcript=args.transcript)
        print(json.dumps({"session": args.session, "panel": None if args.no_panel else open_panel(args.session, args.ratio)}, ensure_ascii=False))
    elif args.command == "demo":
        demo(args.session)
        print(json.dumps(open_panel(args.session)))
    elif args.command == "status":
        print(json.dumps(snapshot(args.session), ensure_ascii=False, indent=2))
    elif args.command == "tui":
        from tui import run, plain
        if args.plain:
            print(plain(snapshot(args.session)))
        else:
            run(args.session)
    else:
        db = connect(args.session)
        if db is None:
            raise ValueError("Start the dashboard first")
        try:
            with db:
                db.execute("BEGIN IMMEDIATE")
                if args.command == "plan":
                    tasks = json.loads(Path(args.json[1:]).read_text() if args.json.startswith("@") else args.json)
                    if not isinstance(tasks, list) or not all(isinstance(t, dict) for t in tasks):
                        raise ValueError("Plan must be a JSON array of task objects")
                    db.execute("DELETE FROM tasks")
                    for i, task in enumerate(tasks, 1):
                        normalize_task(db, task.get("id", str(i)), task)
                    event(db, "lead", "plan", f"Plan updated · {len(tasks)} steps")
                elif args.command == "note":
                    event(db, args.actor, "note", args.text)
                elif args.command == "finish":
                    lead = get_item(db, "agents", "lead")
                    lead.update(status="done", ended=time.time(), current="")
                    put_item(db, "agents", lead)
                    set_meta(db, "finished", True)
                    event(db, "lead", "done", args.summary)
        finally:
            db.close()


if __name__ == "__main__":
    try:
        main()
    except (ValueError, RuntimeError, OSError, subprocess.SubprocessError, sqlite3.Error) as exc:
        print(f"rex-dashboard: {exc}", file=sys.stderr)
        sys.exit(1)
