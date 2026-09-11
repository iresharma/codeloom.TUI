package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

type focusKind int

const (
	focusTree focusKind = iota
	focusCode
	focusChat
	focusComposer
	focusGraph
	focusInspect
)

func (f focusKind) name() string {
	return []string{"files", "code", "chat", "insert", "graph", "inspect"}[int(f)]
}

type keySet struct {
	Quit, Abort, Undo, Help     key.Binding
	NextPane, PrevPane, Insert  key.Binding
	Escape, Zoom                key.Binding
	Tree, Code, Chat, GraphPane key.Binding
	Inspect                     key.Binding
	TabPrev, TabNext, TabClose  key.Binding
	Diff                        key.Binding
	Open, NewFile, NewDir       key.Binding
	Rename, Delete, Refresh     key.Binding
	Peek, Ctx, Mem, InspRefresh key.Binding
	SecUp, SecDown              key.Binding
	PromptOK                    key.Binding
}

func newKeySet() keySet {
	return keySet{
		Quit:        key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Abort:       key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "abort")),
		Undo:        key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "undo edit")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		NextPane:    key.NewBinding(key.WithKeys("tab", "ctrl+w"), key.WithHelp("tab", "next pane")),
		PrevPane:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-tab", "prev pane")),
		Insert:      key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "insert")),
		Escape:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Zoom:        key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "zoom graph")),
		Tree:        key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "files")),
		Code:        key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "code")),
		Chat:        key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "chat")),
		GraphPane:   key.NewBinding(key.WithKeys("4", "g"), key.WithHelp("4/g", "graph")),
		Inspect:     key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "inspect")),
		TabPrev:     key.NewBinding(key.WithKeys("[", "h"), key.WithHelp("[/h", "prev tab")),
		TabNext:     key.NewBinding(key.WithKeys("]", "l"), key.WithHelp("]/l", "next tab")),
		TabClose:    key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "close tab")),
		Diff:        key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "diff")),
		Open:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		NewFile:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "new file")),
		NewDir:      key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "new dir")),
		Rename:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
		Delete:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Refresh:     key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reload")),
		Peek:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "transcript")),
		Ctx:         key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "context")),
		Mem:         key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "memory")),
		InspRefresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		SecUp:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "prev section")),
		SecDown:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "next section")),
		PromptOK:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")),
	}
}

func (m Model) ShortHelp() []key.Binding {
	k := m.keys
	if m.promptID != "" && (m.focus == focusChat || m.focus == focusComposer) {
		if len(m.promptChoices) > 0 {
			return []key.Binding{k.PromptOK, k.SecDown, k.SecUp, k.Insert, k.Escape, k.Help}
		}
		return []key.Binding{k.PromptOK, k.Escape, k.Help}
	}
	base := []key.Binding{k.NextPane, k.Insert, k.Help}
	switch m.focus {
	case focusTree:
		return []key.Binding{k.Open, k.NewFile, k.Rename, k.NextPane, k.Insert, k.Help}
	case focusCode:
		return []key.Binding{k.TabPrev, k.TabNext, k.Diff, k.TabClose, k.Insert, k.Help}
	case focusChat:
		return []key.Binding{k.Insert, k.NextPane, k.Help}
	case focusComposer:
		return []key.Binding{k.Escape, k.Abort, k.Help}
	case focusGraph:
		return []key.Binding{k.Peek, k.Zoom, k.Ctx, k.Mem, k.Insert, k.Help}
	case focusInspect:
		if m.rightSub == 0 {
			return []key.Binding{k.TabPrev, k.TabNext, k.SecDown, k.SecUp, k.Help}
		}
		return []key.Binding{k.TabPrev, k.TabNext, k.Help}
	}
	return base
}

