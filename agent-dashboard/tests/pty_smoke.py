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
        drain(1.2)
        assert process.poll() is None, output.decode(errors="replace")
        assert b"DEMO" in output and b"SYNTHETIC" in output
        if layout:
            assert b"dag" in output and b"Declared" in output, "native DAG was not rendered"
            if "--effects" in sys.argv[2:]:
                assert "●".encode() in output, "decorative flow packets were not rendered"
                assert "┏".encode() in output, "heavy focus frame was not rendered"
            if "--round-trip" in sys.argv[2:]:
                assert "⇄".encode() in output and "▲".encode() in output, "reply direction indicators absent"
                assert b"report results" in output, "report-to-parent declaration absent"
        assert b"\x1b[?1049h" in output, "alternate screen was not entered"
        assert re.search(rb"\x1b\[[0-9;]*m", output), "color/style SGR output absent"
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
        print("PASS: macOS POSIX PTY · " + ("native DAG, " if layout else "") + "alt screen/color output, keys, resize, clean quit, terminal restored, collection retained")
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()
        os.close(master)
        os.close(slave)
