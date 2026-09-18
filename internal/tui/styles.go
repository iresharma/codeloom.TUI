package tui

import (
	"sort"

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

// theme bundles the palette used across the TUI so it can be swapped at
// runtime from the settings modal.
type theme struct {
	bg, bgAlt, fg, dim, accent          lipgloss.Color
	green, red, yellow, magenta, border lipgloss.Color
	focusC                              lipgloss.Color
}

// activeThemeName tracks the currently applied theme, persisted via
// internal/tui/config.go.
var activeThemeName = "default"

var themes = map[string]theme{
	// default mirrors the original hardcoded palette above so switching
	// themes and switching back is a no-op visually.
	"default": {
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
	"dracula": {
		bg:      lipgloss.Color("#282a36"),
		bgAlt:   lipgloss.Color("#21222c"),
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
	"solarized-dark": {
		bg:      lipgloss.Color("#002b36"),
		bgAlt:   lipgloss.Color("#073642"),
		fg:      lipgloss.Color("#eee8d5"),
		dim:     lipgloss.Color("#586e75"),
		accent:  lipgloss.Color("#268bd2"),
		green:   lipgloss.Color("#859900"),
		red:     lipgloss.Color("#dc322f"),
		yellow:  lipgloss.Color("#b58900"),
		magenta: lipgloss.Color("#d33682"),
		border:  lipgloss.Color("#073642"),
		focusC:  lipgloss.Color("#2aa198"),
	},
}

// applyTheme reassigns the package-level palette vars from a named theme
// preset. It returns false if the name is unknown, leaving the current
// palette untouched.
func applyTheme(name string) bool {
	t, ok := themes[name]
	if !ok {
		return false
	}
	bg, bgAlt, fg, dim, accent = t.bg, t.bgAlt, t.fg, t.dim, t.accent
	green, red, yellow, magenta, border, focusC = t.green, t.red, t.yellow, t.magenta, t.border, t.focusC
	activeThemeName = name
	return true
}

// themeNames returns the available theme presets in sorted order, for
// listing in the settings modal.
func themeNames() []string {
	names := make([]string, 0, len(themes))
	for n := range themes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
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
