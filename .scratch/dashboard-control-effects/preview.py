#!/usr/bin/env python3
"""Developer-only local demo launcher. NOT shipped, not a product dependency.
Synthetic tasks update every 5 seconds; no real Claude hooks or sessions.
"""
import json
import os
from pathlib import Path
import signal
import subprocess
import tempfile
import threading

repo = Path(__file__).resolve().parents[2]
skill = repo / "agent-dashboard"
binary = skill / "bin/darwin-arm64/agent-dashboard"
stop = threading.Event()
viewer = None


def interrupted(signum, frame):
    raise KeyboardInterrupt


for sig in (signal.SIGTERM, signal.SIGHUP):
    signal.signal(sig, interrupted)

with tempfile.TemporaryDirectory(prefix="native-preview.", dir=Path(__file__).parent) as root:
    env = dict(os.environ, AGENT_DASHBOARD_HOME=root)
    subprocess.run([str(binary), "demo", "--json", "@" + str(skill / "examples/dag/fanout.layout.json")],
                   env=env, check=True, stdout=subprocess.DEVNULL)
    tasks = json.loads(subprocess.check_output([str(binary), "status", "--session", "DEMO-layout"], env=env))["tasks"]

    def publish():
        completed = True
        while not stop.wait(5):
            tasks["2"]["status"] = "completed" if completed else "in_progress"
            result = subprocess.run([str(binary), "plan", "--session", "DEMO-layout", "--json", json.dumps(list(tasks.values()))],
                                    env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if result.returncode:
                break
            completed = not completed

    worker = threading.Thread(target=publish, daemon=True)
    worker.start()
    try:
        viewer = subprocess.Popen([str(binary), "tui", "--session", "DEMO-layout"], env=env)
        viewer.wait()
    except KeyboardInterrupt:
        pass
    finally:
        stop.set()
        worker.join(timeout=5)
        if viewer is not None and viewer.poll() is None:
            viewer.terminate()
            try:
                viewer.wait(timeout=3)
            except subprocess.TimeoutExpired:
                viewer.kill()
                viewer.wait()
