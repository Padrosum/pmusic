package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	pfs "github.com/padros/pmusic/internal/fs"
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
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []pfs.Track{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var q []pfs.Track
	if err := json.NewDecoder(f).Decode(&q); err != nil {
		return []pfs.Track{}, nil
	}
	return q, nil
}

func SaveQueue(q []pfs.Track) error {
	p, err := queuePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(q); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p)
}
