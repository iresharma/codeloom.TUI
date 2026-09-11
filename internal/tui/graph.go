package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) mergeAgents(live []agentRow) {
	seen := map[string]struct{}{}
	for _, a := range live {
		seen[a.id] = struct{}{}
		found := false
		for i := range m.agents {
			if m.agents[i].id == a.id {
				m.agents[i] = a
				found = true
				break
			}
		}
		if !found {
			m.agents = append(m.agents, a)
		}
	}
	for i := range m.agents {
		if _, ok := seen[m.agents[i].id]; !ok {
			m.agents[i].status = "finished"
			m.agents[i].tool = ""
		}
	}
}

func (m *Model) selectedAgent() string {
	if m.agentCursor >= 0 && m.agentCursor < len(m.agents) {
		return m.agents[m.agentCursor].id
	}
	if len(m.agents) == 0 {
		return ""
	}
	return m.agents[0].id
}

func (m Model) viewGraph(w, h int) string {
	var b strings.Builder
	b.WriteString(headerBar("agents", w) + "\n")
	st := m.orchState
	if st == "" {
		st = "idle"
	}
	orch := fmt.Sprintf(" ●  orchestrator  %s", st)
	b.WriteString(zones.Mark("agent-", orch) + "\n")
	for i, a := range m.agents {
		status := a.status
		if status == "" {
			status = "idle"
		}
		label := fmt.Sprintf("%s  %s", a.profile, status)
		if a.tool != "" {
			label += " · " + a.tool
		}
		prefix := "    ├ "
		style := lipgloss.NewStyle().Foreground(fg)
		if status == "finished" {
			style = dimStyle()
		}
		if i == m.agentCursor {
			prefix = "  ▸ ├ "
			style = style.Foreground(accent).Bold(true)
		}
		line := style.Render(prefix + label)
		b.WriteString(zones.Mark("agent-"+a.id, line) + "\n")
		if a.task != "" && i == m.agentCursor {
			task := a.task
			if len(task) > w-8 {
				task = task[:max(0, w-11)] + "…"
			}
			b.WriteString(dimStyle().Render("      "+task) + "\n")
		}
	}
	if len(m.agents) == 0 {
		b.WriteString(dimStyle().Render("    no children") + "\n")
	}
	return b.String()
}
