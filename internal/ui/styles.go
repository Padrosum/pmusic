package ui

import "github.com/charmbracelet/lipgloss"

// Nord palette
var (
	nord0  = lipgloss.Color("#2E3440")
	nord1  = lipgloss.Color("#3B4252")
	nord2  = lipgloss.Color("#434C5E")
	nord3  = lipgloss.Color("#4C566A")
	nord4  = lipgloss.Color("#D8DEE9")
	nord5  = lipgloss.Color("#E5E9F0")
	nord6  = lipgloss.Color("#ECEFF4")
	nord7  = lipgloss.Color("#8FBCBB")
	nord8  = lipgloss.Color("#88C0D0")
	nord9  = lipgloss.Color("#81A1C1")
	nord10 = lipgloss.Color("#5E81AC")
	nord11 = lipgloss.Color("#BF616A")
	nord13 = lipgloss.Color("#EBCB8B")
	nord14 = lipgloss.Color("#A3BE8C")
)

var (
	stylePanel = lipgloss.NewStyle().
			Background(nord0).
			Foreground(nord4)

	stylePanelBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(nord3).
				Background(nord0)

	stylePanelActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(nord8).
				Background(nord0)

	styleTitle = lipgloss.NewStyle().
			Foreground(nord8).
			Bold(true).
			Padding(0, 1)

	styleSelected = lipgloss.NewStyle().
			Background(nord2).
			Foreground(nord6).
			Bold(true)

	styleNormal = lipgloss.NewStyle().
			Foreground(nord4)

	styleDim = lipgloss.NewStyle().
			Foreground(nord3)

	styleNowPlaying = lipgloss.NewStyle().
			Foreground(nord14).
			Bold(true)

	styleProgress = lipgloss.NewStyle().
			Foreground(nord8)

	styleProgressFill = lipgloss.NewStyle().
				Foreground(nord8)

	styleProgressEmpty = lipgloss.NewStyle().
				Foreground(nord3)

	stylePlaying = lipgloss.NewStyle().
			Foreground(nord14)

	stylePaused = lipgloss.NewStyle().
			Foreground(nord13)

	styleStopped = lipgloss.NewStyle().
			Foreground(nord3)

	styleKey = lipgloss.NewStyle().
			Foreground(nord9).
			Bold(true)

	styleStatusBar = lipgloss.NewStyle().
			Background(nord1).
			Foreground(nord4).
			Padding(0, 1)
)
