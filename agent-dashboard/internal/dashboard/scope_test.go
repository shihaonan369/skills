package dashboard_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	d "agent-dashboard/internal/dashboard"
)

func scopeLayout(path string) string {
	encoded, _ := json.Marshal(path)
	q := `{"kind":"actors","scope_file":` + string(encoded) + `}`
	return `{"version":1,"root":{"kind":"column","children":[{"id":"graph","kind":"dag","size":18,"dag":{"version":1,"nodes":[],"select":` + q + `}},{"id":"cards","kind":"agent","size":18,"bind":{"item":true},"repeat":{"select":` + q + `}},{"id":"rows","kind":"agents","select":` + q + `}]}}`
}
func prepareScope(t *testing.T, s *d.State) *d.State {
	t.Helper()
	next, err := d.PreparePresentation(s)
	if err != nil {
		t.Fatal(err)
	}
	return next
}
func scopeAnswer(batch d.ScopeBatch, decision string) []byte {
	ds := d.ScopeDecisions{PolicyHash: batch.PolicyHash}
	for _, r := range batch.Requests {
		ds.Decisions = append(ds.Decisions, d.ScopeDecision{Key: r.Key, Fingerprint: r.Fingerprint, Decision: decision, Origin: "subagent", Evidence: "Synthetic classifier contract; not a real LLM judgement"})
	}
	raw, _ := json.Marshal(ds)
	return raw
}

func TestAgentOwnedScopeMailboxDrivesEveryProjectionWithoutAnLLMClient(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private scope.json")
	if err := d.InitScope(path, "Only implementation agents; exclude reviewers"); err != nil {
		t.Fatal(err)
	}
	s, root := dynamicNative(t, scopeLayout(path))
	s = spawnDynamic(t, root, "worker-one", "general-purpose")
	if err := d.Ingest(root, d.Payload{Session: "dynamic-session", Kind: "PreToolUse", AgentID: "worker-one", ToolName: "Bash", ToolID: "t", Input: map[string]json.RawMessage{"command": json.RawMessage(`"SECRET TOOL INPUT"`)}}); err != nil {
		t.Fatal(err)
	}
	s = prepareScope(t, nativeSnapshot(t, root))
	f := dynamicFrame(s)
	if strings.Contains(f, "general-purpose · worker-one") || !strings.Contains(f, "1 pending") {
		t.Fatal("unclassified worker was admitted", f)
	}
	batch, err := d.PollScope(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Requests) != 1 || batch.Requests[0].Bind.Actor != "worker-one" || batch.Requests[0].Role != "general-purpose" {
		t.Fatal(batch)
	}
	raw, _ := json.Marshal(batch)
	if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "tool_input") {
		t.Fatal("scope request leaked tool inputs")
	}
	response := scopeAnswer(batch, "include")
	if err := d.DecideScope(path, response); err != nil {
		t.Fatal(err)
	}
	s = prepareScope(t, nativeSnapshot(t, root))
	f = dynamicFrame(s)
	if strings.Count(f, "worker-one") < 3 {
		t.Fatal("one agent decision did not reach DAG/card/list", f)
	}
	if !strings.Contains(f, "Agent-classified scope") {
		t.Fatal("classification provenance missing", f)
	}
	if batch, err = d.PollScope(path, 0); err != nil || len(batch.Requests) != 0 {
		t.Fatal("unchanged observations requeued", err, batch)
	}
	// Render is pure: it neither polls/writes the mailbox nor dispatches an agent.
	before, _ := os.ReadFile(path)
	_ = dynamicFrame(s)
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("render performed classifier IO")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Fatal("scope metadata not private")
		}
	}
}

func TestScopeInvalidatesChangedIdentityAndRejectsLateDecisionsAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	if err := d.InitScope(path, "Implementation only"); err != nil {
		t.Fatal(err)
	}
	_, root := dynamicNative(t, scopeLayout(path))
	s := spawnDynamic(t, root, "worker", "implement")
	prepareScope(t, s)
	original, _ := d.PollScope(path, 0)
	s = spawnDynamic(t, root, "worker", "review")
	s = prepareScope(t, s)
	before, _ := os.ReadFile(path)
	if err := d.DecideScope(path, scopeAnswer(original, "include")); err == nil {
		t.Fatal("accepted stale semantic decision")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("late batch partially committed")
	}
	latest, _ := d.PollScope(path, 0)
	if len(latest.Requests) != 1 || latest.Requests[0].Fingerprint == original.Requests[0].Fingerprint {
		t.Fatal("identity metadata did not invalidate cached verdict")
	}
	if err := d.DecideScope(path, scopeAnswer(latest, "unknown")); err != nil {
		t.Fatal(err)
	}
	s = prepareScope(t, s)
	if f := dynamicFrame(s); strings.Contains(f, "review · worker") || !strings.Contains(f, "1 unknown") {
		t.Fatal("unknown was treated as include", f)
	}
	if b, _ := d.PollScope(path, 0); len(b.Requests) != 0 {
		t.Fatal("unknown response caused an idle classification loop")
	}
	if err := d.Ingest(root, d.Payload{Session: "dynamic-session", Kind: "PreToolUse", AgentID: "worker", ToolID: "one", ToolName: "Bash"}); err != nil {
		t.Fatal(err)
	}
	prepareScope(t, nativeSnapshot(t, root))
	if b, _ := d.PollScope(path, 0); len(b.Requests) != 0 {
		t.Fatal("activity/status-only change requeued identity")
	}
	if err := d.InitScope(path, "A different criterion"); err == nil {
		t.Fatal("scope criterion silently replaced")
	}
}

