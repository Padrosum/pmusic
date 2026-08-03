package ui

import (
	"strings"

	"github.com/Padrosum/pmusic/internal/cover"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// toggleCover opens or closes the cover art overlay, requesting the art for
// the current track asynchronously when it opens.
func (m *Model) toggleCover() tea.Cmd {
	if m.showCover {
		m.showCover = false
		return nil
	}
	m.showCover = true
	return m.requestCover()
}

func (m *Model) openCover() tea.Cmd {
	m.showCover = true
	return m.requestCover()
}

// requestCover resolves and renders the current track's cover art in the
// background, returning a coverReadyMsg. Already-rendered art is reused.
func (m *Model) requestCover() tea.Cmd {
	if m.nowPlaying == nil {
		m.coverError = "No track is playing."
		m.coverLines = nil
		m.coverInline = nil
		return nil
	}
	path := m.nowPlaying.Path
	if entry, ok := m.coverCache[path]; ok {
		m.coverLines = entry.lines
		m.coverInline = entry.inline
		m.coverSource = entry.source
		m.coverError = ""
		return nil
	}
	m.coverLoading = true
	m.coverError = ""
	info := m.trackSearchInfo(*m.nowPlaying)
	artist, album := info.meta.Artist, info.meta.Album
	cacheDir := cover.DefaultCacheDir()
	overlayW, overlayH := m.coverOverlaySize()
	inlineW, inlineH := m.coverInlineSize()
	return func() tea.Msg {
		art, err := cover.Resolve(path, artist, album, cacheDir)
		if err != nil {
			return coverReadyMsg{path: path, err: err}
		}
		overlay, err := cover.Render(art, overlayW, overlayH)
		if err != nil {
			return coverReadyMsg{path: path, err: err}
		}
		inline, err := cover.Render(art, inlineW, inlineH)
		if err != nil {
			return coverReadyMsg{path: path, err: err}
		}
		return coverReadyMsg{path: path, lines: strings.Split(overlay, "\n"), inline: strings.Split(inline, "\n"), source: art.Source}
	}
}

func (m *Model) coverOverlaySize() (w, h int) {
	return max(24, min(44, m.width-8)), max(10, min(20, m.height-8))
}

// coverThumbW is the width in cells of the compact thumbnail shown beside the
// now-playing info in the bottom bar.
func (m *Model) coverThumbW() int {
	return max(8, min(12, m.width/9))
}

// coverInlineSize picks the thumbnail size for the bottom bar. Height is fixed
// to three rows so the art lines up with the now-playing info block.
func (m *Model) coverInlineSize() (w, h int) {
	return m.coverThumbW(), 3
}

func (m *Model) handleCover(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m.uiFrame = (m.uiFrame + 1) % 120
		if m.watcher != nil && m.watcher.Changed() {
			m.rescan()
		}
		return m, m.tickCmd()
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc || key.Matches(msg, keys.Quit) || key.Matches(msg, keys.Cover) {
			m.showCover = false
		}
	}
	return m, nil
}

func (m *Model) renderCover() string {
	boxW := max(34, min(52, m.width-4))
	contentW := max(1, boxW-4)

	lines := []string{styleTitle.Render("  ♥ Cover Art  ")}
	if m.nowPlaying != nil {
		lines = append(lines, styleHeaderMeta.Render("  "+truncate(m.nowPlaying.Name, contentW-2)))
	}
	lines = append(lines, "")

	switch {
	case m.coverError != "":
		lines = append(lines, styleError.Render("  "+truncate(friendlyCoverError(m.coverError), contentW-2)))
	case m.coverLoading:
		lines = append(lines, stylePlaying.Render("  "+visualizer(m.uiFrame, 5)+" loading cover…"))
	case len(m.coverLines) > 0:
		lines = append(lines, m.coverLines...)
		lines = append(lines, "", styleDim.Render("  source: "+string(m.coverSource)))
	default:
		lines = append(lines, styleDim.Render("  No cover art available."))
	}

	lines = append(lines, "", styleDim.Render("  c/Esc/q:close"))
	box := stylePanelActive.Width(boxW).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func friendlyCoverError(err string) string {
	switch {
	case err == "":
		return ""
	case strings.Contains(err, "chafa"):
		return "Cover art needs chafa installed (e.g. sudo pacman -S chafa)"
	case strings.Contains(err, "no metadata"):
		return "No embedded or folder cover, and no artist/album metadata to search"
	case strings.Contains(err, "no cover art found"):
		return "No cover art found"
	case strings.Contains(err, "404"):
		return "No cover art found online"
	default:
		return err
	}
}
