package tui

import (
	"fmt"
	"strings"
)

func (m *Model) refreshFileVP() {
	t := m.currentTab()
	if t == nil || t.kind != tabFile {
		return
	}
	body := t.body
	if t.showDiff && t.diff != "" {
		body = t.diff
	}
	var b strings.Builder
	for i, line := range strings.Split(body, "\n") {
		b.WriteString(fmt.Sprintf("%4d │ %s\n", i+1, line))
	}
	m.fileVP.SetContent(b.String())
}
