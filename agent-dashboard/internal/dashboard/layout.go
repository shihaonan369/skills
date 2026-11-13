package dashboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Layout is caller-owned presentation, never an orchestration/workflow engine.
// A split tree is the only composition interface; leaves bind to observed data.
type Layout struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Root    Panel  `json:"root"`
}
type Panel struct {
	ID         string   `json:"id,omitempty"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title,omitempty"`
	Color      string   `json:"color,omitempty"`
	Size       int      `json:"size,omitempty"`
	Weight     int      `json:"weight,omitempty"`
	StackBelow int      `json:"stack_below,omitempty"`
	Children   []Panel  `json:"children,omitempty"`
	Actor      string   `json:"actor,omitempty"`
	Actors     []string `json:"actors,omitempty"`
	TaskIDs    []string `json:"task_ids,omitempty"`
	Text       []string `json:"text,omitempty"`
	DAG        *DAG     `json:"dag,omitempty"`
}

func ParseLayout(raw []byte) (*Layout, error) {
	if len(raw) > 256<<10 {
		return nil, errors.New("layout exceeds 256 KiB")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var l Layout
	if e := dec.Decode(&l); e != nil {
		return nil, e
	}
	var extra any
	if e := dec.Decode(&extra); e != io.EOF {
		return nil, errors.New("exactly one layout object required")
	}
	if l.Version != 1 {
		return nil, errors.New("layout version must be 1")
	}
	if len(l.Name) > 160 {
		return nil, errors.New("layout name too long")
	}
	l.Name = Clean(l.Name, 160)
	count := 0
	ids := map[string]bool{}
	var check func(*Panel, int) error
	check = func(p *Panel, depth int) error {
		count++
		if count > 64 || depth > 8 {
			return errors.New("layout limited to 64 nodes and depth 8")
		}
		if p.Size < 0 || p.Size > 1000 || p.Weight < 0 || p.Weight > 1000 || p.StackBelow < 0 || p.StackBelow > 1000 {
			return errors.New("layout sizes/weights must be 0..1000")
		}
		if p.ID != "" {
			if len(p.ID) > 80 || Clean(p.ID, 80) != p.ID || ids[p.ID] {
				return errors.New("panel IDs must be unique, printable and at most 80 bytes")
			}
			ids[p.ID] = true
		}
		if _, ok := palette[p.Color]; !ok && p.Color != "" {
			return errors.New("unknown panel color")
		}
		if len(p.Title) > 160 || len(p.Actor) > 120 || len(p.Actors) > 256 || len(p.TaskIDs) > 1024 || len(p.Text) > 128 {
			return errors.New("panel field exceeds limit")
		}
		p.Title = Clean(p.Title, 160)
		p.Actor = Clean(p.Actor, 120)
		for i, v := range p.Actors {
			if len(v) > 120 {
				return errors.New("actor ID too long")
			}
			p.Actors[i] = Clean(v, 120)
		}
		for i, v := range p.TaskIDs {
			if len(v) > 120 {
				return errors.New("task ID too long")
			}
			p.TaskIDs[i] = Clean(v, 120)
		}
		for i, v := range p.Text {
			if len(v) > 2000 {
				return errors.New("text line exceeds 2000 bytes")
			}
			p.Text[i] = Clean(v, 400)
		}
		if p.Kind != "dag" && p.DAG != nil {
			return errors.New("dag data belongs only to dag panels")
		}
		switch p.Kind {
		case "row", "column":
			if len(p.Children) == 0 {
				return errors.New("split requires children")
			}
			if p.Actor != "" || len(p.Actors) > 0 || len(p.TaskIDs) > 0 || len(p.Text) > 0 {
				return errors.New("split cannot bind data")
			}
			for i := range p.Children {
				if e := check(&p.Children[i], depth+1); e != nil {
					return e
				}
			}
		case "session", "agent", "agents", "tasks", "events", "text", "dag":
			if p.ID == "" {
				return errors.New("each leaf panel needs an ID")
			}
			if len(p.Children) != 0 || p.StackBelow != 0 {
				return errors.New("leaf cannot contain children or stack_below")
			}
			if p.Kind == "dag" {
				if e := validateDAG(p.DAG); e != nil {
					return e
				}
			}
			if p.Kind == "agent" && p.Actor == "" {
				return errors.New("agent panel requires actor ID")
			}
			if len(p.Text) > 0 && p.Kind != "text" {
				return errors.New("only text panels accept caller-authored text")
			}
			if p.Actor != "" && p.Kind != "agent" {
				return errors.New("actor belongs to agent panel; use actors for list filters")
			}
			if len(p.Actors) > 0 && p.Kind != "agents" && p.Kind != "events" {
				return errors.New("actors filter belongs to agents/events")
			}
			if len(p.TaskIDs) > 0 && p.Kind != "tasks" {
				return errors.New("task_ids filter belongs to tasks")
			}
		default:
			return fmt.Errorf("unknown panel kind %q", p.Kind)
		}
		return nil
	}
	if e := check(&l.Root, 0); e != nil {
		return nil, e
	}
	return &l, nil
}
func SetLayout(root, session string, raw []byte) error {
	l, e := ParseLayout(raw)
	if e != nil {
		return e
	}
	return Transaction(root, session, false, func(s **State) error {
		(*s).Layout = l
		(*s).event("lead", "layout", "Caller layout: "+l.Name)
		return nil
	})
}

var palette = map[string]string{"peach": "#F5C2A1", "mint": "#A6E3A1", "blue": "#89B4FA", "purple": "#CBA6F7", "red": "#F38BA8", "gray": "#BAC2DE", "teal": "#5EC5CF", "orange": "#F0AA6E", "green": "#7EC98F", "amber": "#E4B469", "white": "#E6EDF3"}

func leaves(p Panel) []Panel {
	if p.Kind != "row" && p.Kind != "column" {
		return []Panel{p}
	}
	out := []Panel{}
	for _, c := range p.Children {
		out = append(out, leaves(c)...)
	}
	return out
}
func contains(filter []string, id string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, v := range filter {
		if v == id {
			return true
		}
	}
	return false
}
func panelData(p Panel, s *State, now time.Time) []string {
	switch p.Kind {
	case "session":
		completion := "not marked finished"
		if s.Finished {
			completion = "explicitly finished"
		}
		age := now.Sub(time.UnixMilli(s.Started)).Round(time.Second)
		if age < 0 {
			age = 0
		}
		a := s.Agents["lead"]
		return []string{Clean(s.Title, 160), "Claude session · " + Clean(s.Session, 256), "Status " + Clean(a.Status, 80) + " | " + completion, "Model " + unknown(a.Model), "Reported plan " + s.Progress("") + " | elapsed " + age.String(), active(s, "lead")}
	case "agent":
		a := s.Agents[p.Actor]
		if a == nil {
			return []string{"Actor " + p.Actor, "Not observed; state/model/progress unknown"}
		}
		return agentData(s, a)
	case "agents":
		out := []string{}
		for _, a := range sortedAgents(s) {
			if contains(p.Actors, a.ID) {
				out = append(out, agentData(s, a)...)
				out = append(out, "")
			}
		}
		if len(out) == 0 {
			return []string{"No matching workers observed since opt-in.", "No parentage or model inferred."}
		}
		return out
	case "tasks":
		out := []string{}
		for _, t := range sortedTasks(s) {
			if !contains(p.TaskIDs, t.ID) {
				continue
			}
			out = append(out, fmt.Sprintf("#%s [%s] %s", Clean(t.ID, 120), Clean(t.Status, 80), Clean(t.Subject, 240)), "owner "+unknown(t.Owner)+" | blocked by "+Clean(strings.Join(t.BlockedBy, ", "), 400))
		}
		if len(out) == 0 {
			return []string{"No reported tasks; progress unknown."}
		}
		return out
	case "events":
		out := []string{}
		for i := len(s.Events) - 1; i >= 0; i-- {
			e := s.Events[i]
			if contains(p.Actors, e.Actor) {
				out = append(out, time.UnixMilli(e.Time).Format("15:04:05")+" · "+Clean(e.Actor, 120)+" · "+Clean(e.Text, 400))
			}
		}
		return out
	case "text":
		return append([]string{"[Caller-authored text · not hook telemetry]"}, p.Text...)
	}
	return nil
}
func agentData(s *State, a *Agent) []string {
	return []string{Clean(a.Role, 100) + " · " + Clean(a.ID, 120) + " · " + Clean(a.Status, 80), "model " + unknown(a.Model) + " | " + s.Progress(a.ID), fmt.Sprintf("tools %d · errors %d | %s", a.Tools, a.Errors, active(s, a.ID))}
}

// Allocations preserve caller order. Fixed sizes are honored while they fit;
// weights divide the remainder. Shrink/stack is geometry, never workflow inference.
func allocate(children []Panel, total int) []int {
	result := make([]int, len(children))
	fixed, weight := 0, 0
	for _, c := range children {
		if c.Size > 0 {
			fixed += c.Size
		} else {
			v := c.Weight
			if v == 0 {
				v = 1
			}
			weight += v
		}
	}
	if fixed > total {
		used := 0
		for i, c := range children {
			result[i] = total * c.Size / fixed
			used += result[i]
		}
		for i := 0; used < total; i = (i + 1) % len(children) {
			if children[i].Size > 0 {
				result[i]++
				used++
			}
		}
		return result
	}
	used := 0
	for i, c := range children {
		if c.Size > 0 {
			result[i] = c.Size
		} else if weight > 0 {
			v := c.Weight
			if v == 0 {
				v = 1
			}
			result[i] = (total - fixed) * v / weight
		}
		used += result[i]
	}
	for i := 0; weight > 0 && used < total; i = (i + 1) % len(children) {
		if children[i].Size == 0 {
			result[i]++
			used++
		}
	}
	return result
}

func panelBorder(focused bool) lipgloss.Border {
	if focused {
		return lipgloss.ThickBorder()
	}
	return lipgloss.RoundedBorder()
}
func panelHeading(title, color string, width int, focused bool) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(focused).Render(clip(Clean(title, 240), width))
}

