package dashboard

import (
	"encoding/json"
	"os"
)

type Installation struct {
	Configured bool            `json:"configured"`
	Settings   string          `json:"settings"`
	Events     map[string]bool `json:"events"`
	Note       string          `json:"note"`
}

func Inspect(settings, exe string) (Installation, error) {
	result := Installation{Settings: settings, Events: map[string]bool{}, Note: "Configured does not prove hooks are loaded or permitted. Check status hook_count after actual Claude activity."}
	raw, e := os.ReadFile(settings)
	if os.IsNotExist(e) {
		return result, nil
	}
	if e != nil {
		return result, e
	}
	var config map[string]any
	if e = json.Unmarshal(raw, &config); e != nil {
		return result, e
	}
	hooks, _ := config["hooks"].(map[string]any)
	result.Configured = true
	for _, event := range HookEvents {
		groups, _ := hooks[event].([]any)
		for _, g := range groups {
			group, _ := g.(map[string]any)
			handlers, _ := group["hooks"].([]any)
			for _, h := range handlers {
				handler, _ := h.(map[string]any)
				if ownedHook(handler, exe) {
					result.Events[event] = true
				}
			}
		}
		if !result.Events[event] {
			result.Configured = false
			result.Events[event] = false
		}
	}
	return result, nil
}
