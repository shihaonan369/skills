package dashboard_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	d "agent-dashboard/internal/dashboard"
	"github.com/charmbracelet/x/ansi"
)

func writeExternalFixture(t *testing.T, path, raw string) {
	t.Helper()
	stage := path + ".fixture"
	if err := os.WriteFile(stage, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(stage, path); err != nil {
		t.Fatal(err)
	}
}

func TestExternalDataOneSnapshotFeedsTextTitlesAndDAGWithoutLayoutWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "countdown.json")
	writeExternalFixture(t, path, `{"remaining_seconds":92,"unbound":"PRIVATE UNBOUND VALUE"}`)
	encoded, _ := json.Marshal(path)
	raw := `{"version":1,"data_sources":{"timer":{"type":"json-file","path":` + string(encoded) + `}},"root":{"kind":"column","children":[
 {"id":"counter","kind":"text","size":8,"value":{"source":"timer","path":"/remaining_seconds","format":"mm:ss"}},
 {"id":"title","kind":"session","size":10,"title_value":{"source":"timer","path":"/remaining_seconds","format":"mm:ss"}},
 {"id":"graph","kind":"dag","dag":{"version":1,"nodes":[{"id":"timer","label_value":{"source":"timer","path":"/remaining_seconds","format":"mm:ss"}}]}}
 ]}}`
	s, root := dynamicNative(t, raw)
	original, _ := json.Marshal(s)
	prepared, err := d.PreparePresentation(s)
	if err != nil {
		t.Fatal(err)
	}
	frame := dynamicFrame(prepared)
	if strings.Count(frame, "01:32") < 3 || !strings.Contains(frame, "External data") {
		t.Fatal(frame)
	}
	writeExternalFixture(t, path, `{"remaining_seconds":31}`)
	if f := dynamicFrame(prepared); !strings.Contains(f, "01:32") || strings.Contains(f, "00:31") {
		t.Fatal("render read a file instead of the immutable snapshot", f)
	}
	prepared, err = d.PreparePresentation(nativeSnapshot(t, root))
	if err != nil {
		t.Fatal(err)
	}
	frame = dynamicFrame(prepared)
	if strings.Count(frame, "00:31") < 3 || strings.Contains(frame, "01:32") {
		t.Fatal("data update required a layout rewrite", frame)
	}
	after, _ := json.Marshal(prepared)
	if string(after) != string(original) || strings.Contains(string(after), "PRIVATE UNBOUND VALUE") {
		t.Fatal("external data changed/serialized into native telemetry or the caller layout")
	}
}

func externalTextState(t *testing.T, path, pointer, format string, staleMS int) (*d.State, string) {
	t.Helper()
	layout := map[string]any{"version": 1, "data_sources": map[string]any{"data": map[string]any{"type": "json-file", "path": path, "stale_after_ms": staleMS}}, "root": map[string]any{"id": "value", "kind": "text", "value": map[string]any{"source": "data", "path": pointer, "format": format, "prefix": "Bound: "}}}
	raw, _ := json.Marshal(layout)
	return dynamicNative(t, string(raw))
}

func externalFrameAt(t *testing.T, s *d.State, at time.Time) string {
	t.Helper()
	prepared, err := d.PreparePresentation(s)
	if err != nil {
		t.Fatal(err)
	}
	return ansi.Strip((d.Viewer{State: prepared, Width: 200, Height: 40}).Render(at))
}

