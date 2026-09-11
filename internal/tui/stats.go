package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewRight(w, h int) string {
	statsH := 8
	graphH := max(7, h/5)
	ctxH := h - statsH - graphH
	if ctxH < 8 {
		ctxH = 8
		graphH = max(5, h-statsH-ctxH)
	}
	if statsH+graphH+ctxH > h {
		ctxH = max(4, h-statsH-graphH)
	}
	return lipgloss.JoinVertical(lipgloss.Top,
		fit(m.viewStats(w), w, statsH),
		fit(m.viewGraph(w, graphH), w, graphH),
		fit(m.viewContext(w, ctxH), w, ctxH),
	)
}

func (m Model) viewStats(w int) string {
	br := m.git.branch
	if br == "" {
		br = "no git"
	}
	dirty := dimStyle().Render(" clean")
	if m.git.dirty {
		dirty = gitMod().Render(" dirty")
	}
	var b strings.Builder
	b.WriteString(headerBar("session", w) + "\n")
	b.WriteString(fmt.Sprintf(" $%.3f  %s  cache %d\n", m.cost, fmt.Sprintf("%d tok", m.tokens), m.cached))
	b.WriteString(" " + br + dirty + fmt.Sprintf("   +%d  ~%d  ?%d\n",
		len(m.git.staged), len(m.git.unstaged), len(m.git.untracked)))
	shown := 0
	writeGit := func(label string, paths []string, sty lipgloss.Style) {
		for _, p := range paths {
			if p == "" || shown >= 3 {
				continue
			}
			name := p
			if len(name) > w-4 {
				name = "…" + name[len(name)-(w-5):]
			}
			b.WriteString(zones.Mark("git-"+p, sty.Render(" "+label+" "+name)) + "\n")
			shown++
		}
	}
	writeGit("A", m.git.staged, gitAdd())
	writeGit("M", m.git.unstaged, gitMod())
	writeGit("?", m.git.untracked, gitUntracked())
	extra := len(m.git.staged) + len(m.git.unstaged) + len(m.git.untracked) - shown
	if extra > 0 {
		b.WriteString(dimStyle().Render(fmt.Sprintf(" +%d more", extra)) + "\n")
	}
	return b.String()
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
		name := n.name
		line := pad + mark + name + glyph
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
