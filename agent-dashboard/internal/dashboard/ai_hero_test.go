package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func readAIHeroLayout(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "ai-hero", name+".layout.json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestAIHeroLayoutsSeparateLiveAndSyntheticBindings(t *testing.T) {
	for _, name := range []string{"tdd", "review", "review-no-spec"} {
		t.Run(name, func(t *testing.T) {
			root := setup(t)
			live := readAIHeroLayout(t, name)
			if strings.Contains(string(live), "demo-") {
				t.Fatal("synthetic ID in live template")
			}
			if err := SetLayout(root, "s", live); err != nil {
				t.Fatal(err)
			}
			s := snap(t, root)
			if len(s.Agents) != 1 || len(s.Tasks) != 0 || s.Finished {
				t.Fatal("publishing example changed execution")
			}
			hasGraph, hasTasks, hasEvents := false, false, false
			for _, p := range leaves(s.Layout.Root) {
				hasTasks = hasTasks || p.Kind == "tasks"
				hasEvents = hasEvents || p.Kind == "events"
				if p.Kind == "dag" {
					hasGraph = true
				}
			}
			if !hasGraph || !hasTasks || !hasEvents {
				t.Fatal("live workspace does not combine native DAG/task/event Panels")
			}
			frame := (Viewer{State: s, Width: 160, Height: 60}).Render(time.Unix(100, 0))
			if !strings.Contains(frame, "unobserved") {
				t.Fatal("unbound live template pretends to have observed work", frame)
			}
			demo := readAIHeroLayout(t, name+".demo")
			if strings.Contains(string(demo), "replace-with-") {
				t.Fatal("unbound placeholder in demo")
			}
			if err := SeedDemo(root, "preview"); err != nil {
				t.Fatal(err)
			}
			if err := SetLayout(root, "preview", demo); err != nil {
				t.Fatal(err)
			}
			preview, err := Snapshot(root, "preview")
			if err != nil {
				t.Fatal(err)
			}
			actors := map[string]string{}
			graphs := []*DAG{}
			for _, p := range leaves(preview.Layout.Root) {
				if p.Actor != "" {
					if preview.Agents[p.Actor] == nil {
						t.Fatalf("missing demo actor %q", p.Actor)
					}
					actors[p.ID] = p.Actor
				}
				for _, id := range p.Actors {
					if preview.Agents[id] == nil {
						t.Fatalf("missing demo event/list actor %q", id)
					}
				}
				if p.DAG != nil {
					graphs = append(graphs, p.DAG)
				}
			}
			for _, g := range graphs {
				for _, n := range g.Nodes {
					if n.ID == "refactor" {
						t.Fatal("refactor incorrectly put into the local TDD skill's loop")
					}
					if name == "review-no-spec" && n.ID == "spec" {
						t.Fatal("no-spec graph kept spec axis")
					}
					if n.ID == "verdict" && n.Bind != nil {
						t.Fatal("caller verdict disguised as observed execution state")
					}
					if n.Bind == nil {
						continue
					}
					if n.Bind.Actor != "" {
						if preview.Agents[n.Bind.Actor] == nil {
							t.Fatalf("missing graph actor %q", n.Bind.Actor)
						}
						if actor := actors[n.ID]; actor != "" && actor != n.Bind.Actor {
							t.Fatal("graph and actor card use different bindings")
						}
					} else if _, ok := preview.Tasks[n.Bind.Task]; !ok {
						t.Fatalf("missing demo task %q", n.Bind.Task)
					}
				}
			}
			frame = (Viewer{State: preview, Width: 160, Height: 60}).Render(time.Unix(100, 0))
			if !strings.Contains(frame, "DEMO · SYNTHETIC") || strings.Contains(frame, "unobserved") || strings.Contains(frame, "Not observed") {
				t.Fatal("mislabelled or unbound synthetic preview", frame)
			}
			if name == "review-no-spec" {
				for _, p := range leaves(preview.Layout.Root) {
					if p.ID == "spec" {
						t.Fatal("no-spec workspace kept spec actor card")
					}
				}
			}
			if after := snap(t, root); len(after.Agents) != 1 || len(after.Tasks) != 0 {
				t.Fatal("demo contaminated live session")
			}
		})
	}
}

func TestAIHeroGraphCompanionsMatchLiveWorkspace(t *testing.T) {
	for _, name := range []string{"tdd", "review"} {
		t.Run(name, func(t *testing.T) {
			l, err := ParseLayout(readAIHeroLayout(t, name))
			if err != nil {
				t.Fatal(err)
			}
			var workspace *DAG
			for _, p := range leaves(l.Root) {
				if p.DAG != nil {
					workspace = p.DAG
				}
			}
			raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "ai-hero", name+".dag.json"))
			if err != nil {
				t.Fatal(err)
			}
			var standalone DAG
			if err := json.Unmarshal(raw, &standalone); err != nil {
				t.Fatal(err)
			}
			if workspace == nil || !reflect.DeepEqual(workspace, &standalone) {
				t.Fatal("standalone graph diverged from native workspace")
			}
			compact, err := ParseLayout(readAIHeroLayout(t, name+".dag"))
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, p := range leaves(compact.Root) {
				if p.DAG != nil {
					found = true
					if !reflect.DeepEqual(p.DAG, workspace) {
						t.Fatal("compact graph binding differs from live workspace")
					}
				}
			}
			if !found {
				t.Fatal("compact layout lost graph")
			}
		})
	}
}

func TestAIHeroReviewDeclaresWorkerReportsToParent(t *testing.T) {
	for _, name := range []string{"review", "review-no-spec"} {
		t.Run(name, func(t *testing.T) {
			for _, suffix := range []string{"", ".demo"} {
				l, err := ParseLayout(readAIHeroLayout(t, name+suffix))
				if err != nil {
					t.Fatal(err)
				}
				for _, p := range leaves(l.Root) {
					if p.DAG == nil {
						continue
					}
					edges := map[[2]string]DAGEdge{}
					for _, e := range p.DAG.Edges {
						edges[[2]string{e.From, e.To}] = e
					}
					workers := []string{"standards"}
					if name == "review" {
						workers = append(workers, "spec")
					}
					for _, worker := range workers {
						outgoing, ok := edges[[2]string{"lead", worker}]
						if !ok || !strings.Contains(outgoing.Label, "delegate") {
							t.Fatal("missing parent delegation", name, suffix, worker)
						}
						incoming, ok := edges[[2]string{worker, "lead"}]
						if !ok || incoming.Style != "return" || !strings.Contains(incoming.Label, "report") {
							t.Fatal("missing explicit child report to parent", name, suffix, worker)
						}
						if _, ok := edges[[2]string{worker, "verdict"}]; !ok {
							t.Fatal("report edge replaced independent findings aggregation")
						}
					}
				}
			}
		})
	}
}
