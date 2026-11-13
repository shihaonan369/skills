"""Responsive terminal renderer. No dependency beyond Python's curses module."""
from __future__ import annotations

import curses
import locale
import re
import time
import unicodedata

from dashboard import snapshot

PALETTE = {"text": 1, "muted": 2, "orange": 3, "blue": 4, "green": 5, "purple": 6, "red": 7}
SYMBOLS = {"pending": "○", "in_progress": "◉", "completed": "✓", "blocked": "!",
           "running": "◉", "done": "✓", "waiting": "◇", "ready": "○",
           "ended": "□", "interrupted": "!"}


def clean(value):
    # Data is never allowed to emit terminal control sequences (including OSC).
    return "".join(c if unicodedata.category(c)[0] != "C" else " " for c in str(value))


def cell_width(char):
    if unicodedata.combining(char):
        return 0
    return 2 if unicodedata.east_asian_width(char) in {"W", "F"} else 1


def fit(text, width):
    out, used = [], 0
    for char in clean(text):
        size = cell_width(char)
        if used + size > width:
            break
        out.append(char)
        used += size
    return "".join(out)


def width(text):
    return sum(cell_width(c) for c in clean(text))


def model_name(agent):
    value = agent.get("model")
    if not value:
        return "model unknown"
    return re.sub(r"-\d{8}$", "", value.replace("claude-", ""))


def duration(agent, now):
    seconds = max(0, int((agent.get("ended") or now) - agent.get("started", now)))
    return f"{seconds // 60:02d}:{seconds % 60:02d}"


def state_color(state):
    return "green" if state in {"completed", "done"} else "red" if state in {"blocked", "interrupted"} else "blue"


