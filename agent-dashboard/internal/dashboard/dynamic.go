package dashboard

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const dynamicPrefix = "@dynamic/"
const maxExpandedPanels = 256

// Selection describes presentation membership inside the already observed scope.
// It never authorizes a connection, creates an entity or infers a relationship.
type Selection struct {
	Kind        string   `json:"kind"`
	Roles       []string `json:"roles,omitempty"`
	ScopeFile   string   `json:"scope_file,omitempty"`
	Sources     []string `json:"sources,omitempty"`
	States      []string `json:"states,omitempty"`
	IncludeLead bool     `json:"include_lead,omitempty"`
	Limit       int      `json:"limit,omitempty"`
}

// Repeat applies the same panel/subtree template to each selected identity.
// Size/weight on the original panel allocate the group; ItemSize sizes its items.
type Repeat struct {
	Select     Selection `json:"select"`
	Direction  string    `json:"direction,omitempty"`
	StackBelow int       `json:"stack_below,omitempty"`
	ItemSize   int       `json:"item_size,omitempty"`
}

type DAGNodeTemplate struct {
	Label string   `json:"label,omitempty"`
	Color string   `json:"color,omitempty"`
	Meta  []string `json:"meta,omitempty"`
}

func (q Selection) validate() error {
	switch q.Kind {
	case "actors", "tasks", "programs", "sources":
	default:
		return errors.New("select kind must be actors, tasks, programs or sources")
	}
	if q.Limit < 0 || q.Limit > 256 || len(q.Sources) > 16 || len(q.States) > 32 || len(q.Roles) > 32 {
		return errors.New("selection exceeds limits (limit 1..256, sources 16, states 32)")
	}
	if q.IncludeLead && q.Kind != "actors" || len(q.Sources) > 0 && (q.Kind == "actors" || q.Kind == "tasks") {
		return errors.New("include_lead belongs to actors; sources filters belong to Rex collections")
	}
	if q.ScopeFile != "" && (!filepath.IsAbs(q.ScopeFile) || len(q.ScopeFile) > 4096 || Clean(q.ScopeFile, 4096) != q.ScopeFile) {
		return errors.New("scope_file must be an absolute bounded private path")
	}
	if len(q.Roles) > 0 && q.Kind != "actors" {
		return errors.New("roles filters match reported Claude agent_type only; use scope_file for semantic classification")
	}
	for _, values := range [][]string{q.Sources, q.States, q.Roles} {
		seen := map[string]bool{}
		for _, value := range values {
			if value == "" || len(value) > 80 || Clean(value, 80) != value || seen[value] {
				return errors.New("selection filters must be unique bounded printable values")
			}
			seen[value] = true
		}
	}
	return nil
}

func (v *observationView) candidates(q Selection) []Binding {
	var candidates []Binding
	switch q.Kind {
	case "actors":
		candidates = append(candidates, v.Actors...)
		if q.IncludeLead {
			if _, ok := v.Entities[Binding{Actor: "lead"}]; ok {
				candidates = append(candidates, Binding{Actor: "lead"})
			}
		}
	case "tasks":
		candidates = append(candidates, v.Tasks...)
	case "programs":
		candidates = append(candidates, v.Programs...)
	case "sources":
		candidates = append(candidates, v.SourceBindings...)
	}
	// Stable identity order, not status/progress/map order. A status update must
	// not reshuffle cards or reidentify existing graph nodes.
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].label() < candidates[j].label() })
	out := []Binding{}
	seen := map[Binding]bool{}
	for _, b := range candidates {
		e, ok := v.Entities[b]
		if !ok || seen[b] || !contains(q.Sources, e.Source) || !contains(q.States, e.Status) || !contains(q.Roles, e.Role) {
			continue
		}
		seen[b] = true
		out = append(out, b)
	}
	return out
}

func (v *observationView) selected(q Selection) ([]Binding, int) {
	out := []Binding{}
	for _, b := range v.candidates(q) {
		if v.scopeDecision(q, b) == "include" {
			out = append(out, b)
		}
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 64
	}
	limit = min(limit, 256)
	omitted := max(0, len(out)-limit)
	return out[:min(len(out), limit)], omitted
}

func selectionNotice(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("Dynamic selection: %d additional observations omitted by the display limit", n)
}

