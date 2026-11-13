package dashboard

// SeedDemo is a labelled layout fixture, isolated from hook ingestion. It does
// not spawn agents and must never be presented as a real Claude issue run.
func SeedDemo(root, session string) error {
	return Transaction(root, session, true, func(sp **State) error {
		if *sp != nil && !(*sp).Demo {
			return ErrInactive
		}
		s := newState(session, "Synthetic layout preview", true)
		*sp = s
		s.Layout = &Layout{Version: 1, Name: "Synthetic panel composition", Root: Panel{Kind: "column", Children: []Panel{
			{ID: "session", Kind: "session", Title: "Session", Color: "peach", Size: 7},
			{Kind: "row", StackBelow: 70, Children: []Panel{{ID: "workers", Kind: "agents", Title: "Observed workers", Color: "blue"}, {ID: "tasks", Kind: "tasks", Title: "Reported tasks", Color: "mint"}}},
			{ID: "events", Kind: "events", Title: "Event buffer", Size: 6},
		}}}
		s.Agents["lead"].Model = "demo-model"
		for i, role := range []string{"sample-A", "sample-B", "sample-C", "sample-D"} {
			id := []string{"demo-scout", "demo-builder", "demo-tests", "demo-D"}[i]
			s.Agents[id] = &Agent{ID: id, Role: role, Status: "running", Model: "demo-model", Started: s.Started + int64(i), Tools: i + 2}
		}
		s.Tasks["1"] = Task{ID: "1", Subject: "Synthetic reported item A", Status: "completed", Owner: "demo-scout"}
		s.Tasks["2"] = Task{ID: "2", Subject: "Synthetic reported item B", Status: "in_progress", Owner: "demo-builder", BlockedBy: []string{"1"}}
		s.Tasks["3"] = Task{ID: "3", Subject: "Synthetic reported item C", Status: "pending", Owner: "demo-tests", BlockedBy: []string{"2"}}
		s.event("lead", "demo", "Synthetic fixture only; no real agents or tool calls")
		s.event("demo-D", "demo", "Synthetic caller note")
		return nil
	})
}