func TestExternalDataFormatsPointersAndDeadlineArePure(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	path := filepath.Join(t.TempDir(), "values.json")
	fixture := map[string]any{"updated_at": now.Format(time.RFC3339), "seconds": 92, "fraction": 1.1, "percent": 35, "precise": json.Number("9007199254740993"), "text": "safe\x1b[31m\nvalue", "flag": true, "deadline": now.Add(10 * time.Second).Format(time.RFC3339), "null": nil, "a/b": map[string]any{"~key": []any{7}}, "object": map[string]any{"x": 1}, "negative": -1, "excess": 101}
	raw, _ := json.Marshal(fixture)
	writeExternalFixture(t, path, string(raw))
	for _, tc := range []struct{ path, format, want string }{
		{"/seconds", "mm:ss", "01:32"}, {"/fraction", "mm:ss", "00:02"},
		{"/percent", "percent", "35%"}, {"/percent", "bar", "[###-------] 35%"},
		{"/precise", "number", "9007199254740993"}, {"/flag", "text", "true"},
		{"/text", "", "safe[31mvalue"}, {"/a~1b/~0key/0", "number", "7"},
		{"/deadline", "deadline", "00:10"}, {"/missing", "text", "unknown (missing or null value)"},
		{"/null", "text", "unknown (missing or null value)"}, {"/object", "text", "unknown (expected a scalar)"},
		{"/text", "number", "unknown (expected a number)"}, {"/percent", "deadline", "unknown (expected an RFC 3339 deadline)"},
		{"/negative", "mm:ss", "unknown (duration outside"}, {"/excess", "bar", "unknown (percentage outside"},
		{"/a~1b/~0key/00", "number", "unknown (missing or null value)"},
	} {
		t.Run(tc.path+"_"+tc.format, func(t *testing.T) {
			s, _ := externalTextState(t, path, tc.path, tc.format, 60000)
			if frame := externalFrameAt(t, s, now); !strings.Contains(frame, "Bound: "+tc.want) {
				t.Fatal(frame)
			}
		})
	}
	s, _ := externalTextState(t, path, "/deadline", "deadline", 60000)
	prepared, err := d.PreparePresentation(s)
	if err != nil {
		t.Fatal(err)
	}
	writeExternalFixture(t, path, `{"deadline":"invalid"}`)
	for _, tc := range []struct {
		elapsed time.Duration
		want    string
	}{{2 * time.Second, "00:08"}, {12 * time.Second, "00:00"}, {61 * time.Second, "stale (external data expired)"}} {
		frame := ansi.Strip((d.Viewer{State: prepared, Width: 200, Height: 40}).Render(now.Add(tc.elapsed)))
		if !strings.Contains(frame, "Bound: "+tc.want) {
			t.Fatal(frame)
		}
	}
	remaining, _ := externalTextState(t, path, "/seconds", "mm:ss", 60000)
	writeExternalFixture(t, path, string(raw))
	prepared, err = d.PreparePresentation(remaining)
	if err != nil {
		t.Fatal(err)
	}
	if f := ansi.Strip((d.Viewer{State: prepared, Width: 200, Height: 40}).Render(now.Add(10 * time.Second))); !strings.Contains(f, "01:32") {
		t.Fatal("remaining_seconds must not be decremented by the viewer", f)
	}
}

func TestExternalDataSourceFailuresStayLocalAndRecoverWithoutCachedSuccess(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "data.json")
	s, _ := externalTextState(t, path, "/n", "number", 0)
	if f := externalFrameAt(t, s, now); !strings.Contains(f, "Bound: unknown (file unavailable)") {
		t.Fatal(f)
	}
	for _, tc := range []struct{ raw, want string }{
		{`{`, "invalid or oversized JSON"}, {`{"n":1,"n":2}`, "invalid or oversized JSON"},
		{`{"n":1} {}`, "invalid or oversized JSON"}, {strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34), "invalid or oversized JSON"},
		{strings.Repeat(" ", d.MaxDataFileBytes+1), "expected a regular file"},
		{`{"n":1,"updated_at":false}`, "invalid updated_at timestamp"},
		{`{"n":1,"updated_at":"` + now.Add(time.Hour).Format(time.RFC3339Nano) + `"}`, "future updated_at timestamp"},
		{`{"n":1,"updated_at":"` + now.Add(-time.Minute).Format(time.RFC3339Nano) + `"}`, "stale (external data expired)"},
		{`{"n":1e999}`, "expected a finite number"},
	} {
		writeExternalFixture(t, path, tc.raw)
		if f := externalFrameAt(t, s, now); !strings.Contains(f, tc.want) || strings.Contains(f, "Bound: 99") {
			t.Fatal(tc.want, f)
		}
	}
	writeExternalFixture(t, path, `{"n":99}`)
	if f := externalFrameAt(t, s, now); !strings.Contains(f, "Bound: 99") {
		t.Fatal(f)
	}
	writeExternalFixture(t, path, `invalid again`)
	if f := externalFrameAt(t, s, now); !strings.Contains(f, "unknown (invalid or oversized JSON)") || strings.Contains(f, "Bound: 99") {
		t.Fatal("failed read reused a healthy-looking value", f)
	}
	link := filepath.Join(t.TempDir(), "link.json")
	if err := os.Symlink(path, link); err == nil {
		linked, _ := externalTextState(t, link, "/n", "number", 0)
		if f := externalFrameAt(t, linked, now); !strings.Contains(f, "expected a regular file") {
			t.Fatal(f)
		}
	}
	folder, _ := externalTextState(t, filepath.Dir(path), "/n", "number", 0)
	if f := externalFrameAt(t, folder, now); !strings.Contains(f, "expected a regular file") {
		t.Fatal(f)
	}
}

