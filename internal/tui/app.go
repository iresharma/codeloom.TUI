package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iresharma/codeloom.TUI/internal/client"
	"github.com/iresharma/codeloom.TUI/internal/host"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

type pane int

const (
	paneTree pane = iota
	paneCenter
	paneRight
)

type tabKind int

const (
	tabChat tabKind = iota
	tabFile
	tabAgent
)

type modalKind int

const (
	modalNone modalKind = iota
	modalCreateFile
	modalCreateDir
	modalRename
	modalDelete
	modalPrompt
)

type chatLine struct {
	id, role, text, agentID string
}

type tab struct {
	kind     tabKind
	title    string
	path     string
	agentID  string
	body     string
	diff     string
	showDiff bool
}

type agentRow struct {
	id, profile, status, tool, parent, task, batch string
}

type gitInfo struct {
	branch                      string
	dirty                       bool
	staged, unstaged, untracked []string
}

type ctxSection struct {
	name, text    string
	chars, tokens int
}

type memFile struct {
	path, purpose string
	stale         bool
}

type memDec struct{ text string }

type eventMsg protocol.Event
type errMsg struct{ err error }
type tickMsg time.Time

type Model struct {
	workspace     string
	engine        *host.Engine
	conn          *client.Conn
	events        <-chan protocol.Event
	width, height int
	focus         pane
	rightSub      int // 0 context, 1 memory

	tree        *fileTree
	tabs        []tab
	active      int
	input       textarea.Model
	chatVP      viewport.Model
	fileVP      viewport.Model
	chat        []chatLine
	streams     map[string]int
	streamAgent map[string]string
	agentChat   map[string][]chatLine

	cost                     float64
	tokens, cached, requests int
	orchState                string
	git                      gitInfo
	agents                   []agentRow
	agentCursor              int

	ctxBudget, ctxUsed, ctxPrompt      int
	ctxSections                        []ctxSection
	ctxOpen                            int
	memFiles                           []memFile
	memEng, memProd, memCICD, memOther []memDec

	modal             modalKind
	modalBuf          string
	modalPath         string
	promptID, promptQ string
	promptChoices     []string

	status   string
	lastErr  string
	quitting bool
}

func New(workspace string, eng *host.Engine, conn *client.Conn) Model {
	ta := textarea.New()
	ta.Placeholder = "message the orchestrator"
	ta.Prompt = "› "
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Focus()
	ta.KeyMap.InsertNewline.SetEnabled(false)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = dimStyle()
	ta.BlurredStyle.Placeholder = dimStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(accent)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(fg)
	m := Model{
		workspace:   workspace,
		engine:      eng,
		conn:        conn,
		tree:        newFileTree(workspace),
		tabs:        []tab{{kind: tabChat, title: "chat"}},
		input:       ta,
		streams:     map[string]int{},
		streamAgent: map[string]string{},
		agentChat:   map[string][]chatLine{},
		orchState:   "idle",
		status:      "connected",
	}
	if conn != nil {
		m.events = conn.Events()
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.listen(),
		m.send(protocol.StartSession(m.workspace, "")),
		m.send(protocol.RequestSnapshot(true)),
		m.send(protocol.RequestGit()),
		m.send(protocol.RequestMemory()),
		m.send(protocol.RequestContext("")),
		tick(),
	)
}

func tick() tea.Cmd {
	return tea.Tick(4*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) listen() tea.Cmd {
	if m.events == nil {
		return nil
	}
	ch := m.events
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return errMsg{fmt.Errorf("engine disconnected")}
		}
		return eventMsg(ev)
	}
}

