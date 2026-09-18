package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// persistedSettings is the on-disk JSON shape for user TUI preferences.
type persistedSettings struct {
	Theme           string `json:"theme"`
	ShowLineNumbers bool   `json:"show_line_numbers"`
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "codeloom-tui", "settings.json"), nil
}

// loadSettings reads persisted settings from disk. Callers should fall
// back to defaults silently if it returns an error.
func loadSettings() (persistedSettings, error) {
	var s persistedSettings
	path, err := settingsPath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

// saveSettings writes settings to disk, creating the config directory if
// needed. Errors are returned for the caller to ignore/log; they must not
// crash the UI.
func saveSettings(s persistedSettings) error {
	path, err := settingsPath()
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

// currentSettings snapshots the model's current settings for persisting.
func (m Model) currentSettings() persistedSettings {
	return persistedSettings{
		Theme:           currentTheme.name,
		ShowLineNumbers: m.showLineNumbers,
	}
}
