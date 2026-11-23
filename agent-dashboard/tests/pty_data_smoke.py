#!/usr/bin/env python3
"""Developer-only macOS/POSIX PTY with explicit synthetic data publications.

Exercises the real native viewer, not a model or harness dispatcher. Python is
only a developer harness; the product and data-write command require no runtime.
"""
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time

SKILL = Path(__file__).resolve().parents[1]
binary = str(Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="external-data-pty.", dir=SKILL / "target") as tmp:
    root = Path(tmp)
    data = root / "data.json"
    binding = dict(source="timer", path="/seconds", format="mm:ss")
    layout = dict(version=1, name="SYNTHETIC external data PTY",
                  data_sources=dict(timer=dict(type="json-file", path=str(data), stale_after_ms=2500)),
                  root=dict(kind="column", children=[
                      dict(id="timer", kind="text", size=7, value=binding),
                      dict(id="session", kind="session", size=9, title_value=binding),
                      dict(id="graph", kind="dag", dag=dict(version=1, nodes=[dict(id="timer", label_value=binding)]))]))
    env = dict(os.environ, TERM="xterm-256color", COLORTERM="truecolor", AGENT_DASHBOARD_HOME=str(root / "state"))
    subprocess.run([binary, "demo", "--json", json.dumps(layout)], env=env, check=True, stdout=subprocess.DEVNULL)
    before = subprocess.check_output([binary, "status", "--session", "DEMO-layout"], env=env)
    master, slave = pty.openpty()
    original = termios.tcgetattr(slave)
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 50, 170, 0, 0))
    process = subprocess.Popen([binary, "tui", "--session", "DEMO-layout"], env=env,
                               stdin=slave, stdout=slave, stderr=slave, start_new_session=True)
    output = bytearray()

    def drain(seconds=.1):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if select.select([master], [], [], .03)[0]:
                try:
                    output.extend(os.read(master, 65536))
                except OSError:
                    break

    def wait_for(text, start=0):
        deadline = time.monotonic() + 8
        while time.monotonic() < deadline:
            drain()
            assert process.poll() is None, output[-4000:]
            if text in output[start:]:
                return
        raise AssertionError(f"missing {text!r}: {output[-5000:]!r}")

    def publish(seconds):
        subprocess.run([binary, "data-write", "--file", str(data), "--json", json.dumps(dict(seconds=seconds))],
                       env=env, check=True)

    try:
        wait_for(b"file unavailable")
        assert b"External data" in output and b"SYNTHETIC" in output
        start = len(output)
        publish(92)
        wait_for(b"01:32", start)
        drain(.2)
        os.write(master, b"p")
        wait_for(b"viewer PAUSED")
        start = len(output)
        publish(31)
        drain(3.0)
        assert b"00:31" not in output[start:], "paused display consumed a newer file"
        assert b"stale (external data expired)" not in output[start:], "paused display continued its freshness clock"
        publish(17)
        os.write(master, b"p")
        wait_for(b"00:17", start)
        start = len(output)
        wait_for(b"stale (external data expired)", start)
        stage = root / "invalid.stage"
        stage.write_text("invalid JSON")
        stage.replace(data)
        start = len(output)
        wait_for(b"invalid or oversized JSON", start)
        publish(11)
        start = len(output)
        wait_for(b"00:11", start)
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 45, 150, 0, 0))
        process.send_signal(signal.SIGWINCH)
        drain(.3)
        os.write(master, b"q")
        assert process.wait(timeout=5) == 0
        restored = termios.tcgetattr(slave)
        original[3] &= ~getattr(termios, "PENDIN", 0)
        restored[3] &= ~getattr(termios, "PENDIN", 0)
        assert restored == original, "terminal not restored"
        after = subprocess.check_output([binary, "status", "--session", "DEMO-layout"], env=env)
        assert after == before, "external data/rendering rewrote layout or native telemetry"
    finally:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=5)
        os.close(master)
        os.close(slave)
print("PASS native POSIX PTY: external JSON -> text/title/DAG, missing/recovery, live values, pause/freshness freeze, stale/corrupt diagnostics, resume, resize/restore; layout and synthetic telemetry unchanged")