func TestExternalDataValidationDoesNotExecuteSourcesOrReadAtLayoutPublication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-created.json")
	base := func() map[string]any {
		return map[string]any{"version": 1, "data_sources": map[string]any{"data": map[string]any{"type": "json-file", "path": path}}, "root": map[string]any{"id": "value", "kind": "text", "value": map[string]any{"source": "data", "path": "/x"}}}
	}
	for name, change := range map[string]func(map[string]any){
		"command": func(l map[string]any) {
			l["data_sources"].(map[string]any)["data"].(map[string]any)["type"] = "command"
		},
		"relative": func(l map[string]any) {
			l["data_sources"].(map[string]any)["data"].(map[string]any)["path"] = "relative.json"
		},
		"ttl": func(l map[string]any) {
			l["data_sources"].(map[string]any)["data"].(map[string]any)["stale_after_ms"] = 99
		},
		"undeclared":  func(l map[string]any) { l["root"].(map[string]any)["value"].(map[string]any)["source"] = "other" },
		"pointer":     func(l map[string]any) { l["root"].(map[string]any)["value"].(map[string]any)["path"] = "$.x" },
		"escape":      func(l map[string]any) { l["root"].(map[string]any)["value"].(map[string]any)["path"] = "/~2" },
		"expression":  func(l map[string]any) { l["root"].(map[string]any)["value"].(map[string]any)["format"] = "eval" },
		"control":     func(l map[string]any) { l["root"].(map[string]any)["value"].(map[string]any)["prefix"] = "\x1b" },
		"native body": func(l map[string]any) { l["root"].(map[string]any)["kind"] = "session" },
		"too many": func(l map[string]any) {
			for i := 0; i < 17; i++ {
				l["data_sources"].(map[string]any)[fmt.Sprintf("s%d", i)] = map[string]any{"type": "json-file", "path": path}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			l := base()
			change(l)
			raw, _ := json.Marshal(l)
			if _, err := d.ParseLayout(raw); err == nil {
				t.Fatal("invalid binding accepted", string(raw))
			}
		})
	}
	raw, _ := json.Marshal(base())
	state, root := dynamicNative(t, string(raw))
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("publishing a layout created/touched its data file", err)
	}
	if f := dynamicFrame(state); !strings.Contains(f, "data source not loaded") {
		t.Fatal("render must not perform initial file IO", f)
	}
	before, _ := json.Marshal(nativeSnapshot(t, root))
	writeExternalFixture(t, path, `{"x":"UNSERIALIZED DATA"}`)
	_, err := d.PreparePresentation(state)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(nativeSnapshot(t, root))
	if string(before) != string(after) {
		t.Fatal("reading external data rewrote persistent state")
	}
}

