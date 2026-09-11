package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewContext(w, h int) string {
	ctxTab, memTab := "context", "memory"
	if m.rightSub == 0 {
		ctxTab = " CONTEXT "
		memTab = " memory "
	} else {
		ctxTab = " context "
		memTab = " MEMORY "
	}
	on := lipgloss.NewStyle().Foreground(bg).Background(accent).Bold(true)
	off := lipgloss.NewStyle().Foreground(dim)
	if m.rightSub == 0 {
		ctxTab = on.Render(ctxTab)
		memTab = off.Render(memTab)
	} else {
		ctxTab = off.Render(ctxTab)
		memTab = on.Render(memTab)
	}
	head := lipgloss.JoinHorizontal(lipgloss.Top, ctxTab, " ", memTab)
	var b strings.Builder
	b.WriteString(head + "\n")
	if m.rightSub == 1 {
		b.WriteString(m.viewMemory())
		return b.String()
	}
	pct := 0
	if m.ctxBudget > 0 {
		pct = m.ctxUsed * 100 / m.ctxBudget
	}
	barW := min(18, max(8, w-14))
	filled := 0
	if m.ctxBudget > 0 {
		filled = pct * barW / 100
	}
	if filled > barW {
		filled = barW
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	b.WriteString(fmt.Sprintf(" %s %d%%\n", bar, pct))
	b.WriteString(dimStyle().Render(fmt.Sprintf(" %d est · %d measured · %d budget\n", m.ctxUsed, m.ctxPrompt, m.ctxBudget)))
	remain := h - 4
	for i, s := range m.ctxSections {
		mark := "  "
		if i == m.ctxOpen {
			mark = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s  %d\n", mark, s.name, s.tokens))
		remain--
		if i == m.ctxOpen && s.text != "" && remain > 2 {
			clip := strings.ReplaceAll(s.text, "\n", " ")
			maxChars := min(remain*w, 180)
			if len(clip) > maxChars {
				clip = clip[:maxChars] + "…"
			}
			wrapped := dimStyle().Width(max(8, w-2)).Render("  " + clip)
			b.WriteString(wrapped + "\n")
		}
	}
	return b.String()
}

func (m *Model) contextAgent() string {
	t := m.currentTab()
	if t != nil && t.kind == tabAgent {
		return t.agentID
	}
	return ""
}