func TestScopeLongPollConcurrentRepliesAndStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	if err := d.InitScope(path, "Caller criterion"); err != nil {
		t.Fatal(err)
	}
	_, root := dynamicNative(t, scopeLayout(path))
	spawnDynamic(t, root, "worker-a", "general-purpose")
	s := spawnDynamic(t, root, "worker-b", "general-purpose")
	done := make(chan error, 1)
	go func() { time.Sleep(20 * time.Millisecond); _, err := d.PreparePresentation(s); done <- err }()
	batch, err := d.PollScope(path, time.Second)
	if err != nil || len(batch.Requests) != 2 {
		t.Fatal(err, batch)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := range batch.Requests {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := batch
			b.Requests = []d.ScopeRequest{batch.Requests[i]}
			errs <- d.DecideScope(path, scopeAnswer(b, []string{"include", "exclude"}[i]))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if b, _ := d.PollScope(path, 0); len(b.Requests) != 0 {
		t.Fatal("concurrent classifier replies lost an update")
	}
	if err := d.CloseScope(path); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	b, err := d.PollScope(path, time.Second)
	if err != nil || !b.Closed || time.Since(start) > 500*time.Millisecond {
		t.Fatal("closed worker was not released", err, b)
	}
	if err := d.DecideScope(path, scopeAnswer(batch, "include")); err == nil {
		t.Fatal("closed scope accepted a decision")
	}
}

func TestScopeFileIntegrityAndMissingWorkerFailClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	s, root := dynamicNative(t, scopeLayout(path))
	if _, err := d.PreparePresentation(s); err == nil {
		t.Fatal("missing scope silently widened membership")
	}
	if err := d.InitScope(path, "Implementation"); err != nil {
		t.Fatal(err)
	}
	prepareScope(t, spawnDynamic(t, root, "unknown", "general-purpose"))
	batch, _ := d.PollScope(path, 0)
	for _, raw := range [][]byte{[]byte(`{}`), []byte(`null`), []byte(`{"unexpected":true}`), scopeAnswer(batch, "invented")} {
		before, _ := os.ReadFile(path)
		if err := d.DecideScope(path, raw); err == nil {
			t.Fatal("malformed decision accepted", string(raw))
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			t.Fatal("bad reply modified mailbox")
		}
	}
	if err := d.InitScope("relative.json", "x"); err == nil {
		t.Fatal("relative mailbox accepted")
	}
	link := filepath.Join(filepath.Dir(path), "link.json")
	if err := os.Symlink(path, link); err == nil {
		if _, err := d.PollScope(link, 0); err == nil {
			t.Fatal("scope symlink followed")
		}
	}
}

func TestSemanticFilterRunsBeforeDisplayLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope.json")
	if err := d.InitScope(path, "Only the designated implementation fixture"); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(path)
	_, root := dynamicNative(t, `{"version":1,"root":{"id":"selected","kind":"agents","select":{"kind":"actors","limit":1,"scope_file":`+string(encoded)+`}}}`)
	for i := 0; i < 70; i++ {
		spawnDynamic(t, root, fmt.Sprintf("a-review-%02d", i), "general-purpose")
	}
	s := spawnDynamic(t, root, "z-implementation", "general-purpose")
	prepareScope(t, s)
	processed := 0
	for batchIndex := 0; batchIndex < 3; batchIndex++ {
		batch, err := d.PollScope(path, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(batch.Requests) == 0 {
			break
		}
		ds := d.ScopeDecisions{PolicyHash: batch.PolicyHash}
		for _, r := range batch.Requests {
			decision := "exclude"
			if r.Bind.Actor == "z-implementation" {
				decision = "include"
			}
			ds.Decisions = append(ds.Decisions, d.ScopeDecision{Key: r.Key, Fingerprint: r.Fingerprint, Decision: decision, Origin: "subagent"})
		}
		processed += len(batch.Requests)
		raw, _ := json.Marshal(ds)
		if err := d.DecideScope(path, raw); err != nil {
			t.Fatal(err)
		}
	}
	if processed != 71 {
		t.Fatal("display limit truncated the classification candidate scope", processed)
	}
	frame := dynamicFrame(prepareScope(t, s))
	if !strings.Contains(frame, "z-implementation") || strings.Contains(frame, "a-review-") {
		t.Fatal(frame)
	}
}
