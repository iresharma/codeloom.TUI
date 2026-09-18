package tui

import "github.com/charmbracelet/lipgloss"

// theme holds one named color palette. Fields mirror the package-level
// color vars in styles.go.
type theme struct {
	name                                                    string
	bg, bgAlt, fg, dim, accent, green, red, yellow, magenta lipgloss.Color
	border, focusC                                          lipgloss.Color
}

// themes lists the built-in presets. Index 0 (Tokyo Night) is the default
// and matches the hardcoded values styles.go used before theming existed.
var themes = []theme{
	{
		name:    "Tokyo Night",
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
	{
		name:    "Dracula",
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
	{
		name:    "Solarized Dark",
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
	{
		name:    "Gruvbox",
		bg:      lipgloss.Color("#282828"),
		bgAlt:   lipgloss.Color("#1d2021"),
		fg:      lipgloss.Color("#ebdbb2"),
		dim:     lipgloss.Color("#928374"),
		accent:  lipgloss.Color("#d79921"),
		green:   lipgloss.Color("#98971a"),
		red:     lipgloss.Color("#cc241d"),
		yellow:  lipgloss.Color("#d79921"),
		magenta: lipgloss.Color("#b16286"),
		border:  lipgloss.Color("#504945"),
		focusC:  lipgloss.Color("#fabd2f"),
	},
}

// currentTheme is the active palette. It starts as Tokyo Night and is
// reassigned (and mirrored into the package color vars via applyTheme) by
// setTheme / setThemeByIndex.
var currentTheme = themes[0]

func themeNames() []string {
	names := make([]string, len(themes))
	for i, t := range themes {
		names[i] = t.name
	}
	return names
}

func themeIndex(name string) int {
	for i, t := range themes {
		if t.name == name {
			return i
		}
	}
	return -1
}

// setThemeByIndex activates the theme at idx (wrapping into range) and
// applies it to the package-level color vars.
func setThemeByIndex(idx int) {
	n := len(themes)
	if n == 0 {
		return
	}
	idx = ((idx % n) + n) % n
	currentTheme = themes[idx]
	applyTheme(currentTheme)
}

// setTheme activates the named theme, falling back to Tokyo Night if the
// name is unknown.
func setTheme(name string) {
	if idx := themeIndex(name); idx >= 0 {
		setThemeByIndex(idx)
		return
	}
	setThemeByIndex(0)
}
