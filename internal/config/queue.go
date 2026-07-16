package config

import (
	"os"
	"path/filepath"

	pfs "github.com/Padrosum/pmusic/internal/fs"
	"github.com/Padrosum/pmusic/internal/persistence"
)

func queuePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pmusic", "queue.json"), nil
}

func LoadQueue() ([]pfs.Track, error) {
	p, err := queuePath()
	if err != nil {
		return nil, err
	}
	q, found, err := persistence.LoadJSON[[]pfs.Track](p, nil)
	if err != nil {
		return nil, err
	}
	if !found {
		return []pfs.Track{}, nil
	}
	return q, nil
}

func SaveQueue(q []pfs.Track) error {
	p, err := queuePath()
	if err != nil {
		return err
	}
	return persistence.SaveJSON(p, q)
}
