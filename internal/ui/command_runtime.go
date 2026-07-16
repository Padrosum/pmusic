package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	pfs "github.com/Padrosum/pmusic/internal/fs"
	"github.com/Padrosum/pmusic/internal/listening"
	"github.com/Padrosum/pmusic/internal/meta"
	"github.com/Padrosum/pmusic/internal/player"
	"github.com/Padrosum/pmusic/internal/ui/command"
	tea "github.com/charmbracelet/bubbletea"
)

type trackSearchInfo struct {
	display string
	search  string
	meta    meta.Meta
}

func (m *Model) trackSearchInfo(t pfs.Track) trackSearchInfo {
	if info, ok := m.trackSearchCache[t.Path]; ok {
		return info
	}
	md := meta.Read(t.Path)
	display := t.Name
	if md.Title != "" {
		display = md.Title
	}
	if md.Artist != "" {
		display = md.Artist + " — " + display
	}
	info := trackSearchInfo{display: display, search: strings.ToLower(strings.Join([]string{t.Name, md.Title, md.Artist, display}, " ")), meta: md}
	if m.trackSearchCache == nil {
		m.trackSearchCache = make(map[string]trackSearchInfo)
	}
	m.trackSearchCache[t.Path] = info
	return info
}

// The methods in this file adapt command.Runtime to the existing model. They
// deliberately call the same helpers used by keyboard shortcuts.
func (m *Model) Play(query string) (tea.Cmd, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		cmd := m.playSelected()
		if cmd == nil {
			return nil, &command.RuntimeCommandError{Message: "No track is selected."}
		}
		return cmd, nil
	}
	if n, err := strconv.Atoi(query); err == nil {
		tracks := m.currentTracks()
		if n < 1 || n > len(tracks) {
			return nil, &command.InvalidArgumentError{Message: fmt.Sprintf("Track index %d is outside the visible list (1-%d).", n, len(tracks)), Usage: ":play <track|index>"}
		}
		m.trackIdx = n - 1
		m.focused = panelTracks
		return m.playSelected(), nil
	}
	type match struct {
		fi, ti int
		track  pfs.Track
	}
	q := strings.ToLower(query)
	var matches []match
	for fi, f := range m.folders {
		for ti, t := range f.Tracks {
			info := m.trackSearchInfo(t)
			if strings.Contains(info.search, q) {
				matches = append(matches, match{fi, ti, t})
			}
		}
	}
	if len(matches) == 0 {
		return nil, &command.RuntimeCommandError{Message: fmt.Sprintf("No local track matches %q.", query)}
	}
	if len(matches) > 1 {
		m.OpenLocalSearch(query)
		return nil, &command.AmbiguousMatchError{Query: query, Count: len(matches)}
	}
	x := matches[0]
	m.folderIdx, m.trackIdx, m.focused = x.fi, x.ti, panelTracks
	return m.playSelected(), nil
}
func (m *Model) Pause() error {
	if m.player.State() == player.Stopped || m.nowPlaying == nil {
		return &command.RuntimeCommandError{Message: "No track is currently loaded."}
	}
	m.player.Pause()
	m.notify("Playback paused.")
	return nil
}
func (m *Model) TogglePause() error {
	if m.player.State() == player.Stopped || m.nowPlaying == nil {
		return &command.RuntimeCommandError{Message: "No track is currently loaded."}
	}
	m.player.TogglePause()
	return nil
}
func (m *Model) Next() (tea.Cmd, error) {
	cmd := m.playNext()
	if cmd == nil {
		return nil, &command.RuntimeCommandError{Message: "There is no next track."}
	}
	return cmd, nil
}
func (m *Model) Previous() (tea.Cmd, error) {
	cmd := m.playPrev()
	if cmd == nil {
		return nil, &command.RuntimeCommandError{Message: "There is no previous track."}
	}
	return cmd, nil
}
func (m *Model) Volume() int { return int(math.Round(m.player.Volume() * 100)) }
func (m *Model) SetVolume(v int) error {
	m.player.SetVolume(float64(v) / 100)
	if v > 0 {
		m.mutedVolume = v
	}
	return nil
}
func (m *Model) Muted() bool { return m.Volume() == 0 }
func (m *Model) Mute(mode string) error {
	switch mode {
	case "mute":
		if m.Volume() > 0 {
			m.mutedVolume = m.Volume()
		}
		m.player.SetVolume(0)
	case "unmute":
		if m.Volume() == 0 {
			v := m.mutedVolume
			if v <= 0 {
				v = 100
			}
			m.player.SetVolume(float64(v) / 100)
		}
	case "toggle":
		if m.Volume() == 0 {
			return m.Mute("unmute")
		}
		return m.Mute("mute")
	}
	return nil
}
func (m *Model) Position() (time.Duration, time.Duration, bool) {
	_, elapsed, total := m.player.Progress()
	return elapsed, total, m.nowPlaying != nil && m.player.State() != player.Stopped && total > 0
}
func (m *Model) SeekAbsolute(pos time.Duration) error {
	if !m.player.SeekTo(pos) {
		return &command.RuntimeCommandError{Message: "No track is currently loaded."}
	}
	return nil
}
func (m *Model) Loop() bool           { return m.loop }
func (m *Model) SetLoop(v bool) error { m.loop = v; return nil }
func (m *Model) OpenQueue()           { m.showQueue = true; m.queueCursor = 0 }
func (m *Model) ClearQueue() (int, error) {
	n := len(m.queue)
	m.queue = nil
	m.queueCursor = 0
	if err := m.persistQueue(); err != nil {
		return 0, err
	}
	return n, nil
}
func (m *Model) OpenLocalSearch(query string) {
	m.showSearch = true
	m.searchInput.SetValue(query)
	m.searchInput.CursorEnd()
	m.trackIdx = 0
	m.searchResultsValid = false
	m.focused = panelTracks
	m.searchInput.Focus()
}
func (m *Model) OpenOnlineSearch(query string, start bool) tea.Cmd {
	focus := m.openMusicSearch()
	m.musicSearch.input.SetValue(query)
	m.musicSearch.input.CursorEnd()
	if start {
		return m.startMusicSearch()
	}
	return focus
}
func (m *Model) ReloadLua() tea.Cmd { return m.reloadLuaCmd() }
func (m *Model) ReloadLibrary() tea.Cmd {
	rootDir := m.rootDir
	return func() tea.Msg {
		root, err := pfs.Scan(rootDir)
		if err != nil {
			return libraryReloadedMsg{err: err}
		}
		return libraryReloadedMsg{root: root, folders: pfs.FlatFolders(root)}
	}
}
func (m *Model) OpenHelp(topic string) error { return m.openRegistryHelp(topic) }
func (m *Model) OpenHistory(limit int)       { m.openHistoryHelp(limit) }
func (m *Model) OpenStats(scope, query string) error {
	if m.listening == nil {
		return &command.RuntimeCommandError{Message: "Listening statistics are unavailable."}
	}
	var summary listening.Summary
	title := "Today's Listening"
	switch scope {
	case "today":
		summary = m.listening.Period(1, time.Now())
	case "week":
		title = "Listening · Last 7 Days"
		summary = m.listening.Period(7, time.Now())
	case "all":
		title = "Listening · All Time"
		summary = m.listening.Artist("")
	case "artist":
		title = "Artist · " + query
		summary = m.listening.Artist(query)
	}
	lines := []string{
		title, "",
		fmt.Sprintf("Listening time   %s", formatListeningSeconds(summary.ListeningSeconds)),
		fmt.Sprintf("Tracks started   %d", summary.Plays),
		fmt.Sprintf("Completed        %d", summary.Completions),
		fmt.Sprintf("Skipped          %d", summary.Skips),
		"", "Top tracks",
	}
	if len(summary.Top) == 0 {
		lines = append(lines, "  No listening activity recorded yet.")
	} else {
		for i, track := range summary.Top {
			name := track.Name
			if track.Artist != "" {
				name = track.Artist + " — " + name
			}
			lines = append(lines, fmt.Sprintf("  %2d. %-38s %s", i+1, truncate(name, 38), formatListeningSeconds(track.ListeningSeconds)))
		}
	}
	lines = append(lines, "", "j/k:scroll  PgUp/PgDn  g/G  Esc/q:close")
	m.commandHelp.show = true
	m.commandHelp.topic = ""
	m.commandHelp.lines = lines
	m.commandHelp.offset = 0
	m.commandHelp.historyLines = lines
	return nil
}

