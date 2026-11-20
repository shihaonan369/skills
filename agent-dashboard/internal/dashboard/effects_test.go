package dashboard

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestFocusUsesHeavyBorderWithoutChangingPanelGeometry(t *testing.T) {
	root := setup(t)
	raw := []byte(`{"version":1,"root":{"kind":"row","children":[{"id":"graph","kind":"dag","dag":{"version":1,"nodes":[{"id":"a"}]}},{"id":"text","kind":"text","text":["Caller annotation"]}]}}`)
	if err := SetLayout(root, "s", raw); err != nil {
		t.Fatal(err)
	}
	v := Viewer{State: snap(t, root), Width: 100, Height: 24}
	first := ansi.Strip(v.Render(time.Unix(100, 0)))
	row := strings.Split(first, "\n")[2]
	if !strings.HasPrefix(row, "┏") || !strings.Contains(row, "╭") {
		t.Fatal("DAG focus does not use a heavy frame", row)
	}
	m, _ := v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = m.(Viewer)
	second := ansi.Strip(v.Render(time.Unix(100, 0)))
	row = strings.Split(second, "\n")[2]
	if !strings.HasPrefix(row, "╭") || !strings.Contains(row, "┏") {
		t.Fatal("focus frame did not move to text panel", row)
	}
	assertDAGFrame(t, first, 100, 24)
	assertDAGFrame(t, second, 100, 24)
	if len(v.State.Agents) != 1 || v.State.HookCount != 0 {
		t.Fatal("focus changed observed state")
	}
}

func TestDAGDecorativePacketsMoveOnlyOnDeclaredPaths(t *testing.T) {
	raw := []byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"}],"edges":[{"from":"a","to":"b"},{"from":"a","to":"c","style":"dashed"},{"from":"b","to":"d"},{"from":"c","to":"d"},{"from":"a","to":"d"}]}}}`)
	l, err := ParseLayout(raw)
	if err != nil {
		t.Fatal(err)
	}
	s := newState("effects", "effects", false)
	s.Layout = l
	s.Agents["lead"].Status = "waiting"
	v := Viewer{State: s, Width: 120, Height: 50}
	m, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	off := m.(Viewer)
	baseline := ansi.Strip(off.Render(time.Unix(100, 0)))
	if strings.ContainsAny(baseline, "●•") {
		t.Fatal("motion toggle did not remove packets")
	}
	rows := strings.Split(baseline, "\n")
	heads := map[[2]int]bool{}
	tailSeen := false
	for i := 0; i < 50; i++ {
		frame := ansi.Strip(v.Render(time.Unix(100, 0).Add(time.Duration(i) * 100 * time.Millisecond)))
		assertDAGFrame(t, frame, 120, 50)
		if !strings.Contains(frame, "decorative") {
			t.Fatal("flow decoration not labelled")
		}
		tailSeen = tailSeen || strings.Contains(frame, "•")
		for y, row := range strings.Split(frame, "\n") {
			for x, ch := range []rune(row) {
				if ch != '●' {
					continue
				}
				static := []rune(rows[y])[x]
				if !strings.ContainsRune("│─└┐┌┘┤├┼┴┬┆·", static) {
					t.Fatalf("packet left its declared route at %d,%d on %q", x, y, string(static))
				}
				heads[[2]int{x, y}] = true
			}
		}
	}
	if len(heads) < 20 || !tailSeen {
		t.Fatal("missing moving heads or short trails", len(heads))
	}
	if s.Revision != 0 || s.HookCount != 0 || len(s.Tasks) != 0 || s.Finished {
		t.Fatal("animation fabricated telemetry")
	}
}

