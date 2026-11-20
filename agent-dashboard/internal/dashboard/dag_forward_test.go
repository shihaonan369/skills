package dashboard_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	d "agent-dashboard/internal/dashboard"
)

func TestBundledGraphsUseOnlyForwardEdges(t *testing.T) {
	root := filepath.Join("..", "..", "examples")
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		count++
		t.Run(filepath.ToSlash(relative), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var shape map[string]json.RawMessage
			if err := json.Unmarshal(raw, &shape); err != nil {
				t.Fatal(err)
			}
			if _, layout := shape["root"]; !layout {
				var graph d.DAG
				if err := json.Unmarshal(raw, &graph); err != nil {
					t.Fatal(err)
				}
				raw, err = json.Marshal(d.Layout{Version: 1, Root: d.Panel{ID: "fixture", Kind: "dag", DAG: &graph}})
				if err != nil {
					t.Fatal(err)
				}
			}
			// ParseLayout applies the actual public graph contract to plain DAG
			// companions as well as full compositions, including every example family.
			layout, err := d.ParseLayout(raw)
			if err != nil {
				t.Fatal(err)
			}
			frame := (d.Viewer{State: &d.State{Version: 1, Demo: true, Active: true, Layout: layout}, Width: 180, Height: 80}).View()
			for _, obsolete := range []string{"▲", "⇄", "[return]", "+return", "-return"} {
				if strings.Contains(frame, obsolete) {
					t.Fatal("bundled graph still displays reverse flow", obsolete)
				}
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("no bundled compositions were validated")
	}
}

func TestPersistedReturnGraphIsDiagnosedWithoutRewritingCallerState(t *testing.T) {
	layout, err := d.ParseLayout([]byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"parent"},{"id":"child"}],"edges":[{"from":"parent","to":"child"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a snapshot written by an older build. It must be diagnosed rather
	// than silently deleting, reversing or reinterpreting the caller's old edge.
	layout.Root.DAG.Edges[0].Style = "return"
	state := &d.State{Version: 1, Demo: true, Active: true, Layout: layout}
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	frame := (d.Viewer{State: state, Width: 180, Height: 24}).View()
	if !strings.Contains(frame, "Invalid DAG") || !strings.Contains(frame, "return edges are no longer supported") || strings.Contains(frame, "Forward flow") {
		t.Fatal("old graph was silently drawn or migrated", frame)
	}
	after, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("rendering changed the caller's persisted composition")
	}
}