func formatListeningSeconds(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	if d >= time.Hour {
		return fmt.Sprintf("%dh %02dm", int(d/time.Hour), int(d%time.Hour/time.Minute))
	}
	return fmt.Sprintf("%dm %02ds", int(d/time.Minute), int(d%time.Minute/time.Second))
}
func (m *Model) ClearHistory() error {
	if err := m.commandLine.history.Clear(); err != nil {
		return err
	}
	m.notify("Command history cleared.")
	return nil
}
func (m *Model) Notify(message string) { m.notify(message) }
func (m *Model) Quit(force bool) (tea.Cmd, error) {
	if m.downloadCancel != nil && !force {
		return nil, &command.RuntimeCommandError{Message: "A download is active. Use :q! to cancel it and quit."}
	}
	m.shutdown()
	return tea.Quit, nil
}
func (m *Model) TrackCompletions(query string, limit int) []command.CompletionItem {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []command.CompletionItem
	seen := map[string]bool{}
	for _, f := range m.folders {
		for _, t := range f.Tracks {
			info := m.trackSearchInfo(t)
			value := info.display
			hay := info.search
			if q != "" && !strings.Contains(hay, q) {
				continue
			}
			key := value + "\x00" + f.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, command.CompletionItem{Value: value, Display: value, Description: f.Name, Kind: command.CompletionArgument})
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}