// Validate template scope separately from ordinary geometry/binding validation.
// Item references are syntax, not identities; none may reach record resolution.
func validateDynamicSyntax(p *Panel, itemKind string) error {
	count := 0
	return walkDynamicSyntax(p, itemKind, &count, 0)
}
func walkDynamicSyntax(p *Panel, itemKind string, count *int, depth int) error {
	*count = *count + 1
	if *count > 64 || depth > 8 {
		return errors.New("layout limited to 64 declarations and depth 8")
	}

	if strings.HasPrefix(p.ID, dynamicPrefix) {
		return errors.New("panel IDs beginning @dynamic/ are reserved")
	}
	if p.Repeat != nil {
		r := p.Repeat
		if err := r.Select.validate(); err != nil {
			return err
		}
		if p.ID == "" || r.Direction != "" && r.Direction != "row" && r.Direction != "column" || r.StackBelow < 0 || r.StackBelow > 1000 || r.ItemSize < 0 || r.ItemSize > 1000 {
			return errors.New("repeat needs an ID, row/column direction and bounded geometry")
		}
		if r.StackBelow > 0 && r.Direction != "row" {
			return errors.New("repeat stack_below requires row direction")
		}
		itemKind = r.Select.Kind
	}
	check := func(b Binding) error {
		if err := b.validate(); err != nil {
			return err
		}
		if b.Item && itemKind == "" {
			return errors.New("item binding requires an enclosing repeat")
		}

		if p.Kind == "programs" && (b.Item && itemKind != "programs" || !b.Item && b.Program == "") || p.Kind == "sources" && (b.Item && itemKind != "sources" || !b.Item && b.Source == "") {
			return errors.New("programs/sources bindings must keep their corresponding entity type")
		}
		return nil
	}
	if p.Bind != nil {
		if err := check(*p.Bind); err != nil {
			return err
		}
	}
	for _, b := range p.Bindings {
		if err := check(b); err != nil {
			return err
		}
	}
	if p.Select != nil {
		if err := p.Select.validate(); err != nil {
			return err
		}
		switch p.Kind {
		case "agents", "tasks", "events", "programs", "sources":
		default:
			return errors.New("select belongs to collection panels; repeat supplies single cards, text and split trees")
		}
		if p.Bind != nil || len(p.Bindings)+len(p.Actors)+len(p.TaskIDs)+len(p.Sources)+len(p.ProgramIDs) > 0 || p.Actor != "" {
			return errors.New("select cannot mix with explicit bindings or legacy filters")
		}
		if p.Kind == "programs" && p.Select.Kind != "programs" || p.Kind == "sources" && p.Select.Kind != "sources" {
			return errors.New("programs/sources panels select their corresponding typed collection")
		}
	}
	if p.DAG != nil {
		for _, n := range p.DAG.Nodes {
			if strings.HasPrefix(n.ID, dynamicPrefix) {
				return errors.New("DAG IDs beginning @dynamic/ are reserved")
			}
			if n.Bind != nil {
				if err := check(*n.Bind); err != nil {
					return err
				}
			}
		}
	}
	for i := range p.Children {
		if err := walkDynamicSyntax(&p.Children[i], itemKind, count, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func dynamicID(context, template string, b Binding) string {
	raw, _ := json.Marshal([]any{context, template, b})
	return fmt.Sprintf("%s%x", dynamicPrefix, sha256.Sum256(raw))
}

func itemBinding(b Binding, item *Binding) Binding {
	if b.Item && item != nil {
		return *item
	}
	return b
}

// Materialization is a pure presentation projection. Author declarations and
// snapshots remain unchanged; every input path (draw, focus, scrolling, link
// inspection and history anchoring) uses the same effective layout.
func presentationLayout(s *State, now time.Time) *Layout {
	if s == nil || s.Layout == nil {
		return nil
	}
	v := observations(s, now)
	count := 0
	var expand func(Panel, *Binding, string, int) (Panel, error)
	expand = func(p Panel, item *Binding, context string, depth int) (Panel, error) {
		count++
		if count > maxExpandedPanels || depth > 16 {
			return Panel{}, errors.New("dynamic layout exceeds 256 expanded panels or depth 16; narrow selections")
		}
		if p.Repeat != nil {
			r := *p.Repeat
			p.Repeat = nil
			members, omitted := v.selected(r.Select)
			group := Panel{ID: p.ID, Kind: r.Direction, Size: p.Size, Weight: p.Weight, StackBelow: r.StackBelow}
			if context != "" && item != nil {
				group.ID = dynamicID(context, p.ID, *item)
			}
			if group.Kind == "" {
				group.Kind = "column"
			}
			for _, b := range members {
				template := p
				template.Size, template.Weight = r.ItemSize, 1
				child, err := expand(template, &b, group.ID, depth+1)
				if err != nil {
					return Panel{}, err
				}
				group.Children = append(group.Children, child)
			}
			if len(members) == 0 || omitted > 0 || r.Select.ScopeFile != "" {
				message := selectionNotice(omitted)
				if len(members) == 0 {
					message = "No matching observations; waiting within the declared scope."
				}
				if scope := v.scopeNotice(r.Select); scope != "" {
					message += " · " + scope
				}
				count++
				group.Children = append(group.Children, Panel{ID: dynamicID(group.ID, "notice", Binding{}), Kind: "text", notice: message})
			}
			return group, nil
		}
		if context != "" && item != nil {
			if p.ID != "" {
				p.ID = dynamicID(context, p.ID, *item)
			}
			if p.Title == "" {
				e, _ := v.resolve(*item)
				p.Title = e.Label
				if p.Title == "" {
					p.Title = item.label()
				}
			}
		}
		if p.Bind != nil {
			b := itemBinding(*p.Bind, item)
			p.Bind = &b
		}

		bindings := p.Bindings
		p.Bindings = nil
		seenBindings := map[Binding]bool{}
		for _, b := range bindings {
			resolved := itemBinding(b, item)
			if !seenBindings[resolved] {
				p.Bindings = append(p.Bindings, resolved)
				seenBindings[resolved] = true
			}
		}
		if p.DAG != nil {
			g := *p.DAG
			g.Nodes = append([]DAGNode(nil), g.Nodes...)
			g.Edges = append([]DAGEdge(nil), g.Edges...)
			for i, n := range g.Nodes {
				if n.Bind != nil {
					b := itemBinding(*n.Bind, item)
					g.Nodes[i].Bind = &b
				}
			}
			if g.Select != nil {
				members, omitted := v.selected(*g.Select)
				seen := map[Binding]bool{}
				for _, n := range g.Nodes {
					if n.Bind != nil {
						seen[*n.Bind] = true
					}
				}
				for _, b := range members {
					if seen[b] {
						continue
					}
					if len(g.Nodes) == 64 {
						omitted++
						continue
					}
					seen[b] = true
					e, _ := v.resolve(b)
					n := DAGNode{ID: dynamicID(p.ID, "node", b), Label: e.Label, Bind: &b}
					if n.Label == "" {
						n.Label = b.label()
					}
					if g.Node != nil {
						n.Color, n.Meta = g.Node.Color, g.Node.Meta
						if g.Node.Label != "" {
							n.Label = g.Node.Label
						}
					}
					n.Label = boundedLabel(n.Label)
					g.Nodes = append(g.Nodes, n)
					if g.AttachFrom != "" {
						if len(g.Edges) == 256 {
							return Panel{}, errors.New("dynamic DAG exceeds 256 edges; narrow selection")
						}
						g.Edges = append(g.Edges, DAGEdge{From: g.AttachFrom, To: n.ID, Label: "caller-declared grouping"})
					}
				}
				g.notice = selectionNotice(omitted)
				if scope := v.scopeNotice(*g.Select); scope != "" {
					if g.notice != "" {
						g.notice += " · "
					}
					g.notice += scope
				}
			}
			p.DAG = &g
		}
		p.Children = append([]Panel(nil), p.Children...)
		for i, child := range p.Children {
			var err error
			p.Children[i], err = expand(child, item, context, depth+1)
			if err != nil {
				return Panel{}, err
			}
		}
		return p, nil
	}
	root, err := expand(s.Layout.Root, nil, "", 0)
	if err != nil || count > maxExpandedPanels {
		message := "dynamic layout exceeds 256 expanded panels; narrow selections"
		if err != nil {
			message = err.Error()
		}
		root = Panel{ID: dynamicPrefix + "error", Kind: "text", notice: "Dynamic layout diagnostic: " + message}
	}
	return &Layout{Version: s.Layout.Version, Name: s.Layout.Name, Root: root}
}

func (v Viewer) canvasLayout() *Layout { return presentationLayout(v.State, time.Time{}) }

func (v *Viewer) prunePresentationState(layout *Layout) {
	present := map[string]bool{}
	for _, p := range leaves(layout.Root) {
		present[p.ID] = true
		if p.Kind != "dag" && v.Offsets[p.ID] > 0 {
			v.Offsets[p.ID] = min(v.Offsets[p.ID], max(0, len(observationRows(p, v.State, time.Time{}))-1))
		}
	}
	for id := range v.Offsets {
		if !present[id] {
			delete(v.Offsets, id)
		}
	}
	for id := range v.XOffsets {
		if !present[id] {
			delete(v.XOffsets, id)
		}
	}
}

func boundedLabel(value string) string {
	value = Clean(value, 160)
	for len(value) > 160 {
		value = string([]rune(value)[:len([]rune(value))-1])
	}
	return value
}

func hasAgentScope(p Panel) bool {
	if p.Select != nil && p.Select.ScopeFile != "" || p.Repeat != nil && p.Repeat.Select.ScopeFile != "" || p.DAG != nil && p.DAG.Select != nil && p.DAG.Select.ScopeFile != "" {
		return true
	}
	for _, child := range p.Children {
		if hasAgentScope(child) {
			return true
		}
	}
	return false
}
