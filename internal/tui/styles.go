package tui

import (
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var (
	bg      lipgloss.Color
	bgAlt   lipgloss.Color
	fg      lipgloss.Color
	dim     lipgloss.Color
	accent  lipgloss.Color
	green   lipgloss.Color
	red     lipgloss.Color
	yellow  lipgloss.Color
	magenta lipgloss.Color
	border  lipgloss.Color
	focusC  lipgloss.Color
	zones   = zone.New()
)

// themeName identifies one of the built-in color presets.
type themeName string

const (
	ThemeTokyoNight themeName = "Tokyo Night"
	ThemeDracula    themeName = "Dracula"
	ThemeLight      themeName = "Light"
)

// theme holds the full flat palette for one preset.
type theme struct {
	bg, bgAlt, fg, dim, accent, green, red, yellow, magenta, border, focusC lipgloss.Color
}

var themes = map[themeName]theme{
	ThemeTokyoNight: {
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
	ThemeDracula: {
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
	ThemeLight: {
		bg:      lipgloss.Color("#fafafa"),
		bgAlt:   lipgloss.Color("#eaeaea"),
		fg:      lipgloss.Color("#202020"),
		dim:     lipgloss.Color("#767676"),
		accent:  lipgloss.Color("#2563eb"),
		green:   lipgloss.Color("#15803d"),
		red:     lipgloss.Color("#b91c1c"),
		yellow:  lipgloss.Color("#a16207"),
		magenta: lipgloss.Color("#7e22ce"),
		border:  lipgloss.Color("#c7c7c7"),
		focusC:  lipgloss.Color("#2563eb"),
	},
}

// themeOrder fixes the cycle order used by the settings modal.
var themeOrder = []themeName{ThemeTokyoNight, ThemeDracula, ThemeLight}

// currentTheme tracks the active preset name.
var currentTheme = ThemeTokyoNight

// setTheme reassigns the package-level palette vars to the named preset.
// Unknown names are ignored (palette stays as-is).
func setTheme(name themeName) {
	t, ok := themes[name]
	if !ok {
		return
	}
	currentTheme = name
	bg, bgAlt, fg, dim = t.bg, t.bgAlt, t.fg, t.dim
	accent, green, red, yellow = t.accent, t.green, t.red, t.yellow
	magenta, border, focusC = t.magenta, t.border, t.focusC
}

func init() {
	setTheme(ThemeTokyoNight)
}

// nextTheme returns the preset name that follows the current one in
// themeOrder, wrapping around.
func nextTheme() themeName {
	for i, n := range themeOrder {
		if n == currentTheme {
			return themeOrder[(i+1)%len(themeOrder)]
		}
	}
	return themeOrder[0]
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
