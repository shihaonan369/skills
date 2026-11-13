package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/gofrs/flock"
)

var HookEvents = []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "PostToolUseFailure", "PermissionRequest", "SubagentStart", "SubagentStop", "Stop", "StopFailure", "SessionEnd", "PostModelSwitch"}
var hookArgs = []any{"hook", "--source", "agent-dashboard-v1"}

func ownedHook(h map[string]any, exe string) bool {
	return h["type"] == "command" && h["command"] == exe && reflect.DeepEqual(h["args"], hookArgs)
}

// Configure never touches unrelated hook handlers, settings, or skill links.
// Invalid settings are rejected rather than replaced with an empty object.
func Configure(settings, exe string, remove bool) (string, error) {
	if !filepath.IsAbs(exe) {
		return "", errors.New("hook executable must be absolute")
	}
	if e := os.MkdirAll(filepath.Dir(settings), 0700); e != nil {
		return "", e
	}
	lock := flock.New(settings+".agent-dashboard.lock", flock.SetPermissions(0600))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ok, e := lock.TryLockContext(ctx, 20*time.Millisecond)
	if e != nil {
		return "", e
	}
	if !ok {
		return "", errors.New("settings locked")
	}
	defer lock.Close()
	raw, e := os.ReadFile(settings)
	exists := e == nil
	if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	config := map[string]any{}
	if exists {
		if e = json.Unmarshal(raw, &config); e != nil {
			return "", fmt.Errorf("invalid settings: %w", e)
		}
		if config == nil {
			return "", errors.New("settings must be an object")
		}
	}
	old, _ := json.Marshal(config)
	hooks := map[string]any{}
	if h, ok := config["hooks"]; ok {
		var valid bool
		hooks, valid = h.(map[string]any)
		if !valid {
			return "", errors.New("hooks must be an object")
		}
	}
	// Reject malformed hook arrays; never lose settings through a type assertion.
	for _, event := range HookEvents {
		groups := []any{}
		if g, ok := hooks[event]; ok {
			var valid bool
			groups, valid = g.([]any)
			if !valid {
				return "", fmt.Errorf("%s hooks must be an array", event)
			}
		}
		next := []any{}
		for _, g := range groups {
			group, ok := g.(map[string]any)
			if !ok {
				return "", errors.New("invalid hook group")
			}
			handlers, ok := group["hooks"].([]any)
			if !ok {
				return "", errors.New("invalid hook handlers")
			}
			kept := []any{}
			removed := false
			for _, handler := range handlers {
				h, ok := handler.(map[string]any)
				if !ok {
					return "", errors.New("invalid hook handler")
				}
				if ownedHook(h, exe) {
					removed = true
					continue
				}
				kept = append(kept, handler)
			}
			if len(kept) > 0 || !removed {
				group["hooks"] = kept
				next = append(next, group)
			}
		}
		if !remove {
			next = append(next, map[string]any{"hooks": []any{map[string]any{"type": "command", "command": exe, "args": hookArgs, "timeout": 3, "onFailure": "continue"}}})
		}
		if len(next) > 0 {
			hooks[event] = next
		} else {
			delete(hooks, event)
		}
	}
	if len(hooks) > 0 {
		config["hooks"] = hooks
	} else {
		delete(config, "hooks")
	}
	encoded, e := json.MarshalIndent(config, "", "  ")
	if e != nil {
		return "", e
	}
	normalized, _ := json.Marshal(config)
	if string(old) == string(normalized) {
		return "", nil
	}
	backup := ""
	if exists {
		f, e := os.OpenFile(settings+".agent-dashboard-backup-"+time.Now().UTC().Format("20060102T150405.000000000"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return "", e
		}
		backup = f.Name()
		_, e = f.Write(raw)
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return "", e
		}
		// Detect a competing editor that does not use our advisory lock.
		current, e := os.ReadFile(settings)
		if e != nil {
			return "", e
		}
		if string(current) != string(raw) {
			return "", errors.New("settings changed during install; retry")
		}
	}
	encoded = append(encoded, '\n')
	return backup, atomicWrite(settings, encoded, 0600)
}
func LinkSkill(skillDir, skillsDir string) error {
	target := filepath.Join(skillsDir, "agent-dashboard")
	if e := os.MkdirAll(skillsDir, 0700); e != nil {
		return e
	}
	if _, e := os.Lstat(target); e == nil {
		resolved, e := filepath.EvalSymlinks(target)
		if e != nil {
			return e
		}
		source, e := filepath.EvalSymlinks(skillDir)
		if e != nil {
			return e
		}
		if resolved == source {
			return nil
		}
		return errors.New("agent-dashboard skill already exists at a different path; preserved")
	} else if !os.IsNotExist(e) {
		return e
	}
	return os.Symlink(skillDir, target)
}
func UnlinkSkill(skillDir, skillsDir string) error {
	target := filepath.Join(skillsDir, "agent-dashboard")
	info, e := os.Lstat(target)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	resolved, e := filepath.EvalSymlinks(target)
	if e != nil {
		return e
	}
	source, e := filepath.EvalSymlinks(skillDir)
	if e != nil {
		return e
	}
	if resolved == source {
		return os.Remove(target)
	}
	return nil
}
