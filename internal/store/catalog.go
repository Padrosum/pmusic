package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Padrosum/pmusic/internal/persistence"
)

const (
	storeRelease = "5a5e04212a0462a7de60d62cd1eb3584939f0454"
	maxFileSize  = int64(1 << 20)
)

var storeHTTPClient = &http.Client{Timeout: 15 * time.Second}

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
}

var Themes = []Item{
	{"gruvbox", "Gruvbox Dark", "theme"},
	{"catppuccin", "Catppuccin Mocha", "theme"},
	{"tokyo-night", "Tokyo Night Storm", "theme"},
}

type Manifest struct {
	Version int            `json:"version"`
	Release string         `json:"release"`
	Files   []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	Destination string `json:"destination,omitempty"`
}

var defaultManifest = Manifest{
	Version: 1,
	Release: storeRelease,
	Files: []ManifestFile{
		{Name: "keymaps", Kind: "plugin", URL: immutableURL("plugins/keymaps.lua"), SHA256: "ac244b711ed5fa2fd9bf86d8db4cd908fbc781cf9e367046b07c46aada7abf7d"},
		{Name: "listen-time", Kind: "plugin", URL: immutableURL("plugins/listen-time.lua"), SHA256: "bd3a72300fbc2ed6b96be15ac4084b4d68d2884abcae940a7fb81caf8df76b0f"},
		{Name: "logger", Kind: "plugin", URL: immutableURL("plugins/logger.lua"), SHA256: "6eb806c141abc704df466774b9e2dd3b3ccd77723d5d335dfedeab7792a60706"},
		{Name: "notify-send", Kind: "plugin", URL: immutableURL("plugins/notify-send.lua"), SHA256: "be07ee1ae331e70e89e06635218f80047e8f2f82fae2d82efafa076f81f6eca5"},
		{Name: "stats", Kind: "plugin", URL: immutableURL("plugins/stats.lua"), SHA256: "99e07743dd6f207634d400c938880991af5b21e7bb6506d24618f0ee72894719"},
		{Name: "statusline", Kind: "plugin", URL: immutableURL("plugins/statusline.lua"), SHA256: "32363ff8dc5c056cd7298a571406f651fe9f6ebd2c1b127a7879c5bbb2bf7275"},
		{Name: "theme-scheduler", Kind: "plugin", URL: immutableURL("plugins/theme-scheduler.lua"), SHA256: "1412eaa1bb5e02cf5e45a94f0bf9af2ff5f7713b1e036ef52045bf06a5d7a17a"},
		{Name: "catppuccin", Kind: "theme", URL: immutableURL("themes/catppuccin.lua"), SHA256: "828b8d02f8dc2f633854f20bdebd94b82c19d7a6119b105f548acaed29604e71"},
		{Name: "gruvbox", Kind: "theme", URL: immutableURL("themes/gruvbox.lua"), SHA256: "740135910a17846f702a412f831ade06f926b8ded3ede912b8b446a0c4a14698"},
		{Name: "tokyo-night", Kind: "theme", URL: immutableURL("themes/tokyo-night.lua"), SHA256: "f24145660a1dec773fb0055e88dd9776052cba70cc136417695b7d2e3ea24a85"},
	},
}

func immutableURL(path string) string {
	return "https://raw.githubusercontent.com/Padrosum/pmusic/" + storeRelease + "/lua/" + path
}

// Sync downloads all known plugins and themes from the GitHub repo.
// Files are saved to luaDir/plugins/ and luaDir/themes/.
func Sync(luaDir string) error {
	return syncManifest(context.Background(), luaDir, defaultManifest, storeHTTPClient)
}

func syncManifest(ctx context.Context, luaDir string, manifest Manifest, client *http.Client) error {
	if manifest.Version != 1 || strings.TrimSpace(manifest.Release) == "" {
		return fmt.Errorf("unsupported or incomplete store manifest")
	}
	fmt.Printf("Store release: %s\n", manifest.Release)
	for _, file := range manifest.Files {
		if file.Kind != "plugin" && file.Kind != "theme" {
			return fmt.Errorf("%s: invalid kind %q", file.Name, file.Kind)
		}
		destination := file.Destination
		if destination == "" {
			subdir := file.Kind + "s"
			destination = filepath.Join(subdir, file.Name+".lua")
		}
		if filepath.IsAbs(destination) || strings.Contains(filepath.Clean(destination), "..") {
			return fmt.Errorf("%s: unsafe destination", file.Name)
		}
		data, err := fetchVerified(ctx, client, file)
		if err != nil {
			return fmt.Errorf("%s: %w", file.Name, err)
		}
		dest := filepath.Join(luaDir, destination)
		if err := persistence.WriteFileAtomic(dest, data, persistence.PrivateFileMode); err != nil {
			return fmt.Errorf("%s: %w", file.Name, err)
		}
		fmt.Printf("  ✓ %s (%s)\n", destination, file.URL)
	}
	return nil
}

func fetchVerified(ctx context.Context, client *http.Client, file ManifestFile) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", file.URL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download %s: HTTP %d", file.URL, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file.URL, err)
	}
	if int64(len(data)) > maxFileSize {
		return nil, fmt.Errorf("download %s exceeds %d bytes", file.URL, maxFileSize)
	}
	expected, err := hex.DecodeString(file.SHA256)
	if err != nil || len(expected) != sha256.Size {
		return nil, fmt.Errorf("invalid SHA-256 in manifest")
	}
	actual := sha256.Sum256(data)
	if !bytesEqual(actual[:], expected) {
		return nil, fmt.Errorf("SHA-256 mismatch for %s", file.URL)
	}
	return data, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
