#!/usr/bin/env python3
"""agent-dashboard DAG view · caller-declared orchestration graph.

Local extension to the agent-dashboard skill: renders a caller-authored DAG
(nodes + edges, explicitly declared, never inferred) and binds node states to
real observed telemetry via `agent-dashboard status`. Changing the DAG changes
presentation only; it never starts/stops agents or tasks.

Usage:
  dag-view.py --dag FILE [--session ID] [--interval 1.0] [--once]

Schema: see docs/DAG.md. Edges between adjacent ranks render inline; edges with
"style":"return" route along the right margin and never affect ranking.
"""
import argparse, json, os, platform, shutil, subprocess, sys, threading, time

RESET = "\x1b[0m"; BOLD = "\x1b[1m"; DIM = "\x1b[2m"
def FG(c): return f"\x1b[38;2;{c[0]};{c[1]};{c[2]}m"

PALETTE = {
    "orange": (240,170,110), "blue": (110,168,216), "green": (126,201,143),
    "purple": (155,142,196), "teal": (94,197,207), "gray": (139,148,158),
    "red": (224,106,106), "amber": (228,180,105), "white": (230,237,243),
}

W, H = shutil.get_terminal_size((100, 34))
CELLS = {}

def put(x, y, s, color=None, style=""):
    if y < 0 or y >= H: return
    for i, ch in enumerate(s):
        if 0 <= x + i < W:
            CELLS[(x + i, y)] = (ch, color, style)

# ---------------- telemetry ----------------
STATUS = {"data": None, "stale": False}
def bin_path():
    m = platform.machine().lower()
    arch = "arm64" if "arm64" in m or "aarch64" in m else "amd64"
    name = "agent-dashboard.exe" if os.name == "nt" else "agent-dashboard"
    p = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
                     "bin", f"{platform.system().lower()}-{arch}", name)
    return p

def poll_loop(binpath, sid, interval, dag_path=None, on_reload=None):
    global STATUS
    while True:
        try:
            r = subprocess.run([binpath, "status", "--session", sid],
                               capture_output=True, timeout=3)
            STATUS["data"] = json.loads(r.stdout)
            STATUS["stale"] = False
        except Exception:
            STATUS["stale"] = True
        if dag_path and on_reload:
            try: on_reload(dag_path)
            except BaseException: pass
        time.sleep(interval)

def node_state(node, t):
    """Return (text, state_color, active) from bound telemetry, honestly."""
    d = STATUS["data"]
    bind = node.get("bind") or {}
    spin = "◐◓◑◒"[int(t * 8) % 4]
    if bind.get("actor"):
        a = (d or {}).get("agents", {}).get(bind["actor"])
        if not a: return ("declared · unobserved", PALETTE["gray"], False)
        if a.get("status") == "running":
            return (f"{spin} running · tools {a.get('tools',0)} · err {a.get('errors',0)}",
                    PALETTE["orange"], True)
        return (f"✓ {a.get('status','stopped')} · tools {a.get('tools',0)}",
                PALETTE["green"], False)
    if bind.get("task"):
        k = (d or {}).get("tasks", {}).get(bind["task"])
        if not k: return ("declared · unobserved", PALETTE["gray"], False)
        s = k.get("status")
        if s == "completed": return ("✓ completed", PALETTE["green"], False)
        if s == "in_progress": return (f"{spin} in_progress", PALETTE["orange"], True)
        if s == "blocked": return ("⛔ blocked", PALETTE["red"], False)
        if s == "deleted": return ("✗ deleted", PALETTE["gray"], False)
        return ("○ pending", PALETTE["gray"], False)
    return ("declared only", PALETTE["gray"], False)

