package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the persisted user configuration for the TUI. It is loaded on
// startup, mutated by the settings modal, and saved back to disk whenever
// the modal is dismissed.
type Config struct {
	Theme        string `json:"theme"`
	MouseEnabled bool   `json:"mouse_enabled"`
}

// defaultConfig returns the config used when no file exists yet.
func defaultConfig() Config {
	return Config{
		Theme:        string(ThemeTokyoNight),
		MouseEnabled: true,
	}
}

// configPath returns the on-disk location of the config file, creating its
// parent directory if needed. It falls back to a relative path if the user
// config dir cannot be determined.
func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "codeloom", "config.json")
}

// LoadConfig reads the config file from disk, returning defaults if it does
// not exist or cannot be parsed.
func LoadConfig() Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return cfg
	}
	if loaded.Theme == "" {
		loaded.Theme = cfg.Theme
	}
	return loaded
}

// Save writes the config to disk as JSON, creating the parent directory if
// needed.
func (c Config) Save() error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
