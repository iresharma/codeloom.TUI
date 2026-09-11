package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type layout struct {
	left, center, right, body int
	compact, hideInspect      bool
}

func (m Model) layout() layout {
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	d := layout{body: max(8, h-1)}
	switch {
	case w < 90:
		d.compact = true
		d.hideInspect = true
		d.center = w
	case w < 120:
		d.left, d.right = 24, 32
		d.center = max(20, w-d.left-d.right)
	default:
		d.left, d.right = 28, 36
		d.center = max(20, w-d.left-d.right)
	}
	return d
}

func (m Model) hideInspect() bool {
	return m.layout().hideInspect
}

func (m *Model) layoutViewports() {
	d := m.layout()
	centerW := d.center
	if d.compact {
		centerW = max(20, m.width)
	}
	innerW := max(10, centerW-2)
	codeH := max(6, (d.body*3)/5)
	chatH := max(8, d.body-codeH)
	m.fileVP.Width = innerW
	m.fileVP.Height = max(3, codeH-4)
	compH := m.composerHeight()
	promptH := m.promptCardHeight(innerW)
	m.chatVP.Width = innerW
	m.chatVP.Height = max(3, chatH-2-compH-1-promptH)
	m.input.SetWidth(max(12, innerW-4))
	m.input.SetHeight(max(1, compH-1))

	rightW := d.right
	if d.compact {
		rightW = min(36, max(24, m.width/2))
	}
	graphInner := max(10, rightW-2)
	graphH := d.body
	if !d.hideInspect {
		graphH = max(10, (d.body*3)/5)
	}
	m.graphVP.Width = graphInner
	m.graphVP.Height = max(4, graphH-7)
	m.inspectVP.Width = graphInner
	inspInner := d.body - 2 - 1
	if !d.hideInspect {
		inspInner = d.body - graphH - 2 - 1
	}
	m.inspectVP.Height = max(3, inspInner)
	m.help.Width = max(20, m.width-24)
	m.refreshGraph()
	m.refreshInspect()
	m.refreshFileVP()
	m.refreshChat()
}

func (m Model) View() string {
	if m.width == 0 {
		return "starting…"
	}
	d := m.layout()
	var cols string
	if d.compact {
		cols = m.viewCompact(d)
	} else {
		cols = m.viewWide(d)
	}
	ui := lipgloss.JoinVertical(lipgloss.Left, cols, m.viewFooter())
	if m.help.ShowAll {
		ui = overlay(m.width, m.height, ui, m.viewFullHelp())
	}
	if m.modal != modalNone {
		ui = overlay(m.width, m.height, ui, m.viewModal())
	}
	return zones.Scan(ui)
}