class Canvas:
    """Cell-aware clipping shared by curses output and plain diagnostic snapshots."""
    def __init__(self, rows, cols):
        self.rows, self.cols = max(1, rows), max(1, cols)
        self.cells = [[(" ", "text", False) for _ in range(self.cols)] for _ in range(self.rows)]

    def put(self, y, x, text, color="text", bold=False, limit=None):
        if y < 0 or y >= self.rows:
            return
        end = min(self.cols, x + limit) if limit is not None else self.cols
        for char in clean(text):
            size = cell_width(char)
            if size == 0:
                continue
            if x + size > end:
                break
            if x >= 0:
                self.cells[y][x] = (char, color, bold)
                for i in range(1, size):
                    self.cells[y][x + i] = ("", color, bold)
            x += size

    def hline(self, y, x, length, char="─", color="muted"):
        self.put(y, x, char * max(0, length), color)

    def box(self, y, x, h, w, title="", color="muted"):
        if h < 2 or w < 3:
            return
        self.put(y, x, "╭" + "─" * (w - 2) + "╮", color)
        self.put(y + h - 1, x, "╰" + "─" * (w - 2) + "╯", color)
        for row in range(y + 1, y + h - 1):
            self.put(row, x, "│", color)
            self.put(row, x + w - 1, "│", color)
        if title:
            self.put(y, x + 2, " " + fit(title, w - 7) + " ", color, True)

    def centered(self, y, x, w, text, color="text", bold=False):
        text = fit(text, w)
        self.put(y, x + max(0, (w - width(text)) // 2), text, color, bold)

    def lines(self):
        return ["".join(c[0] for c in row).rstrip() for row in self.cells]

    def paint(self, screen):
        screen.erase()
        for y, row in enumerate(self.cells):
            x = 0
            while x < len(row):
                char, color, bold = row[x]
                if not char:
                    x += 1
                    continue
                attr = curses.color_pair(PALETTE[color]) if curses.has_colors() else 0
                if bold:
                    attr |= curses.A_BOLD
                chars = [char]
                start = x
                x += cell_width(char)
                while x < len(row):
                    nxt, col, strong = row[x]
                    if not nxt or col != color or strong != bold:
                        break
                    chars.append(nxt)
                    x += cell_width(nxt)
                try:
                    screen.addstr(y, start, "".join(chars), attr)
                except curses.error:
                    pass  # Bottom-right cell and concurrent resize are safe.
        screen.noutrefresh()
        curses.doupdate()


def progress(tasks):
    done = sum(t.get("status") == "completed" for t in tasks)
    return done, len(tasks)


def bar(done, total, size):
    filled = int(size * done / total) if total else 0
    return "█" * filled + "░" * (size - filled)


def render(data, rows=48, cols=96, selected=0, scroll=0, paused=False, logs_only=False, help_on=False, now=None):
    now = now or time.time()
    c = Canvas(rows, cols)
    meta, agents, tasks, events = (data.get(k, {} if k == "meta" else []) for k in ("meta", "agents", "tasks", "events"))
    lead = next((a for a in agents if a["id"] == "lead"), {"id": "lead", "name": "Claude · lead", "status": "ready"})
    workers = [a for a in agents if a["id"] != "lead" and "advisor" not in a.get("role", "").lower()]
    advisors = [a for a in agents if a["id"] != "lead" and "advisor" in a.get("role", "").lower()]
    selected = max(0, min(selected, len(workers) - 1))
    done, total = progress(tasks)
    if rows < 12 or cols < 32:
        c.put(0, 0, "AGENT DASHBOARD", "orange", True)
        c.put(2, 0, "Resize panel to ≥32 cols × 12 rows", "muted")
        c.put(4, 0, f"Agents {len(workers)} · Steps {done}/{total}", "green")
        c.put(rows - 1, 0, "q quit · resize to show dashboard", "muted")
        return c
    c.put(0, 2, "MULTI-AGENT ORCHESTRATION", "text", True, cols - 16)
    badge = "DEMO" if meta.get("demo") else "PAUSED" if paused else "LIVE"
    c.put(0, cols - 12, "● " + badge, "purple" if meta.get("demo") else "green", True)
    c.put(1, 2, meta.get("title", "Untitled issue"), "orange", True, cols - 4)
    c.hline(2, 1, cols - 2)
    active = sum(a.get("status") in {"running", "blocked"} for a in workers)
    c.put(3, 2, f"● lead   ● workers {active}/{len(workers)}   ◆ advisor {len(advisors)}", "blue", limit=cols - 4)
    if total:
        c.put(4, 2, f"{bar(done, total, min(20, max(4, cols // 5)))}  {done}/{total} plan steps complete", "green", limit=cols - 4)
    else:
        c.put(4, 2, "Plan not reported · progress unknown", "muted", limit=cols - 4)
    log_h = min(9, max(4, rows // 5))
    log_y = rows - log_h - 2
    body_y = 6
    if logs_only:
        log_y = 6
        log_h = rows - 8
    else:
        body_h = log_y - body_y - 1
        if body_h >= 9:
            _body(c, body_y, body_h, lead, workers, advisors, tasks, events, selected, now)
        else:
            c.put(body_y, 2, f"LEAD · {lead.get('status', 'ready')} · {model_name(lead)}", "orange", limit=cols - 4)
            for i, a in enumerate(workers[:max(0, body_h - 1)]):
                c.put(body_y + i + 1, 2, f"{SYMBOLS.get(a['status'], '○')} {a['name']} · {a['status']} · {a.get('current', '')}", state_color(a["status"]), limit=cols - 4)
    _logs(c, log_y, log_h, events, scroll)
    hook = meta.get("last_hook")
    if hook:
        age = max(0, int(now - hook))
        feed = f"last hook {age}s ago" if age < 60 else f"quiet {age // 60}m · no heartbeat"
    else:
        feed = "awaiting hooks · restart Claude if newly installed"
    c.put(rows - 2, 2, feed, "muted", limit=cols - 4)
    c.put(rows - 1, 2, "q quit  Tab agent  ↑↓ logs  p pause  l logs  ? help", "muted", limit=cols - 4)
    if help_on:
        w, h = min(68, cols - 4), min(14, rows - 4)
        x, y = (cols - w) // 2, (rows - h) // 2
        for yy in range(y, y + h):
            c.put(yy, x, " " * w)
        c.box(y, x, h, w, "KEYS & DATA", "purple")
        help_lines = ["Tab / j / k      select a worker", "↑ / ↓ / PgUp     scroll session log", "p                pause / resume local rendering",
                      "l                expand session log", "q                close viewer (Claude keeps running)",
                      "? / Esc          close this help", "", "Progress = completed / reported plan steps.",
                      "Models appear only when found in transcripts.", "Quiet hooks do not prove an agent is stuck."]
        for i, text in enumerate(help_lines[:h - 2]):
            c.put(y + 1 + i, x + 2, text, "text", limit=w - 4)
    return c


def _body(c, y, h, lead, workers, advisors, tasks, events, selected, now):
    # Wide layout mirrors the reference: purple rail + orange/green/blue flow.
    wide = c.cols >= 80 and h >= 20
    rail_w = min(25, c.cols // 4) if wide else 0
    gx = rail_w + 3 if wide else 2
    gw = c.cols - gx - 2
    if wide:
        c.box(y, 1, h, rail_w, "ADVISOR", "purple")
        if advisors:
            advisor = advisors[-1]
            lines = [(advisor["name"], True), (model_name(advisor), False),
                     (f"◇ {advisor['status']}", False), ("", False),
                     (advisor.get("current") or "On call", False), ("", False),
                     ("Latest advice:", False)]
            advice = next((e["text"] for e in events if e["actor"] == advisor["id"] and e["kind"] == "note"), "No advice reported")
            lines.extend((s, False) for s in wrap(advice, rail_w - 4))
        else:
            lines = [("◇ not attached", True), ("", False)]
            lines.extend((s, False) for s in wrap("Advisor appears when a real advisor subagent starts. No synthetic agents.", rail_w - 4))
        for i, (text, bold) in enumerate(lines[:h - 4]):
            c.put(y + 2 + i, 3, text, "purple" if bold else "muted", bold, rail_w - 4)
        c.put(y + h - 2, 3, "read-only telemetry", "muted", limit=rail_w - 4)
    lead_w = min(48, gw)
    lead_x = gx + (gw - lead_w) // 2
    c.box(y, lead_x, 4, lead_w, "MAIN SESSION", "orange")
    c.centered(y + 1, lead_x + 1, lead_w - 2, f"{model_name(lead)} · {lead.get('status', 'ready')}", "orange", True)
    c.centered(y + 2, lead_x + 1, lead_w - 2, lead.get("current") or "plan → delegate → implement → verify", "text")
    c.centered(y + 4, gx, gw, "│", "orange")
    routing_y = y + 5
    room_for_cards = h >= 17
    if not room_for_cards:
        c.box(y + 5, gx, h - 5, gw, "SUBAGENTS · Tab to select", "blue")
        capacity = max(1, h - 7)
        start = (selected // capacity) * capacity
        for i, a in enumerate(workers[start:start + capacity]):
            text = f"{'›' if start + i == selected else ' '} {a['name']} · {a['status']} · {model_name(a)} · {a.get('current') or str(a.get('tools', 0)) + ' tools'}"
            c.put(y + 6 + i, gx + 2, text, state_color(a["status"]), limit=gw - 4)
        if not workers:
            c.put(y + 6, gx + 2, "No subagents yet · waiting for delegation", "muted", limit=gw - 4)
        return
    c.box(routing_y, gx, 4, gw, "ORCHESTRATION", "green")
    running_tools = sum(bool(a.get("current")) for a in [lead, *workers])
    c.put(routing_y + 1, gx + 2, f"{len(workers)} delegated  ·  {running_tools} active agents  ·  {sum(a.get('tools', 0) for a in [lead, *workers])} tools done", "green", limit=gw - 4)
    current = next((t for t in tasks if t.get("status") == "in_progress"), None)
    c.put(routing_y + 2, gx + 2, "Now: " + (current.get("activeForm") or current.get("subject", "")) if current else "Next: " + next((t.get("subject", "") for t in tasks if t.get("status") == "pending"), "awaiting plan / delegation"), "text", limit=gw - 4)
    cy = routing_y + 5
    card_h = 6
    capacity = max(1, min(3, gw // 19))
    start = (selected // capacity) * capacity
    visible = workers[start:start + capacity]
    if visible:
        card_w = (gw - (len(visible) - 1)) // len(visible)
        centers = [gx + i * (card_w + 1) + card_w // 2 for i in range(len(visible))]
        c.hline(routing_y + 4, centers[0], centers[-1] - centers[0] + 1, color="blue")
        for center in centers:
            c.put(routing_y + 4, center, "┬", "blue")
        c.put(routing_y + 4, gx + gw // 2, "┴", "green")
        for i, agent in enumerate(visible):
            xx = gx + i * (card_w + 1)
            color = state_color(agent["status"])
            c.box(cy, xx, card_h, card_w, ("› " if start + i == selected else "") + agent["name"], "blue")
            c.put(cy + 1, xx + 2, model_name(agent), "blue", limit=card_w - 4)
            owned = [t for t in tasks if t.get("owner") in {agent["id"], agent["name"]}]
            done, total = progress(owned)
            progress_text = f"{done}/{total} steps" if total else "progress —"
            status = agent["status"]
            if card_w < 24:
                status = {"running": "run", "waiting": "wait", "interrupted": "stop", "blocked": "blocked"}.get(status, status)
                progress_text = f"{done}/{total}" if total else "—"
            c.put(cy + 2, xx + 2, f"{SYMBOLS.get(agent['status'], '○')} {status} {duration(agent, now)}", color, limit=card_w - 4)
            c.put(cy + 3, xx + 2, progress_text + f" · {agent.get('tools', 0)} tools", "muted", limit=card_w - 4)
            c.put(cy + 4, xx + 2, agent.get("current") or agent.get("last_error") or ("Returned to lead" if agent["status"] == "done" else "Awaiting next event"), "text", limit=card_w - 4)
    else:
        c.centered(routing_y + 4, gx, gw, "│", "green")
        c.box(cy, gx, card_h, gw, "SUBAGENTS", "blue")
        c.centered(cy + 2, gx + 1, gw - 2, "No subagents yet · waiting for real delegation", "muted")
        c.centered(cy + 3, gx + 1, gw - 2, "Single-agent execution is shown honestly", "muted")
    after = cy + card_h
    if after < y + h:
        page = f" {start + 1}–{start + len(visible)}/{len(workers)} · Tab to browse " if len(workers) > capacity else " ↓ back to lead · review + verify "
        c.centered(after, gx, gw, page, "orange")
    plan_y = after + 1
    remaining = y + h - plan_y
    if remaining >= 3:
        plan_h = min(remaining, max(5, min(len(tasks) + 2, 9)))
        _plan(c, plan_y, gx, plan_h, gw, tasks)
        detail_y = plan_y + plan_h + 1
        detail_h = y + h - detail_y
        if detail_h >= 4:
            focus = workers[selected] if workers else lead
            c.box(detail_y, gx, detail_h, gw, "FOCUS · " + focus["name"], "blue")
            details = [(f"ID {focus['id']}", "muted"),
                       (f"{model_name(focus)} · {focus['status']} · {duration(focus, now)}", "blue"),
                       (focus.get("current") or focus.get("last_error") or "No active tool reported", "text")]
            details += [(time.strftime("%H:%M:%S", time.localtime(e["time"])) + " " + e["text"], "muted")
                        for e in events if e["actor"] == focus["id"]][:4]
            for i, (text, color) in enumerate(details[:detail_h - 2]):
                c.put(detail_y + i + 1, gx + 2, text, color, limit=gw - 4)


def wrap(text, columns):
    lines = []
    while text:
        line = fit(text, max(1, columns))
        if not line:
            break
        if len(line) < len(text) and " " in line:
            boundary = line.rfind(" ")
            if boundary > len(line) // 2:
                line = line[:boundary]
        lines.append(line)
        text = text[len(line):].lstrip()
    return lines


def _plan(c, y, x, h, w, tasks):
    c.box(y, x, h, w, "PLAN / DEPENDENCIES", "muted")
    if not tasks:
        c.put(y + 1, x + 2, "Waiting for TaskCreate / TodoWrite / plan", "muted", limit=w - 4)
        return
    capacity = h - 2
    active = next((i for i, t in enumerate(tasks) if t.get("status") == "in_progress"), 0)
    start = max(0, min(active - 1, len(tasks) - capacity))
    for i, task in enumerate(tasks[start:start + capacity]):
        state = task.get("status", "pending")
        deps = task.get("blockedBy") or []
        text = f"{SYMBOLS.get(state, '○')} {task['id']} {task.get('subject', '')}"
        if deps:
            text += " ← " + ",".join(str(d) for d in deps)
        if i == capacity - 1 and start + capacity < len(tasks):
            text = f"… {len(tasks) - start - capacity + 1} more steps"
        c.put(y + 1 + i, x + 2, text, state_color(state) if state != "pending" else "muted", limit=w - 4)


def _logs(c, y, h, events, scroll):
    c.box(y, 1, h, c.cols - 2, "SESSION LOG" + (f" · offset {scroll}" if scroll else ""), "muted")
    limit = max(0, h - 2)
    if not events:
        c.put(y + 1, 3, "Waiting for events…", "muted", limit=c.cols - 6)
    for i, ev in enumerate(reversed(events[scroll:scroll + limit])):
        stamp = time.strftime("%H:%M:%S", time.localtime(ev["time"]))
        col = "red" if ev["kind"] in {"error", "blocked"} else "green" if ev["kind"] == "done" else "purple" if ev["kind"] == "note" else "blue"
        c.put(y + 1 + i, 3, stamp, "muted")
        actor_w = min(13, max(6, c.cols // 7))
        c.put(y + 1 + i, 12, ev["actor"], col, True, actor_w)
        c.put(y + 1 + i, 13 + actor_w, ev["text"], "text", limit=c.cols - actor_w - 17)


def plain(data, rows=48, cols=96):
    return "\n".join(render(data, rows, cols).lines())


def run(session):
    locale.setlocale(locale.LC_ALL, "")
    def loop(screen):
        try:
            curses.curs_set(0)
        except curses.error:
            pass
        if curses.has_colors():
            curses.start_color()
            try:
                curses.use_default_colors()
                bg = -1
            except curses.error:
                bg = curses.COLOR_BLACK
            colors = [252, 244, 216, 111, 114, 183, 210] if curses.COLORS >= 256 else [7, 7, 3, 6, 2, 5, 1]
            for i, fg in enumerate(colors, 1):
                curses.init_pair(i, fg, bg)
        screen.timeout(250)
        selected = scroll = 0
        paused = logs_only = help_on = False
        data = snapshot(session)
        error = None
        last_read = 0
        while True:
            if not paused and time.monotonic() - last_read >= 0.5:
                try:
                    data = snapshot(session)
                    error = None
                except Exception as exc:
                    error = f"Telemetry read error · {type(exc).__name__} · retaining last snapshot"
                last_read = time.monotonic()
            rows, cols = screen.getmaxyx()
            workers = [a for a in data["agents"] if a["id"] != "lead" and "advisor" not in a.get("role", "").lower()]
            selected = min(selected, max(0, len(workers) - 1))
            scroll = min(scroll, max(0, len(data["events"]) - 1))
            canvas = render(data, rows, cols, selected, scroll, paused, logs_only, help_on)
            if error:
                canvas.put(rows - 2, 2, error, "red", limit=cols - 4)
            canvas.paint(screen)
            key = screen.getch()
            if key in (ord("q"), 3):
                break
            if key in (ord("?"), 27):
                help_on = not help_on if key == ord("?") else False
            elif key == ord("p"):
                paused = not paused
            elif key == ord("l"):
                logs_only = not logs_only
            elif key in (9, ord("j")):
                selected = (selected + 1) % max(1, len(workers))
            elif key == ord("k"):
                selected = (selected - 1) % max(1, len(workers))
            elif key in (curses.KEY_UP, curses.KEY_PPAGE):
                scroll += 1 if key == curses.KEY_UP else 5
            elif key in (curses.KEY_DOWN, curses.KEY_NPAGE):
                scroll = max(0, scroll - (1 if key == curses.KEY_DOWN else 5))
            elif key == curses.KEY_HOME:
                scroll = 0
    try:
        curses.wrapper(loop)
    except KeyboardInterrupt:
        pass
