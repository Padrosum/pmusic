package ui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Padrosum/pmusic/internal/blackjack"
	pmcfg "github.com/Padrosum/pmusic/internal/config"
	pmdownload "github.com/Padrosum/pmusic/internal/download"
	pfs "github.com/Padrosum/pmusic/internal/fs"
	"github.com/Padrosum/pmusic/internal/inhibit"
	"github.com/Padrosum/pmusic/internal/listening"
	luaeng "github.com/Padrosum/pmusic/internal/lua"
	"github.com/Padrosum/pmusic/internal/meta"
	"github.com/Padrosum/pmusic/internal/player"
	pmsearch "github.com/Padrosum/pmusic/internal/search"
	pmstore "github.com/Padrosum/pmusic/internal/store"
	"github.com/Padrosum/pmusic/internal/watcher"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type panel int

const (
	panelFolders panel = iota
	panelTracks
)

type tickMsg time.Time

// luaReloadedMsg is sent back to the Update loop after a Lua hot-reload attempt.
type luaReloadedMsg struct{ err error }
type libraryReloadedMsg struct {
	root    *pfs.Folder
	folders []*pfs.Folder
	err     error
}

type playbackOrigin string

const (
	playbackSelected playbackOrigin = "selection"
	playbackNext     playbackOrigin = "next"
	playbackPrevious playbackOrigin = "previous"
	playbackLoop     playbackOrigin = "loop"
	playbackQueue    playbackOrigin = "queue"
	playbackMouse    playbackOrigin = "mouse"
)

type playbackStartedMsg struct {
	track      pfs.Track
	origin     playbackOrigin
	folder     int
	trackIndex int
	requestID  uint64
}

type playbackFailedMsg struct {
	track     pfs.Track
	origin    playbackOrigin
	err       error
	requestID uint64
}

// mascot animation constants — each frame line is rendered at fixed width via lipgloss.
const mascotW = 8

