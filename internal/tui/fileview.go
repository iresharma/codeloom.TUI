package tui

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) refreshFileVP() {
	t := m.currentTab()
	if t == nil {
		m.fileVP.SetContent("")
		return
	}
	body := t.body
	if t.showDiff && t.diff != "" {
		m.fileVP.SetContent(renderDiff(t.diff))
		return
	}
	m.fileVP.SetContent(renderSource(t.path, body))
}

func renderSource(path, content string) string {
	highlighted, ok := highlight(path, content)
	if !ok {
		highlighted = content
	}
	return withLineNumbers(highlighted)
}

func highlight(path, content string) (string, bool) {
	if content == "" {
		return "", false
	}
	lexer := lexers.Match(filepath.Base(path))
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, content)
	if err != nil {
		return "", false
	}
	style := styles.Get("tokyonight-night")
	if style == nil {
		style = styles.Fallback
	}
	var buf bytes.Buffer
	if err := formatters.TTY16m.Format(&buf, style, it); err != nil {
		return "", false
	}
	return buf.String(), true
}

func withLineNumbers(content string) string {
	content = strings.TrimSuffix(content, "\n")
	lines := strings.Split(content, "\n")
	var b strings.Builder
	gutter := lipgloss.NewStyle().Foreground(dim)
	for i, line := range lines {
		b.WriteString(gutter.Render(fmt.Sprintf("%4d │ ", i+1)))
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func renderDiff(content string) string {
	add := gitAdd()
	del := gitUntracked()
	hunk := lipgloss.NewStyle().Foreground(accent)
	gutter := lipgloss.NewStyle().Foreground(dim)
	var b strings.Builder
	for i, line := range strings.Split(content, "\n") {
		b.WriteString(gutter.Render(fmt.Sprintf("%4d │ ", i+1)))
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			b.WriteString(add.Render(line))
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			b.WriteString(del.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(hunk.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
