package dashboard

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type Payload struct {
	Session         string                     `json:"session_id"`
	Kind            string                     `json:"hook_event_name"`
	AgentID         string                     `json:"agent_id"`
	AgentType       string                     `json:"agent_type"`
	Transcript      string                     `json:"transcript_path"`
	AgentTranscript string                     `json:"agent_transcript_path"`
	ToolID          string                     `json:"tool_use_id"`
	ToolName        string                     `json:"tool_name"`
	Input           map[string]json.RawMessage `json:"tool_input"`
	Response        json.RawMessage            `json:"tool_response"`
	Model           string                     `json:"model"`
	ToModel         string                     `json:"to_model"`
}

func str(m map[string]json.RawMessage, k string) string {
	var s string
	if json.Unmarshal(m[k], &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(m[k], &n) == nil {
		return n.String()
	}
	return ""
}
func list(m map[string]json.RawMessage, k string) []string {
	var a []string
	_ = json.Unmarshal(m[k], &a)
	if len(a) > 128 {
		a = a[:128]
	}
	for i := range a {
		a[i] = Clean(a[i], 120)
	}
	return a
}
func toolKey(actor, id string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(actor+"\x00"+id)))
}
func Ingest(root string, p Payload) error {
	if p.Session == "" {
		return nil
	}
	return Transaction(root, p.Session, false, func(sp **State) error {
		s := *sp
		if !s.Active || s.Demo {
			return ErrInactive
		}
		if (p.AgentID != "" && (len(p.AgentID) > 120 || Clean(p.AgentID, 120) != p.AgentID)) || len(p.Transcript) > 4096 || len(p.AgentTranscript) > 4096 {
			s.Dropped++
			return nil
		}
		now := time.Now().UnixMilli()
		s.LastHook = now
		s.HookCount++
		actor := Clean(p.AgentID, 120)
		if actor == "" {
			actor = "lead"
		}
		// Session-level events always belong to the lead, even in --agent sessions.
		if p.Kind == "SessionEnd" || p.Kind == "SessionStart" {
			actor = "lead"
		}
		a := s.Agents[actor]
		if a == nil {
			if len(s.Agents) >= MaxAgents {
				s.Dropped++
				return nil
			}
			role := Clean(p.AgentType, 100)
			if role == "" {
				role = "subagent"
			}
			a = &Agent{ID: actor, Role: role, Status: "observed", Started: now}
			s.Agents[actor] = a
		}
		if actor == "lead" && p.Transcript != "" {
			a.Transcript = p.Transcript
		}
		if actor != "lead" {
			if p.AgentTranscript != "" {
				a.Transcript = p.AgentTranscript
			} else if a.Transcript == "" && p.Transcript != "" && filepath.Base(actor) == actor && !strings.ContainsAny(actor, "/\\") {
				a.Transcript = filepath.Join(strings.TrimSuffix(p.Transcript, filepath.Ext(p.Transcript)), "subagents", "agent-"+actor+".jsonl")
			}
		}
		// Do not interpret an Agent tool's requested model as its actual runtime model.
		switch p.Kind {
		case "SessionStart":
			if p.Model != "" {
				a.Model = Clean(p.Model, 100)
			}
			a.Status = "running"
		case "PostModelSwitch":
			if p.ToModel != "" {
				a.Model = Clean(p.ToModel, 100)
				a.Transcript = ""
			}
			s.event(actor, "model", "Observed model switch")
		case "UserPromptSubmit":
			if actor == "lead" {
				s.Finished = false
			}
			a.Status = "running"
			a.Ended = 0
			s.event(actor, "turn", "New user turn")
		case "SubagentStart":
			a.Status = "running"
			a.Ended = 0
			a.Started = now
			if p.AgentType != "" {
				a.Role = Clean(p.AgentType, 100)
			}
			s.event(actor, "spawn", "Subagent started; parentage not supplied")
		case "SubagentStop":
			a.Status = "returned"
			a.Ended = now
			for k, t := range s.Tools {
				if t.Actor == actor {
					delete(s.Tools, k)
				}
			}
			s.event(actor, "return", "Subagent turn returned (not an issue completion)")
		case "PreToolUse", "PostToolUse", "PostToolUseFailure", "PermissionRequest":
			name := Clean(p.ToolName, 100)
			if name == "" {
				name = "unknown tool"
			}
			key := ""
			if p.ToolID != "" {
				key = toolKey(actor, p.ToolID)
			}
			if p.Kind == "PreToolUse" {
				a.Status = "running"
				a.Ended = 0
				if actor == "lead" {
					s.Finished = false
				}
				if key != "" {
					if _, ok := s.Tools[key]; ok || len(s.Tools) < MaxTools {
						s.Tools[key] = Tool{Actor: actor, Name: name, Started: now}
					} else {
						s.Dropped++
					}
				} else {
					s.event(actor, "warning", "Tool ID absent; activity cannot be correlated")
				}
				for _, t := range s.Tools {
					if t.Actor == actor && t.Waiting {
						a.Status = "permission"
					}
				}
				s.event(actor, "tool", name+" started")
			} else if p.Kind == "PermissionRequest" {
				// PermissionRequest has no tool_use_id on some Claude versions. Mark the
				// matching outstanding tools without discarding other parallel activity.
				a.Status = "permission"
				for k, t := range s.Tools {
					if t.Actor == actor && (k == key || (key == "" && t.Name == name)) {
						t.Waiting = true
						s.Tools[k] = t
					}
				}
				s.event(actor, "permission", "Awaiting permission for "+name)
			} else {
				dedup := p.Kind + ":" + key
				if key != "" {
					for _, v := range s.Seen {
						if v == dedup {
							return nil
						}
					}
					s.Seen = append(s.Seen, dedup)
					if len(s.Seen) > 2048 {
						s.Seen = s.Seen[len(s.Seen)-2048:]
					}
					delete(s.Tools, key)
				}
				a.Tools++
				if !(actor == "lead" && s.Finished) {
					a.Status = "running"
				}
				for _, t := range s.Tools {
					if t.Actor == actor && t.Waiting {
						a.Status = "permission"
					}
				}
				if p.Kind == "PostToolUseFailure" {
					a.Errors++
					s.event(actor, "error", name+" failed; inspect Claude for details")
				} else {
					s.syncTasks(p.ToolName, p.Input, p.Response)
					s.event(actor, "result", name+" completed")
				}
			}
		case "Stop":
			if actor == "lead" && s.Finished {
				a.Status = "done"
			} else {
				a.Status = "waiting"
			}
			s.event(actor, "turn", "Agent turn ended; other actors retain last observed state")
		case "StopFailure":
			a.Status = "API error"
			a.Errors++
			s.event(actor, "error", "Agent turn stopped on API error; issue not completed")
		case "SessionEnd":
			a.Status = "ended"
			a.Ended = now
			for id, w := range s.Agents {
				if id != "lead" && (w.Status == "running" || w.Status == "permission" || w.Status == "observed") {
					w.Status = "unconfirmed"
				}
			}
			s.Tools = map[string]Tool{}
			s.event(actor, "session", "Session ended; worker termination not confirmed")
		default:
			return errors.New("unsupported hook event")
		}
		return nil
	})
}
func normalizeTask(m map[string]json.RawMessage, id string, old Task) Task {
	t := old
	t.ID = Clean(id, 120)
	if t.Status == "" {
		t.Status = "pending"
	}
	if _, ok := m["subject"]; ok {
		t.Subject = Clean(str(m, "subject"), 240)
	}
	if _, ok := m["content"]; ok {
		t.Subject = Clean(str(m, "content"), 240)
	}
	if _, ok := m["status"]; ok {
		switch v := str(m, "status"); v {
		case "pending", "in_progress", "completed", "blocked", "deleted":
			t.Status = v
		default:
			t.Status = "unknown"
		}
	}
	if _, ok := m["owner"]; ok {
		t.Owner = Clean(str(m, "owner"), 120)
	}
	if _, ok := m["blockedBy"]; ok {
		t.BlockedBy = list(m, "blockedBy")
	}
	for _, id := range list(m, "addBlockedBy") {
		found := false
		for _, v := range t.BlockedBy {
			if id == v {
				found = true
			}
		}
		if !found && len(t.BlockedBy) < 128 {
			t.BlockedBy = append(t.BlockedBy, id)
		}
	}
	return t
}
func (s *State) putTask(m map[string]json.RawMessage, id string) {
	id = Clean(id, 120)
	if id == "" {
		return
	}
	if _, ok := s.Tasks[id]; !ok && len(s.Tasks) >= MaxTasks {
		s.Dropped++
		return
	}
	s.Tasks[id] = normalizeTask(m, id, s.Tasks[id])
}
func (s *State) syncTasks(name string, input map[string]json.RawMessage, response json.RawMessage) {
	var r map[string]json.RawMessage
	_ = json.Unmarshal(response, &r)
	switch name {
	case "TaskCreate":
		var t map[string]json.RawMessage
		_ = json.Unmarshal(r["task"], &t)
		if t == nil {
			t = r
		}
		id := str(t, "id")
		if id == "" {
			id = str(r, "taskId")
		}
		if id != "" {
			s.putTask(input, id)
			s.putTask(t, id)
		}
	case "TaskUpdate":
		if v, ok := r["success"]; ok && string(v) == "false" {
			return
		}
		s.putTask(input, str(input, "taskId"))
	case "TaskList":
		var tasks []map[string]json.RawMessage
		if json.Unmarshal(r["tasks"], &tasks) != nil {
			_ = json.Unmarshal(response, &tasks)
		}
		// A successful complete list is authoritative, including removal of old tasks.
		if tasks != nil {
			s.Tasks = map[string]Task{}
			for _, t := range tasks {
				s.putTask(t, str(t, "id"))
			}
		}
	case "TodoWrite":
		var tasks []map[string]json.RawMessage
		if json.Unmarshal(input["todos"], &tasks) == nil {
			s.Tasks = map[string]Task{}
			for i, t := range tasks {
				id := str(t, "id")
				if id == "" {
					id = fmt.Sprint(i + 1)
				}
				s.putTask(t, id)
			}
		}
	}
}
func Plan(root, session string, raw []byte) error {
	var tasks []map[string]json.RawMessage
	if e := json.Unmarshal(raw, &tasks); e != nil {
		return e
	}
	if len(tasks) > MaxTasks {
		return errors.New("too many tasks")
	}
	return Transaction(root, session, false, func(sp **State) error {
		s := *sp
		next := map[string]Task{}
		for _, m := range tasks {
			id := Clean(str(m, "id"), 120)
			if id == "" {
				return errors.New("every task requires id")
			}
			if _, ok := next[id]; ok {
				return errors.New("duplicate task id")
			}
			t := normalizeTask(m, id, Task{})
			if t.Status == "unknown" {
				return errors.New("invalid task status")
			}
			next[id] = t
		}
		s.Tasks = next
		s.event("lead", "plan", "Explicit plan updated")
		return nil
	})
}
