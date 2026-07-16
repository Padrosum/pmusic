package urlutil

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

var ErrInvalidURL = errors.New("URL must use http or https and include a host")

func ValidateHTTP(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsFunc(value, unicode.IsControl) {
		return "", ErrInvalidURL
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return "", ErrInvalidURL
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrInvalidURL
	}
	return value, nil
}
