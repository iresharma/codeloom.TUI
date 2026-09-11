package tui

import (
	"strings"
	"testing"
)

func TestMergeAgentsDropsFinished(t *testing.T) {
	m := Model{}
	m.mergeAgents([]agentRow{{id: "a1", profile: "coder", status: "thinking"}})
	m.mergeAgents([]agentRow{{id: "a2", profile: "explore", status: "running"}})
	if len(m.agents) != 1 || m.agents[0].id != "a2" {
		t.Fatalf("finished agent should leave the graph, got %#v", m.agents)
	}
	m.mergeAgents(nil)
	if len(m.agents) != 0 {
		t.Fatalf("empty snapshot should clear graph, got %#v", m.agents)
	}
}

func TestMergeAgentsDropsDoneStatus(t *testing.T) {
	m := Model{}
	m.mergeAgents([]agentRow{
		{id: "a1", profile: "coder", status: "running"},
		{id: "a2", profile: "explore", status: "finished"},
		{id: "a3", profile: "reviewer", status: "completed"},
	})
	if len(m.agents) != 1 || m.agents[0].id != "a1" {
		t.Fatalf("got %#v", m.agents)
	}
}

func TestDagRanksFromParent(t *testing.T) {
	agents := []agentRow{
		{id: "c1", profile: "coder", parent: ""},
		{id: "c2", profile: "explore", parent: ""},
		{id: "c3", profile: "reviewer", parent: "c1"},
	}
	ranks := rankAgents(agents)
	if len(ranks) != 2 {
		t.Fatalf("ranks %d want 2: %#v", len(ranks), ranks)
	}
	if len(ranks[0]) != 2 {
		t.Fatalf("rank0 %d want 2", len(ranks[0]))
	}
	if len(ranks[1]) != 1 || ranks[1][0].id != "c3" {
		t.Fatalf("rank1 %#v", ranks[1])
	}
	order := nodeOrder(agents)
	if len(order) < 4 || order[0] != "" {
		t.Fatalf("order %#v", order)
	}
	got := map[string]int{}
	for i, id := range order {
		got[id] = i
	}
	if got["c3"] <= got["c1"] {
		t.Fatalf("child should follow parent: %#v", order)
	}
}

func TestRenderDAGLooksLikeGraph(t *testing.T) {
	agents := []agentRow{
		{id: "c1", profile: "coder", status: "running", tool: "Read", parent: ""},
		{id: "c2", profile: "explore", status: "idle", parent: ""},
	}
	out := renderDAG(40, "thinking", agents, 0, "")
	if !strings.Contains(out, "orchestrator") {
		t.Fatalf("missing orchestrator:\n%s", out)
	}
	if !strings.Contains(out, "coder") || !strings.Contains(out, "explore") {
		t.Fatalf("missing children:\n%s", out)
	}
	if !strings.Contains(out, "╭") || !strings.Contains(out, "▼") {
		t.Fatalf("missing box/edge drawing:\n%s", out)
	}
}

func TestRenderDAGManySiblings(t *testing.T) {
	agents := make([]agentRow, 8)
	for i := range agents {
		agents[i] = agentRow{id: string(rune('a' + i)), profile: "agent" + string(rune('1'+i)), status: "running"}
	}
	out := renderDAG(34, "thinking", agents, 0, "")
	for _, a := range agents {
		if !strings.Contains(out, a.profile) {
			t.Fatalf("missing %s in:\n%s", a.profile, out)
		}
	}
	if strings.Contains(out, "finished") {
		t.Fatalf("done agents should not render:\n%s", out)
	}
}

func TestRenderDAGHidesFinished(t *testing.T) {
	out := renderDAG(40, "idle", []agentRow{
		{id: "a1", profile: "coder", status: "running"},
		{id: "a2", profile: "explore", status: "finished"},
	}, 0, "")
	if !strings.Contains(out, "coder") {
		t.Fatalf("live agent missing:\n%s", out)
	}
	if strings.Contains(out, "explore") {
		t.Fatalf("finished agent still drawn:\n%s", out)
	}
}
