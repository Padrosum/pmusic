package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	pmsearch "github.com/padros/pmusic/internal/search"
)

func TestClampSelection(t *testing.T) {
	tests := []struct{ index, count, want int }{
		{-1, 3, 0},
		{0, 0, 0},
		{1, 3, 1},
		{9, 3, 2},
	}
	for _, tt := range tests {
		if got := clampSelection(tt.index, tt.count); got != tt.want {
			t.Errorf("clampSelection(%d, %d) = %d, want %d", tt.index, tt.count, got, tt.want)
		}
	}
}

func TestStaleSearchResultIsIgnored(t *testing.T) {
	m := &Model{musicSearch: newMusicSearchModel()}
	m.musicSearch.state = musicSearchLoading
	m.musicSearch.activeRequest = 2
	_, handled := m.handleMusicSearchMessage(musicSearchCompletedMsg{
		requestID: 1,
		results:   []pmsearch.Result{{Title: "stale"}},
	})
	if !handled {
		t.Fatal("completion message was not handled")
	}
	if len(m.musicSearch.results) != 0 || m.musicSearch.state != musicSearchLoading {
		t.Fatalf("stale result changed model: %#v", m.musicSearch)
	}
}

func TestEmptyResultsBecomeErrorState(t *testing.T) {
	m := &Model{musicSearch: newMusicSearchModel()}
	m.musicSearch.state = musicSearchLoading
	m.musicSearch.activeRequest = 3
	m.handleMusicSearchMessage(musicSearchCompletedMsg{requestID: 3})
	if m.musicSearch.state != musicSearchError || !strings.Contains(m.musicSearch.errorText, "No results") {
		t.Fatalf("state=%v error=%q", m.musicSearch.state, m.musicSearch.errorText)
	}
}

func TestFormatSeconds(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{{59, "0:59"}, {60, "1:00"}, {3599, "59:59"}, {3600, "1:00:00"}}
	for _, tt := range tests {
		if got := formatSeconds(tt.seconds, true); got != tt.want {
			t.Errorf("formatSeconds(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
	if got := formatSeconds(0, false); got != "--:--" {
		t.Fatalf("unknown duration = %q", got)
	}
}

func TestTruncateIsDisplayWidthSafe(t *testing.T) {
	for _, input := range []string{"A very long title", "Çok uzun bir şarkı", "東京の長い曲名"} {
		got := truncate(input, 8)
		if width := lipgloss.Width(got); width > 8 {
			t.Fatalf("truncate(%q) width = %d: %q", input, width, got)
		}
	}
}

func TestMusicSearchOverlayDoesNotPanicAtMinimumTerminal(t *testing.T) {
	states := []musicSearchState{
		musicSearchInput,
		musicSearchLoading,
		musicSearchResults,
		musicSearchDownloading,
		musicSearchSuccess,
		musicSearchError,
	}
	for _, state := range states {
		m := &Model{width: 52, height: 12, showMusicSearch: true, musicSearch: newMusicSearchModel()}
		m.musicSearch.state = state
		m.musicSearch.query = "çok uzun bir Unicode sorgusu 東京"
		m.musicSearch.errorText = "example error"
		m.musicSearch.results = []pmsearch.Result{{Title: "A long result title", Uploader: "Uploader", URL: "https://example.com", Provider: "youtube"}}
		m.musicSearch.downloading = m.musicSearch.results[0]
		view := m.View()
		if view == "" {
			t.Fatalf("state %v rendered an empty view", state)
		}
		if width := lipgloss.Width(view); width > m.width {
			t.Fatalf("state %v rendered width %d, terminal width %d", state, width, m.width)
		}
	}
}
