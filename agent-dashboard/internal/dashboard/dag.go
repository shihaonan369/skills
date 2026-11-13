package dashboard

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// DAG is presentation only. Bindings never create agents/tasks and edges never
// execute work. Return edges are excluded from the acyclic ranking constraint.
type DAG struct {
	Version int       `json:"version"`
	Name    string    `json:"name,omitempty"`
	Nodes   []DAGNode `json:"nodes"`
	Edges   []DAGEdge `json:"edges,omitempty"`
}
type DAGNode struct {
	ID    string      `json:"id"`
	Label string      `json:"label,omitempty"`
	Color string      `json:"color,omitempty"`
	Bind  *DAGBinding `json:"bind,omitempty"`
	Meta  []string    `json:"meta,omitempty"`
}
type DAGBinding struct {
	Actor string `json:"actor,omitempty"`
	Task  string `json:"task,omitempty"`
}
type DAGEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Style string `json:"style,omitempty"`
}

func validateDAG(g *DAG) error {
	if g == nil || g.Version != 1 {
		return errors.New("dag requires version 1")
	}
	if len(g.Nodes) == 0 || len(g.Nodes) > 64 || len(g.Edges) > 256 {
		return errors.New("dag limited to 1..64 nodes and 256 edges")
	}
	if len(g.Name) > 160 {
		return errors.New("dag name too long")
	}
	g.Name = Clean(g.Name, 160)
	ids := map[string]bool{}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.ID == "" || len(n.ID) > 80 || Clean(n.ID, 80) != n.ID || ids[n.ID] {
			return errors.New("dag node IDs must be unique, printable, nonempty and at most 80 bytes")
		}
		ids[n.ID] = true
		if len(n.Label) > 160 || len(n.Meta) > 4 {
			return errors.New("dag label/meta exceeds limit")
		}
		n.Label = Clean(n.Label, 160)
		if _, ok := palette[n.Color]; !ok && n.Color != "" {
			return errors.New("unknown dag node color")
		}
		for j, m := range n.Meta {
			if len(m) > 2000 {
				return errors.New("dag meta line exceeds 2000 bytes")
			}
			n.Meta[j] = Clean(m, 400)
		}
		if n.Bind != nil {
			b := n.Bind
			if (b.Actor == "") == (b.Task == "") {
				return errors.New("dag bind requires exactly one actor or task")
			}
			for _, id := range []string{b.Actor, b.Task} {
				if len(id) > 120 || Clean(id, 120) != id {
					return errors.New("dag binding ID must be printable and at most 120 bytes")
				}
			}
		}
	}
	seen := map[string]bool{}
	for i := range g.Edges {
		e := &g.Edges[i]
		if !ids[e.From] || !ids[e.To] {
			return errors.New("dag edge references unknown node")
		}
		if e.Style != "" && e.Style != "solid" && e.Style != "dashed" && e.Style != "return" {
			return errors.New("unknown dag edge style")
		}
		key := e.From + "\x00" + e.To
		if seen[key] {
			return errors.New("duplicate dag edge")
		}
		seen[key] = true
		if len(e.Label) > 160 {
			return errors.New("dag edge label too long")
		}
		e.Label = Clean(e.Label, 160)
	}
	_, err := dagRanks(g)
	return err
}
func dagRanks(g *DAG) (map[string]int, error) {
	ranks := map[string]int{}
	degree := map[string]int{}
	next := map[string][]string{}
	for _, n := range g.Nodes {
		degree[n.ID] = 0
	}
	for _, e := range g.Edges {
		if e.Style != "return" {
			degree[e.To]++
			next[e.From] = append(next[e.From], e.To)
		}
	}
	queue := []string{}
	for _, n := range g.Nodes {
		if degree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, to := range next[id] {
			ranks[to] = max(ranks[to], ranks[id]+1)
			degree[to]--
			if degree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}
	if visited != len(g.Nodes) {
		return nil, errors.New("dag cycle among ordinary edges; mark the closing edge style return")
	}
	return ranks, nil
}

// State is always the literal last observed state. In particular waiting,
// returned, stopped and failed must not turn into a green completion check.
func dagNodeState(n DAGNode, s *State, now time.Time) (string, string, bool) {
	if n.Bind == nil {
		return "declared only", palette["gray"], false
	}
	state := ""
	prefix := ""
	if n.Bind.Actor != "" {
		a := s.Agents[n.Bind.Actor]
		if a == nil {
			return "unobserved · unknown", palette["gray"], false
		}
		state = a.Status
		if state == "running" {
			prefix = fmt.Sprintf(" · tools %d", a.Tools)
		}
	} else {
		t, ok := s.Tasks[n.Bind.Task]
		if !ok {
			return "unobserved · unknown", palette["gray"], false
		}
		state = t.Status
	}
	switch state {
	case "running", "in_progress":
		spinner := []string{"◐", "◓", "◑", "◒"}[((now.UnixMilli()/500)%4+4)%4]
		return spinner + " " + state + prefix, palette["amber"], true
	case "completed", "done":
		return "✓ " + state, palette["green"], false
	case "failed", "blocked", "api_error", "API error":
		return "! " + state, palette["red"], false
	case "":
		return "unknown", palette["gray"], false
	default:
		return Clean(state, 80), palette["gray"], false
	}
}

type dagBox struct{ x, y, w, h int }
type dagRoute struct {
	edge           DAGEdge
	source, target dagBox
	out, in, rail  int
}
type dagPoint struct{ x, y int }

func (r dagRoute) points() []dagPoint {
	a, b := r.source, r.target
	sx, sy, tx, ty := a.x+a.w/2, a.y+a.h-1, b.x+b.w/2, b.y-1
	points := []dagPoint{{sx, sy}, {sx, r.out}}
	if r.rail >= 0 {
		points = append(points, dagPoint{r.rail, r.out}, dagPoint{r.rail, r.in}, dagPoint{tx, r.in})
	} else {
		points = append(points, dagPoint{tx, r.out})
	}
	return append(points, dagPoint{tx, ty})
}
func effectColor(hex string, brightness float64) string {
	rgb, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 24)
	if err != nil {
		return hex
	}
	scale := func(v uint64) int { return min(255, max(0, int(float64(v)*brightness))) }
	return fmt.Sprintf("#%02X%02X%02X", scale((rgb>>16)&255), scale((rgb>>8)&255), scale(rgb&255))
}

