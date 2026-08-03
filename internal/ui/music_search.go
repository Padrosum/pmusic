package ui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pmdownload "github.com/Padrosum/pmusic/internal/download"
	pmsearch "github.com/Padrosum/pmusic/internal/search"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type musicSearchState int

const (
	musicSearchInput musicSearchState = iota
	musicSearchLoading
	musicSearchResults
	musicSearchDownloading
	musicSearchSuccess
	musicSearchError
)

type musicSearchModel struct {
	state         musicSearchState
	input         textinput.Model
	query         string
	results       []pmsearch.Result
	cursor        int
	errorText     string
	directURL     bool
	activeRequest uint64
	downloadID    uint64
	downloading   pmsearch.Result
	destDir       string
}

type musicSearchCompletedMsg struct {
	requestID uint64
	query     string
	directURL bool
	results   []pmsearch.Result
	err       error
}

type musicDownloadCompletedMsg struct {
	downloadID uint64
	result     pmsearch.Result
	err        error
}

func newMusicSearchModel() musicSearchModel {
	input := textinput.New()
	input.Placeholder = "Song, artist, or a supported URL..."
	input.CharLimit = 500
	return musicSearchModel{state: musicSearchInput, input: input}
}

func (m *Model) openMusicSearch() tea.Cmd {
	m.showMusicSearch = true
	if m.musicSearch.state == musicSearchDownloading {
		return nil
	}
	m.musicSearch.destDir = ""
	m.resetMusicSearchInput(true)
	return m.musicSearch.input.Focus()
}

func (m *Model) resetMusicSearchInput(clear bool) {
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
	m.musicSearch.activeRequest++
	m.musicSearch.state = musicSearchInput
	m.musicSearch.results = nil
	m.musicSearch.cursor = 0
	m.musicSearch.errorText = ""
	m.musicSearch.directURL = false
	m.musicSearch.query = ""
	if clear {
		m.musicSearch.input.SetValue("")
	}
	m.musicSearch.input.Focus()
}

func (m *Model) closeMusicSearch() {
	if m.musicSearch.state == musicSearchLoading {
		if m.searchCancel != nil {
			m.searchCancel()
			m.searchCancel = nil
		}
		m.musicSearch.activeRequest++
	}
	m.showMusicSearch = false
	m.musicSearch.input.Blur()
}