func (m Model) send(cmd protocol.Command) tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return errMsg{fmt.Errorf("not connected")}
		}
		if err := m.conn.Send(cmd); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layoutViewports()
	case tickMsg:
		cmds = append(cmds, tick())
	case errMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
		}
		cmds = append(cmds, m.listen())
	case eventMsg:
		if cmd := m.handleEvent(protocol.Event(msg)); cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, m.listen())
	case tea.MouseMsg:
		if cmd := m.handleMouse(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.modal == modalNone && m.focus != paneTree {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	for i := range m.tabs {
		if zones.Get("tab-" + strconv.Itoa(i)).InBounds(msg) {
			m.active = i
			m.refreshFileVP()
			return nil
		}
	}
	if zones.Get("agent-").InBounds(msg) {
		return nil
	}
	for _, a := range m.agents {
		if zones.Get("agent-" + a.id).InBounds(msg) {
			return m.openAgent(a.id)
		}
	}
	openGit := func(paths []string) tea.Cmd {
		for _, p := range paths {
			if p != "" && zones.Get("git-"+p).InBounds(msg) {
				return m.send(protocol.OpenFile(p))
			}
		}
		return nil
	}
	if cmd := openGit(m.git.staged); cmd != nil {
		return cmd
	}
	if cmd := openGit(m.git.unstaged); cmd != nil {
		return cmd
	}
	if cmd := openGit(m.git.untracked); cmd != nil {
		return cmd
	}
	for _, f := range m.memFiles {
		if zones.Get("mem-" + f.path).InBounds(msg) {
			return m.send(protocol.OpenFile(f.path))
		}
	}
	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if m.modal != modalNone {
		return m.handleModalKey(msg)
	}
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return tea.Quit
	case "ctrl+x":
		id := ""
		if t := m.currentTab(); t != nil && t.kind == tabAgent {
			id = t.agentID
		}
		return m.send(protocol.AbortAgent(id))
	case "ctrl+w", "tab":
		m.focus = (m.focus + 1) % 3
		return nil
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
		return nil
	case "ctrl+u":
		return m.send(protocol.UndoLastEdit())
	case "ctrl+e":
		return m.closeTab()
	}
	if m.focus == paneTree {
		return m.handleTreeKey(msg)
	}
	if m.focus == paneRight {
		switch msg.String() {
		case "h", "left":
			m.rightSub = 0
			return m.send(protocol.RequestContext(m.contextAgent()))
		case "l", "right":
			m.rightSub = 1
			return m.send(protocol.RequestMemory())
		case "j", "down":
			if len(m.agents) > 0 && m.agentCursor < len(m.agents)-1 {
				m.agentCursor++
			}
		case "k", "up":
			if m.agentCursor > 0 {
				m.agentCursor--
			}
		case "[", "ctrl+k":
			if m.ctxOpen > 0 {
				m.ctxOpen--
			}
		case "]", "ctrl+j":
			if m.ctxOpen < len(m.ctxSections)-1 {
				m.ctxOpen++
			}
		case "r":
			if m.rightSub == 1 {
				return m.send(protocol.RequestMemory())
			}
			return m.send(protocol.RequestContext(m.contextAgent()))
		case "enter":
			if a := m.selectedAgent(); a != "" {
				return m.openAgent(a)
			}
			return m.send(protocol.RequestContext(m.contextAgent()))
		}
		return nil
	}
	if m.focus == paneCenter {
		switch msg.String() {
		case "ctrl+t":
			if m.active < len(m.tabs)-1 {
				m.active++
				m.refreshFileVP()
			}
		case "ctrl+p":
			if m.active > 0 {
				m.active--
				m.refreshFileVP()
			}
		case "ctrl+d":
			t := m.currentTab()
			if t != nil && t.kind == tabFile {
				t.showDiff = !t.showDiff
				m.refreshFileVP()
				return nil
			}
		case "enter":
			if strings.TrimSpace(m.input.Value()) == "" {
				return nil
			}
			text := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			if m.promptID != "" {
				id := m.promptID
				m.promptID = ""
				return m.send(protocol.AnswerPrompt(id, text))
			}
			return m.send(protocol.SubmitUserMessage(text))
		}
	}
	return nil
}

func (m *Model) handleModalKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.modal = modalNone
		m.modalBuf = ""
		return nil
	case "enter":
		return m.commitModal()
	case "backspace":
		if len(m.modalBuf) > 0 {
			m.modalBuf = m.modalBuf[:len(m.modalBuf)-1]
		}
	default:
		if len(msg.Runes) == 1 {
			m.modalBuf += string(msg.Runes)
		}
	}
	return nil
}

