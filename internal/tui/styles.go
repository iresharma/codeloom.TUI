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

type themePalette struct {
	bg, bgAlt, fg, dim, accent, green, red, yellow, magenta, border, focusC lipgloss.Color
}

var themes = map[string]themePalette{
	"tokyo-night": {
		bg:      lipgloss.Color("#1a1b26"),
		bgAlt:   lipgloss.Color("#16161e"),
		fg:      lipgloss.Color("#c0caf5"),
		dim:     lipgloss.Color("#565f89"),
		accent:  lipgloss.Color("#7aa2f7"),
		green:   lipgloss.Color("#9ece6a"),
		red:     lipgloss.Color("#f7768e"),
		yellow:  lipgloss.Color("#e0af68"),
		magenta: lipgloss.Color("#bb9af7"),
		border:  lipgloss.Color("#3b4261"),
		focusC:  lipgloss.Color("#7aa2f7"),
	},
	"light": {
		bg:      lipgloss.Color("#f4f4f5"),
		bgAlt:   lipgloss.Color("#e4e4e7"),
		fg:      lipgloss.Color("#1f2430"),
		dim:     lipgloss.Color("#8a8f98"),
		accent:  lipgloss.Color("#2563eb"),
		green:   lipgloss.Color("#15803d"),
		red:     lipgloss.Color("#dc2626"),
		yellow:  lipgloss.Color("#b45309"),
		magenta: lipgloss.Color("#7e22ce"),
		border:  lipgloss.Color("#c4c7cc"),
		focusC:  lipgloss.Color("#2563eb"),
	},
	"dracula": {
		bg:      lipgloss.Color("#282a36"),
		bgAlt:   lipgloss.Color("#1e1f29"),
		fg:      lipgloss.Color("#f8f8f2"),
		dim:     lipgloss.Color("#6272a4"),
		accent:  lipgloss.Color("#bd93f9"),
		green:   lipgloss.Color("#50fa7b"),
		red:     lipgloss.Color("#ff5555"),
		yellow:  lipgloss.Color("#f1fa8c"),
		magenta: lipgloss.Color("#ff79c6"),
		border:  lipgloss.Color("#44475a"),
		focusC:  lipgloss.Color("#bd93f9"),
	},
}

var themeOrder = []string{"tokyo-night", "light", "dracula"}

var currentThemeName = "tokyo-night"

func applyTheme(name string) {
	p, ok := themes[name]
	if !ok {
		return
	}
	bg, bgAlt, fg, dim, accent, green, red, yellow, magenta, border, focusC =
		p.bg, p.bgAlt, p.fg, p.dim, p.accent, p.green, p.red, p.yellow, p.magenta, p.border, p.focusC
	currentThemeName = name
}

// ApplyTheme is the exported entry point for setting the active theme from
// outside the tui package (e.g. cmd/codeloom-tui/main.go on startup).
func ApplyTheme(name string) {
	applyTheme(name)
}

// glamourStyleForTheme picks a glamour renderer style that matches the
// currently active theme.
func glamourStyleForTheme() string {
	if currentThemeName == "light" {
		return "light"
	}
	return "dark"
}

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

func sectionBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(border)
}
