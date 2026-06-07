package store

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const repoBase = "https://raw.githubusercontent.com/Padrosum/pmusic/main"

type Item struct {
	Name string
	Desc string
	Kind string // "plugin" | "theme"
}

var Plugins = []Item{
	{"logger", "Log played tracks with timestamps", "plugin"},
	{"stats", "Session play-count tracker", "plugin"},
	{"keymaps", "Extra key binding presets", "plugin"},
	{"listen-time", "Session listening-time tracker", "plugin"},
	{"notify-send", "Desktop notifications on track change", "plugin"},
	{"statusline", "Write status to /tmp/pmusic-status.json", "plugin"},
	{"theme-scheduler", "Automatic time-based theme switching", "plugin"},
	{"yt-dlp", "Download audio from YouTube via yt-dlp", "plugin"},
}

var Themes = []Item{
	{"gruvbox", "Gruvbox Dark", "theme"},
	{"catppuccin", "Catppuccin Mocha", "theme"},
	{"tokyo-night", "Tokyo Night Storm", "theme"},
}

// Sync downloads all known plugins and themes from the GitHub repo.
// Files are saved to luaDir/plugins/ and luaDir/themes/.
func Sync(luaDir string) error {
	all := make([]Item, 0, len(Plugins)+len(Themes))
	all = append(all, Plugins...)
	all = append(all, Themes...)
	for _, item := range all {
		subdir := "plugins"
		if item.Kind == "theme" {
			subdir = "themes"
		}
		url := fmt.Sprintf("%s/lua/%s/%s.lua", repoBase, subdir, item.Name)
		dest := filepath.Join(luaDir, subdir, item.Name+".lua")
		if err := download(url, dest); err != nil {
			return fmt.Errorf("%s: %w", item.Name, err)
		}
		fmt.Printf("  ✓ %s/%s\n", subdir, item.Name)
	}
	return nil
}

func download(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