# ---------------- dag model ----------------
def load_dag(path):
    try:
        with open(path, encoding="utf-8") as f:
            g = json.load(f)
    except FileNotFoundError:
        sys.exit(f"dag error: file not found: {path}")
    except json.JSONDecodeError as ex:
        sys.exit(f"dag error: invalid JSON in {path}: {ex}")
    if g.get("version") not in (None, 1):
        sys.exit(f"dag error: unsupported schema version {g.get('version')!r} (expected 1)")
    nodes = g.get("nodes") or []
    ids = [n.get("id") for n in nodes]
    if len(ids) != len(set(ids)) or not ids:
        sys.exit("dag error: node ids must be unique and nonempty")
    by_id = {n["id"]: n for n in nodes}
    edges = g.get("edges") or []
    for e in edges:
        if e.get("from") not in by_id or e.get("to") not in by_id:
            sys.exit(f"dag error: edge {e} references unknown node")
    return g, nodes, edges

def ranks_of(nodes, edges):
    fwd = [e for e in edges if e.get("style") != "return"]
    preds = {n["id"]: [] for n in nodes}
    succs = {n["id"]: [] for n in nodes}
    for e in fwd:
        preds[e["to"]].append(e["from"]); succs[e["from"]].append(e["to"])
    rank, deg, q = {}, {i: len(preds[i]) for i in preds}, [n["id"] for n in nodes if not preds[n["id"]]]
    for i in q: rank[i] = 0
    while q:
        u = q.pop()
        for v in succs[u]:
            rank[v] = max(rank.get(v, 0), rank[u] + 1)
            deg[v] -= 1
            if deg[v] == 0: q.append(v)
    cyclic = [i for i in deg if deg[i] > 0]
    if cyclic:
        sys.exit(f"dag error: cycle among non-return edges through {cyclic}; "
                 "mark the closing edge \"style\":\"return\"")
    return rank

