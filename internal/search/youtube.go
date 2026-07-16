package search

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Padrosum/pmusic/internal/urlutil"
)

const (
	defaultResultLimit = 10
	maxMetadataLine    = 2 << 20
	maxStderrBytes     = 32 << 10
)

type YouTube struct {
	Binary string
}

func NewYouTube() *YouTube { return &YouTube{Binary: "yt-dlp"} }

func (p *YouTube) ID() string   { return "youtube" }
func (p *YouTube) Name() string { return "YouTube" }

func (p *YouTube) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	if limit <= 0 {
		limit = defaultResultLimit
	}
	args := buildSearchArgs(query, limit)
	return p.runJSONLines(ctx, args, limit)
}

func buildSearchArgs(query string, limit int) []string {
	return []string{"--flat-playlist", "--dump-json", "--no-warnings", "--skip-download", fmt.Sprintf("ytsearch%d:%s", limit, query)}
}

func (p *YouTube) Resolve(ctx context.Context, rawURL string) (Result, error) {
	rawURL, err := urlutil.ValidateHTTP(rawURL)
	if err != nil {
		return Result{}, err
	}
	args := []string{"--dump-json", "--no-warnings", "--skip-download", "--playlist-end", "1", "--", rawURL}
	results, err := p.runJSONLines(ctx, args, 1)
	if err != nil {
		return Result{}, err
	}
	result := results[0]
	// Preserve the user's original URL so playlists and non-YouTube extractors
	// retain the same download behavior after the metadata preview.
	result.URL = rawURL
	return result, nil
}

func (p *YouTube) runJSONLines(ctx context.Context, args []string, limit int) ([]Result, error) {
	binary := p.Binary
	if binary == "" {
		binary = "yt-dlp"
	}
	if _, err := exec.LookPath(binary); err != nil {
		return nil, ErrYTDLP
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("prepare yt-dlp output: %w", err)
	}
	var stderr limitedBuffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, processError(err, stderr.String())
	}

	results, malformed, readErr := parseJSONLines(stdout, p.ID(), limit)
	if readErr != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, processError(waitErr, stderr.String())
	}
	if len(results) == 0 {
		if malformed > 0 {
			return nil, fmt.Errorf("yt-dlp returned invalid metadata")
		}
		return nil, ErrNoResults
	}
	return results, nil
}

type metadata struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Uploader   string   `json:"uploader"`
	Channel    string   `json:"channel"`
	Duration   *float64 `json:"duration"`
	URL        string   `json:"url"`
	WebpageURL string   `json:"webpage_url"`
	Extractor  string   `json:"extractor_key"`
}

func parseJSONLines(r io.Reader, fallbackProvider string, limit int) ([]Result, int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxMetadataLine)
	capacity := max(0, limit)
	results := make([]Result, 0, capacity)
	malformed := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if limit > 0 && len(results) >= limit {
			continue
		}
		var item metadata
		if err := json.Unmarshal(line, &item); err != nil {
			malformed++
			continue
		}
		result := metadataResult(item, fallbackProvider)
		if result.Title == "" && result.URL == "" {
			malformed++
			continue
		}
		if result.URL != "" {
			if _, err := urlutil.ValidateHTTP(result.URL); err != nil {
				malformed++
				continue
			}
		}
		results = append(results, result)
	}
	if err := scanner.Err(); err != nil {
		return nil, malformed, fmt.Errorf("read yt-dlp metadata: %w", err)
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, malformed, nil
}

func metadataResult(item metadata, fallbackProvider string) Result {
	uploader := strings.TrimSpace(item.Uploader)
	if uploader == "" {
		uploader = strings.TrimSpace(item.Channel)
	}
	if uploader == "" {
		uploader = "Unknown uploader"
	}
	provider := strings.ToLower(strings.TrimSpace(item.Extractor))
	if provider == "" {
		provider = fallbackProvider
	}
	result := Result{
		ID:       item.ID,
		Title:    strings.TrimSpace(item.Title),
		Uploader: uploader,
		URL:      resultURL(item),
		Provider: provider,
	}
	if item.Duration != nil && *item.Duration >= 0 {
		result.Duration = int(*item.Duration)
		result.DurationKnown = true
	}
	return result
}

func resultURL(item metadata) string {
	if strings.TrimSpace(item.WebpageURL) != "" {
		return item.WebpageURL
	}
	if item.ID != "" {
		return "https://www.youtube.com/watch?v=" + item.ID
	}
	return item.URL
}

func processError(err error, stderr string) error {
	detail := strings.TrimSpace(stderr)
	if detail != "" {
		if line, _, ok := strings.Cut(detail, "\n"); ok {
			detail = line
		}
		return fmt.Errorf("yt-dlp failed: %s", detail)
	}
	return fmt.Errorf("yt-dlp failed: %w", err)
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := maxStderrBytes - b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return original, nil
}

var _ Provider = (*YouTube)(nil)
var _ URLResolver = (*YouTube)(nil)
