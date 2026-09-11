package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

func (m *Model) mergeAgents(live []agentRow) {
	sel := m.selectedGraphID()
	out := make([]agentRow, 0, len(live))
	for _, a := range live {
		if a.id == "" || agentDone(a.status) {
			continue
		}
		out = append(out, a)
	}
	m.agents = out
	ids := m.graphIDs()
	m.agentCursor = 0
	for i, id := range ids {
		if id == sel {
			m.agentCursor = i
			break
		}
	}
	if m.agentCursor >= len(ids) {
		m.agentCursor = max(0, len(ids)-1)
	}
	m.refreshGraph()
}

func (m *Model) dropAgent(id string) {
	if id == "" {
		return
	}
	out := m.agents[:0]
	for _, a := range m.agents {
		if a.id != id {
			out = append(out, a)
		}
	}
	m.agents = out
	if m.agentCursor >= len(m.graphIDs()) {
		m.agentCursor = max(0, len(m.graphIDs())-1)
	}
}

func agentDone(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "finished", "complete", "completed", "done", "failed", "error", "aborted", "cancelled", "canceled", "stopped", "success":
		return true
	}
	return false
}

func nodeOrder(agents []agentRow) []string {
	ids := map[string]struct{}{}
	kids := map[string][]string{}
	for _, a := range agents {
		if a.id == "" {
			continue
		}
		ids[a.id] = struct{}{}
	}
	for _, a := range agents {
		if a.id == "" {
			continue
		}
		p := a.parent
		if _, ok := ids[p]; !ok {
			p = ""
		}
		kids[p] = append(kids[p], a.id)
	}
	order := []string{""}
	seen := map[string]bool{"": true}
	var walk func(string)
	walk = func(p string) {
		for _, id := range kids[p] {
			if seen[id] {
				continue
			}
			seen[id] = true
			order = append(order, id)
			walk(id)
		}
	}
	walk("")
	for _, a := range agents {
		if a.id != "" && !seen[a.id] {
			seen[a.id] = true
			order = append(order, a.id)
			walk(a.id)
		}
	}
	return order
}

func rankAgents(agents []agentRow) [][]agentRow {
	byID := map[string]agentRow{}
	for _, a := range agents {
		if a.id != "" {
			byID[a.id] = a
		}
	}
	depth := map[string]int{}
	var dep func(string) int
	dep = func(id string) int {
		if d, ok := depth[id]; ok {
			return d
		}
		a, ok := byID[id]
		if !ok {
			return 0
		}
		if a.parent == "" || a.parent == id {
			depth[id] = 0
			return 0
		}
		if _, ok := byID[a.parent]; !ok {
			depth[id] = 0
			return 0
		}
		depth[id] = -1 // cycle guard
		d := dep(a.parent) + 1
		depth[id] = d
		return d
	}
	maxd := -1
	for _, a := range agents {
		if a.id == "" {
			continue
		}
		d := dep(a.id)
		if d > maxd {
			maxd = d
		}
	}
	if maxd < 0 {
		return nil
	}
	ranks := make([][]agentRow, maxd+1)
	for _, a := range agents {
		if a.id == "" {
			continue
		}
		ranks[depth[a.id]] = append(ranks[depth[a.id]], a)
	}
	return ranks
}

func (m *Model) graphIDs() []string {
	return nodeOrder(m.agents)
}

func (m *Model) selectedGraphID() string {
	ids := m.graphIDs()
	if m.agentCursor < 0 || m.agentCursor >= len(ids) {
		return ""
	}
	return ids[m.agentCursor]
}

func (m *Model) selectedAgent() string {
	id := m.selectedGraphID()
	if id == "" {
		return ""
	}
	return id
}

func (m *Model) moveGraphCursor(delta int) {
	ids := m.graphIDs()
	if len(ids) == 0 {
		return
	}
	m.agentCursor += delta
	if m.agentCursor < 0 {
		m.agentCursor = 0
	}
	if m.agentCursor >= len(ids) {
		m.agentCursor = len(ids) - 1
	}
	m.refreshGraph()
}

func (m *Model) peekSelectedAgent() tea.Cmd {
	id := m.selectedGraphID()
	m.chatFilter = id
	m.chatFollow = true
	m.refreshChat()
	if id == "" {
		return nil
	}
	return m.send(protocol.RequestAgentTranscript(id))
}

