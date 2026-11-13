"""Public telemetry/CLI/render contracts; run with unittest discover."""
import concurrent.futures
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
import dashboard as d
import tui


class DashboardTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.env = patch.dict(os.environ, {"REX_DASHBOARD_HOME": self.temp.name + "/state", "HOME": self.temp.name})
        self.env.start()
        d.initialize("one", "Issue #12")

    def tearDown(self):
        self.env.stop()
        self.temp.cleanup()

    def cli(self, *args, input=None):
        return subprocess.run([sys.executable, str(ROOT / "dashboard.py"), *args], input=input,
                              capture_output=True, text=True, timeout=15)

    def hook(self, kind, **fields):
        d.ingest({"session_id": "one", "hook_event_name": kind, **fields})

    def actor(self, actor="lead"):
        return next(a for a in d.snapshot("one")["agents"] if a["id"] == actor)

    def test_global_hook_is_inert_for_inactive_sessions(self):
        d.ingest({"session_id": "inactive", "hook_event_name": "PreToolUse"})
        self.assertFalse(d.session_dir("inactive").exists())

    def test_session_isolation_and_safe_paths(self):
        d.initialize("../../escape", "Other")
        self.hook("SubagentStart", agent_id="a", agent_type="Explore")
        self.assertEqual(len(d.snapshot("../../escape")["agents"]), 1)
        self.assertEqual(d.session_dir("../../escape").parent, d.data_root())
        self.assertEqual(len(d.snapshot("one")["agents"]), 2)

    def test_lifecycle_and_permission_wait(self):
        self.hook("SubagentStart", agent_id="a", agent_type="Explore")
        self.hook("PermissionRequest", agent_id="a", tool_name="Bash")
        self.assertEqual(self.actor("a")["status"], "blocked")
        self.hook("PreToolUse", agent_id="a", tool_name="Read", tool_use_id="r")
        self.assertEqual(self.actor("a")["status"], "running")
        self.hook("SubagentStop", agent_id="a", agent_type="Explore")
        self.assertEqual(self.actor("a")["status"], "done")
        self.assertEqual(d.snapshot("one")["tools"], [])

    def test_parallel_tool_completion_does_not_clear_other_tool(self):
        self.hook("PreToolUse", tool_name="Read", tool_use_id="a")
        self.hook("PreToolUse", tool_name="Grep", tool_use_id="b")
        self.hook("PostToolUse", tool_name="Read", tool_use_id="a")
        self.assertEqual(self.actor()["current"], "Grep")
        self.assertEqual(self.actor()["tools"], 1)

    def test_tool_failure_does_not_claim_agent_failed_or_complete_plan(self):
        self.hook("PostToolUseFailure", tool_name="TaskUpdate", tool_input={"taskId": "1", "status": "completed"})
        state = d.snapshot("one")
        self.assertEqual(state["tasks"], [])
        self.assertEqual(self.actor()["status"], "running")
        self.assertEqual(state["events"][0]["kind"], "error")

    def test_taskcreate_update_and_dependencies(self):
        self.hook("PostToolUse", tool_name="TaskCreate", tool_input={"subject": "Fix it"}, tool_response={"task": {"id": "1", "subject": "Fix it"}})
        self.hook("PostToolUse", tool_name="TaskUpdate", tool_input={"taskId": "1", "status": "in_progress", "addBlockedBy": ["0"]})
        task = d.snapshot("one")["tasks"][0]
        self.assertEqual((task["id"], task["status"], task["blockedBy"]), ("1", "in_progress", ["0"]))
        self.hook("PostToolUse", tool_name="TaskUpdate", tool_input={"taskId": "1", "status": "deleted"})
        self.assertEqual(d.snapshot("one")["tasks"], [])

    def test_todowrite_replaces_plan(self):
        for todos in ([{"content": "Old", "status": "pending"}],
                      [{"content": "New", "status": "completed"}]):
            self.hook("PostToolUse", tool_name="TodoWrite", tool_input={"todos": todos})
        self.assertEqual([t["subject"] for t in d.snapshot("one")["tasks"]], ["New"])

    def test_stop_is_waiting_not_issue_completion(self):
        self.hook("Stop")
        self.assertEqual(self.actor()["status"], "waiting")
        self.hook("UserPromptSubmit")
        self.assertEqual(self.actor()["status"], "running")

    def test_finish_survives_its_own_posttool_and_stop_then_new_turn_resumes(self):
        result = self.cli("finish", "--session", "one", "--summary", "Checked")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.hook("PostToolUse", tool_name="Bash", tool_use_id="finish")
        self.hook("Stop")
        self.assertEqual(self.actor()["status"], "done")
        self.hook("UserPromptSubmit")
        self.assertEqual(self.actor()["status"], "running")

    def test_session_end_interrupts_active_agents(self):
        self.hook("SubagentStart", agent_id="a")
        self.hook("SessionEnd")
        self.assertEqual(self.actor("a")["status"], "interrupted")
        self.assertEqual(self.actor()["status"], "ended")

    def test_concurrent_hooks_are_not_lost(self):
        def work(i):
            self.hook("PostToolUse", agent_id=f"worker{i % 4}", tool_name="Read", tool_use_id=str(i))
        with concurrent.futures.ThreadPoolExecutor(max_workers=12) as executor:
            list(executor.map(work, range(80)))
        state = d.snapshot("one")
        self.assertEqual(state["meta"]["hook_count"], 80)
        self.assertEqual(sum(a.get("tools", 0) for a in state["agents"]), 80)

    def test_agent_display_order_is_stable_during_updates(self):
        self.hook("SubagentStart", agent_id="a")
        self.hook("SubagentStart", agent_id="b")
        self.hook("PostToolUse", agent_id="a", tool_name="Read")
        self.assertEqual([a["id"] for a in d.snapshot("one")["agents"]], ["lead", "a", "b"])

    def test_transcript_models_are_scoped_to_each_agent(self):
        main = Path(self.temp.name) / "projects/one.jsonl"
        sub = main.parent / "one/subagents/agent-a.jsonl"
        sub.parent.mkdir(parents=True)
        def message(model):
            return json.dumps({"type": "assistant", "message": {"model": model}}) + "\n"
        main.write_text(message("claude-sonnet-4-6"))
        sub.write_text(message("claude-haiku-4-5"))
        self.hook("SubagentStart", agent_id="a", transcript_path=str(main))
        self.assertEqual(self.actor()["model"], "claude-sonnet-4-6")
        self.assertEqual(self.actor("a")["model"], "claude-haiku-4-5")
        self.hook("SubagentStart", agent_id="unknown", transcript_path=str(main))
        self.assertNotIn("model", self.actor("unknown"))

    def test_hook_cli_fails_open_on_corrupt_payloads(self):
        for text in ("garbage", "[]", "{}", '{"session_id":"one","tool_input":42,"hook_event_name":"PreToolUse"}'):
            result = self.cli("hook", input=text)
            self.assertEqual((result.returncode, result.stdout, result.stderr), (0, "", ""))

    def test_plan_cli_is_atomic_on_bad_status(self):
        self.assertEqual(self.cli("plan", "--session", "one", "--json", '[{"subject":"Keep me"}]').returncode, 0)
        result = self.cli("plan", "--session", "one", "--json", '[{"subject":"Wrong","status":"imaginary"}]')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(d.snapshot("one")["tasks"][0]["subject"], "Keep me")

    def test_installer_preserves_existing_hooks_and_is_idempotent(self):
        settings = Path(self.temp.name) / ".claude/settings.json"
        settings.parent.mkdir(parents=True)
        original = {"someSetting": True, "hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "echo keep"}]}]}}
        settings.write_text(json.dumps(original))
        for _ in range(2):
            result = self.cli("install", "--settings", str(settings))
            self.assertEqual(result.returncode, 0, result.stderr)
        data = json.loads(settings.read_text())
        self.assertTrue(data["someSetting"])
        self.assertEqual(data["hooks"]["SessionStart"], original["hooks"]["SessionStart"])
        self.assertEqual(len(data["hooks"]["PreToolUse"]), 1)
        self.assertEqual(len(list(settings.parent.glob("*.bak"))), 1)
        result = self.cli("uninstall", "--settings", str(settings))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(settings.read_text()), original)

    def test_private_state_permissions(self):
        self.assertEqual(d.session_dir("one").stat().st_mode & 0o777, 0o700)
        self.assertEqual((d.session_dir("one") / "telemetry.sqlite3").stat().st_mode & 0o777, 0o600)

    def test_rex_flags_do_not_leak_into_dashboard_command(self):
        completed = subprocess.CompletedProcess([], 0, '{"block_ids":["new"]}', '')
        with patch.object(d.subprocess, "run", return_value=completed) as run:
            d.rex_json(["split", "--", sys.executable, "dashboard.py", "tui"])
            args = run.call_args.args[0]
            self.assertIn("--json", args[:args.index("--")])
            self.assertNotIn("--json", args[args.index("--") + 1:])
            d.rex_json(["block", "call", "process"])
            self.assertNotIn("--json", run.call_args.args[0])

    def test_panel_reuses_its_saved_live_block(self):
        with patch.dict(os.environ, {"REX_SESSION": "rex-session", "REX_BLOCK": "anchor"}), \
             patch.object(d.shutil, "which", return_value="/usr/bin/rex"), \
             patch.object(d, "rex_json", side_effect=[{"block_ids": ["panel"]}, {"block": {}}, {"last_exit": None}] ) as rex:
            first = d.open_panel("one")
            second = d.open_panel("one")
            self.assertEqual(first["block_id"], second["block_id"])
            self.assertTrue(second["reused"])
            self.assertEqual(sum(call.args[0][0] == "split" for call in rex.call_args_list), 1)

    def test_panel_connection_failure_does_not_create_duplicate(self):
        with patch.dict(os.environ, {"REX_SESSION": "rex-session", "REX_BLOCK": "anchor"}), \
             patch.object(d.shutil, "which", return_value="/usr/bin/rex"), \
             patch.object(d, "rex_json", return_value={"block_ids": ["panel"]}):
            d.open_panel("one")
        with patch.dict(os.environ, {"REX_SESSION": "rex-session", "REX_BLOCK": "anchor"}), \
             patch.object(d.shutil, "which", return_value="/usr/bin/rex"), \
             patch.object(d, "rex_json", side_effect=RuntimeError("connection timed out")) as rex:
            with self.assertRaises(RuntimeError):
                d.open_panel("one")
            self.assertEqual(rex.call_count, 1)

    def test_timed_out_creation_is_reconciled_before_retry(self):
        context = {"REX_SESSION": "rex-session", "REX_BLOCK": "anchor"}
        with patch.dict(os.environ, context), patch.object(d.shutil, "which", return_value="rex"), \
             patch.object(d, "rex_json", side_effect=subprocess.TimeoutExpired("rex", 20)):
            with self.assertRaises(subprocess.TimeoutExpired):
                d.open_panel("one")
        label = d.snapshot("one")["meta"]["panel_pending"]["label"]
        with patch.dict(os.environ, context), patch.object(d.shutil, "which", return_value="rex"), \
             patch.object(d, "rex_json", side_effect=[{"blocks": [{"block_id": "arrived-late", "label": label}]}, {"last_exit": None}]) as rex:
            recovered = d.open_panel("one")
            self.assertEqual(recovered["block_id"], "arrived-late")
            self.assertTrue(recovered["reused"])
            self.assertFalse(any(c.args[0][0] == "split" for c in rex.call_args_list))

    def test_responsive_renderer_unicode_and_untrusted_text(self):
        self.hook("SubagentStart", agent_id="a", agent_type="Search agent 👩‍💻\x1b]52;clipboard\x07")
        state = d.snapshot("one")
        for rows in (5, 12, 24, 40, 60):
            for cols in (20, 32, 48, 70, 96, 150):
                for mode in ({}, {"logs_only": True}, {"help_on": True}):
                    lines = tui.render(state, rows, cols, **mode).lines()
                    self.assertEqual(len(lines), rows)
                    self.assertTrue(all(tui.width(line) <= cols for line in lines))
                    self.assertTrue(all("\x1b" not in line and "\x07" not in line for line in lines))

    def test_short_panels_still_show_real_subagents(self):
        self.hook("SubagentStart", agent_id="worker-a", agent_type="Explore")
        self.assertIn("Explore", "\n".join(tui.render(d.snapshot("one"), 24, 70).lines()))

    def test_unknown_progress_and_model_are_not_fabricated(self):
        text = tui.plain(d.snapshot("one"))
        self.assertIn("progress unknown", text)
        self.assertIn("model unknown", text)
        self.assertNotIn("sonnet", text.lower())


if __name__ == "__main__":
    unittest.main()