var mascotPlaying = [4][3]string{
	{`/\_/\`, `(^.^)`, `>♪ < `},
	{`/\_/\`, `(^o^)`, `>♫ < `},
	{`/\_/\`, `(>.~)`, `>♪ < `},
	{`/\_/\`, `(^-^)`, `>♫ < `},
}
var mascotPaused = [3]string{`/\_/\`, `(-_-)`, `~♩~ `}
var mascotStopped = [3]string{`/\_/\`, `(z.z)`, ` zzz`}

type storeEntry struct {
	pmstore.Item
	Installed bool
	Enabled   bool
}

type Model struct {
	width, height int
	focused       panel

	root    *pfs.Folder
	folders []*pfs.Folder

	folderIdx int
	trackIdx  int

	player     *player.Player
	inhibitor  *inhibit.Inhibitor
	nowPlaying *pfs.Track
	nowFolder  int
	nowTrack   int
	loop       bool

	mascotFrame int
	uiFrame     int

	nowMeta  meta.Meta
	showHelp bool

	showStore   bool
	storeItems  []storeEntry
	storeCursor int
	storeTab    int // 0=plugins 1=themes

	showMusicSearch bool
	musicSearch     musicSearchModel
	searchProvider  pmsearch.Provider
	urlResolver     pmsearch.URLResolver
	downloader      *pmdownload.Downloader
	searchCancel    context.CancelFunc
	downloadCancel  context.CancelFunc

	showSearch         bool
	searchInput        textinput.Model
	searchResults      []pfs.Track
	searchQuery        string
	searchResultsValid bool

	showBlackjack bool
	bjGame        *blackjack.Game

	// Play queue (session-only, in-memory). Queued tracks take priority over
	// sequential auto-advance; see playFromQueue.
	queue       []pfs.Track
	showQueue   bool
	queueCursor int
	saveQueueFn func([]pfs.Track) error

	watcher *watcher.Watcher
	rootDir string

	luaEngine *luaeng.Engine

	// Lua hook state tracking
	prevPlayingPath string
	prevState       player.State

	// Notification overlay (shown in status bar, expires after a short time)
	notification string
	notifyUntil  time.Time

	commandLine       commandLineModel
	commandHelp       commandHelpModel
	mutedVolume       int
	trackSearchCache  map[string]trackSearchInfo
	listening         *listening.Store
	lastListeningTick time.Time
	playGeneration    uint64
	statsGeneration   uint64
	playbackRequestID uint64
	playbackFailed    bool
	reducingGlobal    bool
	closeOnce         sync.Once
	closeErr          error
}

func New(rootDir string) (*Model, error) {
	root, err := pfs.Scan(rootDir)
	if err != nil {
		return nil, err
	}
	folders := pfs.FlatFolders(root)

	p, err := player.New()
	if err != nil {
		return nil, fmt.Errorf("audio: %w", err)
	}
	si := textinput.New()
	si.Placeholder = "Search tracks (/)..."
	si.CharLimit = 100

	m := &Model{
		root:             root,
		folders:          folders,
		player:           p,
		rootDir:          rootDir,
		searchInput:      si,
		musicSearch:      newMusicSearchModel(),
		downloader:       pmdownload.New(),
		commandHelp:      newCommandHelpModel(),
		trackSearchCache: make(map[string]trackSearchInfo),
	}
	cl, err := newCommandLine()
	if err != nil {
		return nil, fmt.Errorf("command system: %w", err)
	}
	m.commandLine = cl
	if cl.initWarning != nil {
		m.notification = "history: " + cl.initWarning.Error()
		m.notifyUntil = time.Now().Add(5 * time.Second)
	}
	statsPath, statsPathErr := listening.DefaultPath()
	if statsPathErr == nil {
		m.listening, err = listening.Load(statsPath)
	}
	if m.listening == nil {
		m.listening, err = listening.Load("")
		if err != nil {
			return nil, fmt.Errorf("initialize in-memory listening store: %w", err)
		}
	}
	if statsPathErr != nil || err != nil {
		statsErr := err
		if statsErr == nil {
			statsErr = statsPathErr
		}
		m.notification = "listening stats: " + statsErr.Error()
		m.notifyUntil = time.Now().Add(5 * time.Second)
	}
	youtube := pmsearch.NewYouTube()
	m.searchProvider = youtube
	m.urlResolver = youtube

	q, queueErr := pmcfg.LoadQueue()
	if q != nil {
		m.queue = q
	}
	if queueErr != nil {
		m.notification = "queue: " + queueErr.Error()
		m.notifyUntil = time.Now().Add(10 * time.Second)
	}

	p.SetOnDone(func() {
		// handled via tickMsg checking state
	})

	w, err := watcher.New(rootDir, func() {
		// rescan signal handled in Update
	})
	if err == nil {
		m.watcher = w
	} else {
		m.notification = "library watcher: " + err.Error()
		m.notifyUntil = time.Now().Add(10 * time.Second)
	}

	eng := luaeng.New()
	eng.SetMusicDir(rootDir)
	if err := eng.Load(); err != nil {
		m.notification = "lua: " + err.Error()
		m.notifyUntil = time.Now().Add(10 * time.Second)
	}
	m.luaEngine = eng
	applyTheme(eng.Theme())

	inh, inhErr := inhibit.New()
	if inhErr != nil {
		m.notification = "screen inhibit: " + inhErr.Error()
		m.notifyUntil = time.Now().Add(10 * time.Second)
	}
	m.inhibitor = inh

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		tea.SetWindowTitle("pmusic"),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) tickCmd() tea.Cmd {
	interval := m.tickInterval()
	return tea.Tick(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *Model) tickInterval() time.Duration {
	interval := time.Second
	if m.player != nil && m.player.State() == player.Playing {
		interval = time.Second / 4
	}
	if m.showMusicSearch && (m.musicSearch.state == musicSearchLoading || m.musicSearch.state == musicSearchDownloading) {
		interval = time.Second / 4
	}
	return interval
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := m.handleMusicSearchMessage(msg); handled {
		return m, cmd
	}
	// Overlays own terminal input, not application lifecycle messages. Keeping
	// resize, ticks, reloads, and playback results on the shared reducer path
	// prevents an open modal from freezing or dropping background work.
	if !isTerminalInput(msg) && !m.reducingGlobal {
		return m.reduceGlobal(msg)
	}
	// Command input owns every key while active, before overlays, Lua keymaps,
	// and global shortcuts. This prevents typed q/n/p/space/j/k from leaking.
	if !m.reducingGlobal && m.commandLine.active {
		return m.handleCommandLine(msg)
	}
	if !m.reducingGlobal && m.commandHelp.show {
		return m.handleCommandHelp(msg)
	}

	if !m.reducingGlobal && m.showMusicSearch {
		return m.handleMusicSearch(msg)
	}

	if !m.reducingGlobal && m.showSearch {
		before := m.searchInput.Value()
		var inputCmd tea.Cmd
		m.searchInput, inputCmd = m.searchInput.Update(msg)
		if before != m.searchInput.Value() {
			m.searchResultsValid = false
		}
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
		case tickMsg:
			return m, tea.Batch(m.tickCmd(), inputCmd)
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc, tea.KeyEnter:
				m.showSearch = false
				m.searchInput.Blur()
				return m, nil
			}
		}
		// Reset track index when search query changes
		m.trackIdx = 0
		return m, inputCmd
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case luaReloadedMsg:
		if msg.err != nil {
			m.notification = "lua error: " + msg.err.Error()
		} else {
			m.notification = "lua reloaded"
		}
		m.notifyUntil = time.Now().Add(5 * time.Second)
		applyTheme(m.luaEngine.Theme())
		return m, nil

	case libraryReloadedMsg:
		if msg.err != nil {
			m.notify("library reload: " + msg.err.Error())
		} else {
			m.root, m.folders = msg.root, msg.folders
			m.trackSearchCache = make(map[string]trackSearchInfo)
			m.searchResultsValid = false
			m.folderIdx = min(m.folderIdx, max(0, len(m.folders)-1))
			m.trackIdx = 0
			m.notify(fmt.Sprintf("Library reloaded: %d folders", len(m.folders)))
		}
		return m, nil

	case playbackStartedMsg:
		if msg.requestID != m.playbackRequestID {
			return m, nil
		}
		track := msg.track
		m.nowPlaying = &track
		if msg.folder >= 0 && msg.trackIndex >= 0 {
			m.nowFolder, m.nowTrack = msg.folder, msg.trackIndex
		}
		m.playbackFailed = false
		return m, nil

	case playbackFailedMsg:
		if msg.requestID != m.playbackRequestID {
			return m, nil
		}
		m.playbackFailed = true
		m.notify(fmt.Sprintf("Could not play %s: %v. Playback stopped. (%s)", msg.track.Name, msg.err, msg.track.Path))
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		m.uiFrame = (m.uiFrame + 1) % 120
		// Advance mascot animation frame while playing.
		if m.player.State() == player.Playing {
			m.mascotFrame = (m.mascotFrame + 1) % 4
		}

		// Pop any pending Lua notifications.
		if n := m.luaEngine.PopNotification(); n != "" {
			m.notification = n
			m.notifyUntil = time.Now().Add(5 * time.Second)
		}

		// Rescan library if filesystem changed.
		if m.watcher != nil && m.watcher.Changed() {
			m.rescan()
		}
		if m.watcher != nil {
			if err := m.watcher.TakeError(); err != nil {
				m.notify("library watcher: " + err.Error())
			}
		}

		// Fire on_song_change hook and load metadata when the playing track changes.
		trackChanged := m.nowPlaying != nil && m.nowPlaying.Path != m.prevPlayingPath
		playbackChanged := m.playGeneration != m.statsGeneration
		if trackChanged {
			m.prevPlayingPath = m.nowPlaying.Path
			m.nowMeta = m.trackSearchInfo(*m.nowPlaying).meta
			folder := ""
			if m.nowFolder < len(m.folders) {
				folder = m.folders[m.nowFolder].Name
			}
			m.luaEngine.CallOnSongChange(m.nowPlaying.Name, m.nowPlaying.Path, folder)
		}

		// Cache state once so both the hook check and auto-advance use the same snapshot.
		st := m.player.State()
		wasPlaying := m.prevState == player.Playing
		if m.listening != nil {
			if m.nowPlaying != nil && st != player.Stopped {
				track := listening.Track{Path: m.nowPlaying.Path, Name: m.nowPlaying.Name, Artist: m.nowMeta.Artist, Album: m.nowMeta.Album}
				if playbackChanged {
					m.listening.Restart(track, now)
				} else {
					m.listening.Start(track, now)
				}
				m.statsGeneration = m.playGeneration
			}
			if st == player.Playing && wasPlaying && !playbackChanged && !trackChanged && !m.lastListeningTick.IsZero() {
				m.listening.Listen(now.Sub(m.lastListeningTick), now)
			}
			m.lastListeningTick = now
			if m.listening.ShouldSave(now) {
				if err := m.listening.Save(); err != nil {
					m.notify("listening stats: " + err.Error())
				}
			}
		}

		// Fire on_state_change hook when play/pause/stop state transitions.
		if st != m.prevState {
			m.prevState = st
			var stateStr string
			switch st {
			case player.Playing:
				stateStr = "playing"
				if m.inhibitor != nil {
					_ = m.inhibitor.Inhibit()
				}
			case player.Paused:
				stateStr = "paused"
				if m.inhibitor != nil {
					_ = m.inhibitor.UnInhibit()
				}
			default:
				stateStr = "stopped"
				if m.inhibitor != nil {
					_ = m.inhibitor.UnInhibit()
				}
			}
			m.luaEngine.CallOnStateChange(stateStr)
		}

		// Auto-advance (or loop) when a track ends naturally.
		if m.nowPlaying != nil && st == player.Stopped && !m.playbackFailed {
			if m.listening != nil {
				m.listening.FinishPath(m.nowPlaying.Path, true, now)
			}
			var cmd tea.Cmd
			if m.loop {
				cmd = m.replayCurrent()
			} else if c := m.playFromQueue(); c != nil {
				cmd = c
			} else {
				cmd = m.playNext()
			}
			if cmd == nil {
				m.nowPlaying = nil
				m.nowMeta = meta.Meta{}
			}
			return m, tea.Batch(m.tickCmd(), cmd)
		}
		return m, m.tickCmd()

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// Store overlay intercepts all keys.
		if m.showStore {
			return m.handleStore(msg)
		}

		// Queue overlay intercepts all keys.
		if m.showQueue {
			return m.handleQueue(msg)
		}

		// Help overlay intercepts all keys — only ? and q close it.
		if m.showHelp {
			if key.Matches(msg, keys.Help) || key.Matches(msg, keys.Quit) {
				m.showHelp = false
			}
			return m, nil
		}

		// Blackjack overlay intercepts all keys.
		if m.showBlackjack {
			return m.updateBlackjack(msg)
		}

		// Text-input overlays are handled above. Other overlays intentionally
		// intercept ':' rather than allowing two simultaneous modal owners.
		if msg.String() == ":" {
			return m, m.openCommandLine()
		}

		// Built-in Download panel takes priority over any Lua keymap on the same key.
		if key.Matches(msg, keys.Download) {
			return m, m.openMusicSearch()
		}

		// Check Lua-registered custom keymaps before built-in bindings.
		if action := m.luaEngine.Keymap(msg.String()); action != "" {
			if cmd := m.dispatchAction(action); cmd != nil {
				return m, cmd
			}
			return m, nil
		}

		// Check Lua function keymaps (register_keymap with a function).
		if m.luaEngine.HasKeyFunc(msg.String()) {
			if err := m.luaEngine.CallKeyFunc(msg.String()); err != nil {
				m.notification = "lua: " + err.Error()
				m.notifyUntil = time.Now().Add(5 * time.Second)
			}
			if n := m.luaEngine.PopNotification(); n != "" {
				m.notification = n
				m.notifyUntil = time.Now().Add(5 * time.Second)
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			m.shutdown()
			return m, tea.Quit

		case key.Matches(msg, keys.ReloadLua):
			return m, m.reloadLuaCmd()

		case key.Matches(msg, keys.Left):
			m.focused = panelFolders

		case key.Matches(msg, keys.Right):
			m.focused = panelTracks

		case key.Matches(msg, keys.Up):
			m.moveUp()

		case key.Matches(msg, keys.Down):
			m.moveDown()

		case key.Matches(msg, keys.Enter):
			if m.focused == panelTracks {
				return m, m.playSelected()
			} else {
				m.focused = panelTracks
			}

		case key.Matches(msg, keys.Space):
			m.player.TogglePause()

		case key.Matches(msg, keys.Next):
			return m, m.playNext()

		case key.Matches(msg, keys.Prev):
			return m, m.playPrev()

		case key.Matches(msg, keys.Loop):
			m.loop = !m.loop

		case key.Matches(msg, keys.VolUp):
			m.player.SetVolume(m.player.Volume() + 0.1)

		case key.Matches(msg, keys.VolDown):
			m.player.SetVolume(m.player.Volume() - 0.1)

		case key.Matches(msg, keys.SeekBack5):
			m.player.Seek(-5 * time.Second)

		case key.Matches(msg, keys.SeekFwd5):
			m.player.Seek(5 * time.Second)

		case key.Matches(msg, keys.SeekBack30):
			m.player.Seek(-30 * time.Second)

		case key.Matches(msg, keys.SeekFwd30):
			m.player.Seek(30 * time.Second)

		case key.Matches(msg, keys.Help):
			m.showHelp = true

		case key.Matches(msg, keys.Store):
			m.loadStoreItems()
			m.showStore = true
			m.storeCursor = 0

		case key.Matches(msg, keys.Blackjack):
			if m.bjGame == nil {
				m.bjGame = blackjack.New()
			}
			m.showBlackjack = true

		case key.Matches(msg, keys.AddQueue):
			m.enqueueSelected()

		case key.Matches(msg, keys.Queue):
			m.showQueue = true
			m.queueCursor = 0

		case key.Matches(msg, keys.Search):
			m.showSearch = true
			m.searchInput.SetValue("")
			return m, m.searchInput.Focus()

		}
	}
	return m, nil
}

func isTerminalInput(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		return true
	default:
		return false
	}
}

// reduceGlobal feeds non-input messages back through the normal reducer after
// modal routing has been bypassed. The concrete cases remain in Update so
// there is still one implementation for every lifecycle transition.
func (m *Model) reduceGlobal(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.reducingGlobal = true
	defer func() { m.reducingGlobal = false }()
	return m.Update(msg)
}

// dispatchAction executes a named action — used by Lua keymaps.
func (m *Model) dispatchAction(action string) tea.Cmd {
	switch action {
	case "toggle_pause":
		m.player.TogglePause()
	case "next":
		return m.playNext()
	case "prev":
		return m.playPrev()
	case "loop":
		m.loop = !m.loop
	case "focus_folders":
		m.focused = panelFolders
	case "focus_tracks":
		m.focused = panelTracks
	case "reload_lua":
		return m.reloadLuaCmd()
	case "quit":
		m.shutdown()
		return tea.Quit
	case "vol_up":
		m.player.SetVolume(m.player.Volume() + 0.1)
	case "vol_down":
		m.player.SetVolume(m.player.Volume() - 0.1)
	case "seek_back5":
		m.player.Seek(-5 * time.Second)
	case "seek_fwd5":
		m.player.Seek(5 * time.Second)
	case "seek_back30":
		m.player.Seek(-30 * time.Second)
	case "seek_fwd30":
		m.player.Seek(30 * time.Second)
	}
	return nil
}

func (m *Model) shutdown() {
	if err := m.Close(); err != nil {
		m.notify("shutdown: " + err.Error())
	}
}

// Close is the single idempotent lifecycle boundary for model-owned resources.
func (m *Model) Close() error {
	m.closeOnce.Do(func() {
		var closeErrors []error
		if m.searchCancel != nil {
			m.searchCancel()
		}
		if m.downloadCancel != nil {
			m.downloadCancel()
		}
		if m.listening != nil {
			closeErrors = append(closeErrors, m.listening.Save())
		}
		if m.player != nil {
			closeErrors = append(closeErrors, m.player.Close())
		}
		if m.watcher != nil {
			closeErrors = append(closeErrors, m.watcher.Close())
		}
		if m.luaEngine != nil {
			m.luaEngine.Close()
		}
		if m.inhibitor != nil {
			closeErrors = append(closeErrors, m.inhibitor.Close())
		}
		m.closeErr = errors.Join(closeErrors...)
	})
	return m.closeErr
}

func (m *Model) markPlaybackPending() {
	m.playGeneration++
	m.playbackFailed = false
	m.player.MarkPending()
}

func (m *Model) startPlayback(track pfs.Track, origin playbackOrigin, folder, trackIndex int) tea.Cmd {
	m.playbackRequestID++
	requestID := m.playbackRequestID
	m.markPlaybackPending()
	return func() tea.Msg {
		if err := m.player.Play(track.Path); err != nil {
			return playbackFailedMsg{track: track, origin: origin, err: err, requestID: requestID}
		}
		return playbackStartedMsg{track: track, origin: origin, folder: folder, trackIndex: trackIndex, requestID: requestID}
	}
}

// reloadLuaCmd returns a Cmd that reloads the Lua VM in a background goroutine.
func (m *Model) reloadLuaCmd() tea.Cmd {
	eng := m.luaEngine
	return func() tea.Msg {
		return luaReloadedMsg{err: eng.Load()}
	}
}

func (m *Model) moveUp() {
	if m.focused == panelFolders {
		if m.folderIdx > 0 {
			m.folderIdx--
			m.trackIdx = 0
		}
	} else {
		if m.trackIdx > 0 {
			m.trackIdx--
		}
	}
}

func (m *Model) moveDown() {
	if m.focused == panelFolders {
		if m.folderIdx < len(m.folders)-1 {
			m.folderIdx++
			m.trackIdx = 0
		}
	} else {
		tracks := m.currentTracks()
		if m.trackIdx < len(tracks)-1 {
			m.trackIdx++
		}
	}
}

func (m *Model) currentTracks() []pfs.Track {
	if len(m.folders) == 0 {
		return nil
	}
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		return m.folders[m.folderIdx].Tracks
	}
	if m.searchResultsValid && query == m.searchQuery {
		return m.searchResults
	}
	var filtered []pfs.Track
	for _, f := range m.folders {
		for _, t := range f.Tracks {
			if strings.Contains(m.trackSearchInfo(t).search, query) {
				filtered = append(filtered, t)
			}
		}
	}
	m.searchQuery, m.searchResults, m.searchResultsValid = query, filtered, true
	return m.searchResults
}

func (m *Model) playSelected() tea.Cmd {
	tracks := m.currentTracks()
	if len(tracks) == 0 || m.trackIdx >= len(tracks) {
		return nil
	}
	t := tracks[m.trackIdx]
	fi, ti := m.folderIdx, m.trackIdx
	if fi, ti, ok := m.locateTrack(t.Path); ok {
		return m.startPlayback(t, playbackSelected, fi, ti)
	}
	return m.startPlayback(t, playbackSelected, fi, ti)
}

func (m *Model) playNext() tea.Cmd {
	if len(m.folders) == 0 {
		return nil
	}
	var fi, ti int
	if m.nowPlaying == nil {
		// Nothing playing: start from cursor.
		fi = m.folderIdx
		ti = m.trackIdx
	} else {
		fi = m.nowFolder
		ti = m.nowTrack + 1
		if ti >= len(m.folders[fi].Tracks) {
			fi++
			ti = 0
		}
	}
	if fi >= len(m.folders) || len(m.folders[fi].Tracks) == 0 {
		return nil
	}
	t := m.folders[fi].Tracks[ti]
	return m.startPlayback(t, playbackNext, fi, ti)
}

func (m *Model) playPrev() tea.Cmd {
	if len(m.folders) == 0 {
		return nil
	}
	var fi, ti int
	if m.nowPlaying == nil {
		fi = m.folderIdx
		ti = m.trackIdx
	} else {
		fi = m.nowFolder
		ti = m.nowTrack - 1
		if ti < 0 {
			fi--
			if fi < 0 {
				fi = 0
				ti = 0
			} else {
				ti = len(m.folders[fi].Tracks) - 1
			}
		}
	}
	if fi >= len(m.folders) || len(m.folders[fi].Tracks) == 0 {
		return nil
	}
	t := m.folders[fi].Tracks[ti]
	return m.startPlayback(t, playbackPrevious, fi, ti)
}

func (m *Model) replayCurrent() tea.Cmd {
	if m.nowPlaying == nil {
		return nil
	}
	fi := m.nowFolder
	ti := m.nowTrack
	if fi >= len(m.folders) || ti >= len(m.folders[fi].Tracks) {
		return nil
	}
	t := m.folders[fi].Tracks[ti]
	return m.startPlayback(t, playbackLoop, fi, ti)
}

// locateTrack returns the folder/track indices of the track with the given path.
func (m *Model) locateTrack(path string) (int, int, bool) {
	for fi, f := range m.folders {
		for ti, t := range f.Tracks {
			if t.Path == path {
				return fi, ti, true
			}
		}
	}
	return 0, 0, false
}

func (m *Model) persistQueue() error {
	if m.saveQueueFn != nil {
		return m.saveQueueFn(m.queue)
	}
	return pmcfg.SaveQueue(m.queue)
}

func (m *Model) saveQueue() {
	if err := m.persistQueue(); err != nil {
		m.notify("queue save: " + err.Error())
	}
}

// enqueueSelected appends the cursor's track (or the whole highlighted folder)
// to the play queue and shows a confirmation notification.
func (m *Model) enqueueSelected() {
	if m.focused == panelFolders {
		tracks := m.currentTracks()
		if len(tracks) == 0 {
			return
		}
		m.queue = append(m.queue, tracks...)
		m.saveQueue()
		m.notify(fmt.Sprintf("folder queued: %d tracks (%d)", len(tracks), len(m.queue)))
		return
	}
	tracks := m.currentTracks()
	if len(tracks) == 0 || m.trackIdx >= len(tracks) {
		return
	}
	t := tracks[m.trackIdx]
	m.queue = append(m.queue, t)
	m.saveQueue()
	m.notify(fmt.Sprintf("queued: %s  (%d)", t.Name, len(m.queue)))
}

// notify shows msg in the status bar for ~5 seconds.
func (m *Model) notify(msg string) {
	m.notification = msg
	m.notifyUntil = time.Now().Add(5 * time.Second)
}

// playFromQueue pops the head of the queue and plays it, returning nil when the
// queue is empty so the caller can fall back to sequential auto-advance.
func (m *Model) playFromQueue() tea.Cmd {
	if len(m.queue) == 0 {
		return nil
	}
	t := m.queue[0]
	m.queue = m.queue[1:]
	m.saveQueue()
	fi, ti := m.nowFolder, m.nowTrack
	// Anchor sequential fallback to this track's library position when possible.
	if foundFolder, foundTrack, ok := m.locateTrack(t.Path); ok {
		fi, ti = foundFolder, foundTrack
	}
	return m.startPlayback(t, playbackQueue, fi, ti)
}

func (m *Model) rescan() {
	root, err := pfs.Scan(m.rootDir)
	if err != nil {
		m.notification = "rescan: " + err.Error()
		m.notifyUntil = time.Now().Add(5 * time.Second)
		return
	}
	m.root = root
	m.folders = pfs.FlatFolders(root)
	m.trackSearchCache = make(map[string]trackSearchInfo)
	m.searchResultsValid = false
	if m.folderIdx >= len(m.folders) {
		m.folderIdx = max(0, len(m.folders)-1)
	}
}

// handleMouse processes mouse events: scroll to navigate, left click to select/play.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.moveUp()
		return m, nil
	case tea.MouseButtonWheelDown:
		m.moveDown()
		return m, nil
	}

	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}

	col, row := msg.X, msg.Y
	leftW := m.width / 3

	header := m.renderHeader(m.width)
	topH := lipgloss.Height(header)

	// mainH = m.height - bottomH(4) - topH - 2
	mainH := m.height - 4 - topH - 2
	innerH := mainH - 2
	visibleRows := innerH - 1 // title row consumed

	if row < topH || row >= topH+mainH {
		return m, nil
	}

	if col < leftW {
		// Left panel (folders): content begins at screen row topH + 2.
		itemRow := row - (topH + 2)
		if itemRow < 0 {
			m.focused = panelFolders
			return m, nil
		}
		offset := scrollOffset(m.folderIdx, visibleRows)
		idx := offset + itemRow
		if idx >= 0 && idx < len(m.folders) {
			m.folderIdx = idx
			m.trackIdx = 0
			m.focused = panelFolders
		}
	} else {
		// Right panel (tracks).
		itemRow := row - (topH + 2)
		if itemRow < 0 {
			m.focused = panelTracks
			return m, nil
		}
		offset := scrollOffset(m.trackIdx, visibleRows)
		idx := offset + itemRow
		tracks := m.currentTracks()
		if idx >= 0 && idx < len(tracks) {
			m.trackIdx = idx
			m.focused = panelTracks
			t := tracks[idx]
			fi, ti := m.folderIdx, idx
			if foundFolder, foundTrack, ok := m.locateTrack(t.Path); ok {
				fi, ti = foundFolder, foundTrack
			}
			return m, m.startPlayback(t, playbackMouse, fi, ti)
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	if m.commandHelp.show {
		return m.renderCommandHelp()
	}
	if m.width < 52 || m.height < 12 {
		if m.commandLine.active {
			return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Bottom, m.renderCommandLine(m.width))
		}
		content := styleLogo.Render("♪ PMUSIC") + "  " + styleVisualizer.Render(visualizer(m.uiFrame, 5)) +
			"\n\n" + styleTitle.Render("Terminal is a little too cozy") +
			"\n" + styleHeaderMeta.Render("Resize to at least 52 × 12")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	if m.showMusicSearch {
		return m.renderMusicSearch()
	}
	if m.showStore {
		return m.renderStore()
	}
	if m.showQueue {
		return m.renderQueue()
	}
	if m.showHelp {
		return m.renderHelp()
	}
	if m.showBlackjack && m.bjGame != nil {
		return blackjack.Render(m.bjGame, m.width, m.height)
	}

	header := m.renderHeader(m.width)

	bottomH := m.commandLineHeight()
	topH := lipgloss.Height(header)
	mainH := m.height - bottomH - topH - 2

	leftW := m.width / 3
	rightW := m.width - leftW - 1

	left := m.renderFolders(leftW, mainH)
	right := m.renderTracks(rightW, mainH)

	middle := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	bottom := m.renderBottom(m.width)
	if m.commandLine.active {
		bottom = m.renderCommandLine(m.width)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, middle, bottom)
}

func (m *Model) renderHeader(w int) string {
	logo := styleLogo.Render("♪ PMUSIC")
	trackCount := 0
	for _, folder := range m.folders {
		trackCount += len(folder.Tracks)
	}

	meta := styleHeaderMeta.Render(fmt.Sprintf("  LIBRARY  %d folders · %d tracks", len(m.folders), trackCount))
	if w < 72 {
		meta = styleHeaderMeta.Render(fmt.Sprintf("  %d folders · %d tracks", len(m.folders), trackCount))
	}

	right := ""
	if len(m.queue) > 0 {
		right += styleBadge.Render(fmt.Sprintf("QUEUE %d", len(m.queue))) + " "
	}
	state := "READY"
	stateStyle := styleHeaderMeta
	if m.nowPlaying != nil {
		switch m.player.State() {
		case player.Playing:
			state = visualizer(m.uiFrame, 4) + " LIVE"
			stateStyle = stylePlaying
		case player.Paused:
			state = "Ⅱ PAUSED"
			stateStyle = stylePaused
		}
	}
	right += stateStyle.Bold(true).Render(state)

	space := w - lipgloss.Width(logo) - lipgloss.Width(meta) - lipgloss.Width(right)
	if space < 1 {
		meta = ""
		space = w - lipgloss.Width(logo) - lipgloss.Width(right)
	}
	if space < 1 {
		right = ""
		space = max(0, w-lipgloss.Width(logo))
	}
	return styleHeader.Width(w).Render(logo + meta + strings.Repeat(" ", space) + right)
}

func (m *Model) renderFolders(w, h int) string {
	innerW := w - 4
	innerH := h - 2

	var sb strings.Builder
	title := styleTitle.Render("  COLLECTIONS") + styleHeaderMeta.Render(fmt.Sprintf("  %02d", len(m.folders)))
	sb.WriteString(title + "\n")

	if len(m.folders) == 0 {
		sb.WriteString(styleDim.Render("  no music found"))
	} else {
		offset := scrollOffset(m.folderIdx, innerH-1)
		for i := offset; i < len(m.folders) && i < offset+innerH-1; i++ {
			f := m.folders[i]
			name := truncate(f.Name, innerW-4)
			prefix := "  "
			if i == m.folderIdx {
				line := "› " + name
				if m.focused == panelFolders {
					sb.WriteString(styleSelected.Width(innerW).Render(line))
				} else {
					sb.WriteString(styleNormal.Bold(true).Render(line))
				}
			} else {
				sb.WriteString(styleDim.Render(prefix + name))
			}
			sb.WriteString("\n")
		}
	}

	content := sb.String()
	border := stylePanelBorder
	if m.focused == panelFolders {
		border = stylePanelActive
	}
	return border.Width(w - 2).Height(h - 2).Render(content)
}

// mascotLines returns the three display lines for the current mascot state.
func (m *Model) mascotLines() [3]string {
	switch m.player.State() {
	case player.Playing:
		return mascotPlaying[m.mascotFrame%4]
	case player.Paused:
		return mascotPaused
	default:
		return mascotStopped
	}
}

func (m *Model) renderTracks(w, h int) string {
	innerW := w - 4
	innerH := h - 2

	// Reserve space for the mascot column on the right.
	trackW := innerW - mascotW
	cat := m.mascotLines()
	mStyle := styleMascot.Width(mascotW)
	catStr := [3]string{
		mStyle.Render(cat[0]),
		mStyle.Render(cat[1]),
		mStyle.Render(cat[2]),
	}
	blank := strings.Repeat(" ", mascotW)

	var sb strings.Builder
	folderName := ""
	if len(m.folders) > 0 {
		folderName = m.folders[m.folderIdx].Name
	}

	var title string
	if m.showSearch || m.searchInput.Value() != "" {
		title = styleTitle.Render("  SEARCH  ") + m.searchInput.View()
	} else {
		title = styleTitle.Render("  TRACKS") + styleHeaderMeta.Render("  "+truncate(folderName, trackW-12))
	}

	sb.WriteString(padRight(title, innerW-mascotW) + catStr[0] + "\n")

	tracks := m.currentTracks()
	if len(tracks) == 0 {
		sb.WriteString(padRight(styleDim.Render("  no tracks"), innerW-mascotW) + catStr[1] + "\n")
	} else {
		offset := scrollOffset(m.trackIdx, innerH-1)
		for i := offset; i < len(tracks) && i < offset+innerH-1; i++ {
			t := tracks[i]
			rowInPanel := i - offset
			num := fmt.Sprintf("%3d. ", i+1)
			name := truncate(t.Name, trackW-6)
			isNow := m.nowPlaying != nil && t.Path == m.nowPlaying.Path

			var line string
			if isNow {
				icon := playIcon(m.player.State())
				line = styleNowPlaying.Render(num + icon + " " + name)
			} else if i == m.trackIdx {
				if m.focused == panelTracks {
					line = styleSelected.Width(trackW - 4).Render(num + "  " + name)
				} else {
					line = styleNormal.Bold(true).Render(num + "  " + name)
				}
			} else {
				line = styleDim.Render(num + "  " + name)
			}

			// Pad line to track area width then append mascot or blank.
			line = padRight(line, innerW-mascotW)
			if rowInPanel == 0 {
				line += catStr[1]
			} else if rowInPanel == 1 {
				line += catStr[2]
			} else {
				line += blank
			}
			sb.WriteString(line + "\n")
		}
	}

	content := sb.String()
	border := stylePanelBorder
	if m.focused == panelTracks {
		border = stylePanelActive
	}
	return border.Width(w - 2).Height(h - 2).Render(content)
}

func (m *Model) renderBottom(w int) string {
	var sb strings.Builder

	// Notification takes precedence over now-playing info for its duration.
	if m.notification != "" && time.Now().Before(m.notifyUntil) {
		sb.WriteString(styleNotify.Render("  ⦿ "+m.notification) + "\n")
		sb.WriteString(renderProgress(w-4, 0) + "\n")
	} else if m.nowPlaying != nil {
		icon := playIcon(m.player.State())
		stateStyle := stateLabelStyle(m.player.State())
		loopMark := ""
		if m.loop {
			loopMark = stylePlaying.Render(" ↺")
		}
		vol := m.player.Volume()
		volStr := ""
		if vol < 0.995 {
			volStr = styleDim.Render(fmt.Sprintf("  vol:%d%%", int(math.Round(vol*100))))
		}
		// Build display title from metadata, fallback to filename.
		displayTitle := m.nowPlaying.Name
		if m.nowMeta.Title != "" {
			displayTitle = m.nowMeta.Title
		}
		if m.nowMeta.Artist != "" {
			displayTitle = m.nowMeta.Artist + " — " + displayTitle
		}
		ratio, elapsed, total := m.player.Progress()
		available := max(12, w-len(fmtDur(elapsed))-len(fmtDur(total))-28)
		displayTitle = marquee(displayTitle, available, m.uiFrame/2)
		motion := ""
		if m.player.State() == player.Playing {
			motion = styleVisualizer.Render(visualizer(m.uiFrame, 7)) + "  "
		}
		label := motion + stateStyle.Render(icon+" "+displayTitle) + loopMark + volStr
		timeStr := fmt.Sprintf(" %s / %s", fmtDur(elapsed), fmtDur(total))
		sb.WriteString(label + styleDim.Render(timeStr) + "\n")
		if m.nowMeta.Album != "" {
			sb.WriteString(styleDim.Render("  "+m.nowMeta.Album) + "\n")
		} else {
			sb.WriteString("\n")
		}
		sb.WriteString(renderProgress(w-4, ratio) + "\n")
	} else {
		sb.WriteString(styleDim.Render("  nothing playing") + "\n")
		sb.WriteString(renderProgress(w-4, 0) + "\n")
	}

	// Key hints
	hints := []string{"j/k:move", "h/l:panel", "enter:play", "spc:pause", ":command", "?:help", "q:quit"}
	if w >= 110 {
		hints = []string{
			"j/k:move", "h/l:panel", "enter:play", "spc:pause", "n/p:skip",
			"r:loop", "+/-:vol", "/:search", "a:queue+", "u:queue", ":command", "?:help", "q:quit",
		}
	}
	var hintParts []string
	for _, h := range hints {
		parts := strings.SplitN(h, ":", 2)
		label := parts[1]
		if parts[0] == "r" && m.loop {
			hintParts = append(hintParts, styleKey.Render(parts[0])+stylePlaying.Render(":"+label+"[on]"))
		} else {
			hintParts = append(hintParts, styleKey.Render(parts[0])+styleHeaderMeta.Render(":"+label))
		}
	}
	sb.WriteString(styleDim.Render("  " + strings.Join(hintParts, "  ")))

	return styleStatusBar.Width(w).Render(sb.String())
}

func renderProgress(w int, ratio float64) string {
	if w <= 0 {
		return ""
	}
	filled := int(float64(w) * ratio)
	if filled > w {
		filled = w
	}
	bar := ""
	if filled > 0 && filled < w {
		bar = styleProgressFill.Render(strings.Repeat("━", filled-1)+"●") +
			styleProgressEmpty.Render(strings.Repeat("─", w-filled))
	} else {
		bar = styleProgressFill.Render(strings.Repeat("━", filled)) +
			styleProgressEmpty.Render(strings.Repeat("─", w-filled))
	}
	return "  " + bar
}

func visualizer(frame, width int) string {
	levels := []rune("▁▂▃▄▅▆▇█")
	var b strings.Builder
	for i := 0; i < width; i++ {
		level := (frame + i*3 + (frame/3+i*i)%5) % len(levels)
		b.WriteRune(levels[level])
	}
	return b.String()
}

func marquee(s string, width, offset int) string {
	r := []rune(s)
	if width <= 0 || len(r) <= width {
		return s
	}
	gap := []rune("   •   ")
	loop := append(append(append([]rune{}, r...), gap...), r...)
	start := offset % (len(r) + len(gap))
	return string(loop[start : start+width])
}

func playIcon(s player.State) string {
	switch s {
	case player.Playing:
		return "▶"
	case player.Paused:
		return "⏸"
	default:
		return "■"
	}
}

func stateLabelStyle(s player.State) lipgloss.Style {
	switch s {
	case player.Playing:
		return stylePlaying
	case player.Paused:
		return stylePaused
	default:
		return styleStopped
	}
}

func fmtDur(d time.Duration) string {
	d = d.Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return "…"
	}
	var b strings.Builder
	width := 0
	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if width+runeWidth > maxLen-1 {
			break
		}
		b.WriteRune(r)
		width += runeWidth
	}
	return b.String() + "…"
}

// padRight pads s with spaces until its visible width (ANSI-aware) equals width.
func padRight(s string, width int) string {
	n := lipgloss.Width(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func scrollOffset(cursor, visible int) int {
	if cursor < visible {
		return 0
	}
	return cursor - visible + 1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func luaConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pmusic", "lua"), nil
}

func (m *Model) loadStoreItems() {
	luaDir, _ := luaConfigDir()
	enabled, _ := pmcfg.LoadEnabled()
	items := make([]storeEntry, 0, len(pmstore.Plugins)+len(pmstore.Themes))
	for _, item := range append(append([]pmstore.Item{}, pmstore.Plugins...), pmstore.Themes...) {
		subdir := "plugins"
		if item.Kind == "theme" {
			subdir = "themes"
		}
		p := filepath.Join(luaDir, subdir, item.Name+".lua")
		_, err := os.Stat(p)
		items = append(items, storeEntry{
			Item:      item,
			Installed: err == nil,
			Enabled:   enabled.Has(item.Kind, item.Name),
		})
	}
	m.storeItems = items
}

func (m *Model) visibleStoreItems() []storeEntry {
	var out []storeEntry
	for _, item := range m.storeItems {
		if (m.storeTab == 0 && item.Kind == "plugin") ||
			(m.storeTab == 1 && item.Kind == "theme") {
			out = append(out, item)
		}
	}
	return out
}

func (m *Model) toggleStoreItem() tea.Cmd {
	visible := m.visibleStoreItems()
	if len(visible) == 0 || m.storeCursor >= len(visible) {
		return nil
	}
	item := visible[m.storeCursor]
	if !item.Installed {
		m.notification = item.Name + " not installed — run pmusic -s first"
		m.notifyUntil = time.Now().Add(5 * time.Second)
		return nil
	}
	enabled, _ := pmcfg.LoadEnabled()
	enabled.Toggle(item.Kind, item.Name)
	if err := pmcfg.SaveEnabled(enabled); err != nil {
		m.notification = "save error: " + err.Error()
		m.notifyUntil = time.Now().Add(5 * time.Second)
		return nil
	}
	m.loadStoreItems()
	return m.reloadLuaCmd()
}

func (m *Model) handleStore(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visibleStoreItems()
	switch {
	case key.Matches(msg, keys.Store), key.Matches(msg, keys.Quit):
		m.showStore = false
	case key.Matches(msg, keys.Down):
		if m.storeCursor < len(visible)-1 {
			m.storeCursor++
		}
	case key.Matches(msg, keys.Up):
		if m.storeCursor > 0 {
			m.storeCursor--
		}
	case key.Matches(msg, keys.Left):
		m.storeTab = 0
		m.storeCursor = 0
	case key.Matches(msg, keys.Right):
		m.storeTab = 1
		m.storeCursor = 0
	case key.Matches(msg, keys.Space), key.Matches(msg, keys.Enter):
		return m, m.toggleStoreItem()
	}
	return m, nil
}

func (m *Model) renderStore() string {
	visible := m.visibleStoreItems()

	tab0 := styleDim.Render("Plugins")
	tab1 := styleDim.Render("Themes")
	if m.storeTab == 0 {
		tab0 = styleNowPlaying.Render("[Plugins]")
	} else {
		tab1 = styleNowPlaying.Render("[Themes]")
	}

	var lines []string
	lines = append(lines, "  "+tab0+"  "+tab1+"   "+styleDim.Render("download with pmusic -s"))
	lines = append(lines, "")

	boxW := min(66, m.width-4)
	contentW := boxW - 4

	for i, item := range visible {
		var icon string
		switch {
		case item.Enabled:
			icon = styleNowPlaying.Render("✓")
		case item.Installed:
			icon = styleNormal.Render("○")
		default:
			icon = styleDim.Render("✗")
		}

		suffix := ""
		if !item.Installed {
			suffix = styleDim.Render(" [not installed]")
		}

		descW := contentW - 2 - 24
		if descW < 0 {
			descW = 0
		}
		nameCol := fmt.Sprintf("%-20s", item.Name)
		descCol := truncate(item.Desc, descW)
		rowText := fmt.Sprintf("  %s  %s  %s%s", icon, nameCol, descCol, suffix)

		if i == m.storeCursor {
			rowText = styleSelected.Width(contentW).Render(
				fmt.Sprintf("  %s  %-20s  %s%s", icon, item.Name, truncate(item.Desc, descW), suffix),
			)
		}
		lines = append(lines, rowText)
	}

	lines = append(lines, "")
	lines = append(lines, styleDim.Render("  Space:toggle  h/l:tabs  g/q:close"))

	content := styleTitle.Render("  Plugin Store  ") + "\n\n" + strings.Join(lines, "\n")
	box := stylePanelActive.Width(boxW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// handleQueue processes key input while the queue overlay is active.
func (m *Model) handleQueue(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.queue)
	mutated := false
	switch {
	case key.Matches(msg, keys.Queue), key.Matches(msg, keys.Quit):
		m.showQueue = false
	case key.Matches(msg, keys.Down):
		if m.queueCursor < n-1 {
			m.queueCursor++
		}
	case key.Matches(msg, keys.Up):
		if m.queueCursor > 0 {
			m.queueCursor--
		}
	case msg.String() == "J": // move selected item down
		if m.queueCursor < n-1 {
			m.queue[m.queueCursor], m.queue[m.queueCursor+1] = m.queue[m.queueCursor+1], m.queue[m.queueCursor]
			m.queueCursor++
			mutated = true
		}
	case msg.String() == "K": // move selected item up
		if m.queueCursor > 0 {
			m.queue[m.queueCursor], m.queue[m.queueCursor-1] = m.queue[m.queueCursor-1], m.queue[m.queueCursor]
			m.queueCursor--
			mutated = true
		}
	case msg.String() == "d", msg.String() == "x": // remove selected
		if n > 0 && m.queueCursor < n {
			m.queue = append(m.queue[:m.queueCursor], m.queue[m.queueCursor+1:]...)
			if m.queueCursor >= len(m.queue) {
				m.queueCursor = max(0, len(m.queue)-1)
			}
			mutated = true
		}
	case msg.String() == "c": // clear the queue
		if len(m.queue) > 0 {
			m.queue = nil
			m.queueCursor = 0
			mutated = true
		}
	case key.Matches(msg, keys.Enter): // play the selected item now
		if n > 0 && m.queueCursor < n {
			t := m.queue[m.queueCursor]
			m.queue = append(m.queue[:m.queueCursor], m.queue[m.queueCursor+1:]...)
			if m.queueCursor >= len(m.queue) {
				m.queueCursor = max(0, len(m.queue)-1)
			}
			m.saveQueue()
			fi, ti := m.nowFolder, m.nowTrack
			if fi, ti, ok := m.locateTrack(t.Path); ok {
				return m, m.startPlayback(t, playbackQueue, fi, ti)
			}
			return m, m.startPlayback(t, playbackQueue, fi, ti)
		}
	}
	if mutated {
		m.saveQueue()
	}
	return m, nil
}

// renderQueue returns the centered queue overlay listing all queued tracks.
func (m *Model) renderQueue() string {
	boxW := min(66, m.width-4)
	contentW := boxW - 4

	var lines []string
	lines = append(lines, "  "+styleDim.Render(fmt.Sprintf("%d tracks queued", len(m.queue))))
	lines = append(lines, "")

	if len(m.queue) == 0 {
		lines = append(lines, styleDim.Render("  queue empty — select a track/folder and press [a]"))
	} else {
		// Cap visible rows so the box stays on screen.
		visible := m.height - 10
		if visible < 1 {
			visible = 1
		}
		offset := scrollOffset(m.queueCursor, visible)
		for i := offset; i < len(m.queue) && i < offset+visible; i++ {
			t := m.queue[i]
			num := fmt.Sprintf("%3d. ", i+1)
			name := truncate(t.Name, contentW-6)
			if i == m.queueCursor {
				lines = append(lines, styleSelected.Width(contentW).Render(num+name))
			} else {
				lines = append(lines, styleDim.Render(num+name))
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, styleDim.Render("  j/k:nav  K/J:move  d:remove  c:clear  Enter:play  u/q:close"))

	content := styleTitle.Render("  ♪  Queue  ") + "\n\n" + strings.Join(lines, "\n")
	box := stylePanelActive.Width(boxW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// updateBlackjack handles all key input while the blackjack overlay is active.
func (m *Model) updateBlackjack(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	g := m.bjGame
	switch msg.String() {
	case "b", "esc":
		m.showBlackjack = false
	case "n":
		if g.Phase == blackjack.PhaseMenu || g.Phase == blackjack.PhaseResult {
			if g.Balance <= 0 {
				g.NewRound()
			} else {
				g.Deal()
			}
		}
	case "h":
		if g.Phase == blackjack.PhasePlaying {
			g.Hit()
		}
	case "s":
		if g.Phase == blackjack.PhasePlaying {
			g.Stand()
		}
	case "d":
		if g.Phase == blackjack.PhasePlaying {
			g.Double()
		}
	case "+", "=":
		g.AdjustBet(10)
	case "-":
		g.AdjustBet(-10)
	}
	return m, nil
}

// renderHelp returns a full-screen centered help overlay listing all key bindings.
func (m *Model) renderHelp() string {
	type section struct{ title, bindings string }
	sections := []section{
		{"Navigation", "j / k      up / down\nh / l      folders / tracks\nEnter      select & play"},
		{"Playback", "Space      pause / resume\nn          next track\np          previous track\nr          loop mode"},
		{"Queue", "a          queue selected track/folder\nu          open / close queue\nK / J      move up / down in queue\nd          remove from queue\nc          clear queue"},
		{"Seek", "[  /  ]    ±5 seconds\n{  /  }    ±30 seconds"},
		{"Volume", "+  /  =    volume up (10%)\n-          volume down (10%)"},
		{"Music Search", "Y          search and download music\n/          new search inside the overlay"},
		{"System", ":          open command mode\n?          toggle this help\nCtrl+R     reload Lua config\nq          quit"},
	}

	var lines []string
	lines = append(lines, styleTitle.Render("  ♪  Keyboard Shortcuts  "))
	lines = append(lines, "")
	for _, s := range sections {
		lines = append(lines, stylePlaying.Render("  "+s.title))
		for _, b := range strings.Split(s.bindings, "\n") {
			lines = append(lines, styleDim.Render("    "+b))
		}
		lines = append(lines, "")
	}
	lines = append(lines, styleDim.Render("  close with [ ? ] or [ q ]"))

	content := strings.Join(lines, "\n")
	boxW := min(62, m.width-4)
	box := stylePanelActive.Width(boxW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