// Packets are decorative, never evidence of data moving. They travel in source
// to destination order on the exact stroke polyline, including external rails.
func drawFlowPacket(c *dagCanvas, r dagRoute, index int, clock time.Time, color string) {
	points := r.points()
	length := 0
	for i := 1; i < len(points); i++ {
		length += abs(points[i].x-points[i-1].x) + abs(points[i].y-points[i-1].y)
	}
	phase := int((clock.UnixMilli()/70 + int64(index*7)) % 24)
	if phase < 0 {
		phase += 24
	}
	tailColor, midColor, headColor := effectColor(color, 0.45), effectColor(color, 0.8), effectColor(color, 1.3)
	for head := phase; head < length; head += 24 {
		for trail := 3; trail >= 0; trail-- {
			distance := head - trail
			if distance <= 0 {
				continue
			}
			for i := 1; i < len(points); i++ {
				a, b := points[i-1], points[i]
				segment := abs(b.x-a.x) + abs(b.y-a.y)
				if distance > segment {
					distance -= segment
					continue
				}
				if segment == 0 {
					continue
				}
				x, y := a.x, a.y
				if b.x > a.x {
					x += distance
				}
				if b.x < a.x {
					x -= distance
				}
				if b.y > a.y {
					y += distance
				}
				if b.y < a.y {
					y -= distance
				}
				glyph, tint := "·", tailColor
				if trail == 1 {
					glyph, tint = "•", midColor
				}
				if trail == 0 {
					glyph, tint = "●", headColor
				}
				c.text(x, y, glyph, tint)
				break
			}
		}
	}
}

