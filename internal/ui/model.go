package ui

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	pmcfg "github.com/padros/pmusic/internal/config"
	pfs "github.com/padros/pmusic/internal/fs"
	luaeng "github.com/padros/pmusic/internal/lua"
	"github.com/padros/pmusic/internal/meta"
	"github.com/padros/pmusic/internal/player"
	pmstore "github.com/padros/pmusic/internal/store"
	"github.com/padros/pmusic/internal/watcher"
)

type panel int

const (
	panelFolders panel = iota
	panelTracks
)

type tickMsg time.Time

// luaReloadedMsg is sent back to the Update loop after a Lua hot-reload attempt.
type luaReloadedMsg struct{ err error }

// ytdlpMsg is returned when the yt-dlp process has been launched.
type ytdlpMsg struct {
	query string
	err   error
}

// mascot animation constants — each frame line is rendered at fixed width via lipgloss.
const mascotW = 8

var mascotPlaying = [4][3]string{
	{`/\_/\`, `(^.^)`, `>♪ < `},
	{`/\_/\`, `(^o^)`, `>♫ < `},
	{`/\_/\`, `(>.~)`, `>♪ < `},
	{`/\_/\`, `(^-^)`, `>♫ < `},
}
var mascotPaused  = [3]string{`/\_/\`, `(-_-)`, `~♩~ `}
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
	nowPlaying *pfs.Track
	nowFolder  int
	nowTrack   int
	loop       bool

	mascotFrame int

	nowMeta  meta.Meta
	showHelp bool

	showStore   bool
	storeItems  []storeEntry
	storeCursor int
	storeTab    int // 0=plugins 1=themes

	showDownload  bool
	downloadInput textinput.Model

	watcher *watcher.Watcher
	rootDir string

	luaEngine *luaeng.Engine

	// Lua hook state tracking
	prevPlayingPath string
	prevState       player.State

	// Notification overlay (shown in status bar, expires after a short time)
	notification string
	notifyUntil  time.Time
}

func New(rootDir string) (*Model, error) {
	root, err := pfs.Scan(rootDir)
	if err != nil {
		return nil, err
	}
	folders := pfs.FlatFolders(root)

	p := player.New()
	ti := textinput.New()
	ti.Placeholder = "search YouTube..."
	ti.CharLimit = 120
	m := &Model{
		root:          root,
		folders:       folders,
		player:        p,
		rootDir:       rootDir,
		downloadInput: ti,
	}

	p.SetOnDone(func() {
		// handled via tickMsg checking state
	})

	w, err := watcher.New(rootDir, func() {
		// rescan signal handled in Update
	})
	if err == nil {
		m.watcher = w
	}

	eng := luaeng.New()
	eng.SetMusicDir(rootDir)
	if err := eng.Load(); err != nil {
		m.notification = "lua: " + err.Error()
		m.notifyUntil = time.Now().Add(10 * time.Second)
	}
	m.luaEngine = eng
	applyTheme(eng.Theme())

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.SetWindowTitle("pmusic"),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Pass every message through the download input so its cursor blinks.
	if m.showDownload {
		var inputCmd tea.Cmd
		m.downloadInput, inputCmd = m.downloadInput.Update(msg)
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.Type {
			case tea.KeyEsc:
				m.showDownload = false
				m.downloadInput.Blur()
				return m, nil
			case tea.KeyEnter:
				query := strings.TrimSpace(m.downloadInput.Value())
				m.showDownload = false
				m.downloadInput.Blur()
				if query != "" {
					return m, m.runYtDlp(query)
				}
				return m, nil
			}
		}
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

	case ytdlpMsg:
		if msg.err != nil {
			m.notification = "yt-dlp: " + msg.err.Error()
		} else {
			m.notification = "indiriliyor: " + msg.query
		}
		m.notifyUntil = time.Now().Add(8 * time.Second)
		return m, nil

	case tickMsg:
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

		// Fire on_song_change hook and load metadata when the playing track changes.
		if m.nowPlaying != nil && m.nowPlaying.Path != m.prevPlayingPath {
			m.prevPlayingPath = m.nowPlaying.Path
			m.nowMeta = meta.Read(m.nowPlaying.Path)
			folder := ""
			if m.nowFolder < len(m.folders) {
				folder = m.folders[m.nowFolder].Name
			}
			m.luaEngine.CallOnSongChange(m.nowPlaying.Name, m.nowPlaying.Path, folder)
		}

		// Cache state once so both the hook check and auto-advance use the same snapshot.
		st := m.player.State()

		// Fire on_state_change hook when play/pause/stop state transitions.
		if st != m.prevState {
			m.prevState = st
			var stateStr string
			switch st {
			case player.Playing:
				stateStr = "playing"
			case player.Paused:
				stateStr = "paused"
			default:
				stateStr = "stopped"
			}
			m.luaEngine.CallOnStateChange(stateStr)
		}

		// Auto-advance (or loop) when a track ends naturally.
		if m.nowPlaying != nil && st == player.Stopped {
			var cmd tea.Cmd
			if m.loop {
				cmd = m.replayCurrent()
			} else {
				cmd = m.playNext()
			}
			if cmd == nil {
				m.nowPlaying = nil
				m.nowMeta = meta.Meta{}
			}
			return m, tea.Batch(tickCmd(), cmd)
		}
		return m, tickCmd()

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// Store overlay intercepts all keys.
		if m.showStore {
			return m.handleStore(msg)
		}

		// Help overlay intercepts all keys — only ? and q close it.
		if m.showHelp {
			if key.Matches(msg, keys.Help) || key.Matches(msg, keys.Quit) {
				m.showHelp = false
			}
			return m, nil
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
			m.player.Stop()
			if m.watcher != nil {
				m.watcher.Close()
			}
			m.luaEngine.Close()
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

		case key.Matches(msg, keys.Download):
			m.openDownloadPanel()
			return m, m.downloadInput.Focus()
		}
	}
	return m, nil
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
		m.player.Stop()
		if m.watcher != nil {
			m.watcher.Close()
		}
		m.luaEngine.Close()
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
	return m.folders[m.folderIdx].Tracks
}

func (m *Model) playSelected() tea.Cmd {
	tracks := m.currentTracks()
	if len(tracks) == 0 || m.trackIdx >= len(tracks) {
		return nil
	}
	t := tracks[m.trackIdx]
	m.nowPlaying = &t
	m.nowFolder = m.folderIdx
	m.nowTrack = m.trackIdx
	// Mark pending before goroutine starts so tick doesn't trigger auto-advance.
	m.player.MarkPending()
	path := t.Path
	return func() tea.Msg {
		m.player.Play(path)
		return tickMsg(time.Now())
	}
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
	m.nowFolder = fi
	m.nowTrack = ti
	t := m.folders[fi].Tracks[ti]
	m.nowPlaying = &t
	m.player.MarkPending()
	path := t.Path
	return func() tea.Msg {
		m.player.Play(path)
		return tickMsg(time.Now())
	}
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
	m.nowFolder = fi
	m.nowTrack = ti
	t := m.folders[fi].Tracks[ti]
	m.nowPlaying = &t
	m.player.MarkPending()
	path := t.Path
	return func() tea.Msg {
		m.player.Play(path)
		return tickMsg(time.Now())
	}
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
	m.nowPlaying = &t
	m.player.MarkPending()
	path := t.Path
	return func() tea.Msg {
		m.player.Play(path)
		return tickMsg(time.Now())
	}
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
	// mainH = m.height - bottomH(4) - 2; panel content rows start at screen row 2.
	mainH := m.height - 4 - 2
	innerH := mainH - 2
	visibleRows := innerH - 1 // title row consumed

	if row <= 0 || row > mainH {
		return m, nil
	}

	if col < leftW {
		// Left panel (folders): content begins at screen row 2.
		itemRow := row - 2
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
		itemRow := row - 2
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
			m.nowPlaying = &t
			m.nowFolder = m.folderIdx
			m.nowTrack = idx
			m.player.MarkPending()
			path := t.Path
			return m, func() tea.Msg {
				m.player.Play(path)
				return tickMsg(time.Now())
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	if m.showDownload {
		return m.renderDownload()
	}
	if m.showStore {
		return m.renderStore()
	}
	if m.showHelp {
		return m.renderHelp()
	}

	bottomH := 4
	mainH := m.height - bottomH - 2

	leftW := m.width / 3
	rightW := m.width - leftW - 1

	left := m.renderFolders(leftW, mainH)
	right := m.renderTracks(rightW, mainH)

	top := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	bottom := m.renderBottom(m.width)

	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (m *Model) renderFolders(w, h int) string {
	innerW := w - 4
	innerH := h - 2

	var sb strings.Builder
	title := styleTitle.Render("  Folders")
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
				line := prefix + name
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
	title := styleTitle.Render("  " + truncate(folderName, trackW-6))
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
		label := stateStyle.Render(icon+" "+displayTitle) + loopMark + volStr
		ratio, elapsed, total := m.player.Progress()
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
	hints := []string{
		"j/k:move", "h/l:panel", "enter:play", "spc:pause",
		"n/p:next/prev", "r:loop", "+/-:vol", "[/]:seek5s",
		"Y:indir", "g:store", "?:help", "q:quit",
	}
	var hintParts []string
	for _, h := range hints {
		parts := strings.SplitN(h, ":", 2)
		label := parts[1]
		if parts[0] == "r" && m.loop {
			hintParts = append(hintParts, styleKey.Render(parts[0])+stylePlaying.Render(":"+label+"[on]"))
		} else {
			hintParts = append(hintParts, styleKey.Render(parts[0])+styleDim.Render(":"+label))
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
	bar := styleProgressFill.Render(strings.Repeat("━", filled)) +
		styleProgressEmpty.Render(strings.Repeat("─", w-filled))
	return "  " + bar
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
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
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

func (m *Model) openDownloadPanel() {
	query := ""
	if m.nowPlaying != nil {
		query = m.nowPlaying.Name
		if m.nowFolder < len(m.folders) && m.folders[m.nowFolder].Name != "" {
			query = m.folders[m.nowFolder].Name + " " + query
		}
	}
	m.downloadInput.SetValue(query)
	m.downloadInput.CursorEnd()
	m.showDownload = true
}

func (m *Model) runYtDlp(query string) tea.Cmd {
	musicDir := m.rootDir
	return func() tea.Msg {
		if _, err := exec.LookPath("yt-dlp"); err != nil {
			return ytdlpMsg{query: query, err: fmt.Errorf("yt-dlp bulunamadı — PATH'te kurulu olmalı")}
		}
		cmd := exec.Command("yt-dlp",
			"-x", "--audio-format", "mp3", "--audio-quality", "0",
			"-o", filepath.Join(musicDir, "%(title)s.%(ext)s"),
			"ytsearch1:"+query,
		)
		if err := cmd.Start(); err != nil {
			return ytdlpMsg{query: query, err: err}
		}
		go cmd.Wait() // prevent zombie process
		return ytdlpMsg{query: query}
	}
}

func (m *Model) renderDownload() string {
	boxW := min(64, m.width-4)
	inputW := boxW - 12 // "  Ara:  " prefix + border padding

	m.downloadInput.Width = inputW

	var lines []string
	lines = append(lines, styleTitle.Render("  YouTube'dan indir  "))
	lines = append(lines, "")
	lines = append(lines, styleDim.Render("  Ara:  ")+m.downloadInput.View())
	lines = append(lines, "")
	lines = append(lines, styleDim.Render("  Enter:indir   Esc:kapat"))

	content := strings.Join(lines, "\n")
	box := stylePanelActive.Width(boxW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
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
		m.notification = item.Name + " kurulu değil — önce pmusic -s çalıştır"
		m.notifyUntil = time.Now().Add(5 * time.Second)
		return nil
	}
	enabled, _ := pmcfg.LoadEnabled()
	enabled.Toggle(item.Kind, item.Name)
	if err := pmcfg.SaveEnabled(enabled); err != nil {
		m.notification = "kayıt hatası: " + err.Error()
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
	lines = append(lines, "  "+tab0+"  "+tab1+"   "+styleDim.Render("pmusic -s ile indir"))
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
			suffix = styleDim.Render(" [kurulu değil]")
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
	lines = append(lines, styleDim.Render("  Space:toggle  h/l:sekme  g/q:kapat"))

	content := styleTitle.Render("  Plugin Store  ") + "\n\n" + strings.Join(lines, "\n")
	box := stylePanelActive.Width(boxW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// renderHelp returns a full-screen centered help overlay listing all key bindings.
func (m *Model) renderHelp() string {
	type section struct{ title, bindings string }
	sections := []section{
		{"Navigasyon", "j / k      yukarı / aşağı\nh / l      klasörler / parçalar\nEnter      seç ve çal"},
		{"Oynatma", "Space      duraklat / devam\nn          sonraki parça\np          önceki parça\nr          döngü modu"},
		{"Sarma", "[  /  ]    ±5 saniye\n{  /  }    ±30 saniye"},
		{"Ses", "+  /  =    ses arttır (%10)\n-          ses azalt (%10)"},
		{"Sistem", "?          bu yardımı göster / kapat\nCtrl+R     Lua config yenile\nq          çık"},
	}

	var lines []string
	lines = append(lines, styleTitle.Render("  ♪  Klavye Kısayolları  "))
	lines = append(lines, "")
	for _, s := range sections {
		lines = append(lines, stylePlaying.Render("  "+s.title))
		for _, b := range strings.Split(s.bindings, "\n") {
			lines = append(lines, styleDim.Render("    "+b))
		}
		lines = append(lines, "")
	}
	lines = append(lines, styleDim.Render("  [ ? ] veya [ q ] tuşuyla kapat"))

	content := strings.Join(lines, "\n")
	boxW := min(62, m.width-4)
	box := stylePanelActive.Width(boxW).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