func (m Model) FullHelp() [][]key.Binding {
	k := m.keys
	return [][]key.Binding{
		{k.NextPane, k.PrevPane, k.Insert, k.Escape, k.Help},
		{k.Tree, k.Code, k.Chat, k.GraphPane, k.Inspect, k.Zoom},
		{k.TabPrev, k.TabNext, k.TabClose, k.Diff},
		{k.Open, k.NewFile, k.NewDir, k.Rename, k.Delete, k.Refresh},
		{k.Peek, k.Ctx, k.Mem, k.Undo, k.Abort, k.Quit},
	}
}

var _ help.KeyMap = Model{}

func (m *Model) setFocus(f focusKind) {
	if m.focus == focusComposer && f != focusComposer {
		m.input.Blur()
	}
	if f == focusComposer && m.focus != focusComposer {
		m.prevFocus = m.focus
		m.input.Focus()
	}
	m.focus = f
}

func (m Model) focusOrder() []focusKind {
	return []focusKind{focusTree, focusCode, focusComposer, focusGraph, focusInspect}
}

func (m *Model) cycleFocus(delta int) {
	order := m.focusOrder()
	cur := m.focus
	if cur == focusChat {
		cur = focusComposer
	}
	idx := 0
	for i, f := range order {
		if f == cur {
			idx = i
			break
		}
	}
	n := len(order)
	m.setFocus(order[(idx+delta%n+n)%n])
}

func (m *Model) handleKey(msg tea.KeyMsg) (cmd tea.Cmd, handled bool) {
	if m.modal != modalNone {
		return m.handleModalKey(msg), true
	}
	k := m.keys
	switch {
	case key.Matches(msg, k.Quit):
		m.quitting = true
		return tea.Quit, true
	case key.Matches(msg, k.Help):
		m.help.ShowAll = !m.help.ShowAll
		return nil, true
	case key.Matches(msg, k.Abort):
		return m.send(protocol.AbortAgent(m.chatFilter)), true
	}
	if m.help.ShowAll {
		if key.Matches(msg, k.Escape) {
			m.help.ShowAll = false
			return nil, true
		}
		return nil, true
	}
	if cmd, handled := m.handlePromptKey(msg); handled {
		return cmd, true
	}
	if m.focus == focusComposer {
		switch {
		case key.Matches(msg, k.Escape):
			back := m.prevFocus
			if back == focusComposer {
				back = focusChat
			}
			m.setFocus(back)
			return nil, true
		case msg.String() == "enter":
			return m.submitComposer(), true
		case key.Matches(msg, k.NextPane), key.Matches(msg, k.PrevPane):
			if key.Matches(msg, k.PrevPane) {
				m.cycleFocus(-1)
			} else {
				m.cycleFocus(1)
			}
			return nil, true
		}
		return nil, false
	}
	switch {
	case key.Matches(msg, k.Escape):
		if m.graphZoom {
			m.graphZoom = false
			m.refreshGraph()
			return nil, true
		}
		return nil, true
	case key.Matches(msg, k.NextPane):
		m.cycleFocus(1)
		return nil, true
	case key.Matches(msg, k.PrevPane):
		m.cycleFocus(-1)
		return nil, true
	case key.Matches(msg, k.Insert):
		m.setFocus(focusComposer)
		return nil, true
	case key.Matches(msg, k.Tree):
		m.setFocus(focusTree)
		return nil, true
	case key.Matches(msg, k.Code):
		m.setFocus(focusCode)
		return nil, true
	case key.Matches(msg, k.Chat):
		m.setFocus(focusComposer)
		return nil, true
	case key.Matches(msg, k.GraphPane):
		m.setFocus(focusGraph)
		return nil, true
	case key.Matches(msg, k.Inspect):
		m.setFocus(focusInspect)
		return nil, true
	case key.Matches(msg, k.Undo):
		return m.send(protocol.UndoLastEdit()), true
	}

	switch m.focus {
	case focusTree:
		return m.handleTreeKey(msg), true
	case focusCode:
		return m.handleCodeKey(msg)
	case focusGraph:
		return m.handleGraphKey(msg)
	case focusInspect:
		return m.handleInspectKey(msg)
	}
	return nil, false
}

