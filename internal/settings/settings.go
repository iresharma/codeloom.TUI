package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user-tunable TUI preferences that persist across runs.
type Settings struct {
	Theme        string `json:"theme"`
	MouseEnabled bool   `json:"mouseEnabled"`
}

// Default returns the settings used when none are stored yet.
func Default() Settings {
	return Settings{Theme: "tokyo-night", MouseEnabled: true}
}

// Path returns the on-disk location of the settings file, creating its
// parent directory if necessary.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "codeloom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// Load reads settings from disk. Any problem reading or parsing the file
// (missing file, corrupt JSON, permission issues) falls back to Default()
// without surfacing an error, since settings are not correctness-critical
// and should never block TUI startup.
func Load() (Settings, error) {
	path, err := Path()
	if err != nil {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Default(), nil
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Default(), nil
	}
	return s, nil
}

// Save writes settings to disk as indented JSON.
func Save(s Settings) error {
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
