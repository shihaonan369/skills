#!/usr/bin/env python3
"""Developer-only POSIX PTY smoke test. Not a product runtime dependency."""
import fcntl
import json
import os
import pty
import re
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time
from pathlib import Path

binary = str(Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="agent-dashboard-pty-") as root:
    env = dict(os.environ, AGENT_DASHBOARD_HOME=root, TERM="xterm-256color", COLORTERM="truecolor")
    layout = next((a.split("=", 1)[1] for a in sys.argv[2:] if a.startswith("--layout=")), None)
    demo_args = [binary, "demo"]
    if layout:
        demo_args.extend(["--json", "@" + str(Path(layout).resolve())])
    subprocess.run(demo_args, env=env, check=True, capture_output=True)
    master, slave = pty.openpty()
    original = termios.tcgetattr(slave)
    size = next((a.split("=", 1)[1] for a in sys.argv[2:] if a.startswith("--size=")), "110x32")
    width, height = map(int, size.split("x"))
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", height, width, 0, 0))
    process = subprocess.Popen([binary, "tui", "--session", "DEMO-layout"], env=env,
                               stdin=slave, stdout=slave, stderr=slave, start_new_session=True)
    output = bytearray()

    def drain(seconds=0.3):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if select.select([master], [], [], 0.05)[0]:
                try:
                    output.extend(os.read(master, 65536))
                except OSError:
                    break

    try:
        # A short forward edge exposes a decorative head only briefly each
        # cycle. Wait for actual startup/effect output instead of assuming a
        # fixed sampling window happens to include that phase.
        deadline = time.monotonic() + 8
        while time.monotonic() < deadline:
            drain(.2)
            if process.poll() is not None:
                break
            ready = b"DEMO" in output and b"SYNTHETIC" in output
            if "--effects" in sys.argv[2:]:
                ready = ready and "●".encode() in output and "┏".encode() in output
            if ready:
                break
        assert process.poll() is None, output.decode(errors="replace")
        assert b"DEMO" in output and b"SYNTHETIC" in output
        if layout:
            assert b"dag" in output and b"Declared" in output, "native DAG was not rendered"
            for obsolete in ("▲".encode(), "⇄".encode(), b"[return]"):
                assert obsolete not in output, "obsolete reverse flow was rendered"
            if "--effects" in sys.argv[2:]:
                assert "●".encode() in output, "decorative flow packets were not rendered"
                assert "┏".encode() in output, "heavy focus frame was not rendered"
        assert b"\x1b[?1049h" in output, "alternate screen was not entered"
        assert re.search(rb"\x1b\[[0-9;]*m", output), "color/style SGR output absent"
        if "--controls" in sys.argv[2:]:
            assert layout, "--controls needs a DAG/task composition (fanout recommended)"
            initial = json.loads(subprocess.check_output([binary, "status", "--session", "DEMO-layout"], env=env))
            # This fixture starts on the DAG. The graph leaf must be at least
            # 82 columns wide for its optional in-leaf sidebar.
            for key, expected in ((b"]", b":neighbors"), (b"f", b":upstream"),
                                  (b"i", b"DETAILS")):
                start = len(output)
                os.write(master, key)
                drain(0.3)
                assert expected in output[start:], f"DAG control {key!r} did not render {expected!r}"
            for key in (b"c", b"i"):  # clear selection and close sidebar
                os.write(master, key)
                drain(0.15)
            tasks = list(initial["tasks"].values())

            def plan(status):
                for task in tasks:
                    if task["id"] == "2":
                        task["status"] = status
                subprocess.run([binary, "plan", "--session", "DEMO-layout", "--json", json.dumps(tasks)],
                               env=env, check=True, capture_output=True)

            start = len(output)
            plan("completed")
            drain(0.9)
            assert "▸ #2 [completed]".encode() in output[start:], "changed task row did not echo"
            assert "Δ".encode() in output[start:], "static change summary absent"
            drain(1.5)
            start = len(output)
            os.write(master, b"a")
            drain(0.2)
            plan("in_progress")
            drain(0.9)
            assert "· #2 [in_progress]".encode() in output[start:], "motion-off lost static change marker"
            assert "▸".encode() not in output[start:], "motion-off retained transient emphasis"
            for key in (b"a", b"x", b"p"):  # motion on, clear summary, freeze display
                os.write(master, key)
                drain(0.15)
            plan("pending")
            drain(0.8)
            start = len(output)
            os.write(master, b"p")
            drain(0.9)
            assert b"#2 [pending]" in output[start:], "resume failed to adopt new baseline"
            assert "▸".encode() not in output[start:], "resume replayed accumulated changes"
            current = json.loads(subprocess.check_output([binary, "status", "--session", "DEMO-layout"], env=env))
            assert current["hook_count"] == initial["hook_count"], "viewer controls manufactured hooks"
            assert current["demo"] and current["active"] and not current["finished"]
        # Navigation, focus, independent scroll, pause/help and resume.
        for key in (b"\x1b[C", b"\x1b[D", b"\x1b[6~", b"\x1b[H", b"2", b"3", b"1", b"\t", b"\x1b[B", b"a", b"a", b"p", b"?", b"\x1b", b"p"):
            os.write(master, key)
            drain(0.1)
        # Exercise a real SIGWINCH resize, not just a render function.
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 14, 45, 0, 0))
        process.send_signal(signal.SIGWINCH)
        drain(0.5)
        os.write(master, b"\x03" if '--ctrl-c' in sys.argv[2:] else b"q")
        drain(0.4)
        assert process.wait(timeout=5) == 0
        drain()
        assert b"\x1b[?1049l" in output, "alternate screen was not left"
        restored = termios.tcgetattr(slave)
        # macOS sets PENDIN (pending-input redisplay) when returning to canonical
        # mode. It is kernel bookkeeping, not an altered input/echo configuration.
        pending = getattr(termios, "PENDIN", 0)
        restored[3] &= ~pending
        original[3] &= ~pending
        assert restored == original, f"terminal configuration was not restored: before={original!r} after={restored!r}"
        state = json.loads(subprocess.check_output([binary, "status", "--session", "DEMO-layout"], env=env))
        assert state["active"] is True, "viewer quit disabled collection"
        print("PASS: macOS POSIX PTY · " + ("native DAG, " if layout else "") + ("link focus/sidebar + task echo/motion-off/resume, " if "--controls" in sys.argv[2:] else "") + "alt screen/color output, keys, resize, clean quit, terminal restored, collection retained")
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()
        os.close(master)
        os.close(slave)
