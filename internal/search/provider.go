package search

import (
	"context"
	"errors"
	"strings"

	"github.com/Padrosum/pmusic/internal/urlutil"
)

var (
	ErrEmptyQuery = errors.New("search query cannot be empty")
	ErrNoResults  = errors.New("no results found")
	ErrYTDLP      = errors.New("yt-dlp is not installed or not available on PATH")
)

type Result struct {
	ID            string
	Title         string
	Uploader      string
	Duration      int
	DurationKnown bool
	URL           string
	Provider      string
}

type Provider interface {
	ID() string
	Name() string
	Search(ctx context.Context, query string, limit int) ([]Result, error)
}

type URLResolver interface {
	Resolve(ctx context.Context, rawURL string) (Result, error)
}

type InputKind int

const (
	InputQuery InputKind = iota
	InputURL
)

func ClassifyInput(input string) (InputKind, string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return InputQuery, "", ErrEmptyQuery
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		validated, err := urlutil.ValidateHTTP(value)
		return InputURL, validated, err
	}
	if strings.Contains(value, "://") {
		_, err := urlutil.ValidateHTTP(value)
		return InputURL, "", err
	}
	return InputQuery, value, nil
}
