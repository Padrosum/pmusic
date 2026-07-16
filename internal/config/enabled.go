package config

import (
	"os"
	"path/filepath"

	"github.com/Padrosum/pmusic/internal/persistence"
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
	e, found, err := persistence.LoadJSON[Enabled](p, nil)
	if err != nil {
		return Enabled{}, err
	}
	if !found {
		return Enabled{}, nil
	}
	return e, nil
}

func SaveEnabled(e Enabled) error {
	p, err := enabledPath()
	if err != nil {
		return err
	}
	return persistence.SaveJSON(p, e)
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
