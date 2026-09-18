package tui

import (
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var (
	bg      = lipgloss.Color("#1a1b26")
	bgAlt   = lipgloss.Color("#16161e")
	fg      = lipgloss.Color("#c0caf5")
	dim     = lipgloss.Color("#565f89")
	accent  = lipgloss.Color("#7aa2f7")
	green   = lipgloss.Color("#9ece6a")
	red     = lipgloss.Color("#f7768e")
	yellow  = lipgloss.Color("#e0af68")
	magenta = lipgloss.Color("#bb9af7")
	border  = lipgloss.Color("#3b4261")
	focusC  = lipgloss.Color("#7aa2f7")
	zones   = zone.New()
)

func paneStyle(focused bool, w, h int) lipgloss.Style {
	c := border
	if focused {
		c = focusC
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c).
		Width(max(1, w-2)).
		Height(max(1, h-2)).
		Foreground(fg).
		Background(bg)
}

func titleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(accent).Bold(true)
}

func dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(dim)
}

func gitMod() lipgloss.Style       { return lipgloss.NewStyle().Foreground(yellow) }
func gitAdd() lipgloss.Style       { return lipgloss.NewStyle().Foreground(green) }
func gitUntracked() lipgloss.Style { return lipgloss.NewStyle().Foreground(red) }

func headerBar(text string, w int) string {
	return lipgloss.NewStyle().
		Foreground(magenta).
		Bold(true).
		Background(bgAlt).
		Width(max(1, w)).
		Render(" " + text)
}

func fit(s string, w, h int) string {
	return lipgloss.NewStyle().
		Width(max(1, w)).
		Height(max(1, h)).
		MaxWidth(max(1, w)).
		MaxHeight(max(1, h)).
		Background(bg).
		Foreground(fg).
		Render(s)
}

// applyTheme copies the given theme's colors into the package-level color
// vars above so every existing call site (chat.go, tabs.go, graph.go,
// layout.go, context.go, fileview.go, stats.go, app.go, ...) picks up the
// change without needing to be touched.
func applyTheme(t theme) {
	bg = t.bg
	bgAlt = t.bgAlt
	fg = t.fg
	dim = t.dim
	accent = t.accent
	green = t.green
	red = t.red
	yellow = t.yellow
	magenta = t.magenta
	border = t.border
	focusC = t.focusC
}

func sectionBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(border)
}
