package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Padrosum/pmusic/internal/catalog"
	pfs "github.com/Padrosum/pmusic/internal/fs"
	"github.com/Padrosum/pmusic/internal/ui/command"
)

func (m *Model) loadCatalog(folders []*pfs.Folder) (*catalog.Overlay, error) {
	opt := catalog.Options{MusicDir: m.rootDir}
	if m.catalogConverter != nil {
		opt.Converter = m.catalogConverter
	}
	if m.catalogConfigDir != "" {
		opt.ConfigDir = m.catalogConfigDir
	}
	return catalog.Load(context.Background(), folders, opt)
}

func (m *Model) applyCatalog(overlay *catalog.Overlay, loadErr error) {
	m.catalog = overlay
	if m.viewingPlaylist {
		if pl, ok := m.playlistAt(m.playlistIdx); ok {
			m.setCatalogTracks(pl.Name, pl.Tracks)
		} else {
			m.clearCatalogView()
		}
	} else if m.viewingCatalog && m.catalogTitle != "" && overlay != nil {
		if name, tracks, _, ok := overlay.TracksOfType(m.catalogTitle); ok {
			m.setCatalogTracks(name, tracks)
		}
	}
	m.trackSearchCache = make(map[string]trackSearchInfo)
	if loadErr != nil {
		m.notify(catalogLoadMessage(loadErr))
	}
}

func catalogLoadMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, catalog.ErrGenLangMissing) {
		return "library.gl found, but genlang is not on PATH"
	}
	return "catalog: " + err.Error()
}

func (m *Model) playlists() []catalog.Playlist {
	if m.catalog == nil {
		return nil
	}
	return m.catalog.Playlists
}

func (m *Model) playlistAt(i int) (catalog.Playlist, bool) {
	lists := m.playlists()
	if i < 0 || i >= len(lists) {
		return catalog.Playlist{}, false
	}
	return lists[i], true
}

func (m *Model) leftLen() int {
	return len(m.folders) + len(m.playlists())
}

func (m *Model) leftCursor() int {
	if m.viewingPlaylist {
		return len(m.folders) + m.playlistIdx
	}
	if m.folderIdx < 0 {
		return 0
	}
	return m.folderIdx
}

func (m *Model) selectLeft(i int) {
	n := m.leftLen()
	if n == 0 {
		m.clearCatalogView()
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	if i < len(m.folders) {
		m.folderIdx = i
		m.viewingPlaylist = false
		m.viewingCatalog = false
		m.catalogTitle = ""
		m.catalogTracks = nil
		m.trackIdx = 0
		return
	}
	m.playlistIdx = i - len(m.folders)
	m.viewingPlaylist = true
	if pl, ok := m.playlistAt(m.playlistIdx); ok {
		m.setCatalogTracks(pl.Name, pl.Tracks)
	}
	m.trackIdx = 0
}

func (m *Model) clearCatalogView() {
	m.viewingPlaylist = false
	m.viewingCatalog = false
	m.catalogTitle = ""
	m.catalogTracks = nil
}

func (m *Model) setCatalogTracks(title string, tracks []pfs.Track) {
	m.viewingCatalog = true
	m.catalogTitle = title
	m.catalogTracks = tracks
}

func (m *Model) OpenPlaylist(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return m.ListPlaylists()
	}
	if m.catalog == nil {
		return &command.RuntimeCommandError{Message: "No GenLang catalog is loaded. Place library.gl in ~/.config/pmusic/ or your music directory and install genlang."}
	}
	pl, ok := m.catalog.Playlist(name)
	if !ok {
		return &command.InvalidArgumentError{Message: fmt.Sprintf("Unknown playlist %q.", name), Usage: ":playlist [list|show|queue] <name>"}
	}
	for i, item := range m.playlists() {
		if strings.EqualFold(item.Name, pl.Name) {
			m.selectLeft(len(m.folders) + i)
			m.focused = panelTracks
			if len(pl.Missing) > 0 {
				m.notify(fmt.Sprintf("%s: %d tracks, %d unmatched", pl.Name, len(pl.Tracks), len(pl.Missing)))
			} else {
				m.notify(fmt.Sprintf("Playlist %s: %d tracks", pl.Name, len(pl.Tracks)))
			}
			return nil
		}
	}
	m.setCatalogTracks(pl.Name, pl.Tracks)
	m.focused = panelTracks
	return nil
}

func (m *Model) QueuePlaylist(name string) (int, error) {
	if m.catalog == nil {
		return 0, &command.RuntimeCommandError{Message: "No GenLang catalog is loaded."}
	}
	pl, ok := m.catalog.Playlist(name)
	if !ok {
		return 0, &command.InvalidArgumentError{Message: fmt.Sprintf("Unknown playlist %q.", name), Usage: ":playlist queue <name>"}
	}
	if len(pl.Tracks) == 0 {
		return 0, &command.RuntimeCommandError{Message: fmt.Sprintf("Playlist %s has no matching local tracks.", pl.Name)}
	}
	m.queue = append(m.queue, pl.Tracks...)
	if err := m.persistQueue(); err != nil {
		return 0, err
	}
	return len(pl.Tracks), nil
}

