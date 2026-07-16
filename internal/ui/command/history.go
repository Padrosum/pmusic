package command

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const MaxHistory = 500

type History struct {
	entries []string
	path    string
	index   int
	draft   string
}

func DefaultHistoryPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "pmusic", "command-history"), nil
}

func LoadHistory(path string) (*History, error) {
	h := &History{path: path, index: 0}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return h, nil
	}
	if err != nil {
		return h, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if line := strings.TrimSpace(s.Text()); line != "" {
			h.entries = append(h.entries, line)
		}
	}
	if len(h.entries) > MaxHistory {
		h.entries = h.entries[len(h.entries)-MaxHistory:]
	}
	h.index = len(h.entries)
	return h, s.Err()
}

func (h *History) Entries() []string  { return append([]string(nil), h.entries...) }
func (h *History) Reset(draft string) { h.index = len(h.entries); h.draft = draft }
func (h *History) Add(value string) error {
	v := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), ":"))
	if v == "" {
		return nil
	}
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == v {
		h.Reset("")
		return nil
	}
	h.entries = append(h.entries, v)
	if len(h.entries) > MaxHistory {
		h.entries = h.entries[len(h.entries)-MaxHistory:]
	}
	h.Reset("")
	return h.persist()
}
func (h *History) Previous(current string) string {
	if len(h.entries) == 0 {
		return current
	}
	if h.index == len(h.entries) {
		h.draft = current
	}
	if h.index > 0 {
		h.index--
	}
	return h.entries[h.index]
}
func (h *History) Next() string {
	if h.index < len(h.entries) {
		h.index++
	}
	if h.index == len(h.entries) {
		return h.draft
	}
	return h.entries[h.index]
}
func (h *History) Clear() error { h.entries = nil; h.index = 0; h.draft = ""; return h.persist() }
func (h *History) persist() error {
	if h.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return err
	}
	tmp := h.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, v := range h.entries {
		if _, err = w.WriteString(v + "\n"); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err = w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err = os.Chmod(tmp, 0o600); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, h.path)
}
