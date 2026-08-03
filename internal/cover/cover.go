package cover

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// Source describes where the cover art came from, shown in the UI.
type Source string

const (
	SourceEmbedded Source = "embedded"
	SourceFolder   Source = "folder file"
	SourceCache    Source = "cache"
	SourceOnline   Source = "iTunes"
)

var (
	ErrNotFound   = errors.New("no cover art found")
	ErrNoChafa    = errors.New("chafa is not installed")
	ErrNoMetadata = errors.New("no embedded or folder art, and no metadata to search online")
)

// itunesBaseURL is overridable so tests can serve a fake API.
var itunesBaseURL = "https://itunes.apple.com"

// Art is a resolved cover image plus where it came from.
type Art struct {
	Path   string
	Data   []byte
	Source Source
}

var folderNames = []string{"cover", "folder", "front", "albumart", "album"}
var folderExts = []string{".jpg", ".jpeg", ".png", ".webp"}

// Resolve finds cover art for a track: embedded tag art first, then a cover
// file beside the track, then an online fetch cached on disk.
func Resolve(trackPath, artist, album, cacheDir string) (*Art, error) {
	if a, err := embedded(trackPath); err == nil {
		return a, nil
	}
	if a, err := folderFile(trackPath); err == nil {
		return a, nil
	}
	return online(artist, album, cacheDir)
}

// DefaultCacheDir returns the per-user cache directory for downloaded covers.
func DefaultCacheDir() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "pmusic-covers")
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "pmusic", "covers")
}

// Render converts art to ANSI symbols via chafa and returns its display lines.
func Render(art *Art, width, height int) (string, error) {
	if width < 4 || height < 2 {
		return "", fmt.Errorf("art area too small (%dx%d)", width, height)
	}
	chafa, err := exec.LookPath("chafa")
	if err != nil {
		return "", ErrNoChafa
	}
	cmd := exec.Command(chafa,
		"--format", "symbols",
		"--colors", "full",
		"--size", fmt.Sprintf("%dx%d", width, height),
		"-")
	cmd.Stdin = bytes.NewReader(art.Data)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("chafa failed: %w", err)
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

func embedded(path string) (*Art, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	p := m.Picture()
	if p == nil || len(p.Data) == 0 {
		return nil, ErrNotFound
	}
	return &Art{Path: "embedded", Data: p.Data, Source: SourceEmbedded}, nil
}

func folderFile(trackPath string) (*Art, error) {
	dir := filepath.Dir(trackPath)
	for _, name := range folderNames {
		for _, ext := range folderExts {
			p := filepath.Join(dir, name+ext)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				data, err := os.ReadFile(p)
				if err != nil {
					continue
				}
				return &Art{Path: p, Data: data, Source: SourceFolder}, nil
			}
		}
	}
	return nil, ErrNotFound
}

func online(artist, album, cacheDir string) (*Art, error) {
	if strings.TrimSpace(artist) == "" && strings.TrimSpace(album) == "" {
		return nil, ErrNoMetadata
	}
	if cacheDir == "" {
		cacheDir = DefaultCacheDir()
	}
	key := strings.ToLower(strings.TrimSpace(artist)) + "|" + strings.ToLower(strings.TrimSpace(album))
	h := sha1.Sum([]byte(key))
	hash := hex.EncodeToString(h[:])
	path := filepath.Join(cacheDir, hash+".jpg")
	if data, err := os.ReadFile(path); err == nil {
		return &Art{Path: path, Data: data, Source: SourceCache}, nil
	}
	url, err := searchArtworkURL(artist, album)
	if err != nil {
		return nil, err
	}
	data, err := download(url)
	if err != nil {
		return nil, err
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("downloaded cover is not a valid image: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		_ = os.WriteFile(path, data, 0o644)
	}
	return &Art{Path: path, Data: data, Source: SourceOnline}, nil
}

func searchArtworkURL(artist, album string) (string, error) {
	term := strings.Join([]string{strings.TrimSpace(artist), strings.TrimSpace(album)}, " ")
	term = strings.Join(strings.Fields(term), "+")
	url := fmt.Sprintf("%s/search?term=%s&media=music&entity=album&limit=1", itunesBaseURL, term)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("iTunes search returned %s", resp.Status)
	}
	var out struct {
		Results []struct {
			ArtworkURL string `json:"artworkUrl100"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Results) == 0 || out.Results[0].ArtworkURL == "" {
		return "", ErrNotFound
	}
	return strings.Replace(out.Results[0].ArtworkURL, "100x100", "600x600", 1), nil
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover download returned %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}
