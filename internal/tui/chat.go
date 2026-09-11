package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

func (m *Model) refreshChat() {
	m.chatVP.SetContent(renderLines(m.chat, len(m.streams) > 0))
	m.chatVP.GotoBottom()
}

func (m *Model) viewAgentChat(id string) string {
	lines := m.agentChat[id]
	if len(lines) == 0 {
		return dimStyle().Render("(no transcript yet — enter on an agent to replay)")
	}
	return renderLines(lines, false)
}

func (m *Model) appendAgent(id, role, text string, reset bool) {
	if id == "" {
		return
	}
	if m.agentChat == nil {
		m.agentChat = map[string][]chatLine{}
	}
	if reset {
		m.agentChat[id] = nil
	}
	m.agentChat[id] = append(m.agentChat[id], chatLine{role: role, text: text, agentID: id})
}

func (m *Model) appendAgentDelta(id, text string) {
	lines := m.agentChat[id]
	if len(lines) == 0 {
		m.appendAgent(id, "assistant", text, false)
		return
	}
	lines[len(lines)-1].text += text
	m.agentChat[id] = lines
}

func renderLines(lines []chatLine, streaming bool) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(titleStyle().Render(line.role))
		b.WriteString("\n")
		if !streaming && (line.role == "assistant" || line.role == "engine") {
			if rendered, err := glamour.Render(line.text, "dark"); err == nil {
				b.WriteString(rendered)
			} else {
				b.WriteString(line.text)
			}
		} else {
			b.WriteString(line.text)
		}
		b.WriteString("\n")
	}
	return b.String()
}