type canvasOptions struct {
	Focus      string
	Offsets    map[string]int
	XOffsets   map[string]int
	Hidden     *int
	Motion     bool
	Clock      time.Time
	Links      map[string]linkFocus
	Echo       changeEcho
	EchoNow    time.Time
	EchoMotion bool
}

func spaces(w, h int) []string {
	out := make([]string, h)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
	}
	return out
}
func fit(lines []string, w, h int) []string {
	out := spaces(w, h)
	for i := 0; i < len(lines) && i < h; i++ {
		out[i] = clip(lines[i], w)
		out[i] += strings.Repeat(" ", max(0, w-ansi.StringWidth(out[i])))
	}
	return out
}
func renderPanel(p Panel, s *State, w, h int, now time.Time, o canvasOptions) []string {
	if w <= 0 || h <= 0 {
		*o.Hidden += len(leaves(p))
		return nil
	}
	if p.Kind == "row" || p.Kind == "column" {
		horizontal := p.Kind == "row"
		if horizontal && p.StackBelow > 0 && w < p.StackBelow {
			horizontal = false
		}
		total := h
		if horizontal {
			total = w
		}
		sizes := allocate(p.Children, total)
		out := spaces(w, h)
		pos := 0
		for i, c := range p.Children {
			cw, ch := w, sizes[i]
			if horizontal {
				cw, ch = sizes[i], h
			}
			piece := renderPanel(c, s, cw, ch, now, o)
			if horizontal {
				for row := 0; row < h; row++ {
					segment := strings.Repeat(" ", cw)
					if row < len(piece) {
						segment = piece[row]
					}
					if pos == 0 {
						out[row] = segment
					} else {
						out[row] += segment
					}
				}
			} else {
				for row := 0; row < len(piece) && pos+row < h; row++ {
					out[pos+row] = piece[row]
				}
			}
			pos += sizes[i]
		}
		return fit(out, w, h)
	}
	if w < 8 || h < 4 {
		*o.Hidden++
		return fit([]string{"…"}, w, h)
	}
	color := palette[p.Color]
	if color == "" {
		color = palette["blue"]
	}
	focused := o.Focus == p.ID
	if focused {
		color = palette["purple"]
	}
	title := p.Title
	if title == "" {
		title = p.ID
	}
	if focused {
		title = "▶ " + title
	}
	title += " · " + p.Kind
	if p.Kind == "dag" {
		return renderDAGPanel(p, s, w, h, o, title, color)
	}
	data := panelData(p, s, now)
	keys := panelEchoKeys(p, s)
	offset := o.Offsets[p.ID]
	capacity := h - 3
	if offset > max(0, len(data)-1) {
		offset = max(0, len(data)-1)
	}
	if offset < 0 {
		offset = 0
	}
	if len(data) > capacity {
		title += fmt.Sprintf(" [%d/%d]", offset+1, len(data))
	}
	body := []string{panelHeading(title, color, w-2, focused)}
	for i := offset; i < len(data) && len(body) < h-2; i++ {
		row := Clean(data[i], 2000)
		if p.Kind == "session" || p.Kind == "agent" || p.Kind == "agents" || p.Kind == "tasks" {
			var rowKeys []echoKey
			if i < len(keys) {
				rowKeys = keys[i]
			}
			row = echoLine(row, rowKeys, o.Echo, o.EchoNow, o.EchoMotion, w-2)
		}
		body = append(body, row)
	}
	for len(body) < h-2 {
		body = append(body, "")
	}
	for i := range body {
		body[i] = clip(body[i], w-2)
	}
	styled := lipgloss.NewStyle().Border(panelBorder(focused)).BorderForeground(lipgloss.Color(color)).Width(w - 2).Render(strings.Join(body, "\n"))
	return fit(strings.Split(styled, "\n"), w, h)
}

// panelDimensions uses the same allocations as rendering, so DAG navigation
// clamps at the visible edge instead of accumulating invisible overscroll.
func panelDimensions(p Panel, id string, w, h int) (int, int, bool) {
	if p.Kind != "row" && p.Kind != "column" {
		return w, h, p.ID == id
	}
	horizontal := p.Kind == "row" && !(p.StackBelow > 0 && w < p.StackBelow)
	total := h
	if horizontal {
		total = w
	}
	sizes := allocate(p.Children, total)
	for i, child := range p.Children {
		cw, ch := w, sizes[i]
		if horizontal {
			cw, ch = sizes[i], h
		}
		if fw, fh, ok := panelDimensions(child, id, cw, ch); ok {
			return fw, fh, true
		}
	}
	return 0, 0, false
}