func (m *Model) handlePromptKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.promptID == "" {
		return nil, false
	}
	if m.focus != focusChat && m.focus != focusComposer {
		return nil, false
	}
	k := m.keys
	empty := strings.TrimSpace(m.input.Value()) == ""
	switch {
	case key.Matches(msg, k.Escape):
		if m.focus == focusComposer && !empty {
			m.input.Reset()
			return nil, true
		}
		return m.answerPrompt(m.promptDecline()), true
	case msg.String() == "enter":
		if !empty {
			return m.submitComposer(), true
		}
		if len(m.promptChoices) == 0 {
			return nil, true
		}
		return m.answerPrompt(m.selectedPromptChoice()), true
	case msg.String() == "up" || (empty && msg.String() == "k"):
		m.movePromptChoice(-1)
		return nil, true
	case msg.String() == "down" || (empty && msg.String() == "j"):
		m.movePromptChoice(1)
		return nil, true
	}
	if empty && len(m.promptChoices) > 0 {
		s := msg.String()
		if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
			idx := int(s[0] - '1')
			if idx < len(m.promptChoices) {
				return m.answerPrompt(m.promptChoices[idx]), true
			}
		}
	}
	return nil, false
}

func (m *Model) handleCodeKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	k := m.keys
	switch {
	case key.Matches(msg, k.TabNext):
		m.nextTab(1)
		return nil, true
	case key.Matches(msg, k.TabPrev):
		m.nextTab(-1)
		return nil, true
	case key.Matches(msg, k.TabClose):
		return m.closeTab(), true
	case key.Matches(msg, k.Diff):
		t := m.currentTab()
		if t != nil && t.diff != "" {
			t.showDiff = !t.showDiff
			m.refreshFileVP()
		}
		return nil, true
	}
	return nil, false
}

func (m *Model) handleGraphKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	k := m.keys
	switch {
	case key.Matches(msg, k.Zoom):
		m.graphZoom = !m.graphZoom
		m.refreshGraph()
		return nil, true
	case key.Matches(msg, k.Peek), msg.String() == "enter":
		return m.peekSelectedAgent(), true
	case key.Matches(msg, k.TabPrev), key.Matches(msg, k.Ctx):
		return m.switchInspect(0), true
	case key.Matches(msg, k.TabNext), key.Matches(msg, k.Mem):
		return m.switchInspect(1), true
	case msg.String() == "j", msg.String() == "down":
		m.moveGraphCursor(1)
		return nil, true
	case msg.String() == "k", msg.String() == "up":
		m.moveGraphCursor(-1)
		return nil, true
	}
	return nil, false
}

func (m *Model) handleInspectKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	k := m.keys
	switch {
	case key.Matches(msg, k.TabPrev), key.Matches(msg, k.Ctx):
		return m.switchInspect(0), true
	case key.Matches(msg, k.TabNext), key.Matches(msg, k.Mem):
		return m.switchInspect(1), true
	case m.rightSub == 0 && key.Matches(msg, k.SecUp):
		m.moveContextSection(-1)
		return nil, true
	case m.rightSub == 0 && key.Matches(msg, k.SecDown):
		m.moveContextSection(1)
		return nil, true
	case key.Matches(msg, k.InspRefresh):
		return m.switchInspect(m.rightSub), true
	}
	return nil, false
}

func (m *Model) switchInspect(sub int) tea.Cmd {
	m.rightSub = sub
	m.inspectVP.YOffset = 0
	if m.hideInspect() {
		m.setFocus(focusInspect)
	}
	m.refreshInspect()
	if sub == 1 {
		return m.send(protocol.RequestMemory())
	}
	return m.send(protocol.RequestContext(m.contextAgent()))
}
