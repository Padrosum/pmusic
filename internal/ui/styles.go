package ui

import (
	"github.com/charmbracelet/lipgloss"
	luaeng "github.com/padros/pmusic/internal/lua"
)

var (
	styleHeader        lipgloss.Style
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
	styleMascot        lipgloss.Style
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
	fg := lipgloss.Color("#CAD3F5") // Macchiato Text
	fg2 := lipgloss.Color("#F4DBD6") // Macchiato Rosewater
	amber := lipgloss.Color("#EED49F") // Macchiato Yellow

	styleHeader = lipgloss.NewStyle().Background(accent).Foreground(lipgloss.Color("#181825")).Bold(true).Padding(0, 1)
	stylePanel = lipgloss.NewStyle().Background(bg).Foreground(fg)
	stylePanelBorder = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(bdr).Background(bg).Padding(0, 1)
	stylePanelActive = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(bdrA).Background(bg).Padding(0, 1)
	styleTitle = lipgloss.NewStyle().Foreground(ttl).Bold(true).Padding(0, 1)
	styleSelected = lipgloss.NewStyle().Background(sel).Foreground(fg2).Bold(true).Padding(0, 1)
	styleNormal = lipgloss.NewStyle().Foreground(fg).Padding(0, 1)
	styleDim = lipgloss.NewStyle().Foreground(dim).Padding(0, 1)
	styleNowPlaying = lipgloss.NewStyle().Foreground(np).Bold(true).Padding(0, 1)
	styleProgress = lipgloss.NewStyle().Foreground(accent)
	styleProgressFill = lipgloss.NewStyle().Foreground(accent)
	styleProgressEmpty = lipgloss.NewStyle().Foreground(dim)
	stylePlaying = lipgloss.NewStyle().Foreground(np)
	stylePaused = lipgloss.NewStyle().Foreground(amber)
	styleStopped = lipgloss.NewStyle().Foreground(dim)
	styleKey = lipgloss.NewStyle().Foreground(k).Bold(true)
	styleStatusBar = lipgloss.NewStyle().Background(sbg).Foreground(fg).Padding(0, 1)
	styleNotify = lipgloss.NewStyle().Foreground(amber).Bold(true).Padding(0, 1)
	styleMascot = lipgloss.NewStyle().Foreground(accent).Bold(true)
}
