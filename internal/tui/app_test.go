package tui

import (
	"testing"

	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

func TestFileContentOpensTab(t *testing.T) {
	m := New(t.TempDir(), nil, nil)
	m.handleEvent(mustEvent(`{"type":"FileContent","path":"a.py","content":"print(1)"}`))
	if len(m.tabs) != 2 || m.tabs[1].kind != tabFile || m.tabs[1].body != "print(1)" {
		t.Fatalf("tabs %#v", m.tabs)
	}
	if m.active != 1 {
		t.Fatalf("active %d", m.active)
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

func mustEvent(raw string) protocol.Event {
	ev, err := protocol.DecodeEvent([]byte(raw))
	if err != nil {
		panic(err)
	}
	return ev
}
