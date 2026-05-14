package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	pfs "github.com/padros/pmusic/internal/fs"
	"github.com/padros/pmusic/internal/player"
	"github.com/padros/pmusic/internal/watcher"
)

type panel int

const (
	panelFolders panel = iota
	panelTracks
)

type tickMsg time.Time

type Model struct {
	width, height int
	focused       panel

	root    *pfs.Folder
	folders []*pfs.Folder

	folderIdx int
	trackIdx  int

	player      *player.Player
	nowPlaying  *pfs.Track
	nowFolder   int
	nowTrack    int

	watcher *watcher.Watcher
	rootDir string
}

func New(rootDir string) (*Model, error) {
	root, err := pfs.Scan(rootDir)
	if err != nil {
		return nil, err
	}
	folders := pfs.FlatFolders(root)

	p := player.New()
	m := &Model{
		root:    root,
		folders: folders,
		player:  p,
		rootDir: rootDir,
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
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.watcher != nil && m.watcher.Changed() {
			m.rescan()
		}
		// auto-advance when a track ends naturally
		if m.nowPlaying != nil && m.player.State() == player.Stopped {
			cmd := m.playNext()
			if cmd == nil {
				// end of library
				m.nowPlaying = nil
			}
			return m, tea.Batch(tickCmd(), cmd)
		}
		return m, tickCmd()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.player.Stop()
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit

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
		}
	}
	return m, nil
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

func (m *Model) rescan() {
	root, err := pfs.Scan(m.rootDir)
	if err != nil {
		return
	}
	m.root = root
	m.folders = pfs.FlatFolders(root)
	if m.folderIdx >= len(m.folders) {
		m.folderIdx = max(0, len(m.folders)-1)
	}
}

func (m *Model) View() string {
	if m.width == 0 {
		return "loading..."
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

func (m *Model) renderTracks(w, h int) string {
	innerW := w - 4
	innerH := h - 2

	var sb strings.Builder
	folderName := ""
	if len(m.folders) > 0 {
		folderName = m.folders[m.folderIdx].Name
	}
	title := styleTitle.Render("  " + truncate(folderName, innerW-6))
	sb.WriteString(title + "\n")

	tracks := m.currentTracks()
	if len(tracks) == 0 {
		sb.WriteString(styleDim.Render("  no tracks"))
	} else {
		offset := scrollOffset(m.trackIdx, innerH-1)
		for i := offset; i < len(tracks) && i < offset+innerH-1; i++ {
			t := tracks[i]
			num := fmt.Sprintf("%3d. ", i+1)
			name := truncate(t.Name, innerW-6)
			isNow := m.nowPlaying != nil && t.Path == m.nowPlaying.Path

			var line string
			if isNow {
				icon := playIcon(m.player.State())
				line = styleNowPlaying.Render(num+icon+" "+name)
			} else if i == m.trackIdx {
				if m.focused == panelTracks {
					line = styleSelected.Width(innerW).Render(num + "  " + name)
				} else {
					line = styleNormal.Bold(true).Render(num + "  " + name)
				}
			} else {
				line = styleDim.Render(num + "  " + name)
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

	// Now playing line
	if m.nowPlaying != nil {
		icon := playIcon(m.player.State())
		stateStyle := stateLabelStyle(m.player.State())
		label := stateStyle.Render(icon + " " + m.nowPlaying.Name)
		ratio, elapsed, total := m.player.Progress()
		timeStr := fmt.Sprintf(" %s / %s", fmtDur(elapsed), fmtDur(total))
		sb.WriteString(label + styleDim.Render(timeStr) + "\n")
		sb.WriteString(renderProgress(w-4, ratio) + "\n")
	} else {
		sb.WriteString(styleDim.Render("  nothing playing") + "\n")
		sb.WriteString(renderProgress(w-4, 0) + "\n")
	}

	// Key hints
	hints := []string{"j/k:move", "h/l:panel", "enter:play", "spc:pause", "n/p:next/prev", "q:quit"}
	var hintParts []string
	for _, h := range hints {
		parts := strings.SplitN(h, ":", 2)
		hintParts = append(hintParts, styleKey.Render(parts[0])+styleDim.Render(":"+parts[1]))
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
