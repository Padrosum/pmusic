package catalog

import (
	"os"
	"path/filepath"
	"strings"
)

const FileName = "library.gl"

// Options control where library.gl is found and how it is converted.
type Options struct {
	MusicDir  string
	ConfigDir string // if empty, UserConfigDir/pmusic
	Converter Converter
}

func configCatalogDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pmusic"), nil
}

// FindPath returns the overlay file to load. Config wins over the music
// directory when both exist. An empty path means there is no overlay.
func FindPath(opt Options) (string, error) {
	configDir := opt.ConfigDir
	if configDir == "" {
		dir, err := configCatalogDir()
		if err != nil {
			return "", err
		}
		configDir = dir
	}
	candidates := []string{
		filepath.Join(configDir, FileName),
	}
	if strings.TrimSpace(opt.MusicDir) != "" {
		candidates = append(candidates, filepath.Join(opt.MusicDir, FileName))
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if info.IsDir() {
			continue
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	return "", nil
}