// Opposite endpoint pairs are independent declarations, not inferred replies.
func hasOpposingEdges(g *DAG) bool {
	seen := map[[2]string]bool{}
	for _, e := range g.Edges {
		if e.From != e.To && seen[[2]string{e.To, e.From}] {
			return true
		}
		seen[[2]string{e.From, e.To}] = true
	}
	return false
}
func flowCaption(g *DAG, motion bool) string {
	if hasOpposingEdges(g) {
		if !motion {
			return "Flow off · ⇄ paired arrows"
		}
		return "Flow ⇄ paired arrows · decorative"
	}
	if !motion {
		return "Flow off · a toggles motion"
	}
	return "Flow animation · decorative"
}

type dagGeometry struct {
	boxes                 map[string]dagBox
	routes                []dagRoute
	width, height, legend int
}

// Use stable declared order within longest-path ranks. Every long/return edge
// has an external rail, so it cannot cut through an intermediate rank's boxes.
func placeDAG(g *DAG) (dagGeometry, error) {
	ranks, err := dagRanks(g)
	if err != nil {
		return dagGeometry{}, err
	}
	rows := make([][]DAGNode, len(g.Nodes))
	maxrank := 0
	for _, n := range g.Nodes {
		r := ranks[n.ID]
		rows[r] = append(rows[r], n)
		maxrank = max(maxrank, r)
	}
	incoming := map[string][]int{}
	outgoing := map[string][]int{}
	external := map[int]int{}
	width := 0
	for _, row := range rows {
		width = max(width, len(row)*34-4)
	}
	for i, e := range g.Edges {
		outgoing[e.From] = append(outgoing[e.From], i)
		if e.Style == "return" || ranks[e.To] != ranks[e.From]+1 {
			external[i] = width + 2 + 2*len(external)
			incoming[e.To] = append(incoming[e.To], i)
		}
	}
	geo := dagGeometry{boxes: map[string]dagBox{}, routes: make([]dagRoute, len(g.Edges)), width: width + 2*len(external) + 3}
	y := 2
	for r := 0; r <= maxrank; r++ {
		row := rows[r]
		ins, outs, meta := 0, 0, 0
		for _, n := range row {
			ins += len(incoming[n.ID])
			outs += len(outgoing[n.ID])
			meta = max(meta, len(n.Meta))
		}
		boxy := y + ins
		x := (width - (len(row)*34 - 4)) / 2
		inIndex, outIndex := 0, 0
		for _, n := range row {
			b := dagBox{x, boxy, 30, 4 + meta}
			geo.boxes[n.ID] = b
			x += 34
			for _, idx := range incoming[n.ID] {
				geo.routes[idx].in = y + inIndex
				inIndex++
			}
			for _, idx := range outgoing[n.ID] {
				geo.routes[idx].out = boxy + b.h + outIndex
				outIndex++
			}
		}
		y = boxy + 4 + meta + outs + 1
	}
	for i, e := range g.Edges {
		p := &geo.routes[i]
		p.edge = e
		p.source = geo.boxes[e.From]
		p.target = geo.boxes[e.To]
		p.rail = -1
		if rail, ok := external[i]; ok {
			p.rail = rail
		}
	}
	geo.legend = y
	geo.height = y + len(g.Edges)
	geo.width = max(geo.width, ansi.StringWidth(flowCaption(g, true)))
	for _, e := range g.Edges {
		geo.width = max(geo.width, ansi.StringWidth(dagEdgeText(e)))
	}
	return geo, nil
}

type dagCell struct {
	text, color string
	mask        uint8
	dashed      bool
}
type dagCanvas struct {
	cells      [][]dagCell
	x, y, w, h int
}