func TestMotionClockDoesNotReadTelemetryAndPauseFreezesEffects(t *testing.T) {
	l, err := ParseLayout([]byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"a"},{"id":"b"}],"edges":[{"from":"a","to":"b"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	s := newState("clock", "clock", false)
	s.Layout = l
	v := Viewer{Root: "/not-a-state-root", State: s, Width: 100, Height: 30}
	t0 := time.Unix(100, 0)
	m, cmd := v.Update(motionMsg(t0))
	v = m.(Viewer)
	if !v.MotionTime.Equal(t0) || v.State != s || s.HookCount != 0 {
		t.Fatal("frame changed telemetry instead of visual clock")
	}
	if cmd == nil {
		t.Fatal("frame timer did not reschedule")
	}
	if _, ok := cmd().(motionMsg); !ok {
		t.Fatal("animation scheduled telemetry IO instead of its own timer")
	}
	key := func(k rune) { m, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{k}}); v = m.(Viewer) }
	key('p')
	before := ansi.Strip(v.Render(t0))
	m, _ = v.Update(motionMsg(t0.Add(time.Second)))
	v = m.(Viewer)
	if !v.MotionTime.Equal(t0) || ansi.Strip(v.Render(t0)) != before {
		t.Fatal("pause did not freeze effects")
	}
	newer := *s
	newer.Revision = 9
	m, _ = v.Update(snapshotMsg{state: &newer})
	v = m.(Viewer)
	if v.State != s {
		t.Fatal("in-flight snapshot changed paused view")
	}
	key('p')
	m, _ = v.Update(motionMsg(t0.Add(time.Second)))
	v = m.(Viewer)
	if v.MotionTime.Equal(t0) {
		t.Fatal("resume did not restart visual clock")
	}
	key('a')
	if !v.MotionOff {
		t.Fatal("motion preference did not toggle")
	}
	frozen := v.MotionTime
	m, _ = v.Update(motionMsg(t0.Add(2 * time.Second)))
	v = m.(Viewer)
	if !v.MotionTime.Equal(frozen) {
		t.Fatal("motion-off clock advanced")
	}
	if strings.ContainsAny(ansi.Strip(v.Render(t0)), "●•") {
		t.Fatal("motion-off retained animated packets")
	}
	key('a')
	if v.MotionOff {
		t.Fatal("motion preference failed to re-enable")
	}
	if !s.Active || s.Finished || len(s.Agents) != 1 {
		t.Fatal("visual controls changed collection or lifecycle")
	}
}

func BenchmarkAnimatedAIHero(b *testing.B) {
	raw, err := os.ReadFile("../../examples/ai-hero/review.demo.layout.json")
	if err != nil {
		b.Fatal(err)
	}
	layout, err := ParseLayout(raw)
	if err != nil {
		b.Fatal(err)
	}
	root := b.TempDir()
	if err := SeedDemo(root, "bench"); err != nil {
		b.Fatal(err)
	}
	s, err := Snapshot(root, "bench")
	if err != nil {
		b.Fatal(err)
	}
	s.Layout = layout
	v := Viewer{State: s, Width: 140, Height: 56, Selected: 1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Render(time.Unix(100, 0).Add(time.Duration(i) * 100 * time.Millisecond))
	}
}

func TestDAGDelegationDisplaysOnlyForwardFlow(t *testing.T) {
	root := setup(t)
	raw := []byte(`{"version":1,"root":{"id":"g","kind":"dag","dag":{"version":1,"nodes":[{"id":"lead","label":"Parent","bind":{"actor":"lead"}},{"id":"worker","label":"Child","bind":{"actor":"actual-worker"}}],"edges":[{"from":"lead","to":"worker","label":"delegate work"}]}}}`)
	if err := SetLayout(root, "s", raw); err != nil {
		t.Fatal(err)
	}
	send(t, root, Payload{Kind: "SubagentStart", AgentID: "actual-worker", AgentType: "caller-chosen-worker"})
	state := snap(t, root)
	viewer := Viewer{State: state, Width: 100, Height: 32}
	for i := 0; i < 20; i++ {
		frame := ansi.Strip(viewer.Render(time.Unix(100, 0).Add(time.Duration(i) * 100 * time.Millisecond)))
		if !strings.Contains(frame, "lead → worker · delegate work") || strings.Index(frame, "Parent") > strings.Index(frame, "Child") {
			t.Fatal("forward delegation order lost", frame)
		}
		for _, obsolete := range []string{"worker → lead", "▲", "⇄", "[return]", "+return", "-return"} {
			if strings.Contains(frame, obsolete) {
				t.Fatal("reverse flow UI remains", obsolete, frame)
			}
		}
	}
	// The retired key has no hidden effect and no longer creates link state.
	before := viewer.Render(time.Unix(100, 0))
	viewer = viewerKey(viewer, "r")
	if viewer.Render(time.Unix(100, 0)) != before || len(viewer.Links) != 0 {
		t.Fatal("retired return key still changes the viewer")
	}
	if state.HookCount != 1 || len(state.Agents) != 2 || len(state.Tasks) != 0 || state.Finished {
		t.Fatal("forward presentation manufactured execution")
	}
}

func BenchmarkLinkFocusWithSidebar(b *testing.B) {
	raw, err := os.ReadFile("../../examples/ai-hero/review.demo.layout.json")
	if err != nil {
		b.Fatal(err)
	}
	layout, err := ParseLayout(raw)
	if err != nil {
		b.Fatal(err)
	}
	root := b.TempDir()
	if err := SeedDemo(root, "bench"); err != nil {
		b.Fatal(err)
	}
	s, err := Snapshot(root, "bench")
	if err != nil {
		b.Fatal(err)
	}
	s.Layout = layout
	v := Viewer{State: s, Width: 140, Height: 56, Selected: 1, Links: map[string]linkFocus{"graph": {Node: "lead", Scope: 3, Inspect: true}}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Render(time.Unix(100, 0).Add(time.Duration(i) * 100 * time.Millisecond))
	}
}
