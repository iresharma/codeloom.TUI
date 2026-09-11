package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iresharma/codeloom.TUI/internal/client"
	"github.com/iresharma/codeloom.TUI/internal/host"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

type tab struct {
	title, path, body, diff string
	showDiff, edited        bool
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

type chatLine struct {
	id, role, text, agentID string
}

type modalKind int

const (
	modalNone modalKind = iota
	modalCreateFile
	modalCreateDir
	modalRename
	modalDelete
)

type eventMsg protocol.Event
type errMsg struct{ err error }
type tickMsg time.Time

type Model struct {
	workspace     string
	engine        *host.Engine
	conn          *client.Conn
	events        <-chan protocol.Event
	width, height int
	focus         focusKind
	prevFocus     focusKind
	rightSub      int
	keys          keySet
	help          help.Model
	spin          spinner.Model
	ctxProg       progress.Model
	graphZoom     bool
	chatFilter    string
	chatFollow    bool

	tree        *fileTree
	tabs        []tab
	active      int
	input       textarea.Model
	modalInput  textinput.Model
	chatVP      viewport.Model
	fileVP      viewport.Model
	graphVP     viewport.Model
	inspectVP   viewport.Model
	chat        []chatLine
	streams     map[string]int
	streamAgent map[string]string
	agentChat   map[string][]chatLine

	cost, elapsed             float64
	tokens, cached, requests  int
	promptTok, compTok, turns int
	orchState                 string
	git                       gitInfo
	agents                    []agentRow
	agentCursor               int

	ctxBudget, ctxUsed, ctxPrompt      int
	ctxSections                        []ctxSection
	ctxOpen                            int
	memFiles                           []memFile
	memEng, memProd, memCICD, memOther []memDec

	modal             modalKind
	modalPath         string
	promptID, promptQ string
	promptKind        string
	promptChoices     []string
	promptCursor      int

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
	ta.Blur()
	ta.KeyMap.InsertNewline.SetEnabled(false)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = dimStyle()
	ta.BlurredStyle.Placeholder = dimStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(accent)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(fg)

	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 256
	ti.Placeholder = ""

	fileVP := viewport.New(20, 10)
	km := viewport.DefaultKeyMap()
	km.HalfPageDown.SetEnabled(false)
	km.HalfPageUp.SetEnabled(false)
	km.Left.SetEnabled(false)
	km.Right.SetEnabled(false)
	fileVP.KeyMap = km
	fileVP.MouseWheelEnabled = true

	chatVP := viewport.New(20, 8)
	chatVP.MouseWheelEnabled = true

	graphVP := viewport.New(20, 10)
	gkm := viewport.DefaultKeyMap()
	gkm.Up.SetEnabled(false)
	gkm.Down.SetEnabled(false)
	gkm.Left.SetEnabled(false)
	gkm.Right.SetEnabled(false)
	graphVP.KeyMap = gkm

	inspectVP := viewport.New(20, 8)
	ikm := viewport.DefaultKeyMap()
	ikm.Left.SetEnabled(false)
	ikm.Right.SetEnabled(false)
	inspectVP.KeyMap = ikm
	inspectVP.MouseWheelEnabled = true

	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(accent)))
	prog := progress.New(progress.WithSolidFill(string(accent)), progress.WithoutPercentage(), progress.WithWidth(14))

	h := help.New()
	h.ShowAll = false
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(accent)
	h.Styles.ShortDesc = dimStyle()
	h.Styles.ShortSeparator = dimStyle()
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(accent)
	h.Styles.FullDesc = dimStyle()

	m := Model{
		workspace:   workspace,
		engine:      eng,
		conn:        conn,
		focus:       focusTree,
		prevFocus:   focusTree,
		keys:        newKeySet(),
		help:        h,
		spin:        sp,
		ctxProg:     prog,
		chatFollow:  true,
		tree:        newFileTree(workspace),
		input:       ta,
		modalInput:  ti,
		fileVP:      fileVP,
		chatVP:      chatVP,
		graphVP:     graphVP,
		inspectVP:   inspectVP,
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
		m.spin.Tick,
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
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		if m.busy() {
			cmds = append(cmds, cmd)
			m.refreshGraph()
		}
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
		if m.busy() {
			cmds = append(cmds, m.spin.Tick)
		}
	case tea.MouseMsg:
		if cmd := m.handleMouse(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		cmd, handled := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if handled {
			return m, tea.Batch(cmds...)
		}
		var c tea.Cmd
		switch {
		case m.modal != modalNone:
			m.modalInput, c = m.modalInput.Update(msg)
			cmds = append(cmds, c)
		case m.focus == focusComposer:
			m.input, c = m.input.Update(msg)
			cmds = append(cmds, c)
			m.layoutViewports()
		case m.focus == focusCode:
			m.fileVP, c = m.fileVP.Update(msg)
			cmds = append(cmds, c)
		case m.focus == focusChat:
			m.chatVP, c = m.chatVP.Update(msg)
			cmds = append(cmds, c)
			m.chatFollow = m.chatVP.AtBottom()
		case m.focus == focusGraph:
			m.graphVP, c = m.graphVP.Update(msg)
			cmds = append(cmds, c)
		case m.focus == focusInspect:
			m.inspectVP, c = m.inspectVP.Update(msg)
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
		var cmd tea.Cmd
		switch m.paneAt(msg) {
		case focusCode:
			m.fileVP, cmd = m.fileVP.Update(msg)
		case focusChat, focusComposer:
			m.chatVP, cmd = m.chatVP.Update(msg)
			m.chatFollow = m.chatVP.AtBottom()
		case focusGraph:
			m.graphVP, cmd = m.graphVP.Update(msg)
		case focusInspect:
			m.inspectVP, cmd = m.inspectVP.Update(msg)
		}
		return cmd
	}
	if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	if zones.Get("insp-ctx").InBounds(msg) {
		m.setFocus(focusInspect)
		return m.switchInspect(0)
	}
	if zones.Get("insp-mem").InBounds(msg) {
		m.setFocus(focusInspect)
		return m.switchInspect(1)
	}
	for i := range m.promptChoices {
		if zones.Get("prompt-" + strconv.Itoa(i)).InBounds(msg) {
			m.setFocus(focusChat)
			if m.promptCursor == i {
				return m.answerPrompt(m.promptChoices[i])
			}
			m.promptCursor = i
			return nil
		}
	}
	for i := range m.tabs {
		if zones.Get("tab-" + strconv.Itoa(i)).InBounds(msg) {
			m.active = i
			m.fileVP.YOffset = 0
			m.refreshFileVP()
			m.setFocus(focusCode)
			return nil
		}
	}
	if zones.Get("agent-").InBounds(msg) {
		m.setFocus(focusGraph)
		return m.peekAgent("")
	}
	for _, a := range m.agents {
		if zones.Get("agent-" + a.id).InBounds(msg) {
			m.setFocus(focusGraph)
			return m.peekAgent(a.id)
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
	m.setFocus(m.paneAt(msg))
	return nil
}

func (m *Model) openModal(kind modalKind, initial, path string) {
	m.modal = kind
	m.modalPath = path
	m.modalInput.SetValue(initial)
	m.modalInput.CursorEnd()
	m.modalInput.Focus()
}

func (m *Model) handleModalKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.modal = modalNone
		m.modalInput.Blur()
		return nil
	case "enter":
		return m.commitModal()
	}
	var cmd tea.Cmd
	m.modalInput, cmd = m.modalInput.Update(msg)
	return cmd
}

func (m *Model) commitModal() tea.Cmd {
	kind := m.modal
	buf := strings.TrimSpace(m.modalInput.Value())
	path := m.modalPath
	m.modal = modalNone
	m.modalInput.Blur()
	m.modalInput.SetValue("")
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
		m.openModal(modalCreateFile, "", "")
	case "A":
		m.openModal(modalCreateDir, "", "")
	case "r":
		n := m.tree.current()
		if n == nil {
			return nil
		}
		m.openModal(modalRename, n.name, n.path)
	case "d":
		n := m.tree.current()
		if n == nil {
			return nil
		}
		m.openModal(modalDelete, "", n.path)
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
					if m.chatFilter == aid {
						m.refreshChat()
					}
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
		found := false
		for i := range m.tabs {
			if m.tabs[i].path == path {
				m.tabs[i].diff = diff
				m.tabs[i].edited = true
				found = true
			}
		}
		m.tree.reload()
		if !found && path != "" {
			return m.send(protocol.OpenFile(path))
		}
	case "FileTreeUpdated", "PathChanged":
		m.tree.reload()
		m.status = ev.Type
	case "GitStateUpdated":
		m.applyGit(ev.Map("git"))
		m.refreshInspect()
	case "SnapshotReady":
		snap := ev.Map("snapshot")
		if g, ok := snap["git"].(map[string]any); ok {
			m.applyGit(g)
		}
		if s, ok := snap["stats"].(map[string]any); ok {
			m.applyStats(s)
		} else {
			m.applyStats(snap)
		}
		m.refreshInspect()
	case "StatsUpdated":
		m.applyStats(ev.Raw)
	case "AgentStateChanged":
		if ev.Str("agent_id") == "" {
			m.orchState = ev.Str("state")
		}
		id := ev.Str("agent_id")
		st := ev.Str("state")
		if id != "" && agentDone(st) {
			m.dropAgent(id)
		} else if id != "" {
			for i := range m.agents {
				if m.agents[i].id == id {
					m.agents[i].status = st
				}
			}
		}
		m.refreshGraph()
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
		m.refreshGraph()
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
		if m.ctxOpen >= len(m.ctxSections) {
			m.ctxOpen = max(0, len(m.ctxSections)-1)
		}
		m.refreshInspect()
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
		m.refreshInspect()
	case "UserPromptRequested":
		m.promptID = ev.Str("prompt_id")
		m.promptQ = ev.Str("question")
		m.promptKind = ev.Str("kind")
		m.promptChoices = protocol.AsStringSlice(ev.Raw["choices"])
		m.promptCursor = 0
		m.chatFollow = true
		m.input.Reset()
		m.setFocus(focusComposer)
		m.status = "prompt: " + m.promptKind
		m.layoutViewports()
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
	if s == nil {
		return
	}
	if nested, ok := s["stats"].(map[string]any); ok {
		m.applyStatsFields(nested)
		return
	}
	m.applyStatsFields(s)
}

func (m *Model) applyStatsFields(s map[string]any) {
	if v, ok := lookupNum(s, "cost", "cost_usd", "usd", "total_cost"); ok {
		m.cost = v
	}
	if v, ok := lookupNum(s, "prompt_tokens", "input_tokens"); ok {
		m.promptTok = int(v)
	}
	if v, ok := lookupNum(s, "completion_tokens", "output_tokens"); ok {
		m.compTok = int(v)
	}
	if v, ok := lookupNum(s, "total_tokens", "tokens"); ok {
		m.tokens = int(v)
	} else if m.promptTok+m.compTok > 0 {
		m.tokens = m.promptTok + m.compTok
	}
	if v, ok := lookupNum(s, "cached_tokens", "cache_tokens", "cached"); ok {
		m.cached = int(v)
	}
	if v, ok := lookupNum(s, "requests", "request_count"); ok {
		m.requests = int(v)
	}
	if v, ok := lookupNum(s, "elapsed_s", "elapsed"); ok {
		m.elapsed = v
	}
	if v, ok := lookupNum(s, "turns"); ok {
		m.turns = int(v)
	}
	if runs := asMaps(s["agent_runs"]); len(runs) > 0 {
		sum := 0.0
		for _, row := range runs {
			sum += num(row["cost"])
		}
		m.cost += sum
	}
}

func asMaps(v any) []map[string]any {
	arr, _ := v.([]any)
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		row, ok := item.(map[string]any)
		if ok {
			out = append(out, row)
		}
	}
	return out
}

func lookupNum(s map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := s[k]; ok && v != nil {
			if n, ok := numeric(v); ok {
				return n, true
			}
			if nested, ok := v.(map[string]any); ok {
				if n, ok := lookupNum(nested, keys...); ok {
					return n, true
				}
			}
		}
	}
	for _, nest := range []string{"stats", "usage", "metrics", "totals"} {
		m, _ := s[nest].(map[string]any)
		if len(m) == 0 {
			continue
		}
		if v, ok := lookupNum(m, keys...); ok {
			return v, true
		}
	}
	return 0, false
}

func numeric(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case int32:
		return float64(t), true
	case string:
		n, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(t), "$"), 64)
		return n, err == nil
	}
	return 0, false
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
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case int32:
		return float64(t)
	case string:
		n, _ := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(t), "$"), 64)
		return n
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
