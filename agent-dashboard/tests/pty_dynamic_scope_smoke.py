#!/usr/bin/env python3
"""Developer-only PTY: synthetic Rex child and synthetic classifier replies.

This tests the mailbox/selection/UI transport, not a real model or native harness
background-subagent dispatcher. Add --stream to feed both initial and later
requests through one scope-watch process (a stand-in for Monitor's event source).
Python is not a product runtime dependency.
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
with tempfile.TemporaryDirectory(prefix="dynamic-scope-pty.", dir=SKILL / "target") as tmp:
    root = Path(tmp)
    fixture = root / "synthetic-rex"
    fixture.write_text(f"#!{sys.executable}\n" + r'''
import json, os, time
from pathlib import Path
root = Path(__file__).parent
(root / "subscription.pid").write_text(str(os.getpid()))
def emit(seq, kind, **fields):
    print(json.dumps(dict(agent_dashboard_rex=1, host_id="synthetic-host",
        instance_id="synthetic-instance", seq=seq, type=kind, **fields)), flush=True)
pane = dict(session_id="synthetic-session", session_label="Synthetic session",
    block_id="synthetic-block", block_label="Synthetic block", records=[
    dict(id="first", state="working", app="synthetic", title="First implement fixture")])
emit(1, "hello")
emit(2, "snapshot", panes=[pane])
changed = False
third = False
while True:
    if not changed and (root / "add").exists():
        changed = True
        pane["records"] += [
            dict(id="second", state="working", app="synthetic", title="Second implement fixture"),
            dict(id="review", state="working", app="synthetic", title="Excluded review fixture")]
        emit(3, "pane", pane=pane)
    if changed and not third and (root / "third").exists():
        third = True
        pane["records"].append(dict(id="third", state="working", app="synthetic", title="Third implement fixture"))
        emit(4, "pane", pane=pane)
    time.sleep(.05)
''')
    fixture.chmod(0o700)
    sources = root / "sources.json"
    sources.write_text(json.dumps(dict(version=1, sources=[dict(name="Synthetic source",
        endpoint="synthetic-endpoint", host_id="synthetic-host")])))
    scope = root / "scope.json"
    subprocess.run([binary, "scope-init", "--file", str(scope), "--criterion",
                    "Synthetic test: include implement fixture titles; exclude review fixture titles."], check=True)
    q = dict(kind="programs", scope_file=str(scope))
    layout = root / "layout.json"
    layout.write_text(json.dumps(dict(version=1, name="SYNTHETIC dynamic scope PTY", root=dict(
        kind="column", children=[
            dict(id="graph", kind="dag", size=19, dag=dict(version=1, nodes=[], select=q)),
            dict(id="cards", kind="agent", size=13, bind=dict(item=True),
                 repeat=dict(select=q, direction="row")),
            dict(id="rows", kind="tasks", size=14, select=q),
            dict(id="programs", kind="programs", size=10, select=q, compact=True),
            dict(id="sources", kind="sources", size=8, select=dict(kind="sources")),
            dict(id="notes", kind="text", size=8, text=["Caller-authored synthetic note"],
                 repeat=dict(select=q, direction="row"))]))))
    master, slave = pty.openpty()
    original = termios.tcgetattr(slave)
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 90, 200, 0, 0))
    env = dict(os.environ, TERM="xterm-256color", COLORTERM="truecolor",
               AGENT_DASHBOARD_HOME=str(root / "state"))
    process = subprocess.Popen([binary, "rex-watch", "--sources", "@" + str(sources),
                                "--layout", "@" + str(layout), "--rex-bin", str(fixture)],
                               stdin=slave, stdout=slave, stderr=slave, env=env, start_new_session=True)
    output = bytearray()
    watcher = None

    def drain(seconds=.1):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if select.select([master], [], [], .03)[0]:
                try:
                    chunk = os.read(master, 65536)
                    if chunk:
                        output.extend(chunk)
                except OSError:
                    break

    def wait_for(predicate, description):
        deadline = time.monotonic() + 12
        while time.monotonic() < deadline:
            drain()
            if process.poll() is not None:
                raise AssertionError(f"viewer exited before {description}: {output[-4000:]!r}")
            if predicate():
                return
        raise AssertionError(f"timeout: {description}: {output[-5000:]!r}")

    def next_stream_batch():
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            drain(.05)
            if select.select([watcher.stdout], [], [], .05)[0]:
                line = watcher.stdout.readline()
                assert line, "scope-watch exited without the expected event"
                return json.loads(line)
        raise AssertionError("scope-watch event did not arrive")

    def answer_pending(expected):
        # Explicit stand-in for a native worker's tool calls; no LLM is invoked.
        if watcher is None:
            batch = json.loads(subprocess.check_output([binary, "scope-poll", "--file", str(scope), "--wait", "5"], text=True))
        else:
            batch = next_stream_batch()
            while not batch["closed"] and not batch["requests"]:
                batch = next_stream_batch()  # A single empty startup/ready event.
            assert not batch["closed"], batch
        assert len(batch["requests"]) == expected, batch
        answers = dict(policy_hash=batch["policy_hash"], decisions=[dict(
            key=r["key"], fingerprint=r["fingerprint"],
            decision="include" if "implement" in r["label"] else "exclude", origin="subagent",
            evidence="Synthetic classifier reply; not an actual model judgement.") for r in batch["requests"]])
        subprocess.run([binary, "scope-decide", "--file", str(scope), "--json", json.dumps(answers)], check=True)

    try:
        if "--stream" in sys.argv[2:]:
            watcher = subprocess.Popen([binary, "scope-watch", "--file", str(scope),
                                        "--duration", "60"], stdout=subprocess.PIPE,
                                       stderr=subprocess.PIPE, bufsize=0, env=env)
        wait_for(lambda: b"1 pending" in output, "initial unclassified waiting view")
        assert b"First implement fixture" not in output
        answer_pending(1)
        wait_for(lambda: b"First implement fixture" in output, "first included record")
        (root / "add").touch()
        wait_for(lambda: b"2 pending" in output, "new entities discovered without layout rewrite")
        assert b"Second implement fixture" not in output
        answer_pending(2)
        wait_for(lambda: b"Second implement fixture" in output, "new member in the live canvas")
        drain(.8)
        assert b"Excluded review fixture" not in output
        assert b"Caller-authored synthetic note" in output
        assert b"Agent-classified scope" in output
        os.write(master, b"p")
        drain(.3)
        (root / "third").touch()
        answer_pending(1)
        drain(.7)
        assert b"Third implement fixture" not in output, "pause changed the displayed membership"
        os.write(master, b"p")
        wait_for(lambda: b"Third implement fixture" in output, "scope discovery continued while the view was paused")
        os.write(master, b"\t]ipa")
        drain(.2)
        os.write(master, b"p")
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 80, 180, 0, 0))
        os.kill(process.pid, signal.SIGWINCH)
        drain(.3)
        os.write(master, b"q")
        process.wait(timeout=5)
        drain(.1)
        assert process.returncode == 0
        restored = termios.tcgetattr(slave)
        original[3] &= ~getattr(termios, "PENDIN", 0)
        restored[3] &= ~getattr(termios, "PENDIN", 0)
        assert restored == original, "terminal not restored"
        pid = int((root / "subscription.pid").read_text())
        try:
            os.kill(pid, 0)
        except ProcessLookupError:
            pass
        else:
            raise AssertionError("owned synthetic Rex child survived")
        subprocess.run([binary, "scope-close", "--file", str(scope)], check=True)
        assert json.loads(subprocess.check_output([binary, "scope-poll", "--file", str(scope)], text=True))["closed"]
        if watcher is not None:
            closed = next_stream_batch()
            assert closed["closed"] and not closed["requests"], closed
            watcher.wait(timeout=5)
            assert watcher.returncode == 0, watcher.stderr.read()
    finally:
        if watcher is not None:
            if watcher.poll() is None:
                watcher.terminate()
                watcher.wait(timeout=5)
            watcher.stdout.close()
            watcher.stderr.close()
        if process.poll() is None:
            os.killpg(process.pid, signal.SIGTERM)
            process.wait(timeout=5)
        os.close(master)
        os.close(slave)
print("PASS macOS PTY: live dynamic graph/cards/lists/notes, " +
      ("one scope-watch stream for initial/later/paused candidates and close" if "--stream" in sys.argv[2:] else "bounded scope-poll transport") +
      ", pending/include/exclude, controls/resize/restore, child cleanup; synthetic replies, no real LLM or harness subagent")
