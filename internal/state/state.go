package state

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// State represents the persisted UI state
type State struct {
	// Last selected instance
	SelectedInstance string `yaml:"selected_instance,omitempty"`

	// Last active tab
	ActiveTab int `yaml:"active_tab"`

	// Last focused pane (0 = instances, 1 = details)
	FocusedPane int `yaml:"focused_pane"`

	// Log filter if any
	LogFilter string `yaml:"log_filter,omitempty"`

	// Log follow mode
	LogFollow bool `yaml:"log_follow"`

	// Window size (for restoration)
	WindowWidth  int `yaml:"window_width,omitempty"`
	WindowHeight int `yaml:"window_height,omitempty"`
}

// DefaultState returns a new state with default values
func DefaultState() *State {
	return &State{
		ActiveTab:   0,
		FocusedPane: 0,
		LogFollow:   true,
	}
}

// StatePath returns the full path to the state file
func StatePath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "lazyclaw", "state.yml"), nil
}

// Load loads the state from disk
func Load() (*State, error) {
	path, err := StatePath()
	if err != nil {
		return DefaultState(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultState(), nil
		}
		return DefaultState(), err
	}

	state := DefaultState()
	if err := yaml.Unmarshal(data, state); err != nil {
		return DefaultState(), err
	}

	return state, nil
}

// Save writes the state to disk atomically
func Save(state *State) error {
	path, err := StatePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	// Write atomically: write to temp file, then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