func newDAGCanvas(w, h, x, y int) *dagCanvas {
	c := &dagCanvas{w: w, h: h, x: x, y: y, cells: make([][]dagCell, h)}
	for i := range c.cells {
		c.cells[i] = make([]dagCell, w)
	}
	return c
}
func (c *dagCanvas) cell(x, y int) *dagCell {
	x -= c.x
	y -= c.y
	if x < 0 || x >= c.w || y < 0 || y >= c.h {
		return nil
	}
	return &c.cells[y][x]
}
func (c *dagCanvas) text(x, y int, s, color string) {
	// Write whole graphemes only. Partial wide glyphs at either viewport edge
	// stay blank, rather than corrupting the neighboring terminal cell.
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		text := gr.Str()
		w := ansi.StringWidth(text)
		if w == 0 {
			continue
		}
		if x >= c.x && x+w <= c.x+c.w {
			if cell := c.cell(x, y); cell != nil {
				*cell = dagCell{text: text, color: color}
				for i := 1; i < w; i++ {
					*c.cell(x+i, y) = dagCell{text: "\x00"}
				}
			}
		}
		x += w
	}
}
func (c *dagCanvas) line(x1, y1, x2, y2 int, color string, dashed bool) {
	if x1 != x2 && y1 != y2 {
		return
	}
	dx, dy := 0, 0
	if x2 > x1 {
		dx = 1
	}
	if x2 < x1 {
		dx = -1
	}
	if y2 > y1 {
		dy = 1
	}
	if y2 < y1 {
		dy = -1
	}
	steps := max(abs(x2-x1), abs(y2-y1))
	for i := 0; i <= steps; i++ {
		cell := c.cell(x1+dx*i, y1+dy*i)
		if cell == nil {
			continue
		}
		// Bit order: up, right, down, left. Junctions remain crossings, not nodes.
		if dx != 0 {
			if i > 0 {
				if dx > 0 {
					cell.mask |= 8
				} else {
					cell.mask |= 2
				}
			}
			if i < steps {
				if dx > 0 {
					cell.mask |= 2
				} else {
					cell.mask |= 8
				}
			}
		}
		if dy != 0 {
			if i > 0 {
				if dy > 0 {
					cell.mask |= 1
				} else {
					cell.mask |= 4
				}
			}
			if i < steps {
				if dy > 0 {
					cell.mask |= 4
				} else {
					cell.mask |= 1
				}
			}
		}
		cell.color = color
		cell.dashed = dashed
	}
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func (c *dagCanvas) lines() []string {
	glyph := []string{" ", "│", "─", "└", "│", "│", "┌", "├", "─", "┘", "─", "┴", "┐", "┤", "┬", "┼"}
	out := make([]string, c.h)
	for y, row := range c.cells {
		var b, run strings.Builder
		color := ""
		flush := func() {
			if color == "" {
				b.WriteString(run.String())
			} else {
				b.WriteString(shade(color, run.String()))
			}
			run.Reset()
		}
		for _, cell := range row {
			if cell.text == "\x00" {
				continue
			}
			text := cell.text
			if text == "" {
				text = glyph[cell.mask]
				if cell.dashed {
					if cell.mask == 5 || cell.mask == 1 || cell.mask == 4 {
						text = "┆"
					}
					if cell.mask == 10 || cell.mask == 2 || cell.mask == 8 {
						text = "·"
					}
				}
			}
			if cell.color != color {
				flush()
				color = cell.color
			}
			run.WriteString(text)
		}
		flush()
		out[y] = b.String()
	}
	return out
}
func dagEdgeText(e DAGEdge) string {
	label := e.From + " → " + e.To
	if e.Style != "" {
		label += " [" + e.Style + "]"
	}
	if e.Label != "" {
		label += " · " + e.Label
	}
	return label
}

func renderDAGPanel(p Panel, s *State, w, h int, o canvasOptions, title, color string) []string {
	// Layouts loaded from disk may have been edited outside the CLI. Validate a
	// copy before drawing rather than trusting persisted graphs or mutating state.
	g := p.DAG
	if g == nil {
		return fit([]string{"Invalid DAG: missing graph"}, w, h)
	}
	// Geometry is bounded by the validated graph limits; normal publication is
	// validated before committing. Defensive validation also covers manual edits.
	clone := *g
	clone.Nodes = append([]DAGNode(nil), g.Nodes...)
	clone.Edges = append([]DAGEdge(nil), g.Edges...)
	for i := range clone.Nodes {
		clone.Nodes[i].Meta = append([]string(nil), g.Nodes[i].Meta...)
	}
	if err := validateDAG(&clone); err != nil {
		return fit([]string{"Invalid DAG: " + Clean(err.Error(), 200)}, w, h)
	}
	g = &clone
	geo, err := placeDAG(g)
	if err != nil {
		return fit([]string{Clean(err.Error(), 200)}, w, h)
	}
	width, height := w-2, h-3
	x := min(max(0, o.XOffsets[p.ID]), max(0, geo.width-width))
	y := min(max(0, o.Offsets[p.ID]), max(0, geo.height-height))
	c := newDAGCanvas(width, height, x, y)
	// Honest labels scroll with the canvas; the outer session badge remains.
	c.text(0, 0, "Declared edges · observed states", palette["gray"])
	c.text(0, 1, flowCaption(g, o.Motion), palette["gray"])
	colors := map[string]string{}
	for _, n := range g.Nodes {
		colors[n.ID] = palette[n.Color]
		if colors[n.ID] == "" {
			colors[n.ID] = palette["blue"]
		}
	}
	for _, route := range geo.routes {
		points := route.points()
		edgeColor := effectColor(colors[route.edge.From], 0.55)
		if o.Focus == p.ID {
			edgeColor = effectColor(colors[route.edge.From], 0.8)
		}
		dashed := route.edge.Style == "dashed" || route.edge.Style == "return"
		for i := 1; i < len(points); i++ {
			a, b := points[i-1], points[i]
			c.line(a.x, a.y, b.x, b.y, edgeColor, dashed)
		}
	}
	if o.Motion {
		for i, route := range geo.routes {
			drawFlowPacket(c, route, i, o.Clock, colors[route.edge.From])
		}
	}
	for _, route := range geo.routes {
		b := route.target
		c.text(b.x+b.w/2, b.y-1, "▼", palette["gray"])
		// Keep the rail direction visible even as particles travel past it.
		// A report to a higher-ranked parent travels upward, not as a second
		// copy of the parent's downward delegation animation.
		if route.rail >= 0 && route.in != route.out {
			arrow := "▼"
			if route.in < route.out {
				arrow = "▲"
			}
			c.text(route.rail, (route.in+route.out)/2, arrow, colors[route.edge.From])
		}
	}
	for _, n := range g.Nodes {
		b := geo.boxes[n.ID]
		nc := palette[n.Color]
		if nc == "" {
			nc = palette["blue"]
		}
		state, sc, active := dagNodeState(n, s, o.Clock)
		if active && o.Motion {
			// A gentle brightness pulse is conditional on observed activity only.
			phase := float64(o.Clock.UnixMilli()%1800) * 2 * math.Pi / 1800
			nc = effectColor(nc, 1.08+0.18*math.Sin(phase))
		}
		c.text(b.x, b.y, "╭"+strings.Repeat("─", b.w-2)+"╮", nc)
		for yy := b.y + 1; yy < b.y+b.h-1; yy++ {
			c.text(b.x, yy, "│"+strings.Repeat(" ", b.w-2)+"│", nc)
		}
		c.text(b.x, b.y+b.h-1, "╰"+strings.Repeat("─", b.w-2)+"╯", nc)
		label := n.Label
		if label == "" {
			label = n.ID
		}
		c.text(b.x+2, b.y+1, clip(label, b.w-4), nc)
		c.text(b.x+2, b.y+2, clip(state, b.w-4), sc)
		for i, m := range n.Meta {
			c.text(b.x+2, b.y+3+i, clip(m, b.w-4), palette["gray"])
		}
	}
	// Preserve every edge label in a scrollable legend, even when an inline
	// label could not fit. This also makes crossings unambiguous by endpoint ID.
	for i, e := range g.Edges {
		c.text(0, geo.legend+i, dagEdgeText(e), palette["gray"])
	}
	title += fmt.Sprintf(" [x %d/%d · y %d/%d]", x, max(0, geo.width-width), y, max(0, geo.height-height))
	body := append([]string{panelHeading(title, color, width, o.Focus == p.ID)}, c.lines()...)
	styled := lipgloss.NewStyle().Border(panelBorder(o.Focus == p.ID)).BorderForeground(lipgloss.Color(color)).Width(width).Render(strings.Join(body, "\n"))
	return fit(strings.Split(styled, "\n"), w, h)
}
