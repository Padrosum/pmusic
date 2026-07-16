package config

import (
	"os"
	"path/filepath"

	"github.com/Padrosum/pmusic/internal/persistence"
)

type Config struct {
	MusicDir string `json:"music_dir"`
}

func path() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pmusic", "config.json"), nil
}

func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	cfg, found, err := persistence.LoadJSON[Config](p, nil)
	if err != nil {
		return nil, err
	}
	if !found {
		return &Config{}, nil
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	p, err := path()
	if err != nil {
		return err
	}
	return persistence.SaveJSON(p, cfg)
}
