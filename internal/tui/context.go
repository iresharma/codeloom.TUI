package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewContext(w, h int) string {
	return m.viewInspectPane(w, h)
}

func (m *Model) refreshInspect() {
	m.inspectVP.SetContent(m.viewInspectBody(max(8, m.inspectVP.Width)))
}

func (m Model) viewInspectPane(w, h int) string {
	head := m.viewInspectTabs(w)
	return lipgloss.JoinVertical(lipgloss.Top, head, m.inspectVP.View())
}

func (m Model) viewInspect(w int) string {
	return lipgloss.JoinVertical(lipgloss.Top, m.viewInspectTabs(w), m.viewInspectBody(w))
}

func (m Model) viewInspectTabs(w int) string {
	on := lipgloss.NewStyle().Foreground(bg).Background(accent).Bold(true)
	off := lipgloss.NewStyle().Foreground(dim)
	var ctxTab, memTab string
	if m.rightSub == 0 {
		ctxTab = on.Render(" CONTEXT ")
		memTab = off.Render(" memory ")
	} else {
		ctxTab = off.Render(" context ")
		memTab = on.Render(" MEMORY ")
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		zones.Mark("insp-ctx", ctxTab), " ", zones.Mark("insp-mem", memTab),
	)
	return lipgloss.NewStyle().Background(bgAlt).Width(max(1, w)).Render(bar)
}

func (m Model) viewInspectBody(w int) string {
	if m.rightSub == 1 {
		return m.viewMemory()
	}
	return m.viewContextBreakdown(w)
}

func (m Model) viewContextBreakdown(w int) string {
	var b strings.Builder
	pct := 0
	if m.ctxBudget > 0 {
		pct = m.ctxUsed * 100 / m.ctxBudget
		if pct > 100 {
			pct = 100
		}
	}
	barW := min(16, max(8, w-12))
	prog := m.ctxProg
	prog.Width = barW
	fill := 0.0
	if m.ctxBudget > 0 {
		fill = float64(m.ctxUsed) / float64(m.ctxBudget)
		if fill > 1 {
			fill = 1
		}
	}
	b.WriteString(" " + prog.ViewAs(fill) + fmt.Sprintf(" %d%%\n", pct))
	b.WriteString(dimStyle().Render(fmt.Sprintf(" %s used · %s measured · %s budget\n",
		fmtTokens(m.ctxUsed), fmtTokens(m.ctxPrompt), fmtTokens(m.ctxBudget))))
	if len(m.ctxSections) == 0 {
		b.WriteString(dimStyle().Render("\n no sections"))
		return b.String()
	}
	inner := max(8, w-2)
	shareW := min(12, max(6, w-8))
	for i, s := range m.ctxSections {
		share := 0
		if m.ctxUsed > 0 {
			share = s.tokens * 100 / m.ctxUsed
		}
		name := clip(s.name, max(6, inner-12))
		row := fmt.Sprintf("%s  %5s  %3d%%", name, fmtTokens(s.tokens), share)
		if i == m.ctxOpen {
			b.WriteString(lipgloss.NewStyle().Foreground(accent).Bold(true).Render("> "+row) + "\n")
			b.WriteString("  " + dimStyle().Render(miniBar(share, shareW)) + "\n")
			b.WriteString(sectionPreview(s.text, inner) + "\n")
		} else {
			b.WriteString(dimStyle().Render("  "+row) + "\n")
		}
	}
	return b.String()
}

func sectionPreview(text string, w int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return dimStyle().Render("  (no preview)")
	}
	maxLines := 8
	wrapped := lipgloss.NewStyle().
		Foreground(fg).
		Width(max(8, w)).
		Render(text)
	lines := strings.Split(wrapped, "\n")
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], dimStyle().Render("…"))
	}
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func miniBar(pct, w int) string {
	if w < 4 {
		w = 4
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct * w / 100
	return strings.Repeat("█", filled) + strings.Repeat("░", w-filled)
}

func fmtTokens(n int) string {
	switch {
	case n >= 10000:
		return fmt.Sprintf("%.0fk", float64(n)/1000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (m *Model) moveContextSection(delta int) {
	if len(m.ctxSections) == 0 {
		return
	}
	m.ctxOpen += delta
	if m.ctxOpen < 0 {
		m.ctxOpen = 0
	}
	if m.ctxOpen >= len(m.ctxSections) {
		m.ctxOpen = len(m.ctxSections) - 1
	}
	m.refreshInspect()
}

func (m *Model) contextAgent() string {
	return m.chatFilter
}