func (m *Model) commitModal() tea.Cmd {
	kind := m.modal
	buf := strings.TrimSpace(m.modalBuf)
	path := m.modalPath
	m.modal = modalNone
	m.modalBuf = ""
	switch kind {
	case modalCreateFile:
		if buf == "" {
			return nil
		}
		p := buf
		if n := m.tree.current(); n != nil && n.isDir {
			p = filepath.ToSlash(filepath.Join(n.path, buf))
		} else if n != nil {
			p = filepath.ToSlash(filepath.Join(filepath.Dir(n.path), buf))
		}
		return m.send(protocol.CreatePath(p, false, ""))
	case modalCreateDir:
		if buf == "" {
			return nil
		}
		p := buf
		if n := m.tree.current(); n != nil && n.isDir {
			p = filepath.ToSlash(filepath.Join(n.path, buf))
		} else if n != nil {
			p = filepath.ToSlash(filepath.Join(filepath.Dir(n.path), buf))
		}
		return m.send(protocol.CreatePath(p, true, ""))
	case modalRename:
		if buf == "" {
			return nil
		}
		dest := filepath.ToSlash(filepath.Join(filepath.Dir(path), buf))
		return m.send(protocol.RenamePath(path, dest))
	case modalDelete:
		if strings.ToLower(buf) != "y" && strings.ToLower(buf) != "yes" {
			return nil
		}
		return m.send(protocol.DeletePath(path))
	case modalPrompt:
		id := m.promptID
		m.promptID = ""
		return m.send(protocol.AnswerPrompt(id, buf))
	}
	return nil
}

func (m *Model) handleTreeKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		m.tree.move(1)
	case "k", "up":
		m.tree.move(-1)
	case "h", "left":
		n := m.tree.current()
		if n != nil && n.isDir && n.open {
			m.tree.toggle()
		}
	case "l", "right":
		n := m.tree.current()
		if n != nil && n.isDir && !n.open {
			m.tree.toggle()
		}
	case "enter":
		n := m.tree.current()
		if n == nil {
			return nil
		}
		if n.isDir {
			m.tree.toggle()
			return nil
		}
		return m.send(protocol.OpenFile(n.path))
	case "a":
		m.modal = modalCreateFile
		m.modalBuf = ""
	case "A":
		m.modal = modalCreateDir
		m.modalBuf = ""
	case "r":
		n := m.tree.current()
		if n == nil {
			return nil
		}
		m.modal = modalRename
		m.modalPath = n.path
		m.modalBuf = n.name
	case "d":
		n := m.tree.current()
		if n == nil {
			return nil
		}
		m.modal = modalDelete
		m.modalPath = n.path
		m.modalBuf = ""
	case "R":
		m.tree.reload()
	}
	return nil
}