func (m *Model) peekAgent(id string) tea.Cmd {
	m.chatFilter = id
	m.chatFollow = true
	m.refreshChat()
	for i, gid := range m.graphIDs() {
		if gid == id {
			m.agentCursor = i
			break
		}
	}
	m.refreshGraph()
	if id == "" {
		return nil
	}
	return m.send(protocol.RequestAgentTranscript(id))
}

func (m *Model) refreshGraph() {
	w := m.graphVP.Width
	if w < 12 {
		w = 12
	}
	spin := ""
	if m.busy() {
		spin = m.spin.View()
	}
	m.graphVP.SetContent(renderDAG(w, m.orchState, m.agents, m.agentCursor, spin))
}

func (m Model) busy() bool {
	st := strings.ToLower(m.orchState)
	if st == "thinking" || st == "running" || st == "tool" {
		return true
	}
	if len(m.streams) > 0 || len(m.streamAgent) > 0 {
		return true
	}
	for _, a := range m.agents {
		s := strings.ToLower(a.status)
		if s == "thinking" || s == "running" || s == "tool" {
			return true
		}
	}
	return false
}

func renderDAG(w int, orchState string, agents []agentRow, cursor int, spin string) string {
	if w < 12 {
		w = 12
	}
	agents = liveAgents(agents)
	order := nodeOrder(agents)
	sel := ""
	if cursor >= 0 && cursor < len(order) {
		sel = order[cursor]
	}
	ranks := rankAgents(agents)
	fitN := boxesPerRow(w)
	for _, r := range ranks {
		if len(r) > fitN {
			return renderDAGTree(w, orchState, agents, sel, spin)
		}
	}

	maxN := 1
	for _, r := range ranks {
		if len(r) > maxN {
			maxN = len(r)
		}
	}
	bw := boxWidth(w, maxN)

	var b strings.Builder
	orchX := max(0, (w-bw)/2)
	sub := orchState
	if sub == "" {
		sub = "idle"
	}
	if spin != "" && (sub == "thinking" || sub == "running" || sub == "tool") {
		sub = strings.TrimSpace(spin + " " + sub)
	}
	b.WriteString(placeBox(zones.Mark("agent-", nodeBox("orchestrator", sub, bw, sel == "", false)), orchX, w))
	b.WriteByte('\n')

	if len(ranks) == 0 {
		b.WriteString(dimStyle().Render(padCenter("no children", w)))
		return b.String()
	}

	orchMid := orchX + bw/2
	xs0 := boxXs(len(ranks[0]), bw, w)
	b.WriteString(connectDown(w, orchMid, mids(xs0, bw)))
	b.WriteByte('\n')

	for ri, rank := range ranks {
		xs := boxXs(len(rank), bw, w)
		if ri == 0 {
			xs = xs0
		}
		b.WriteString(joinBoxes(rank, xs, bw, sel))
		b.WriteByte('\n')
		if task := selectedTask(rank, sel); task != "" {
			b.WriteString(dimStyle().Render(clip("  "+task, w)) + "\n")
		}
		if ri+1 < len(ranks) {
			nxs := boxXs(len(ranks[ri+1]), bw, w)
			b.WriteString(connectForest(w, rank, xs, bw, ranks[ri+1], nxs))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func liveAgents(agents []agentRow) []agentRow {
	out := make([]agentRow, 0, len(agents))
	for _, a := range agents {
		if a.id == "" || agentDone(a.status) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func boxesPerRow(w int) int {
	const minBox = 8
	n := (w + 1) / (minBox + 1)
	if n < 1 {
		return 1
	}
	return n
}

func boxWidth(w, n int) int {
	if n < 1 {
		n = 1
	}
	bw := min(22, max(10, (w-n)/n))
	if n*bw+(n-1) > w {
		bw = max(8, (w-(n-1))/n)
	}
	return bw
}

func renderDAGTree(w int, orchState string, agents []agentRow, sel, spin string) string {
	bw := min(28, max(12, w-4))
	sub := orchState
	if sub == "" {
		sub = "idle"
	}
	if spin != "" && (sub == "thinking" || sub == "running" || sub == "tool") {
		sub = strings.TrimSpace(spin + " " + sub)
	}
	kids := map[string][]string{}
	byID := map[string]agentRow{}
	ids := map[string]struct{}{}
	for _, a := range agents {
		if a.id == "" {
			continue
		}
		byID[a.id] = a
		ids[a.id] = struct{}{}
	}
	for _, a := range agents {
		p := a.parent
		if _, ok := ids[p]; !ok {
			p = ""
		}
		kids[p] = append(kids[p], a.id)
	}
	var b strings.Builder
	b.WriteString(zones.Mark("agent-", nodeBox("orchestrator", sub, bw, sel == "", false)))
	roots := kids[""]
	for i, id := range roots {
		b.WriteByte('\n')
		writeTreeNode(&b, id, byID, kids, nil, i == len(roots)-1, bw, sel)
	}
	if len(roots) == 0 {
		b.WriteByte('\n')
		b.WriteString(dimStyle().Render("no children"))
	}
	return b.String()
}

func writeTreeNode(b *strings.Builder, id string, byID map[string]agentRow, kids map[string][]string, spines []bool, last bool, bw int, sel string) {
	a := byID[id]
	st := a.status
	if st == "" {
		st = "idle"
	}
	sub := st
	if a.tool != "" {
		sub = st + " · " + a.tool
	}
	box := zones.Mark("agent-"+a.id, nodeBox(a.profile, sub, bw, a.id == sel, false))
	first, rest := treePrefix(spines, last)
	b.WriteString(stampBox(box, first, rest))
	if a.id == sel && a.task != "" {
		b.WriteByte('\n')
		b.WriteString(rest + dimStyle().Render(clip(a.task, max(8, bw))))
	}
	children := kids[id]
	childSpines := append(append([]bool{}, spines...), !last)
	for i, cid := range children {
		b.WriteByte('\n')
		writeTreeNode(b, cid, byID, kids, childSpines, i == len(children)-1, bw, sel)
	}
}

func treePrefix(spines []bool, last bool) (first, rest string) {
	var f, r strings.Builder
	for _, cont := range spines {
		if cont {
			f.WriteString("│ ")
			r.WriteString("│ ")
		} else {
			f.WriteString("  ")
			r.WriteString("  ")
		}
	}
	if last {
		f.WriteString("└▼")
		r.WriteString("  ")
	} else {
		f.WriteString("├▼")
		r.WriteString("│ ")
	}
	return f.String(), r.String()
}

func stampBox(box, first, rest string) string {
	lines := strings.Split(box, "\n")
	for i, line := range lines {
		if i == 0 {
			lines[i] = first + line
		} else {
			lines[i] = rest + line
		}
	}
	return strings.Join(lines, "\n")
}

func selectedTask(rank []agentRow, sel string) string {
	for _, a := range rank {
		if a.id == sel {
			return a.task
		}
	}
	return ""
}

func joinBoxes(rank []agentRow, xs []int, bw int, sel string) string {
	var parts []string
	prev := 0
	for i, a := range rank {
		st := a.status
		if st == "" {
			st = "idle"
		}
		sub := st
		if a.tool != "" {
			sub = st + " · " + a.tool
		}
		box := zones.Mark("agent-"+a.id, nodeBox(a.profile, sub, bw, a.id == sel, a.status == "finished"))
		x := 0
		if i < len(xs) {
			x = xs[i]
		}
		if x > prev {
			parts = append(parts, vspace(x-prev, 4))
		}
		parts = append(parts, box)
		prev = x + bw
	}
	if len(parts) == 0 {
		return ""
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func nodeBox(title, sub string, w int, selected, dimmed bool) string {
	inner := max(1, w-2)
	t := clip(title, inner)
	s := clip(sub, inner)
	pad := func(v string) string {
		extra := inner - lipgloss.Width(v)
		if extra < 0 {
			extra = 0
		}
		return v + strings.Repeat(" ", extra)
	}
	top := "╭" + strings.Repeat("─", inner) + "╮"
	mid1 := "│" + pad(t) + "│"
	mid2 := "│" + pad(s) + "│"
	bot := "╰" + strings.Repeat("─", inner) + "╯"
	body := top + "\n" + mid1 + "\n" + mid2 + "\n" + bot
	st := lipgloss.NewStyle().Foreground(fg)
	if dimmed {
		st = dimStyle()
	}
	if selected {
		st = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
	return st.Render(body)
}

func boxXs(n, bw, w int) []int {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []int{max(0, (w-bw)/2)}
	}
	total := n * bw
	gap := (w - total) / (n + 1)
	if gap < 1 {
		bw = max(6, (w-(n+1))/n)
		total = n * bw
		gap = max(1, (w-total)/(n+1))
	}
	xs := make([]int, n)
	x := gap
	for i := 0; i < n; i++ {
		xs[i] = min(x, max(0, w-bw))
		x += bw + gap
	}
	return xs
}

func mids(xs []int, bw int) []int {
	out := make([]int, len(xs))
	for i, x := range xs {
		out[i] = x + bw/2
	}
	return out
}

func placeBox(box string, x, w int) string {
	if x <= 0 {
		return box
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, vspace(x, 4), box)
}

func vspace(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	line := strings.Repeat(" ", width)
	return strings.Repeat(line+"\n", height-1) + line
}

func padCenter(s string, w int) string {
	return lipgloss.PlaceHorizontal(max(1, w), lipgloss.Center, s)
}

func connectDown(w, parentMid int, childMids []int) string {
	if len(childMids) == 0 {
		return ""
	}
	return overlayFork(w, parentMid, childMids)
}

func connectForest(w int, parents []agentRow, pxs []int, bw int, kids []agentRow, kxs []int) string {
	grouped := map[string][]int{}
	for i, k := range kids {
		mid := bw / 2
		if i < len(kxs) {
			mid = kxs[i] + bw/2
		}
		grouped[k.parent] = append(grouped[k.parent], mid)
	}
	stem := spaces(w)
	bar := spaces(w)
	arrows := spaces(w)
	for i, p := range parents {
		mids := grouped[p.id]
		if len(mids) == 0 {
			continue
		}
		pm := bw / 2
		if i < len(pxs) {
			pm = pxs[i] + bw/2
		}
		mergeFork(stem, bar, arrows, pm, mids)
	}
	return string(stem) + "\n" + string(bar) + "\n" + string(arrows)
}

func overlayFork(w, parentMid int, childMids []int) string {
	stem := spaces(w)
	bar := spaces(w)
	arrows := spaces(w)
	mergeFork(stem, bar, arrows, parentMid, childMids)
	return string(stem) + "\n" + string(bar) + "\n" + string(arrows)
}

func mergeFork(stem, bar, arrows []rune, parentMid int, childMids []int) {
	w := len(stem)
	put := func(row []rune, x int, r rune) {
		if x >= 0 && x < w {
			row[x] = r
		}
	}
	put(stem, parentMid, '│')
	if len(childMids) == 1 && childMids[0] == parentMid {
		put(arrows, childMids[0], '▼')
		return
	}
	lo, hi := parentMid, parentMid
	for _, m := range childMids {
		if m < lo {
			lo = m
		}
		if m > hi {
			hi = m
		}
	}
	for x := lo; x <= hi && x < w; x++ {
		if bar[x] == ' ' {
			bar[x] = '─'
		}
	}
	put(bar, lo, '┌')
	put(bar, hi, '┐')
	for _, m := range childMids {
		switch {
		case m == lo:
			put(bar, m, '┌')
		case m == hi:
			put(bar, m, '┐')
		default:
			put(bar, m, '┬')
		}
		put(arrows, m, '▼')
	}
	if parentMid >= 0 && parentMid < w {
		switch bar[parentMid] {
		case '─':
			bar[parentMid] = '┴'
		case '┬':
			bar[parentMid] = '┼'
		case '┌':
			bar[parentMid] = '├'
		case '┐':
			bar[parentMid] = '┤'
		case ' ':
			bar[parentMid] = '│'
		}
	}
}

func spaces(n int) []rune {
	r := make([]rune, max(1, n))
	for i := range r {
		r[i] = ' '
	}
	return r
}

func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

func (m Model) viewGraphPane(w, h int) string {
	title := headerBar(fmt.Sprintf("agents  %d", len(m.agents)), w)
	head := m.viewSession(w)
	rest := max(1, h-lipgloss.Height(title)-lipgloss.Height(head))
	return lipgloss.JoinVertical(lipgloss.Top, title, head, fit(m.graphVP.View(), w, rest))
}
