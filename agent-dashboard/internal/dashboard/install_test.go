package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestInstallIdempotentNarrowUninstall(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.json")
	exe := filepath.Join(dir, "path with '$ spaces", "agent-dashboard")
	original := []byte(`{"permissions":{"allow":["Read"]},"env":{"KEY":"preserve"},"hooks":{"PreToolUse":[{"matcher":"Read","hooks":[{"type":"command","command":"other-hook"}]}],"Notification":[{"hooks":[{"type":"command","command":"notify"}]}]}}`)
	os.WriteFile(settings, original, 0600)
	backup, e := Configure(settings, exe, false)
	if e != nil {
		t.Fatal(e)
	}
	if b, _ := os.ReadFile(backup); string(b) != string(original) {
		t.Fatal("backup changed")
	}
	first, _ := os.ReadFile(settings)
	backup, e = Configure(settings, exe, false)
	if e != nil || backup != "" {
		t.Fatal("not idempotent", e)
	}
	second, _ := os.ReadFile(settings)
	if string(first) != string(second) {
		t.Fatal("idempotent install changed settings")
	}
	var c map[string]any
	json.Unmarshal(first, &c)
	hooks := c["hooks"].(map[string]any)
	for _, event := range HookEvents {
		count := 0
		for _, g := range hooks[event].([]any) {
			for _, h := range g.(map[string]any)["hooks"].([]any) {
				if ownedHook(h.(map[string]any), exe) {
					count++
				}
			}
		}
		if count != 1 {
			t.Fatalf("%s count %d", event, count)
		}
	}
	if _, e = Configure(settings, filepath.Join(dir, "foreign"), true); e != nil {
		t.Fatal(e)
	}
	second, _ = os.ReadFile(settings)
	if string(first) != string(second) {
		t.Fatal("uninstall removed another install")
	}
	if _, e = Configure(settings, exe, true); e != nil {
		t.Fatal(e)
	}
	second, _ = os.ReadFile(settings)
	var a, b any
	json.Unmarshal(original, &a)
	json.Unmarshal(second, &b)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("other settings changed: %s", second)
	}
}
func TestInvalidSettingsPreserved(t *testing.T) {
	for _, bad := range []string{`{broken`, `null`, `[]`, `{"hooks":null}`, `{"hooks":{"Stop":"bad"}}`, `{"hooks":{"Stop":[{"hooks":null}]}}`} {
		t.Run(bad, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "settings.json")
			os.WriteFile(p, []byte(bad), 0600)
			if _, e := Configure(p, "/fake/executable", false); e == nil {
				t.Fatal("bad config accepted")
			}
			b, _ := os.ReadFile(p)
			if string(b) != bad {
				t.Fatal("bad config clobbered")
			}
		})
	}
}
func TestSkillLinksPreserveOthers(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	skills := filepath.Join(dir, "skills")
	os.MkdirAll(source, 0700)
	os.MkdirAll(skills, 0700)
	os.Mkdir(filepath.Join(skills, "other"), 0700)
	if e := LinkSkill(source, skills); e != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink privilege unavailable: %v", e)
		}
		t.Fatal(e)
	}
	if e := LinkSkill(source, skills); e != nil {
		t.Fatal(e)
	}
	if e := LinkSkill(dir, skills); e == nil {
		t.Fatal("conflicting link overwritten")
	}
	if e := UnlinkSkill(dir, skills); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Lstat(filepath.Join(skills, "agent-dashboard")); e != nil {
		t.Fatal("foreign unlink removed source")
	}
	if e := UnlinkSkill(source, skills); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(skills, "other")); e != nil {
		t.Fatal("other skill removed")
	}
	os.Mkdir(filepath.Join(skills, "agent-dashboard"), 0700)
	if e := LinkSkill(source, skills); e == nil {
		t.Fatal("real directory overwritten")
	}
	if e := UnlinkSkill(source, skills); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(skills, "agent-dashboard")); e != nil {
		t.Fatal("real directory deleted")
	}
}

func TestDoctorDoesNotClaimHookLiveness(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.json")
	exe := filepath.Join(dir, "agent-dashboard")
	info, e := Inspect(settings, exe)
	if e != nil || info.Configured {
		t.Fatal("missing config claims installed")
	}
	if _, e := Configure(settings, exe, false); e != nil {
		t.Fatal(e)
	}
	info, e = Inspect(settings, exe)
	if e != nil || !info.Configured || !strings.Contains(info.Note, "does not prove") {
		t.Fatal("doctor readiness claim wrong", e)
	}
}
