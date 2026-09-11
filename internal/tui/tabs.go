package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

func (m *Model) openFileTab(path, content string) {
	for i, t := range m.tabs {
		if t.path == path {
			m.tabs[i].body = content
			m.active = i
			m.tabs[i].edited = false
			m.fileVP.YOffset = 0
			m.refreshFileVP()
			if m.focus != focusComposer {
				m.setFocus(focusCode)
			}
			return
		}
	}
	title := path
	if base := filepath.Base(path); base != "" && base != "." {
		title = base
	}
	m.tabs = append(m.tabs, tab{title: title, path: path, body: content})
	m.active = len(m.tabs) - 1
	m.fileVP.YOffset = 0
	m.refreshFileVP()
	if m.focus != focusComposer {
		m.setFocus(focusCode)
	}
}

func (m *Model) nextTab(delta int) {
	if len(m.tabs) == 0 {
		return
	}
	m.active = (m.active + delta + len(m.tabs)) % len(m.tabs)
	m.fileVP.YOffset = 0
	m.refreshFileVP()
}

func (m *Model) closeTab() tea.Cmd {
	if m.active < 0 || m.active >= len(m.tabs) {
		return nil
	}
	t := m.tabs[m.active]
	var cmd tea.Cmd
	if t.path != "" {
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

func (m Model) viewCode(w, h int) string {
	head := m.viewTabBar(w)
	bodyH := max(1, h-lipgloss.Height(head))
	body := m.viewCodeBody(w, bodyH)
	return lipgloss.JoinVertical(lipgloss.Top, head, body)
}

func (m Model) viewTabBar(w int) string {
	if len(m.tabs) == 0 {
		return lipgloss.NewStyle().Foreground(dim).Background(bgAlt).Width(max(1, w)).Render(" no file")
	}
	active := lipgloss.NewStyle().Foreground(bg).Background(accent).Bold(true).Padding(0, 1)
	idle := lipgloss.NewStyle().Foreground(dim).Padding(0, 1)
	edited := lipgloss.NewStyle().Foreground(yellow).Padding(0, 1)
	var parts []string
	for i, t := range m.tabs {
		label := t.title
		if t.edited {
			label = "● " + label
		}
		if lipgloss.Width(label) > 22 {
			label = "…" + label[len(label)-21:]
		}
		st := idle
		if i == m.active {
			st = active
		} else if t.edited {
			st = edited
		}
		parts = append(parts, zones.Mark("tab-"+fmt.Sprint(i), st.Render(label)))
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return lipgloss.NewStyle().Background(bgAlt).Width(max(1, w)).Render(bar)
}

func (m Model) viewCodeBody(w, h int) string {
	t := m.currentTab()
	if t == nil {
		recent := m.recentFiles()
		msg := lipgloss.JoinVertical(lipgloss.Center,
			titleStyle().Render("code"),
			"",
			dimStyle().Render("open a file from the tree"),
			dimStyle().Render("enter · 1 files"),
		)
		if len(recent) > 0 {
			msg = lipgloss.JoinVertical(lipgloss.Center, msg, "", dimStyle().Render(strings.Join(recent, "  ")))
		}
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
	}
	hint := ""
	if t.showDiff && t.diff != "" {
		hint = dimStyle().Render(" diff overlay  ctrl+d restore") + "\n"
	}
	return fit(hint+m.fileVP.View(), w, h)
}

func (m Model) recentFiles() []string {
	var out []string
	for _, t := range m.tabs {
		if t.path != "" {
			out = append(out, filepath.Base(t.path))
		}
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func (m Model) viewComposer(w, h int) string {
	c := border
	if m.focus == focusComposer {
		c = focusC
	}
	return zones.Mark("pane-composer", lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(c).
		Width(max(1, w)).
		Height(max(1, h)).
		Padding(0, 1).
		Render(m.input.View()))
}