func (m *Model) ListPlaylists() error {
	lines := []string{"Playlists", ""}
	if m.catalog == nil {
		lines = append(lines,
			"No library.gl catalog is loaded.",
			"",
			"Place a GenLang file at ~/.config/pmusic/library.gl",
			"or <music-dir>/library.gl, then install genlang.",
			"See examples/library.gl.",
		)
	} else if len(m.catalog.Playlists) == 0 {
		lines = append(lines, "The catalog has no sets (kume).")
	} else {
		for _, pl := range m.catalog.Playlists {
			lines = append(lines, fmt.Sprintf("  %-20s %d tracks", pl.Name, len(pl.Tracks)))
			if len(pl.Missing) > 0 {
				lines = append(lines, fmt.Sprintf("    unmatched: %s", strings.Join(pl.Missing, ", ")))
			}
		}
		lines = append(lines, "", ":playlist <name> to open  :playlist queue <name> to queue")
	}
	lines = append(lines, "", "j/k:scroll  Esc/q:close")
	m.showHelpOverlay(lines)
	return nil
}

func (m *Model) OpenType(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return m.listTypes()
	}
	if m.catalog == nil {
		return &command.RuntimeCommandError{Message: "No GenLang catalog is loaded."}
	}
	typeName, tracks, missing, ok := m.catalog.TracksOfType(name)
	if !ok {
		return &command.InvalidArgumentError{Message: fmt.Sprintf("Unknown type %q.", name), Usage: ":type <name>"}
	}
	m.viewingPlaylist = false
	m.setCatalogTracks(typeName, tracks)
	m.trackIdx = 0
	m.focused = panelTracks
	if len(missing) > 0 {
		m.notify(fmt.Sprintf("%s: %d tracks, %d unmatched", typeName, len(tracks), len(missing)))
	} else {
		m.notify(fmt.Sprintf("Type %s: %d tracks", typeName, len(tracks)))
	}
	return nil
}

func (m *Model) listTypes() error {
	lines := []string{"Types", ""}
	if m.catalog == nil {
		lines = append(lines, "No library.gl catalog is loaded.")
	} else if len(m.catalog.Doc.Types) == 0 {
		lines = append(lines, "The catalog has no types.")
	} else {
		for _, t := range m.catalog.Doc.Types {
			parent := ""
			if t.Parent != nil && *t.Parent != "" {
				parent = " -> " + *t.Parent
			}
			lines = append(lines, fmt.Sprintf("  %s %s%s", t.Kind, t.Name, parent))
		}
		lines = append(lines, "", ":type Parca to show matching local tracks")
	}
	lines = append(lines, "", "j/k:scroll  Esc/q:close")
	m.showHelpOverlay(lines)
	return nil
}

func (m *Model) InspectSelection() error {
	track, ok := m.selectedOrPlaying()
	if !ok {
		return &command.RuntimeCommandError{Message: "No track is selected."}
	}
	lines := []string{"Catalog", "", "  " + track.Name, "  " + track.Path, ""}
	if m.catalog == nil {
		lines = append(lines, "No library.gl catalog is loaded.")
	} else if info, found := m.catalog.EntityByPath(track.Path); found {
		lines = append(lines, "Types")
		if len(info.Types) == 0 {
			lines = append(lines, "  (none)")
		} else {
			for _, name := range info.Types {
				lines = append(lines, "  "+name)
			}
		}
		lines = append(lines, "", "Sets")
		if len(info.Sets) == 0 {
			lines = append(lines, "  (none)")
		} else {
			for _, name := range info.Sets {
				lines = append(lines, "  "+name)
			}
		}
		if info.Title != "" && info.Title != track.Name {
			lines = append(lines, "", "  title   "+info.Title)
		}
		if info.Artist != "" {
			lines = append(lines, "  artist  "+info.Artist)
		}
		if info.Album != "" {
			lines = append(lines, "  album   "+info.Album)
		}
	} else {
		lines = append(lines, "This track has no catalog entry.")
	}
	lines = append(lines, "", "j/k:scroll  Esc/q:close")
	m.showHelpOverlay(lines)
	return nil
}

func (m *Model) selectedOrPlaying() (pfs.Track, bool) {
	tracks := m.currentTracks()
	if len(tracks) > 0 && m.trackIdx >= 0 && m.trackIdx < len(tracks) {
		return tracks[m.trackIdx], true
	}
	if m.nowPlaying != nil {
		return *m.nowPlaying, true
	}
	return pfs.Track{}, false
}

func (m *Model) showHelpOverlay(lines []string) {
	m.commandHelp.show = true
	m.commandHelp.topic = ""
	m.commandHelp.lines = lines
	m.commandHelp.offset = 0
	m.commandHelp.historyLines = lines
}

func (m *Model) PlaylistCompletions(query string, limit int) []command.CompletionItem {
	return m.nameCompletions(m.catalog.SetNames(), query, limit)
}

func (m *Model) TypeCompletions(query string, limit int) []command.CompletionItem {
	return m.nameCompletions(m.catalog.TypeNames(), query, limit)
}

func (m *Model) nameCompletions(names []string, query string, limit int) []command.CompletionItem {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []command.CompletionItem
	for _, name := range names {
		if q != "" && !strings.Contains(strings.ToLower(name), q) {
			continue
		}
		out = append(out, command.CompletionItem{Value: name, Display: name, Kind: command.CompletionArgument})
		if len(out) >= limit {
			return out
		}
	}
	return out
}

func (m *Model) enqueueCatalogOrFolder() bool {
	if m.focused != panelFolders {
		return false
	}
	if m.viewingPlaylist || m.viewingCatalog {
		tracks := m.currentTracks()
		if len(tracks) == 0 {
			return true
		}
		m.queue = append(m.queue, tracks...)
		m.saveQueue()
		label := m.catalogTitle
		if label == "" {
			label = "list"
		}
		m.notify(fmt.Sprintf("queued %s: %d tracks (%d)", label, len(tracks), len(m.queue)))
		return true
	}
	return false
}