def layout(nodes, edges, rank, t):
    """Assign boxes per rank; adaptive to current W/H."""
    by_rank = {}
    for n in nodes: by_rank.setdefault(rank[n["id"]], []).append(n)
    ranks = sorted(by_rank)
    d = STATUS["data"]
    boxes = {}
    max_right = 0
    for r in ranks:
        y_needed = 3  # minimum: border+state+border
        for n in by_rank[r]:
            meta = n.get("meta") or []
            w = max(len(n.get("label", n["id"])) + 2,
                    max((len(m) for m in meta), default=0) + 2,
                    len("declared · unobserved") + 2, 12)
            w = min(w, 26)
            boxes[n["id"]] = {"w": w, "meta": meta}
            max_right = max(max_right, w)
    # vertical budget: try full meta, shed meta, then shrink lanes
    lane_h = 2
    def total_h(with_meta):
        th = 0
        for r in ranks:
            hh = max(3 + (max(len(boxes[n["id"]]["meta"]) for n in by_rank[r]) if with_meta else 0), 3)
            th += hh + lane_h
        return th - lane_h
    budget = H - 3 - 1  # header + status
    with_meta = total_h(True) <= budget
    if not with_meta and total_h(False) > budget: lane_h = 1
    y = 2
    for r in ranks:
        row = by_rank[r]
        hmax = 3
        for n in row:
            m = boxes[n["id"]]["meta"] if with_meta else []
            boxes[n["id"]]["h"] = 3 + len(m)
            boxes[n["id"]]["draw_meta"] = m
            hmax = max(hmax, boxes[n["id"]]["h"])
        tw = sum(boxes[n["id"]]["w"] for n in row) + 2 * (len(row) - 1)
        x = max(0, (W - tw) // 2)
        for n in row:
            b = boxes[n["id"]]
            b["x"], b["y"], b["rank_h"] = x, y, hmax
            x += b["w"] + 2
        y += hmax + lane_h
    graph_bottom = y - lane_h
    return boxes, ranks, by_rank, graph_bottom, lane_h

def draw_node(n, b, t):
    color = PALETTE.get(n.get("color", "gray"))
    state, scolor, active = node_state(n, t)
    style = BOLD if active else ""
    put(b["x"], b["y"], "╭" + "─" * (b["w"] - 2) + "╮", color, style)
    for i in range(1, b["h"] - 1):
        put(b["x"], b["y"] + i, "│", color, style); put(b["x"] + b["w"] - 1, b["y"] + i, "│", color, style)
    put(b["x"], b["y"] + b["h"] - 1, "╰" + "─" * (b["w"] - 2) + "╯", color, style)
    put(b["x"] + 2, b["y"], f" {n.get('label', n['id'])} "[:b["w"] - 3], color, BOLD)
    put(b["x"] + 2, b["y"] + 1, state[:b["w"] - 3], scolor, BOLD if active else "")
    for j, m in enumerate(b.get("draw_meta", [])):
        put(b["x"] + 2, b["y"] + 2 + j, m[:b["w"] - 3], PALETTE["gray"], DIM)

def draw_edge(e, boxes, lane_h, t, idx):
    s, d = boxes[e["from"]], boxes[e["to"]]
    ret = e.get("style") == "return"
    dashed = ret or e.get("style") == "dashed"
    color = PALETTE.get(nodes_by_id[e["from"]].get("color", "gray"))
    path = []
    if ret:
        # right-margin route: src right → margin → up → dst right
        mx = W - 3
        sy = s["y"] + 1; dy = d["y"] + 1
        for x in range(s["x"] + s["w"], mx): path.append((x, sy))
        step = -1 if dy < sy else 1
        for y in range(sy, dy, step): path.append((mx, y))
        for x in range(mx, d["x"] + d["w"], -1): path.append((x, dy))
        for (x, y) in path:
            ch = "┆" if x == mx else "·"
            put(x, y, ch, color, DIM)
        if path:
            p = int((t * 9 + idx * 3) % len(path))
            put(*path[p], "●", color, BOLD)
        put(d["x"] + d["w"], dy, "◀", color, BOLD)
        lbl = e.get("label")
        if lbl: put(mx - len(lbl) - 1, (sy + dy) // 2, lbl, color, DIM)
        return
    # adjacent-rank inline route (from the node's actual bottom border)
    sx, sy = s["x"] + s["w"] // 2, s["y"] + s["h"] - 1
    tx, ty = d["x"] + d["w"] // 2, d["y"]
    lane = sy + 1
    for y in range(sy + 1, lane + lane_h): path.append((sx, y))
    lo, hi = min(sx, tx), max(sx, tx)
    for x in range(lo, hi + 1): path.append((x, lane + lane_h - 1))
    for y in range(lane + lane_h, ty): path.append((tx, y))
    hv, vv = ("─", "│") if not dashed else ("·", "┆")
    for (x, y) in path:
        if y == lane + lane_h - 1 and x not in (sx, tx): put(x, y, hv, color, DIM)
        elif x in (sx, tx): put(x, y, vv, color, DIM)
    if path:
        p = int((t * 9 + idx * 3) % len(path))
        put(*path[p], "●", color, BOLD)
    put(tx, ty - 1, "▼", color, BOLD)
    lbl = e.get("label")
    if lbl and hi - lo > len(lbl) + 2:
        put((lo + hi - len(lbl)) // 2, lane, lbl, color, DIM)

nodes_by_id = {}

def render(g, nodes, edges, sid, t):
    global W, H, CELLS
    W, H = shutil.get_terminal_size((100, 34))
    CELLS = {}
    d = STATUS["data"]
    # header
    put(0, 0, "═" * W, PALETTE["gray"], DIM)
    badge = "DECLARED ONLY · NO SESSION"
    if sid: badge = ("DEMO · SYNTHETIC" if (d or {}).get("demo") else "LIVE TELEMETRY") + (f" · {sid[:8]}" if sid else "")
    title = f"◆ DAG · {g.get('name','unnamed')} · {badge}"
    if len(title) > W: title = title[:W - 4] + "…" + title[-3:]
    put(max(0, (W - len(title)) // 2), 0, title, PALETTE["amber"], BOLD)
    put(1, 1, "edges caller-declared · states from observed telemetry", PALETTE["gray"], DIM)
    if STATUS["stale"]: put(W - 14, 1, "stale!", PALETTE["red"], BOLD)

    rank = ranks_of(nodes, edges)
    boxes, ranks, by_rank, bottom, lane_h = layout(nodes, edges, rank, t)
    # edges first, nodes drawn on top
    for i, e in enumerate(edges): draw_edge(e, boxes, lane_h, t, i)
    for n in nodes: draw_node(n, boxes[n["id"]], t)

    # legend
    ly = min(H - 2, bottom + 1)
    used = []
    for n in nodes:
        c = n.get("color", "gray")
        if c not in used: used.append(c)
    lx = 1
    for c in used:
        put(lx, ly, f"■ {c}", PALETTE[c]); lx += len(c) + 5
    put(lx + 1, ly, "● packet · declared flow", PALETTE["amber"], DIM)

    # log strip
    avail = H - 1 - (ly + 1)
    if avail >= 3 and d:
        put(0, ly + 1, "─" * W, PALETTE["gray"], DIM)
        actors = {(n.get("bind") or {}).get("actor") for n in nodes} - {None}
        evs = [e for e in (d.get("events") or []) if not actors or e.get("actor") in actors]
        put(2, ly + 1, f" events · {len(evs)} observed ", PALETTE["gray"])
        for i, e in enumerate(evs[-(avail - 1):]):
            yy = ly + 2 + i
            ts = time.strftime("%H:%M:%S", time.localtime(e.get("time", 0) / 1000))
            put(2, yy, ts, PALETTE["gray"], DIM)
            ac = PALETTE["orange"] if e.get("actor") == "lead" else PALETTE["blue"]
            put(11, yy, f"{e.get('actor','?'):<10}", ac, BOLD)
            put(22, yy, str(e.get("text", ""))[:W - 26], PALETTE["white"])

    # status
    sy = H - 1
    put(1, sy, "q quit · DAG is presentation only", PALETTE["gray"], DIM)
    if d:
        right = f"hooks {d.get('hook_count','?')} · rev {d.get('revision','?')} · {'demo' if d.get('demo') else 'live'}"
        put(W - len(right) - 1, sy, right, PALETTE["amber"], BOLD)

def frame_to_stdout():
    out = ["\x1b[H"]
    for y in range(H):
        row = []; last = None
        for x in range(W):
            ch, color, style = CELLS.get((x, y), (" ", None, ""))
            key = (color, style)
            if key != last:
                row.append(RESET)
                if style: row.append(style)
                if color: row.append(FG(color))
                last = key
            row.append(ch)
        out.append("".join(row) + RESET + "\x1b[K")
        if y < H - 1: out.append("\n")
    sys.stdout.write("".join(out)); sys.stdout.flush()

def main():
    global nodes_by_id
    ap = argparse.ArgumentParser()
    ap.add_argument("--dag", required=True)
    ap.add_argument("--session")
    ap.add_argument("--interval", type=float, default=1.0)
    ap.add_argument("--once", action="store_true")
    args = ap.parse_args()
    g, nodes, edges = load_dag(args.dag)
    nodes_by_id = {n["id"]: n for n in nodes}
    if args.session:
        def reload_dag(path):
            global g, nodes, edges, nodes_by_id
            g2, n2, e2 = load_dag(path)
            g, nodes, edges, nodes_by_id = g2, n2, e2, {n["id"]: n for n in n2}
        threading.Thread(target=poll_loop,
                         args=(bin_path(), args.session, args.interval,
                               args.dag, reload_dag), daemon=True).start()
        time.sleep(0.4)
    sys.stdout.write("\x1b[?25l\x1b[2J")
    t0 = time.time()
    try:
        while True:
            render(g, nodes, edges, args.session, time.time() - t0)
            frame_to_stdout()
            if args.once: break
            time.sleep(0.09)
    except KeyboardInterrupt:
        pass
    finally:
        sys.stdout.write("\x1b[?25h\x1b[0m\x1b[2J\x1b[H"); sys.stdout.flush()

if __name__ == "__main__":
    main()
