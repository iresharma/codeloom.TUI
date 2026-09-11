package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewMemory() string {
	var b strings.Builder
	for _, f := range m.memFiles {
		flag := "fresh"
		if f.stale {
			flag = "STALE"
		}
		line := fmt.Sprintf("· %s [%s]\n  %s", f.path, flag, f.purpose)
		b.WriteString(zones.Mark("mem-"+f.path, line) + "\n")
	}
	writeDec := func(title string, items []memDec) {
		if len(items) == 0 {
			return
		}
		b.WriteString(titleStyle().Render(title) + "\n")
		for _, d := range items {
			b.WriteString("  - " + d.text + "\n")
		}
	}
	writeDec("engineering", m.memEng)
	writeDec("product", m.memProd)
	writeDec("ci/cd", m.memCICD)
	writeDec("other", m.memOther)
	if len(m.memFiles) == 0 && len(m.memEng)+len(m.memProd)+len(m.memCICD)+len(m.memOther) == 0 {
		b.WriteString(dimStyle().Render("(empty memory)"))
	}
	return b.String()
}
