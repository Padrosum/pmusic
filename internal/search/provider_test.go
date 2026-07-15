package search

import (
	"errors"
	"testing"
)

func TestClassifyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		kind  InputKind
		value string
		err   error
	}{
		{name: "query", input: "  metallica orion  ", kind: InputQuery, value: "metallica orion"},
		{name: "https URL", input: "https://soundcloud.com/example/song", kind: InputURL, value: "https://soundcloud.com/example/song"},
		{name: "http URL", input: "http://example.com/song", kind: InputURL, value: "http://example.com/song"},
		{name: "URL-shaped input remains direct", input: "https://", kind: InputURL, value: "https://"},
		{name: "empty", input: " \t\n ", kind: InputQuery, err: ErrEmptyQuery},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, value, err := ClassifyInput(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("error = %v, want %v", err, tt.err)
			}
			if kind != tt.kind || value != tt.value {
				t.Fatalf("got (%v, %q), want (%v, %q)", kind, value, tt.kind, tt.value)
			}
		})
	}
}

func TestYouTubeRejectsEmptyQueryBeforeProcess(t *testing.T) {
	provider := &YouTube{Binary: "definitely-not-a-real-binary"}
	if _, err := provider.Search(t.Context(), "  ", 10); !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("Search error = %v, want ErrEmptyQuery", err)
	}
}
