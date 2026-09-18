package tui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds a full color palette that can be swapped into the package-level
// style vars defined in styles.go.
type Theme struct {
	Name                                           string
	Bg, BgAlt, Fg, Dim, Accent, Green, Red, Yellow lipgloss.Color
	Magenta, Border, FocusC                        lipgloss.Color
}

var themeTokyoNight = Theme{
	Name:    "TokyoNight",
	Bg:      lipgloss.Color("#1a1b26"),
	BgAlt:   lipgloss.Color("#16161e"),
	Fg:      lipgloss.Color("#c0caf5"),
	Dim:     lipgloss.Color("#565f89"),
	Accent:  lipgloss.Color("#7aa2f7"),
	Green:   lipgloss.Color("#9ece6a"),
	Red:     lipgloss.Color("#f7768e"),
	Yellow:  lipgloss.Color("#e0af68"),
	Magenta: lipgloss.Color("#bb9af7"),
	Border:  lipgloss.Color("#3b4261"),
	FocusC:  lipgloss.Color("#7aa2f7"),
}

var themeDracula = Theme{
	Name:    "Dracula",
	Bg:      lipgloss.Color("#282a36"),
	BgAlt:   lipgloss.Color("#21222c"),
	Fg:      lipgloss.Color("#f8f8f2"),
	Dim:     lipgloss.Color("#6272a4"),
	Accent:  lipgloss.Color("#bd93f9"),
	Green:   lipgloss.Color("#50fa7b"),
	Red:     lipgloss.Color("#ff5555"),
	Yellow:  lipgloss.Color("#f1fa8c"),
	Magenta: lipgloss.Color("#ff79c6"),
	Border:  lipgloss.Color("#44475a"),
	FocusC:  lipgloss.Color("#bd93f9"),
}

var themeLight = Theme{
	Name:    "Light",
	Bg:      lipgloss.Color("#fafafa"),
	BgAlt:   lipgloss.Color("#eeeeee"),
	Fg:      lipgloss.Color("#24292e"),
	Dim:     lipgloss.Color("#6a737d"),
	Accent:  lipgloss.Color("#005cc5"),
	Green:   lipgloss.Color("#22863a"),
	Red:     lipgloss.Color("#d73a49"),
	Yellow:  lipgloss.Color("#b08800"),
	Magenta: lipgloss.Color("#8250df"),
	Border:  lipgloss.Color("#d1d5da"),
	FocusC:  lipgloss.Color("#005cc5"),
}

var themes = []Theme{themeTokyoNight, themeDracula, themeLight}

// findTheme returns the theme matching name, or themeTokyoNight if not found.
func findTheme(name string) Theme {
	for _, t := range themes {
		if t.Name == name {
			return t
		}
	}
	return themeTokyoNight
}

// applyTheme mutates the package-level color vars in styles.go so all style
// helper funcs pick up the new palette immediately.
func applyTheme(t Theme) {
	bg = t.Bg
	bgAlt = t.BgAlt
	fg = t.Fg
	dim = t.Dim
	accent = t.Accent
	green = t.Green
	red = t.Red
	yellow = t.Yellow
	magenta = t.Magenta
	border = t.Border
	focusC = t.FocusC
}

// AppSettings is persisted user-configurable state.
type AppSettings struct {
	ThemeName     string `json:"theme"`
	ConfirmDelete bool   `json:"confirm_delete"`
}

func defaultSettings() AppSettings {
	return AppSettings{
		ThemeName:     themeTokyoNight.Name,
		ConfirmDelete: true,
	}
}

func settingsFilePath() (string, error) {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "codeloom-tui", "settings.json"), nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".codeloom-tui", "settings.json"), nil
}

func loadSettings() AppSettings {
	s := defaultSettings()
	path, err := settingsFilePath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return defaultSettings()
	}
	return s
}

func (s AppSettings) save() error {
	path, err := settingsFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// settingsModal holds transient UI state while the settings modal is open.
type settingsModal struct {
	cursor int
}
