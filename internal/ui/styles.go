package ui

import (
	"github.com/charmbracelet/lipgloss"
	luaeng "github.com/padros/pmusic/internal/lua"
)

var (
	stylePanel         lipgloss.Style
	stylePanelBorder   lipgloss.Style
	stylePanelActive   lipgloss.Style
	styleTitle         lipgloss.Style
	styleSelected      lipgloss.Style
	styleNormal        lipgloss.Style
	styleDim           lipgloss.Style
	styleNowPlaying    lipgloss.Style
	styleProgress      lipgloss.Style
	styleProgressFill  lipgloss.Style
	styleProgressEmpty lipgloss.Style
	stylePlaying       lipgloss.Style
	stylePaused        lipgloss.Style
	styleStopped       lipgloss.Style
	styleKey           lipgloss.Style
	styleStatusBar     lipgloss.Style
	styleNotify        lipgloss.Style
)

func init() {
	applyTheme(luaeng.DefaultTheme())
}

// applyTheme regenerates all package-level style vars from the given theme.
// Called on startup and after each Lua reload.
func applyTheme(t luaeng.Theme) {
	bg := lipgloss.Color(t.PanelBg)
	accent := lipgloss.Color(t.Accent)
	dim := lipgloss.Color(t.Dim)
	sel := lipgloss.Color(t.SelectedBg)
	np := lipgloss.Color(t.NowPlaying)
	bdr := lipgloss.Color(t.Border)
	bdrA := lipgloss.Color(t.BorderActive)
	ttl := lipgloss.Color(t.Title)
	sbg := lipgloss.Color(t.StatusBg)
	k := lipgloss.Color(t.Key)
	fg := lipgloss.Color("#D8DEE9")
	fg2 := lipgloss.Color("#ECEFF4")
	amber := lipgloss.Color("#EBCB8B")

	stylePanel = lipgloss.NewStyle().Background(bg).Foreground(fg)
	stylePanelBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(bdr).Background(bg)
	stylePanelActive = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(bdrA).Background(bg)
	styleTitle = lipgloss.NewStyle().Foreground(ttl).Bold(true).Padding(0, 1)
	styleSelected = lipgloss.NewStyle().Background(sel).Foreground(fg2).Bold(true)
	styleNormal = lipgloss.NewStyle().Foreground(fg)
	styleDim = lipgloss.NewStyle().Foreground(dim)
	styleNowPlaying = lipgloss.NewStyle().Foreground(np).Bold(true)
	styleProgress = lipgloss.NewStyle().Foreground(accent)
	styleProgressFill = lipgloss.NewStyle().Foreground(accent)
	styleProgressEmpty = lipgloss.NewStyle().Foreground(dim)
	stylePlaying = lipgloss.NewStyle().Foreground(np)
	stylePaused = lipgloss.NewStyle().Foreground(amber)
	styleStopped = lipgloss.NewStyle().Foreground(dim)
	styleKey = lipgloss.NewStyle().Foreground(k).Bold(true)
	styleStatusBar = lipgloss.NewStyle().Background(sbg).Foreground(fg).Padding(0, 1)
	styleNotify = lipgloss.NewStyle().Foreground(amber).Bold(true)
}
