package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

func (m *Model) openFileTab(path, content string) {
	for i, t := range m.tabs {
		if t.kind == tabFile && t.path == path {
			m.tabs[i].body = content
			m.active = i
			m.refreshFileVP()
			return
		}
	}
	m.tabs = append(m.tabs, tab{kind: tabFile, title: path, path: path, body: content})
	m.active = len(m.tabs) - 1
	m.refreshFileVP()
}

func (m *Model) openAgent(id string) tea.Cmd {
	short := id
	if len(short) > 8 {
		short = short[:8]
	}
	profile := "agent"
	for _, a := range m.agents {
		if a.id == id {
			profile = a.profile
		}
	}
	title := profile + " · " + short
	for i, t := range m.tabs {
		if t.kind == tabAgent && t.agentID == id {
			m.active = i
			return m.send(protocol.RequestAgentTranscript(id))
		}
	}
	m.tabs = append(m.tabs, tab{kind: tabAgent, title: title, agentID: id})
	m.active = len(m.tabs) - 1
	return m.send(protocol.RequestAgentTranscript(id))
}

func (m *Model) closeTab() tea.Cmd {
	if m.active <= 0 || m.active >= len(m.tabs) {
		return nil
	}
	t := m.tabs[m.active]
	var cmd tea.Cmd
	if t.kind == tabFile && t.path != "" {
		cmd = m.send(protocol.CloseFile(t.path))
	}
	m.tabs = append(m.tabs[:m.active], m.tabs[m.active+1:]...)
	if m.active >= len(m.tabs) {
		m.active = len(m.tabs) - 1
	}
	m.refreshFileVP()
	return cmd
}

func (m *Model) currentTab() *tab {
	if m.active < 0 || m.active >= len(m.tabs) {
		return nil
	}
	return &m.tabs[m.active]
}

func (m Model) viewCenter(w, h int) string {
	inputH := 3
	headH := 1
	bodyH := h - inputH - headH
	if bodyH < 1 {
		bodyH = 1
	}
	head := m.viewTabBar(w)
	body := m.viewCenterBody(w, bodyH)
	in := m.viewComposer(w, inputH)
	return lipgloss.JoinVertical(lipgloss.Top, head, body, in)
}

func (m Model) viewTabBar(w int) string {
	active := lipgloss.NewStyle().Foreground(bg).Background(accent).Bold(true).Padding(0, 1)
	idle := lipgloss.NewStyle().Foreground(dim).Padding(0, 1)
	var parts []string
	for i, t := range m.tabs {
		label := t.title
		if len(label) > 24 {
			label = "…" + label[len(label)-23:]
		}
		st := idle
		if i == m.active {
			st = active
		}
		parts = append(parts, zones.Mark("tab-"+fmt.Sprint(i), st.Render(label)))
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return lipgloss.NewStyle().Background(bgAlt).Width(max(1, w)).Render(bar)
}

func (m Model) viewCenterBody(w, h int) string {
	t := m.currentTab()
	switch {
	case t == nil || t.kind == tabChat:
		if len(m.chat) == 0 {
			msg := lipgloss.JoinVertical(lipgloss.Center,
				titleStyle().Render("codeloom"),
				"",
				dimStyle().Render("orchestrator chat"),
				dimStyle().Render("type below · enter to send"),
			)
			return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
		}
		return m.chatVP.View()
	case t.kind == tabFile:
		hint := ""
		if t.showDiff && t.diff != "" {
			hint = dimStyle().Render(" diff overlay  ctrl+d to restore file") + "\n"
		}
		return fit(hint+m.fileVP.View(), w, h)
	case t.kind == tabAgent:
		return fit(m.viewAgentChat(t.agentID), w, h)
	}
	return fit("", w, h)
}

func (m Model) viewComposer(w, h int) string {
	return sectionBorder().
		Width(max(1, w)).
		Height(max(1, h)).
		Padding(0, 1).
		Render(m.input.View())
}