func TestExternalDataPublisherIsAtomicPrivateAndRejectsMalformedUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "data.json")
	if err := d.PublishDataFile(path, []byte(`{"n":12}`)); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(before, &value); err != nil {
		t.Fatal(err)
	}
	if _, err := time.Parse(time.RFC3339Nano, value["updated_at"].(string)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{`, `{"n":1,"n":2}`, `{"updated_at":null}`, `1 2`, strings.Repeat(" ", d.MaxDataFileBytes+1)} {
		if err := d.PublishDataFile(path, []byte(bad)); err == nil {
			t.Fatal("invalid publication accepted")
		}
		after, _ := os.ReadFile(path)
		if string(after) != string(before) {
			t.Fatal("invalid write destroyed prior complete snapshot")
		}
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Fatal(info.Mode())
		}
	}
	if err := d.PublishDataFile("relative.json", []byte(`1`)); err == nil {
		t.Fatal("relative destination accepted")
	}
	if err := d.PublishDataFile(filepath.Dir(path), []byte(`1`)); err == nil {
		t.Fatal("directory destination accepted")
	}
	if err := d.PublishDataFile(path, []byte(`[1,true,"x"]`)); err != nil {
		t.Fatal(err)
	}
	items, _ := os.ReadDir(filepath.Dir(path))
	if len(items) != 1 {
		t.Fatal("producer left staging files", items)
	}
}

func TestExternalDataBindingsDecorateEveryLeafAndDynamicTemplates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	writeExternalFixture(t, path, `{"text":"EXTERNAL TITLE"}`)
	for _, kind := range []string{"session", "agent", "agents", "tasks", "events", "text", "dag", "programs", "sources"} {
		t.Run(kind, func(t *testing.T) {
			panel := map[string]any{"id": "leaf", "kind": kind, "title_value": map[string]any{"source": "data", "path": "/text"}}
			if kind == "agent" {
				panel["actor"] = "lead"
			}
			if kind == "dag" {
				panel["dag"] = map[string]any{"version": 1, "nodes": []any{map[string]any{"id": "x"}}}
			}
			layout := map[string]any{"version": 1, "data_sources": map[string]any{"data": map[string]any{"type": "json-file", "path": path}}, "root": panel}
			raw, _ := json.Marshal(layout)
			s, _ := dynamicNative(t, string(raw))
			prepared, err := d.PreparePresentation(s)
			if err != nil {
				t.Fatal(err)
			}
			if f := dynamicFrame(prepared); !strings.Contains(f, "[data] EXTERNAL TITLE") {
				t.Fatal(kind, f)
			}
		})
	}
	encoded, _ := json.Marshal(path)
	raw := `{"version":1,"data_sources":{"data":{"type":"json-file","path":` + string(encoded) + `}},"root":{"kind":"column","children":[
 {"id":"repeat","kind":"text","value":{"source":"data","path":"/text"},"repeat":{"select":{"kind":"actors","roles":["implement"]}}},
 {"id":"graph","kind":"dag","dag":{"version":1,"nodes":[{"id":"group"}],"attach_from":"group","select":{"kind":"actors","roles":["implement"]},"node":{"label_value":{"source":"data","path":"/text"}}}}
 ]}}`
	s, root := dynamicNative(t, raw)
	s = spawnDynamic(t, root, "worker-one", "implement")
	prepared, err := d.PreparePresentation(s)
	if err != nil {
		t.Fatal(err)
	}
	if f := dynamicFrame(prepared); strings.Count(f, "EXTERNAL TITLE") < 2 {
		t.Fatal("dynamic materialization lost source definitions or template field bindings", f)
	}
}

func TestExternalDataRexAndDAGEdgesUseTheSameProjectionWithoutChangingObservations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.json")
	writeExternalFixture(t, path, `{"label":"EXTERNAL EDGE"}`)
	encoded, _ := json.Marshal(path)
	layout, err := d.ParseLayout([]byte(`{"version":1,"data_sources":{"file":{"type":"json-file","path":` + string(encoded) + `}},"root":{"kind":"dag","id":"graph","dag":{"version":1,"nodes":[{"id":"a"},{"id":"b"}],"edges":[{"from":"a","to":"b","label_value":{"source":"file","path":"/label"}}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := programFixture()
	s := d.ProgramState(snapshot, layout)
	original, _ := json.Marshal(s)
	prepared, err := d.PreparePresentation(s)
	if err != nil {
		t.Fatal(err)
	}
	if frame := dynamicFrame(prepared); !strings.Contains(frame, "EXTERNAL EDGE") {
		t.Fatal(frame)
	}
	after, _ := json.Marshal(prepared)
	if string(original) != string(after) || layout.Root.DAG.Edges[0].Label != "" {
		t.Fatal("external edge projection mutated declared layout or Rex observation", string(after))
	}
}
