package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

func TestFileContentOpensTab(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	m.handleEvent(mustEvent(`{"type":"FileContent","path":"a.py","content":"print(1)"}`))
	if len(m.tabs) != 1 || m.tabs[0].path != "a.py" || m.tabs[0].body != "print(1)" {
		t.Fatalf("tabs %#v", m.tabs)
	}
	if m.active != 0 {
		t.Fatalf("active %d", m.active)
	}
	if m.focus != focusCode {
		t.Fatalf("focus %v want code", m.focus)
	}
}

func TestStatsUpdatedReadsEnginePayload(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	m.handleEvent(mustEvent(`{"type":"StatsUpdated","stats":{"prompt_tokens":12400,"completion_tokens":3100,"cached_tokens":400,"total_tokens":15500,"cost":0.041,"turns":3,"elapsed_s":18.2,"agent_runs":[{"agent_id":"a1","profile":"coder","cost":1.25}]}}`))
	if m.cost < 1.29 || m.cost > 1.30 {
		t.Fatalf("session+agent cost=%v want ~1.291", m.cost)
	}
	if m.tokens != 15500 || m.promptTok != 12400 || m.compTok != 3100 {
		t.Fatalf("tokens total=%d in=%d out=%d", m.tokens, m.promptTok, m.compTok)
	}
	view := m.View()
	if !strings.Contains(view, "$1.29") && !strings.Contains(view, "$1.291") {
		t.Fatalf("cost missing from ui: %q", clip(view, 500))
	}
	if !strings.Contains(view, "in") || !strings.Contains(view, "out") {
		t.Fatalf("token breakdown missing: %q", clip(view, 500))
	}
}

func TestGitAndMemoryEvents(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	m.handleEvent(mustEvent(`{"type":"GitStateUpdated","git":{"branch":"main","dirty":true,"staged":["a.py"],"unstaged":[],"untracked":["b.py"]}}`))
	if m.git.branch != "main" || len(m.git.staged) != 1 || len(m.git.untracked) != 1 {
		t.Fatalf("git %#v", m.git)
	}
	m.handleEvent(mustEvent(`{"type":"MemoryUpdated","files":[{"path":"a.py","stale":true,"purpose":"x"}],"engineering":[{"text":"ship"}],"product":[],"cicd":[],"other":[]}`))
	if len(m.memFiles) != 1 || !m.memFiles[0].stale || len(m.memEng) != 1 {
		t.Fatalf("memory files=%v eng=%v", m.memFiles, m.memEng)
	}
	m.handleEvent(mustEvent(`{"type":"ContextBreakdown","budget":1000,"prompt_tokens":40,"sections":[{"name":"system","chars":8,"tokens_est":2,"text":"hello"}]}`))
	if m.ctxBudget != 1000 || m.ctxUsed != 2 || len(m.ctxSections) != 1 {
		t.Fatalf("ctx budget=%d used=%d secs=%v", m.ctxBudget, m.ctxUsed, m.ctxSections)
	}
}

func TestAssistantMessageWrapsAtPaneWidth(t *testing.T) {
	text := strings.Repeat("hello world ", 16)
	out := renderLines([]chatLine{{role: "assistant", text: text}}, true, 24)
	if !strings.Contains(out, "hello") || strings.Count(out, "\n") < 3 {
		t.Fatalf("expected wrapped assistant text, got %q", out)
	}
	for i, line := range strings.Split(out, "\n") {
		if lipgloss.Width(line) > 24 {
			t.Fatalf("line %d width %d > 24: %q", i, lipgloss.Width(line), line)
		}
	}
	if !strings.Contains(out, "world") {
		t.Fatal("wrap should keep words intact")
	}
}

func TestFocusCodeDoesNotTypeIntoComposer(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	m.openFileTab("a.py", strings.Repeat("line\n", 80))
	m.setFocus(focusCode)
	m.input.SetValue("")
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mod.(Model)
	if m.input.Value() != "" {
		t.Fatalf("composer stole j: %q", m.input.Value())
	}
	if m.fileVP.YOffset < 1 {
		t.Fatalf("expected file viewport to scroll, y=%d", m.fileVP.YOffset)
	}
}

