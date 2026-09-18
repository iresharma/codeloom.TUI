package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings is the persisted TUI preference file, stored as JSON under the
// user's config directory (see configPath).
type Settings struct {
	Theme           string `json:"theme"`
	MouseEnabled    bool   `json:"mouse_enabled"`
	FullHelpDefault bool   `json:"full_help_default"`
}

func defaultSettings() Settings {
	return Settings{
		Theme:           "default",
		MouseEnabled:    true,
		FullHelpDefault: false,
	}
}

// configPath returns the path to the persisted settings file, creating the
// containing directory if it doesn't exist.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "codeloom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "tui-settings.json"), nil
}

// loadSettings reads persisted settings, falling back to defaults if the
// file is missing, unreadable, or unparseable. It never errors out the app.
func loadSettings() Settings {
	def := defaultSettings()
	path, err := configPath()
	if err != nil {
		return def
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return def
	}
	if s.Theme == "" {
		s.Theme = def.Theme
	}
	return s
}

// saveSettings writes settings as indented JSON to configPath(), creating
// directories as needed.
func saveSettings(s Settings) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
