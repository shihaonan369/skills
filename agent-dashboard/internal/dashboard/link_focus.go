package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Link focus is viewer-local presentation of declared reachability, not traffic.
// Zero values deliberately mean no selection, direct neighbors, returns included.
type linkFocus struct {
	Node           string
	Scope          int
	ExcludeReturns bool
	Inspect        bool
}

func (f linkFocus) scopeName() string {
	return []string{"neighbors", "upstream", "downstream", "both"}[f.Scope%4]
}

type linkRelations struct {
	up, down           map[string]bool
	incoming, outgoing map[int]bool
}

func declaredRelations(g *DAG, f linkFocus) linkRelations {
	r := linkRelations{map[string]bool{}, map[string]bool{}, map[int]bool{}, map[int]bool{}}
	if f.Node == "" {
		return r
	}
	trace := func(up bool) {
		seen := map[string]bool{f.Node: true}
		queue := []string{f.Node}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			for i, e := range g.Edges {
				if f.ExcludeReturns && e.Style == "return" {
					continue
				}
				from, to := e.From, e.To
				if up {
					from, to = to, from
				}
				if from != id {
					continue
				}
				if up {
					r.up[to], r.incoming[i] = true, true
				} else {
					r.down[to], r.outgoing[i] = true, true
				}
				if f.Scope != 0 && !seen[to] {
					seen[to] = true
					queue = append(queue, to)
				}
			}
		}
	}
	if f.Scope != 2 {
		trace(true)
	}
	if f.Scope != 1 {
		trace(false)
	}
	return r
}

func relationMark(up, down bool) string {
	if up && down {
		return "="
	}
	if up {
		return "<"
	}
	if down {
		return ">"
	}
	return "."
}
func relationColor(mark string) string {
	switch mark {
	case "*", "=":
		return palette["purple"]
	case "<":
		return palette["peach"]
	case ">":
		return palette["mint"]
	default:
		return "#69778F"
	}
}
func (r linkRelations) nodeMark(id, selected string) string {
	if id == selected {
		return "*"
	}
	return relationMark(r.up[id], r.down[id])
}
func (r linkRelations) edgeMark(i int) string {
	return relationMark(r.incoming[i], r.outgoing[i])
}

// The sidebar lives inside the caller's existing leaf rectangle. Narrow leaves
// retain their graph viewport and get a hint, never an implicit split/stack.
func dagViewport(width int, f linkFocus) (graph, side int) {
	if f.Inspect && width >= 80 {
		return width - 33, 32
	}
	return width, 0
}
func focusedGeometryWidth(g dagGeometry, f linkFocus) int {
	if f.Node != "" {
		return g.width + 4 // prefixes in the existing scrollable edge legend
	}
	return g.width
}

func linkDetails(g *DAG, s *State, f linkFocus, r linkRelations) []string {
	out := []string{"DETAILS · declared relations"}
	if f.Node == "" {
		return append(out, "[ / ] selects a node", "No relationship inferred.")
	}
	out = append(out, "Node "+f.Node)
	for _, n := range g.Nodes {
		if n.ID != f.Node {
			continue
		}
		out = append(out, unknown(n.Label))
		status := "declared only"
		if n.Bind != nil {
			status = "unobserved · unknown"
			if n.Bind.Actor != "" {
				out = append(out, "Actor "+n.Bind.Actor)
				if a := s.Agents[n.Bind.Actor]; a != nil {
					status = unknown(a.Status)
				}
			} else {
				out = append(out, "Task "+n.Bind.Task)
				if t, ok := s.Tasks[n.Bind.Task]; ok {
					status = unknown(t.Status)
				}
			}
		}
		out = append(out, "State "+status)
	}
	returns := "included"
	if f.ExcludeReturns {
		returns = "excluded from focus"
	}
	out = append(out, "Scope "+f.scopeName(), "Return "+returns, "[*] selected [=] both")
	for _, group := range []struct {
		name string
		ids  map[string]bool
	}{{"[<] Upstream", r.up}, {"[>] Downstream", r.down}} {
		ids := []string{}
		for _, n := range g.Nodes {
			if n.ID != f.Node && group.ids[n.ID] {
				ids = append(ids, n.ID)
			}
		}
		if len(ids) == 0 {
			ids = append(ids, "none")
		}
		out = append(out, group.name+": "+strings.Join(ids, ", "))
	}
	out = append(out, "Edges · legend has full labels")
	count := 0
	for i, e := range g.Edges {
		mark := r.edgeMark(i)
		if mark != "." {
			out = append(out, "["+mark+"] "+dagEdgeText(e))
			count++
		}
	}
	if count == 0 {
		out = append(out, "No matching declared edges.")
	}
	return out
}
func drawLinkDetails(graph []string, g *DAG, s *State, f linkFocus, r linkRelations, width, height int) []string {
	details := linkDetails(g, s, f, r)
	if len(details) > height && height > 0 {
		details = append(append([]string(nil), details[:height-1]...), fmt.Sprintf("+%d lines · scroll graph legend", len(details)-height+1))
	}
	side := fit(details, width, height)
	for i := range graph {
		graph[i] += lipgloss.NewStyle().Foreground(lipgloss.Color(palette["purple"])).Render("│") + side[i]
	}
	return graph
}

func (v Viewer) focusedDAG() (Panel, bool) {
	if v.Page != 0 || v.Help || v.State == nil || v.State.Layout == nil {
		return Panel{}, false
	}
	for _, p := range leaves(v.State.Layout.Root) {
		if p.ID == v.focus() && p.Kind == "dag" && p.DAG != nil && len(p.DAG.Nodes) > 0 && len(p.DAG.Nodes) <= 64 {
			return p, true
		}
	}
	return Panel{}, false
}
func (v *Viewer) linkKey(key string) {
	p, ok := v.focusedDAG()
	if !ok {
		return
	}
	if v.Links == nil {
		v.Links = map[string]linkFocus{}
	}
	f := v.Links[p.ID]
	switch key {
	case "[", "]":
		index := -1
		for i, n := range p.DAG.Nodes {
			if n.ID == f.Node {
				index = i
			}
		}
		if key == "]" {
			index = (index + 1) % len(p.DAG.Nodes)
		} else if index <= 0 {
			index = len(p.DAG.Nodes) - 1
		} else {
			index--
		}
		f.Node = p.DAG.Nodes[index].ID
	case "c":
		f.Node = ""
	case "f":
		f.Scope = (f.Scope + 1) % 4
	case "r":
		f.ExcludeReturns = !f.ExcludeReturns
	case "i":
		f.Inspect = !f.Inspect
	}
	v.Links[p.ID] = f
}
func (v *Viewer) reconcileLinks(next *State) {
	retained := map[string]linkFocus{}
	if next.Layout != nil {
		for _, p := range leaves(next.Layout.Root) {
			if p.Kind != "dag" || p.DAG == nil {
				continue
			}
			if f, ok := v.Links[p.ID]; ok {
				found := false
				for _, n := range p.DAG.Nodes {
					found = found || n.ID == f.Node
				}
				if !found {
					f.Node = ""
				}
				retained[p.ID] = f
			}
		}
	}
	v.Links = retained
}