func TestTabSwitchBrackets(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	m.openFileTab("a.py", "a")
	m.openFileTab("b.py", "b")
	if m.active != 1 {
		t.Fatalf("active %d", m.active)
	}
	m.setFocus(focusCode)
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m = mod.(Model)
	if m.active != 0 {
		t.Fatalf("prev tab active=%d", m.active)
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = mod.(Model)
	if m.active != 1 {
		t.Fatalf("next tab active=%d", m.active)
	}
}

func TestLayoutBreakpoints(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	m.width, m.height = 80, 24
	d := m.layout()
	if !d.compact || !d.hideInspect || d.center < 20 {
		t.Fatalf("80-col layout %+v", d)
	}
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mod.(Model)
	if v := m.View(); v == "" || v == "starting…" {
		t.Fatalf("compact view empty")
	}
	m.width, m.height = 100, 30
	d = m.layout()
	if d.compact || d.hideInspect || d.left == 0 || d.right == 0 {
		t.Fatalf("100-col layout %+v", d)
	}
	m.width, m.height = 140, 40
	d = m.layout()
	if d.compact || d.hideInspect || d.left == 0 || d.right == 0 {
		t.Fatalf("140-col layout %+v", d)
	}
	mod, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	view := m.View()
	if !strings.Contains(view, "files") {
		t.Fatalf("wide view missing files pane: %q", clip(view, 200))
	}
	if !strings.Contains(view, "agents") {
		t.Fatalf("wide view missing agents: %q", clip(view, 200))
	}
	if !strings.Contains(view, "context") && !strings.Contains(view, "CONTEXT") {
		t.Fatalf("wide view missing inspector: %q", clip(view, 200))
	}
}

func TestSyntaxHighlighting(t *testing.T) {
	out := renderSource("demo.py", "def greet():\n    return 1\n")
	if !strings.Contains(out, "\x1b") {
		t.Fatalf("expected ANSI highlight, got %q", out)
	}
	if !strings.Contains(out, "1 │") {
		t.Fatalf("expected line numbers, got %q", out)
	}
	diff := renderDiff("@@ -1 +1 @@\n-old\n+new\n")
	if !strings.Contains(diff, "old") || !strings.Contains(diff, "new") {
		t.Fatalf("diff render: %q", diff)
	}
}

func TestSwitchMemoryFromGraphAndInspect(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	m.setFocus(focusGraph)
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = mod.(Model)
	if m.rightSub != 1 {
		t.Fatalf("graph l should open memory, rightSub=%d", m.rightSub)
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = mod.(Model)
	if m.rightSub != 0 {
		t.Fatalf("graph h should open context, rightSub=%d", m.rightSub)
	}
	m.setFocus(focusInspect)
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = mod.(Model)
	if m.rightSub != 1 {
		t.Fatalf("inspect right should open memory, rightSub=%d focus=%v", m.rightSub, m.focus)
	}
	if !strings.Contains(m.viewInspect(40), "MEMORY") {
		t.Fatalf("inspector should show memory tab active")
	}
}

func TestCompactInspectOverlay(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mod.(Model)
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = mod.(Model)
	if m.focus != focusInspect {
		t.Fatalf("5 should focus inspect overlay, got %v", m.focus)
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = mod.(Model)
	if m.rightSub != 1 {
		t.Fatalf("l on inspect overlay should switch memory, got %d", m.rightSub)
	}
}

func TestHelpBindings(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	if len(m.ShortHelp()) == 0 {
		t.Fatal("short help empty")
	}
	if len(m.FullHelp()) == 0 {
		t.Fatal("full help empty")
	}
	if m.help.ShowAll {
		t.Fatal("help should start collapsed")
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = mod.(Model)
	if !m.help.ShowAll {
		t.Fatal("? should open full help")
	}
	foot := m.viewFooter()
	if !strings.Contains(foot, "tab") && !strings.Contains(foot, "insert") && !strings.Contains(foot, "?") {
		t.Fatalf("footer missing help hints: %q", foot)
	}
}

func TestComposerOnlyWhenFocused(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	if m.focus == focusComposer || m.input.Focused() {
		t.Fatal("composer should start blurred")
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = mod.(Model)
	if m.focus != focusComposer || !m.input.Focused() {
		t.Fatalf("i should insert, focus=%v focused=%v", m.focus, m.input.Focused())
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mod.(Model)
	if m.focus == focusComposer || m.input.Focused() {
		t.Fatal("esc should blur composer")
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = mod.(Model)
	if m.focus != focusComposer || !m.input.Focused() {
		t.Fatalf("3 should focus the prompt, focus=%v focused=%v", m.focus, m.input.Focused())
	}
}

func TestAgentPeekDoesNotStealFile(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	m.openFileTab("a.py", "print(1)")
	m.mergeAgents([]agentRow{{id: "a1", profile: "coder", status: "running"}})
	m.setFocus(focusGraph)
	m.agentCursor = 1
	_ = m.peekSelectedAgent()
	if m.chatFilter != "a1" {
		t.Fatalf("filter %q", m.chatFilter)
	}
	if len(m.tabs) != 1 || m.tabs[0].path != "a.py" {
		t.Fatalf("peek replaced file tabs: %#v", m.tabs)
	}
	if m.focus != focusGraph {
		t.Fatalf("peek stole focus: %v", m.focus)
	}
}

func TestInspectTabsUseBrackets(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	m.handleEvent(mustEvent(`{"type":"ContextBreakdown","budget":1000,"prompt_tokens":400,"sections":[{"name":"system","chars":40,"tokens_est":200,"text":"You are the orchestrator.\nKeep replies short."},{"name":"files","chars":80,"tokens_est":800,"text":"internal/tui/app.go\nfunc (m Model) View()"}]}`))
	m.setFocus(focusInspect)
	m.rightSub = 0
	m.ctxOpen = 0
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = mod.(Model)
	if m.rightSub != 1 {
		t.Fatalf("] should open memory tab, rightSub=%d", m.rightSub)
	}
	if m.ctxOpen != 0 {
		t.Fatalf("] should not cycle sections, ctxOpen=%d", m.ctxOpen)
	}
	if !strings.Contains(m.viewInspectPane(36, 12), "MEMORY") {
		t.Fatal("memory tab should be active")
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m = mod.(Model)
	if m.rightSub != 0 {
		t.Fatalf("[ should open context tab, rightSub=%d", m.rightSub)
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mod.(Model)
	if m.ctxOpen != 1 {
		t.Fatalf("j should move to next section, ctxOpen=%d", m.ctxOpen)
	}
	view := m.viewInspectBody(36)
	if !strings.Contains(view, "files") || !strings.Contains(view, "app.go") {
		t.Fatalf("selected section should show readable preview, got %q", view)
	}
	if strings.Contains(view, "internal/tui/app.go func") {
		t.Fatalf("preview should keep line breaks, got %q", view)
	}
}

func TestInlinePromptNotModal(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = mod.(Model)
	m.openFileTab("a.py", "print(1)")
	m.handleEvent(mustEvent(`{"type":"UserPromptRequested","prompt_id":"p1","kind":"permission","question":"Allow go test?","choices":["allow","deny"]}`))
	if m.modal != modalNone {
		t.Fatal("prompt should not open a modal overlay")
	}
	if m.promptID != "p1" || m.focus != focusComposer || !m.input.Focused() {
		t.Fatalf("prompt should focus composer, id=%q focus=%v focused=%v", m.promptID, m.focus, m.input.Focused())
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = mod.(Model)
	if m.input.Value() != "y" {
		t.Fatalf("typed into prompt without i, got %q", m.input.Value())
	}
	m.input.Reset()
	view := m.View()
	if !strings.Contains(view, "Allow go test?") {
		t.Fatalf("inline prompt missing question: %q", clip(view, 400))
	}
	if !strings.Contains(view, "allow") || !strings.Contains(view, "deny") {
		t.Fatalf("inline prompt missing choices: %q", clip(view, 400))
	}
	if !strings.Contains(view, "print(1)") && !strings.Contains(view, "a.py") {
		t.Fatalf("prompt overlay hid the file: %q", clip(view, 400))
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mod.(Model)
	if m.promptCursor != 1 {
		t.Fatalf("j should move to deny, cursor=%d", m.promptCursor)
	}
	mod, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mod.(Model)
	if m.promptID != "" {
		t.Fatal("enter should answer and clear prompt")
	}
	if len(m.chat) < 2 || m.chat[len(m.chat)-1].text != "deny" {
		t.Fatalf("answered chat %#v", m.chat)
	}
}

func mustEvent(raw string) protocol.Event {
	ev, err := protocol.DecodeEvent([]byte(raw))
	if err != nil {
		panic(err)
	}
	return ev
}
