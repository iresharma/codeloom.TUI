package tui

import "testing"

func TestMergeAgentsKeepsFinished(t *testing.T) {
	m := Model{}
	m.mergeAgents([]agentRow{{id: "a1", profile: "coder", status: "thinking"}})
	m.mergeAgents([]agentRow{{id: "a2", profile: "explore", status: "running"}})
	if len(m.agents) != 2 {
		t.Fatalf("got %d agents", len(m.agents))
	}
	if m.agents[0].status != "finished" {
		t.Fatalf("first should stay finished, got %s", m.agents[0].status)
	}
	m.mergeAgents(nil)
	if m.agents[1].status != "finished" {
		t.Fatalf("second should finish, got %s", m.agents[1].status)
	}
}