func (m Model) viewWide(d layout) string {
	left := zones.Mark("pane-tree", paneStyle(m.focus == focusTree, d.left, d.body).
		Render(m.viewTree(d.left-2, d.body-2)))
	var center string
	if m.graphZoom {
		center = zones.Mark("pane-graph", paneStyle(true, d.center, d.body).
			Render(m.graphVP.View()))
	} else {
		codeH := max(6, (d.body*3)/5)
		chatH := max(6, d.body-codeH)
		code := zones.Mark("pane-code", paneStyle(m.focus == focusCode, d.center, codeH).
			Render(m.viewCode(d.center-2, codeH-2)))
		chat := zones.Mark("pane-chat", paneStyle(m.focus == focusChat || m.focus == focusComposer, d.center, chatH).
			Render(m.viewChatStrip(d.center-2, chatH-2)))
		center = lipgloss.JoinVertical(lipgloss.Top, code, chat)
	}
	rightW, rightH := d.right, d.body
	var right string
	if d.hideInspect {
		right = zones.Mark("pane-graph", paneStyle(m.focus == focusGraph, rightW, rightH).
			Render(m.viewGraphPane(rightW-2, rightH-2)))
	} else {
		graphH := max(10, (rightH*3)/5)
		inspH := max(6, rightH-graphH)
		graph := zones.Mark("pane-graph", paneStyle(m.focus == focusGraph, rightW, graphH).
			Render(m.viewGraphPane(rightW-2, graphH-2)))
		insp := zones.Mark("pane-inspect", paneStyle(m.focus == focusInspect, rightW, inspH).
			Render(m.viewInspectPane(rightW-2, inspH-2)))
		right = lipgloss.JoinVertical(lipgloss.Top, graph, insp)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
}

func (m Model) viewCompact(d layout) string {
	codeH := max(6, (d.body*3)/5)
	chatH := max(6, d.body-codeH)
	w := max(20, m.width)
	var main string
	if m.graphZoom {
		main = zones.Mark("pane-graph", paneStyle(true, w, d.body).Render(m.graphVP.View()))
	} else {
		code := zones.Mark("pane-code", paneStyle(m.focus == focusCode, w, codeH).
			Render(m.viewCode(w-2, codeH-2)))
		chat := zones.Mark("pane-chat", paneStyle(m.focus == focusChat || m.focus == focusComposer, w, chatH).
			Render(m.viewChatStrip(w-2, chatH-2)))
		main = lipgloss.JoinVertical(lipgloss.Top, code, chat)
	}
	switch m.focus {
	case focusTree:
		tw := min(28, max(18, w/2))
		tree := zones.Mark("pane-tree", paneStyle(true, tw, d.body).
			Render(m.viewTree(tw-2, d.body-2)))
		return lipgloss.JoinHorizontal(lipgloss.Top, tree, main)
	case focusGraph:
		gw := min(36, max(22, w/2))
		graph := zones.Mark("pane-graph", paneStyle(true, gw, d.body).
			Render(m.viewGraphPane(gw-2, d.body-2)))
		return lipgloss.JoinHorizontal(lipgloss.Top, main, graph)
	case focusInspect:
		iw := min(36, max(22, w/2))
		insp := zones.Mark("pane-inspect", paneStyle(true, iw, d.body).
			Render(m.viewInspectPane(iw-2, d.body-2)))
		return lipgloss.JoinHorizontal(lipgloss.Top, main, insp)
	}
	return main
}

func (m Model) viewFooter() string {
	owned := "attach"
	if m.engine != nil && m.engine.Owned {
		owned = "spawned"
	}
	left := fmt.Sprintf(" %s %s %s", "codeloom", owned, m.focus.name())
	if m.git.branch != "" {
		left += "  " + m.git.branch
	}
	if m.lastErr != "" {
		err := m.lastErr
		if len(err) > 24 {
			err = err[:24]
		}
		left += "  " + err
	}
	cost := lipgloss.NewStyle().Foreground(green).Bold(true).Render("$" + fmtCost(m.cost))
	if m.tokens > 0 {
		cost += dimStyle().Render("  " + fmtTokens(m.tokens))
	}
	helpView := m.help.View(m)
	w := max(1, m.width)
	right := cost
	if lipgloss.Width(left)+lipgloss.Width(helpView)+lipgloss.Width(cost)+4 <= w {
		right = cost + "  " + helpView
	}
	gap := max(1, w-lipgloss.Width(left)-lipgloss.Width(right)-1)
	line := dimStyle().Render(left) + strings.Repeat(" ", gap) + right + " "
	if lipgloss.Width(line) > w {
		line = clip(dimStyle().Render(left)+"  "+cost, w)
	}
	return lipgloss.NewStyle().Background(bgAlt).Width(w).Render(line)
}

func (m Model) viewFullHelp() string {
	m.help.ShowAll = true
	body := m.help.View(m)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(min(72, max(40, m.width-8))).
		Render(titleStyle().Render("keys") + "\n\n" + body + "\n\n" + dimStyle().Render("? or esc close"))
}

func overlay(w, h int, _, box string) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(dim),
	)
}

func (m Model) viewModal() string {
	title := "input"
	hint := "enter confirm · esc cancel"
	switch m.modal {
	case modalCreateFile:
		title = "new file"
	case modalCreateDir:
		title = "new directory"
	case modalRename:
		title = "rename " + m.modalPath
	case modalDelete:
		title = "delete " + m.modalPath + " ? (y/n)"
	}
	body := titleStyle().Render(title) + "\n\n" + m.modalInput.View() + "\n\n" + dimStyle().Render(hint)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(min(60, max(24, m.width-4))).
		Render(body)
}

func (m Model) paneAt(msg tea.MouseMsg) focusKind {
	switch {
	case zones.Get("pane-tree").InBounds(msg):
		return focusTree
	case zones.Get("pane-code").InBounds(msg):
		return focusCode
	case zones.Get("pane-composer").InBounds(msg):
		return focusComposer
	case zones.Get("pane-chat").InBounds(msg):
		return focusComposer
	case zones.Get("pane-graph").InBounds(msg):
		return focusGraph
	case zones.Get("pane-inspect").InBounds(msg):
		return focusInspect
	}
	return m.focus
}
