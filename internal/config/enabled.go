package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Enabled struct {
	Plugins []string `json:"plugins"`
	Themes  []string `json:"themes"`
}

func enabledPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pmusic", "enabled.json"), nil
}

func LoadEnabled() (Enabled, error) {
	p, err := enabledPath()
	if err != nil {
		return Enabled{}, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Enabled{}, nil
	}
	if err != nil {
		return Enabled{}, err
	}
	var e Enabled
	return e, json.Unmarshal(data, &e)
}

func SaveEnabled(e Enabled) error {
	p, err := enabledPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func (e *Enabled) Has(kind, name string) bool {
	list := e.Plugins
	if kind == "theme" {
		list = e.Themes
	}
	for _, n := range list {
		if n == name {
			return true
		}
	}
	return false
}

func (e *Enabled) Toggle(kind, name string) {
	if kind == "theme" {
		e.Themes = toggleSlice(e.Themes, name)
	} else {
		e.Plugins = toggleSlice(e.Plugins, name)
	}
}

func toggleSlice(s []string, name string) []string {
	for i, n := range s {
		if n == name {
			return append(s[:i], s[i+1:]...)
		}
	}
	return append(s, name)
}
