package collect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State holds per-user persistent state
type State struct {
	LastLogin          time.Time            `json:"last_login"`
	LastNewsgroupHigh  map[string]int       `json:"last_newsgroup_high"`
	WeatherLocation    string               `json:"weather_location"`
	PreferredTab       int                  `json:"preferred_tab"`
}

// LoadState reads state from ~/.cache/cosmic-dashboard/state.json
func LoadState() *State {
	s := &State{
		LastNewsgroupHigh: make(map[string]int),
	}
	path := statePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	json.Unmarshal(data, s)
	if s.LastNewsgroupHigh == nil {
		s.LastNewsgroupHigh = make(map[string]int)
	}
	return s
}

// SaveState writes state to disk
func (s *State) SaveState() error {
	path := statePath()
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0700)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func statePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/cosmic-dashboard-state.json"
	}
	return filepath.Join(home, ".cache", "cosmic-dashboard", "state.json")
}
