package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewSession(w int) string {
	in, out := m.promptTok, m.compTok
	if in+out == 0 && m.tokens > 0 {
		out = m.tokens
	}
	line1 := fmt.Sprintf(" $%s  %s in / %s out", fmtCost(m.cost), fmtTokens(in), fmtTokens(out))
	if m.cached > 0 {
		line1 += "  " + fmtTokens(m.cached) + " cache"
	}
	line2 := ""
	if m.elapsed > 0 || m.turns > 0 {
		line2 = fmt.Sprintf(" %s  turn %d", fmtElapsed(m.elapsed), m.turns)
	}
	br := m.git.branch
	if br == "" {
		br = "no git"
	}
	dirty := "clean"
	if m.git.dirty {
		dirty = "dirty"
	}
	git := fmt.Sprintf(" %s %s  +%d ~%d ?%d", br, dirty, len(m.git.staged), len(m.git.unstaged), len(m.git.untracked))
	if line2 == "" {
		line2 = git
	} else {
		line2 += "  " + strings.TrimSpace(git)
	}
	return clip(line1, w) + "\n" + clip(line2, w)
}

func fmtCost(c float64) string {
	switch {
	case c >= 100:
		return fmt.Sprintf("%.0f", c)
	case c >= 1:
		return fmt.Sprintf("%.2f", c)
	case c >= 0.01:
		return fmt.Sprintf("%.3f", c)
	default:
		return fmt.Sprintf("%.3f", c)
	}
}

func fmtElapsed(s float64) string {
	if s < 60 {
		return fmt.Sprintf("%.1fs", s)
	}
	return fmt.Sprintf("%.0fm%02.0fs", s/60, float64(int(s)%60))
}

func (m Model) gitGlyph(path string) string {
	slash := func(p string) string { return strings.ReplaceAll(p, "\\", "/") }
	path = slash(path)
	for _, p := range m.git.staged {
		if slash(p) == path || strings.HasSuffix(slash(p), "/"+path) {
			return gitAdd().Render(" A")
		}
	}
	for _, p := range m.git.unstaged {
		if slash(p) == path || strings.HasSuffix(slash(p), "/"+path) {
			return gitMod().Render(" M")
		}
	}
	for _, p := range m.git.untracked {
		if slash(p) == path || strings.HasSuffix(slash(p), "/"+path) {
			return gitUntracked().Render(" ?")
		}
	}
	return ""
}

func (m Model) viewTree(w, h int) string {
	help := dimStyle().Render(" a file  A dir  r  d  enter")
	inner := h - 2
	if inner < 1 {
		inner = 1
	}
	var b strings.Builder
	start := 0
	if m.tree.cursor >= inner {
		start = m.tree.cursor - inner + 1
	}
	end := start + inner
	if end > len(m.tree.flat) {
		end = len(m.tree.flat)
	}
	for i := start; i < end; i++ {
		n := m.tree.flat[i]
		depth := depthOf(n.path)
		if depth > 0 {
			depth--
		}
		pad := strings.Repeat(" ", depth)
		mark := "  "
		if n.isDir {
			if n.open {
				mark = "▾ "
			} else {
				mark = "▸ "
			}
		}
		glyph := m.gitGlyph(n.path)
		line := pad + mark + n.name + glyph
		if i == m.tree.cursor {
			line = lipgloss.NewStyle().Foreground(bg).Background(accent).Bold(true).Width(max(1, w)).Render(" " + line)
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	body := fit(b.String(), w, inner)
	return lipgloss.JoinVertical(lipgloss.Top,
		headerBar("files", w),
		body,
		help,
	)
}