func (m *Model) handleEvent(ev protocol.Event) tea.Cmd {
	switch ev.Type {
	case "ChatMessageStarted":
		id := ev.Str("id")
		aid := ev.Str("agent_id")
		if aid == "" {
			m.chat = append(m.chat, chatLine{id: id, role: ev.Str("role"), agentID: aid})
			m.streams[id] = len(m.chat) - 1
			m.refreshChat()
			break
		}
		m.appendAgent(aid, ev.Str("role"), "", false)
		m.streamAgent[id] = aid
	case "ChatMessageDelta":
		id := ev.Str("id")
		if aid, ok := m.streamAgent[id]; ok {
			m.appendAgentDelta(aid, ev.Str("text"))
			break
		}
		if i, ok := m.streams[id]; ok && i < len(m.chat) {
			m.chat[i].text += ev.Str("text")
			m.refreshChat()
		}
	case "ChatMessageAdded":
		id := ev.Str("id")
		aid := ev.Str("agent_id")
		if _, ok := m.streamAgent[id]; ok {
			delete(m.streamAgent, id)
			if aid != "" {
				lines := m.agentChat[aid]
				if len(lines) > 0 {
					lines[len(lines)-1].text = ev.Str("text")
					lines[len(lines)-1].role = ev.Str("role")
					m.agentChat[aid] = lines
				} else {
					m.appendAgent(aid, ev.Str("role"), ev.Str("text"), false)
				}
			}
			break
		}
		if i, ok := m.streams[id]; ok && i < len(m.chat) {
			m.chat[i].text = ev.Str("text")
			m.chat[i].role = ev.Str("role")
			delete(m.streams, id)
			m.refreshChat()
			break
		}
		if aid == "" {
			m.chat = append(m.chat, chatLine{id: id, role: ev.Str("role"), text: ev.Str("text")})
			m.refreshChat()
		} else {
			m.appendAgent(aid, ev.Str("role"), ev.Str("text"), false)
		}
	case "ChatHistoryAdded":
		aid := ev.Str("agent_id")
		if aid == "" {
			if ev.Int("index") == 0 {
				m.chat = m.chat[:0]
			}
			m.chat = append(m.chat, chatLine{id: ev.Str("id"), role: ev.Str("role"), text: ev.Str("text")})
			m.refreshChat()
		} else {
			m.appendAgent(aid, ev.Str("role"), ev.Str("text"), ev.Int("index") == 0)
		}
	case "FileContent":
		m.openFileTab(ev.Str("path"), ev.Str("content"))
	case "FileEdited":
		path := ev.Str("path")
		diff := ev.Str("diff")
		for i := range m.tabs {
			if m.tabs[i].kind == tabFile && m.tabs[i].path == path {
				m.tabs[i].diff = diff
			}
		}
		m.tree.reload()
	case "FileTreeUpdated", "PathChanged":
		m.tree.reload()
		m.status = ev.Type
	case "GitStateUpdated":
		m.applyGit(ev.Map("git"))
	case "SnapshotReady":
		snap := ev.Map("snapshot")
		if g, ok := snap["git"].(map[string]any); ok {
			m.applyGit(g)
		}
		if s, ok := snap["stats"].(map[string]any); ok {
			m.applyStats(s)
		}
	case "StatsUpdated":
		m.applyStats(ev.Map("stats"))
	case "AgentStateChanged":
		if ev.Str("agent_id") == "" {
			m.orchState = ev.Str("state")
		}
		id := ev.Str("agent_id")
		for i := range m.agents {
			if m.agents[i].id == id {
				m.agents[i].status = ev.Str("state")
			}
		}
	case "AgentsUpdated":
		var live []agentRow
		for _, raw := range ev.Slice("agents") {
			row, _ := raw.(map[string]any)
			live = append(live, agentRow{
				id:      str(row["id"]),
				profile: str(row["profile"]),
				status:  str(row["status"]),
				tool:    str(row["current_tool"]),
				parent:  str(row["parent_id"]),
				task:    str(row["task"]),
				batch:   str(row["batch_name"]),
			})
		}
		m.mergeAgents(live)
	case "AgentStarted":
		m.status = "spawned " + ev.Str("profile")
	case "ContextBreakdown":
		m.ctxBudget = ev.Int("budget")
		m.ctxPrompt = ev.Int("prompt_tokens")
		m.ctxSections = m.ctxSections[:0]
		used := 0
		for _, raw := range ev.Slice("sections") {
			row, _ := raw.(map[string]any)
			sec := ctxSection{
				name:   str(row["name"]),
				text:   str(row["text"]),
				chars:  int(num(row["chars"])),
				tokens: int(num(row["tokens_est"])),
			}
			used += sec.tokens
			m.ctxSections = append(m.ctxSections, sec)
		}
		m.ctxUsed = used
	case "MemoryUpdated":
		m.memFiles = m.memFiles[:0]
		for _, raw := range ev.Slice("files") {
			row, _ := raw.(map[string]any)
			m.memFiles = append(m.memFiles, memFile{
				path: str(row["path"]), purpose: str(row["purpose"]), stale: truth(row["stale"]),
			})
		}
		m.memEng = decisions(ev.Slice("engineering"))
		m.memProd = decisions(ev.Slice("product"))
		m.memCICD = decisions(ev.Slice("cicd"))
		m.memOther = decisions(ev.Slice("other"))
	case "UserPromptRequested":
		m.promptID = ev.Str("prompt_id")
		m.promptQ = ev.Str("question")
		m.promptChoices = protocol.AsStringSlice(ev.Raw["choices"])
		m.modal = modalPrompt
		m.modalBuf = ""
		m.status = "prompt: " + ev.Str("kind")
	case "ErrorOccurred", "WarningOccurred":
		m.lastErr = ev.Str("message")
	case "SessionEnded":
		m.status = "session ended"
	case "ContextCompacted":
		m.status = "context compacted"
		return m.send(protocol.RequestContext(m.contextAgent()))
	}
	return nil
}

func (m *Model) applyGit(g map[string]any) {
	m.git.branch, _ = g["branch"].(string)
	m.git.dirty, _ = g["dirty"].(bool)
	m.git.staged = protocol.AsStringSlice(g["staged"])
	m.git.unstaged = protocol.AsStringSlice(g["unstaged"])
	m.git.untracked = protocol.AsStringSlice(g["untracked"])
}