func (m *Model) startMusicSearch() tea.Cmd {
	kind, value, err := pmsearch.ClassifyInput(m.musicSearch.input.Value())
	if err != nil {
		m.musicSearch.state = musicSearchInput
		m.musicSearch.errorText = "Enter a song, artist, or URL"
		return nil
	}
	if m.searchCancel != nil {
		m.searchCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	m.searchCancel = cancel
	m.musicSearch.activeRequest++
	requestID := m.musicSearch.activeRequest
	m.musicSearch.state = musicSearchLoading
	m.musicSearch.query = value
	m.musicSearch.results = nil
	m.musicSearch.cursor = 0
	m.musicSearch.errorText = ""
	m.musicSearch.directURL = kind == pmsearch.InputURL
	m.musicSearch.input.Blur()

	provider := m.searchProvider
	resolver := m.urlResolver
	return func() tea.Msg {
		if kind == pmsearch.InputURL {
			result, resolveErr := resolver.Resolve(ctx, value)
			return musicSearchCompletedMsg{
				requestID: requestID,
				query:     value,
				directURL: true,
				results:   []pmsearch.Result{result},
				err:       resolveErr,
			}
		}
		results, searchErr := provider.Search(ctx, value, 10)
		return musicSearchCompletedMsg{requestID: requestID, query: value, results: results, err: searchErr}
	}
}

func (m *Model) startMusicDownload() tea.Cmd {
	if m.musicSearch.state != musicSearchResults || len(m.musicSearch.results) == 0 {
		return nil
	}
	m.musicSearch.cursor = clampSelection(m.musicSearch.cursor, len(m.musicSearch.results))
	result := m.musicSearch.results[m.musicSearch.cursor]
	if strings.TrimSpace(result.URL) == "" {
		m.musicSearch.state = musicSearchError
		m.musicSearch.errorText = "The selected result has no downloadable URL"
		return nil
	}
	m.musicSearch.state = musicSearchDownloading
	m.musicSearch.downloading = result
	m.musicSearch.downloadID++
	downloadID := m.musicSearch.downloadID
	downloader := m.downloader
	musicDir := m.rootDir
	if m.musicSearch.destDir != "" {
		musicDir = filepath.Join(m.rootDir, m.musicSearch.destDir)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.downloadCancel = cancel
	return func() tea.Msg {
		err := downloader.Download(ctx, musicDir, result.URL)
		return musicDownloadCompletedMsg{downloadID: downloadID, result: result, err: err}
	}
}

// handleMusicSearchMessage consumes asynchronous completion messages even when
// the overlay is closed. Search generations prevent stale results from winning;
// downloads remain alive and notify the user after the overlay is dismissed.
func (m *Model) handleMusicSearchMessage(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case musicSearchCompletedMsg:
		if !isCurrentRequest(m.musicSearch.activeRequest, msg.requestID) {
			return nil, true
		}
		if m.searchCancel != nil {
			m.searchCancel()
			m.searchCancel = nil
		}
		if msg.directURL && msg.err != nil {
			m.musicSearch.results = []pmsearch.Result{{
				Title:    "Direct URL",
				Uploader: "Metadata preview unavailable",
				URL:      msg.query,
				Provider: "yt-dlp",
			}}
			m.musicSearch.cursor = 0
			m.musicSearch.state = musicSearchResults
			m.musicSearch.errorText = "Preview unavailable; the URL can still be downloaded"
			return nil, true
		}
		if msg.err != nil {
			m.musicSearch.state = musicSearchError
			m.musicSearch.errorText = friendlySearchError(msg.err)
			return nil, true
		}
		if len(msg.results) == 0 {
			m.musicSearch.state = musicSearchError
			m.musicSearch.errorText = "No results found"
			return nil, true
		}
		m.musicSearch.results = msg.results
		m.musicSearch.cursor = clampSelection(m.musicSearch.cursor, len(msg.results))
		m.musicSearch.state = musicSearchResults
		m.musicSearch.errorText = ""
		return nil, true

	case musicDownloadCompletedMsg:
		if msg.downloadID != m.musicSearch.downloadID {
			return nil, true
		}
		if m.downloadCancel != nil {
			m.downloadCancel()
			m.downloadCancel = nil
		}
		if msg.err != nil {
			m.musicSearch.state = musicSearchError
			m.musicSearch.errorText = friendlyDownloadError(msg.err)
			m.notify("download failed: " + m.musicSearch.errorText)
		} else {
			m.musicSearch.state = musicSearchSuccess
			m.musicSearch.errorText = ""
			m.notify("download complete: " + displayResultTitle(msg.result))
		}
		return nil, true
	}
	return nil, false
}

func (m *Model) handleMusicSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		m.uiFrame = (m.uiFrame + 1) % 120
		if m.watcher != nil && m.watcher.Changed() {
			m.rescan()
		}
		return m, m.tickCmd()
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc || key.Matches(msg, keys.Quit) {
			m.closeMusicSearch()
			return m, nil
		}
		switch m.musicSearch.state {
		case musicSearchInput:
			if msg.Type == tea.KeyEnter {
				return m, m.startMusicSearch()
			}
		case musicSearchLoading:
			if msg.String() == "/" {
				m.resetMusicSearchInput(false)
				return m, m.musicSearch.input.Focus()
			}
			return m, nil
		case musicSearchResults:
			switch {
			case key.Matches(msg, keys.Down):
				m.musicSearch.cursor = clampSelection(m.musicSearch.cursor+1, len(m.musicSearch.results))
			case key.Matches(msg, keys.Up):
				m.musicSearch.cursor = clampSelection(m.musicSearch.cursor-1, len(m.musicSearch.results))
			case msg.Type == tea.KeyEnter:
				return m, m.startMusicDownload()
			case msg.String() == "/":
				m.resetMusicSearchInput(false)
				return m, m.musicSearch.input.Focus()
			}
			return m, nil
		case musicSearchDownloading:
			return m, nil
		case musicSearchSuccess, musicSearchError:
			if msg.String() == "/" {
				m.resetMusicSearchInput(false)
				return m, m.musicSearch.input.Focus()
			}
			return m, nil
		}
	}

	if m.musicSearch.state == musicSearchInput {
		before := m.musicSearch.input.Value()
		var cmd tea.Cmd
		m.musicSearch.input, cmd = m.musicSearch.input.Update(msg)
		if m.musicSearch.input.Value() != before {
			m.musicSearch.errorText = ""
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) renderMusicSearch() string {
	boxW := min(78, max(44, m.width-4))
	contentW := max(20, boxW-4)
	m.musicSearch.input.Width = max(12, contentW-10)

	lines := []string{styleTitle.Render("  ♫  Music Search  "), ""}
	query := m.musicSearch.query
	if m.musicSearch.state == musicSearchInput {
		lines = append(lines, styleHeaderMeta.Render("  Search: ")+m.musicSearch.input.View())
	} else {
		lines = append(lines, styleHeaderMeta.Render("  Search: ")+styleNormal.Render(truncate(query, contentW-12)))
	}
	source := "YouTube (text search)"
	if m.searchProvider != nil {
		source = m.searchProvider.Name() + " (text search)"
	}
	if m.musicSearch.directURL {
		source = "yt-dlp URL"
	}
	lines = append(lines, styleHeaderMeta.Render("  Source: ")+styleNormal.Render(source), "")

	switch m.musicSearch.state {
	case musicSearchInput:
		if m.musicSearch.errorText != "" {
			lines = append(lines, styleError.Render("  "+truncate(m.musicSearch.errorText, contentW-2)))
		}
		lines = append(lines, styleDim.Render("  Enter:search  Esc/q:close"))
	case musicSearchLoading:
		lines = append(lines,
			stylePlaying.Render("  "+visualizer(m.uiFrame, 5)+" Searching…"),
			styleDim.Render("  /:new search  Esc/q:close"),
		)
	case musicSearchResults:
		visible := max(1, min(len(m.musicSearch.results), (m.height-8)/2))
		offset := scrollOffset(m.musicSearch.cursor, visible)
		for i := offset; i < len(m.musicSearch.results) && i < offset+visible; i++ {
			result := m.musicSearch.results[i]
			title := truncate(displayResultTitle(result), contentW-5)
			prefix := "    "
			if i == m.musicSearch.cursor {
				prefix = "  › "
				lines = append(lines, styleSelected.Width(contentW).Render(prefix+title))
			} else {
				lines = append(lines, styleNormal.Render(prefix+title))
			}
			meta := fmt.Sprintf("      %s · %s · %s", result.Uploader, formatSeconds(result.Duration, result.DurationKnown), providerName(result.Provider))
			lines = append(lines, styleDim.Render(truncate(meta, contentW)))
		}
		if m.musicSearch.errorText != "" {
			lines = append(lines, styleError.Render("  "+truncate(m.musicSearch.errorText, contentW-2)))
		}
		lines = append(lines, styleDim.Render("  j/k:select  Enter:download  /:new search  Esc/q:close"))
	case musicSearchDownloading:
		lines = append(lines,
			stylePlaying.Render("  "+visualizer(m.uiFrame, 5)+" Downloading…"),
			styleNormal.Render("  "+truncate(displayResultTitle(m.musicSearch.downloading), contentW-2)),
			styleDim.Render("  Esc/q:close (download continues)"),
		)
	case musicSearchSuccess:
		lines = append(lines,
			styleNowPlaying.Render("  ✓ Added to the local music library"),
			styleNormal.Render("  "+truncate(displayResultTitle(m.musicSearch.downloading), contentW-2)),
			styleDim.Render("  /:new search  Esc/q:close"),
		)
	case musicSearchError:
		lines = append(lines,
			styleError.Render("  "+truncate(m.musicSearch.errorText, contentW-2)),
			styleDim.Render("  /:new search  Esc/q:close"),
		)
	}

	box := stylePanelActive.Width(boxW).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func clampSelection(index, count int) int {
	if count <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}
	return index
}

func isCurrentRequest(active, incoming uint64) bool { return active == incoming }

func formatSeconds(seconds int, known bool) string {
	if !known || seconds < 0 {
		return "--:--"
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func displayResultTitle(result pmsearch.Result) string {
	if strings.TrimSpace(result.Title) == "" {
		return "Untitled item"
	}
	return result.Title
}

func providerName(provider string) string {
	if strings.EqualFold(provider, "youtube") {
		return "YouTube"
	}
	if provider == "" {
		return "yt-dlp"
	}
	return provider
}

func friendlySearchError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "Search timed out"
	case errors.Is(err, context.Canceled):
		return "Search canceled"
	case errors.Is(err, pmsearch.ErrYTDLP):
		return "yt-dlp is not installed or not available on PATH"
	case errors.Is(err, pmsearch.ErrNoResults):
		return "No results found"
	default:
		return truncate(strings.TrimSpace(err.Error()), 100)
	}
}

func friendlyDownloadError(err error) string {
	switch {
	case errors.Is(err, pmdownload.ErrYTDLP):
		return "yt-dlp is not installed or not available on PATH"
	case errors.Is(err, pmdownload.ErrInvalidDest):
		return "Music directory does not exist"
	case errors.Is(err, pmdownload.ErrNotWritable):
		return "Music directory is not writable"
	case errors.Is(err, pmdownload.ErrInvalidURL):
		return "The selected result has no valid URL"
	default:
		return truncate(strings.TrimSpace(err.Error()), 100)
	}
}
