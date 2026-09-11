package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

func (m *Model) refreshChat() {
	lines := m.chat
	streaming := len(m.streams) > 0
	if m.chatFilter != "" {
		lines = m.agentChat[m.chatFilter]
		streaming = false
		for _, aid := range m.streamAgent {
			if aid == m.chatFilter {
				streaming = true
				break
			}
		}
	}
	w := m.chatVP.Width
	if w < 8 {
		w = 8
	}
	m.chatVP.SetContent(renderLines(lines, streaming, w))
	if m.chatFollow {
		m.chatVP.GotoBottom()
	}
}

func (m Model) viewChatStrip(w, h int) string {
	compH := m.composerHeight()
	head := m.viewChatFilter(w)
	card := ""
	cardH := 0
	if m.promptID != "" {
		card = m.viewPromptCard(w)
		cardH = lipgloss.Height(card)
	}
	bodyH := max(1, h-compH-cardH-lipgloss.Height(head))
	body := m.chatVP.View()
	if len(m.chat) == 0 && m.chatFilter == "" && m.promptID == "" {
		body = lipgloss.Place(w, bodyH, lipgloss.Center, lipgloss.Center,
			dimStyle().Render("orchestrator · type a message · enter to send"))
	}
	parts := []string{head, body}
	if card != "" {
		parts = append(parts, card)
	}
	parts = append(parts, m.viewComposer(w, compH))
	return lipgloss.JoinVertical(lipgloss.Top, parts...)
}

func (m Model) viewChatFilter(w int) string {
	label := "orch"
	if m.chatFilter != "" {
		short := m.chatFilter
		if len(short) > 8 {
			short = short[:8]
		}
		profile := "agent"
		for _, a := range m.agents {
			if a.id == m.chatFilter {
				profile = a.profile
			}
		}
		label = profile + " · " + short
	}
	if m.busy() {
		label = m.spin.View() + " " + label
	}
	return lipgloss.NewStyle().Foreground(magenta).Background(bgAlt).Width(max(1, w)).Render(" " + label)
}

func (m Model) composerHeight() int {
	h := strings.Count(m.input.Value(), "\n") + 1
	if h < 1 {
		h = 1
	}
	if h > 5 {
		h = 5
	}
	return h + 1 // border
}

func (m *Model) submitComposer() tea.Cmd {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return nil
	}
	m.input.Reset()
	m.chatFollow = true
	if m.promptID != "" {
		return m.answerPrompt(text)
	}
	return m.send(protocol.SubmitUserMessage(text))
}

func (m Model) promptCardHeight(w int) int {
	if m.promptID == "" {
		return 0
	}
	return lipgloss.Height(m.viewPromptCard(w))
}

func (m Model) viewPromptCard(w int) string {
	inner := max(12, w-4)
	kind := m.promptKind
	if kind == "" {
		kind = "prompt"
	}
	var b strings.Builder
	b.WriteString(titleStyle().Render(kind) + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(fg).Width(inner).Render(m.promptQ))
	b.WriteString("\n")
	if len(m.promptChoices) > 0 {
		b.WriteString("\n")
		for i, c := range m.promptChoices {
			label := fmt.Sprintf("%d  %s", i+1, c)
			mark := "  "
			if i == m.promptCursor {
				mark = "❯ "
				label = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(label)
			} else {
				label = dimStyle().Render(label)
			}
			b.WriteString(zones.Mark("prompt-"+fmt.Sprint(i), mark+label) + "\n")
		}
	} else {
		b.WriteString("\n" + dimStyle().Render("type an answer below") + "\n")
	}
	b.WriteString(dimStyle().Render("enter confirm · esc skip"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Width(max(1, w-2)).
		Padding(0, 1).
		Render(strings.TrimRight(b.String(), "\n"))
}

func (m *Model) movePromptChoice(delta int) {
	n := len(m.promptChoices)
	if n == 0 {
		return
	}
	m.promptCursor += delta
	if m.promptCursor < 0 {
		m.promptCursor = n - 1
	}
	if m.promptCursor >= n {
		m.promptCursor = 0
	}
}

func (m Model) selectedPromptChoice() string {
	if m.promptCursor >= 0 && m.promptCursor < len(m.promptChoices) {
		return m.promptChoices[m.promptCursor]
	}
	return ""
}

func (m Model) promptDecline() string {
	for _, want := range []string{"deny", "no", "cancel", "reject", "skip", "never"} {
		for _, c := range m.promptChoices {
			if strings.EqualFold(c, want) {
				return c
			}
		}
	}
	if n := len(m.promptChoices); n > 0 {
		return m.promptChoices[n-1]
	}
	return ""
}

func (m *Model) answerPrompt(text string) tea.Cmd {
	if m.promptID == "" {
		return nil
	}
	id := m.promptID
	q := m.promptQ
	m.promptID = ""
	m.promptQ = ""
	m.promptKind = ""
	m.promptChoices = nil
	m.promptCursor = 0
	if q != "" {
		m.chat = append(m.chat, chatLine{role: "prompt", text: q})
	}
	m.chat = append(m.chat, chatLine{role: "user", text: text})
	m.chatFollow = true
	m.refreshChat()
	m.layoutViewports()
	return m.send(protocol.AnswerPrompt(id, text))
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
	if m.chatFilter == id {
		m.refreshChat()
	}
}

func (m *Model) appendAgentDelta(id, text string) {
	lines := m.agentChat[id]
	if len(lines) == 0 {
		m.appendAgent(id, "assistant", text, false)
		return
	}
	lines[len(lines)-1].text += text
	m.agentChat[id] = lines
	if m.chatFilter == id {
		m.refreshChat()
	}
}

func renderLines(lines []chatLine, streaming bool, w int) string {
	if w < 8 {
		w = 8
	}
	var b strings.Builder
	for _, line := range lines {
		role := line.role
		if role == "prompt" {
			b.WriteString(titleStyle().Render("prompt"))
		} else {
			b.WriteString(titleStyle().Render(role))
		}
		b.WriteString("\n")
		switch {
		case role == "prompt":
			b.WriteString(lipgloss.NewStyle().Foreground(magenta).Width(w).Render(line.text))
		case !streaming && (line.role == "assistant" || line.role == "engine"):
			b.WriteString(renderMarkdown(line.text, w))
		default:
			b.WriteString(wrapPlain(line.text, w))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderMarkdown(text string, w int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(w),
	)
	if err != nil {
		return wrapPlain(text, w)
	}
	out, err := r.Render(text)
	if err != nil {
		return wrapPlain(text, w)
	}
	return strings.TrimRight(out, "\n")
}

func wrapPlain(text string, w int) string {
	if w < 8 {
		w = 8
	}
	return lipgloss.NewStyle().Width(w).Render(text)
}