func (m *Model) applyStats(s map[string]any) {
	m.cost = num(s["cost"])
	m.tokens = int(num(s["total_tokens"]))
	m.cached = int(num(s["cached_tokens"]))
	m.requests = int(num(s["requests"]))
}

func (m *Model) layoutViewports() {
	_, centerW, _, bodyH := m.dims()
	innerW := max(10, centerW-2)
	innerH := max(8, bodyH-2)
	chatH := max(3, innerH-4) // tab + composer
	m.chatVP.Width = innerW
	m.chatVP.Height = chatH
	m.fileVP.Width = innerW
	m.fileVP.Height = chatH
	m.input.SetWidth(max(12, innerW-4))
	m.input.SetHeight(1)
}

func (m Model) dims() (left, center, right, body int) {
	w, h := m.width, m.height
	if w < 40 {
		w = 80
	}
	if h < 20 {
		h = 24
	}
	left, right = 30, 38
	if w < 120 {
		left, right = 26, 34
	}
	if w < 90 {
		left, right = 22, 28
	}
	center = w - left - right
	if center < 20 {
		center = 20
	}
	body = h - 1
	return
}

func (m Model) View() string {
	if m.width == 0 {
		return "starting…"
	}
	leftW, centerW, rightW, bodyH := m.dims()
	left := paneStyle(m.focus == paneTree, leftW, bodyH).Render(m.viewTree(leftW-2, bodyH-2))
	center := paneStyle(m.focus == paneCenter, centerW, bodyH).Render(m.viewCenter(centerW-2, bodyH-2))
	right := paneStyle(m.focus == paneRight, rightW, bodyH).Render(m.viewRight(rightW-2, bodyH-2))
	cols := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	ui := lipgloss.JoinVertical(lipgloss.Left, cols, m.viewFooter())
	if m.modal != modalNone {
		ui = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewModal(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(dim),
		)
	}
	return zones.Scan(ui)
}

func (m Model) viewFooter() string {
	owned := "attach"
	if m.engine != nil && m.engine.Owned {
		owned = "spawned"
	}
	focus := []string{"files", "chat", "side"}[int(m.focus)]
	err := m.lastErr
	if len(err) > 40 {
		err = err[:40]
	}
	left := fmt.Sprintf(" %s  %s  %s", "codeloom", owned, focus)
	br := m.git.branch
	if br == "" {
		br = "-"
	}
	right := fmt.Sprintf("$%.3f  %s  tab·panes  %s ", m.cost, br, err)
	gap := max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	line := left + strings.Repeat(" ", gap) + right
	if len(line) > m.width && m.width > 0 {
		line = line[:m.width]
	}
	return lipgloss.NewStyle().Background(bgAlt).Foreground(dim).Width(max(1, m.width)).Render(line)
}

func (m Model) viewModal() string {
	title := "input"
	help := "enter confirm · esc cancel"
	switch m.modal {
	case modalCreateFile:
		title = "new file"
	case modalCreateDir:
		title = "new directory"
	case modalRename:
		title = "rename " + m.modalPath
	case modalDelete:
		title = "delete " + m.modalPath + " ? (y/n)"
	case modalPrompt:
		title = m.promptQ
		if len(m.promptChoices) > 0 {
			help = strings.Join(m.promptChoices, " / ")
		}
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(min(60, m.width-4)).
		Render(titleStyle().Render(title) + "\n\n> " + m.modalBuf + "█\n\n" + dimStyle().Render(help))
	return box
}

func (m Model) Close() {
	if m.conn != nil {
		if m.engine != nil && m.engine.Owned {
			_ = m.conn.Send(protocol.Shutdown())
		}
		_ = m.conn.Close()
	}
	if m.engine != nil && m.engine.Owned {
		m.engine.Kill()
	}
}

func num(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	}
	return 0
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func truth(v any) bool {
	b, _ := v.(bool)
	return b
}

func decisions(raw []any) []memDec {
	out := make([]memDec, 0, len(raw))
	for _, item := range raw {
		row, _ := item.(map[string]any)
		out = append(out, memDec{text: str(row["text"])})
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
