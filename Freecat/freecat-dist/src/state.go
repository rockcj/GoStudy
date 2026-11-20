package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// State is persisted in the user's config directory (or FREECAT_HOME).
type State struct {
	Email      string `json:"email,omitempty"`
	Token      string `json:"token,omitempty"`
	AuthData   string `json:"auth_data,omitempty"`
	Host       string `json:"host,omitempty"`
	UpdatedAt  int64  `json:"updated_at,omitempty"`
	Traffic    string `json:"traffic,omitempty"`
	Port       int    `json:"port,omitempty"`
	Controller string `json:"controller,omitempty"`
	Secret     string `json:"secret,omitempty"`
	TUN        bool   `json:"tun,omitempty"`
	ServerHost string `json:"server_host,omitempty"`
	// ServerIP is the legacy field used by 0.1.0; kept for migration.
	ServerIP string `json:"server_ip,omitempty"`
}

// endpoint returns the host written into every proxy's `server:` field.
// "keep" disables the rewrite and keeps the subscription's own domains.
func (s *State) endpoint() string {
	h := s.ServerHost
	if h == "" {
		// migrate the 0.1.0 default IP to the DNS endpoint
		if s.ServerIP != "" && s.ServerIP != "34.92.4.52" {
			h = s.ServerIP
		} else {
			h = defaultServerHost
		}
	}
	if h == "keep" || h == "none" {
		return ""
	}
	return h
}

func configDir() string {
	if h := os.Getenv("FREECAT_HOME"); h != "" {
		_ = os.MkdirAll(h, 0o700)
		return h
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	d := filepath.Join(base, "freecat")
	_ = os.MkdirAll(d, 0o700)
	return d
}

func statePath() string   { return filepath.Join(configDir(), "state.json") }
func profilePath() string { return filepath.Join(configDir(), "profile.yaml") }
func runtimePath() string { return filepath.Join(configDir(), "config.yaml") }
func coreDir() string     { return configDir() }

func loadState() (*State, error) {
	b, err := os.ReadFile(statePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("state.json 损坏: %w", err)
	}
	return &s, nil
}

func saveState(s *State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := statePath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, statePath())
}

func writeFile0600(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
