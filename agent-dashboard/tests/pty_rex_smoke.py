#!/usr/bin/env python3
"""Developer-only POSIX PTY check with a labelled synthetic CLI process.

This does not contact a Rex server or validate native Linux/Windows execution.
Python is a test harness, not a shipped dashboard dependency.
"""
import fcntl
import hashlib
import json
import os
from pathlib import Path
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

SKILL = Path(__file__).resolve().parents[1]
binary = str(Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="rex-pty.", dir=SKILL / "target") as tmp:
    root = Path(tmp).resolve()
    fixture = root / "synthetic-rex"
    fixture.write_text(f"#!{sys.executable}\n" + r'''
import json
import os
from pathlib import Path
import time
root = Path(__file__).parent
(root / "subscription.pid").write_text(str(os.getpid()))
def emit(seq, kind, **fields):
    print(json.dumps(dict(agent_dashboard_rex=1, host_id="synthetic-host",
        instance_id="synthetic-instance", seq=seq, type=kind, **fields)), flush=True)
pane = dict(session_id="synthetic-session", session_label="Synthetic PTY session",
    block_id="synthetic-block", block_label="Synthetic fixture",
    records=[dict(state="working", app="synthetic-app", progress=0, msg="Initial synthetic report")])
emit(1, "hello")
emit(2, "snapshot", panes=[pane])
seq = 2
while True:
    if (root / "change").exists():
        (root / "change").unlink()
        seq += 1
        pane["records"] = [dict(state="blocked", app="synthetic-app", kind="permission",
            progress=42, msg="Changed synthetic report")]
        emit(seq, "pane", pane=pane)
    time.sleep(.05)
''', encoding="utf-8")
    fixture.chmod(0o700)
    source = root / "sources.json"
    source.write_text(json.dumps(dict(version=1, sources=[dict(name="Synthetic PTY source",
        endpoint="unix:///synthetic-pty-fixture", host_id="synthetic-host")])), encoding="utf-8")
    identity = json.dumps(["synthetic-host", "synthetic-session", "synthetic-block", ""], separators=(",", ":"))
    key = "rex:" + hashlib.sha256(identity.encode()).hexdigest()
    template = (SKILL / "examples/rex/records.template.layout.json").read_text(encoding="utf-8")
    layout = root / "layout.json"
    layout.write_text(template.replace("rex:" + "0" * 64, key), encoding="utf-8")
    env = dict(os.environ, AGENT_DASHBOARD_HOME=str(root / "state"), TERM="xterm-256color", COLORTERM="truecolor")
    master, slave = pty.openpty()
    original = termios.tcgetattr(slave)
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 64, 180, 0, 0))
    process = subprocess.Popen([binary, "rex-watch", "--sources", "@" + str(source),
        "--layout", "@" + str(layout), "--rex-bin", str(fixture)], env=env,
        stdin=slave, stdout=slave, stderr=slave, start_new_session=True)
    output = bytearray()

    def drain(seconds=.3):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if select.select([master], [], [], .05)[0]:
                try:
                    output.extend(os.read(master, 65536))
                except OSError:
                    break

    def alive(pid):
        try:
            os.kill(pid, 0)
            return True
        except ProcessLookupError:
            return False

    try:
        drain(1.5)
        assert process.poll() is None, output.decode(errors="replace")
        for text in (b"REX PROGRAM REPORTS", b"Rex observation scope", b"Rex program report",
                     b"Initial synthetic report", b"Local Rex observation history", b"baseline", b"working"):
            assert text in output, (text, output.decode(errors="replace"))
        for kind in (b"session", b"agent", b"agents", b"tasks", b"events", b"sources", b"programs", b"dag", b"text"):
            assert b" \xc2\xb7 " + kind in output, (kind, output.decode(errors="replace"))
        assert b"\x1b[?1049h" in output
        assert re.search(rb"\x1b\[[0-9;]*m", output)
        pid = int((root / "subscription.pid").read_text())
        start = len(output)
        (root / "change").touch()
        drain(1)
        changed = output[start:]
        assert b"Changed synthetic report" in changed and b"blocked" in changed, changed.decode(errors="replace")
        assert "▸".encode() in changed and "Δ".encode() in changed, "shared record echo missing"
        for keypress in (b"2", b"3", b"1", b"\t", b"\t", b"\t", b"\t", b"\t", b"]", b"i", b"p", b"?", b"\x1b", b"p"):
            os.write(master, keypress)
            drain(.08)
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 14, 45, 0, 0))
        process.send_signal(signal.SIGWINCH)
        drain(.4)
        os.write(master, b"\x03" if "--ctrl-c" in sys.argv else b"q")
        drain(.4)
        assert process.wait(timeout=5) == 0
        assert b"\x1b[?1049l" in output
        restored = termios.tcgetattr(slave)
        pending = getattr(termios, "PENDIN", 0)
        restored[3] &= ~pending
        original[3] &= ~pending
        assert restored == original, "terminal settings not restored"
        assert not alive(pid), "owned synthetic CLI child was not reaped"
        print("PASS: native POSIX PTY with synthetic Rex CLI; all panel shapes, live report/echo, keys, resize, terminal restoration, child reaping")
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()
        pid_path = root / "subscription.pid"
        if pid_path.exists():
            pid = int(pid_path.read_text())
            if alive(pid):
                os.kill(pid, signal.SIGKILL)
        os.close(master)
        os.close(slave)
