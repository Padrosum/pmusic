package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) commandLineHeight() int {
	if !m.commandLine.active {
		return 4
	}
	visible := min(len(m.commandLine.completion.Items), max(0, min(8, m.height-12)))
	return 2 + visible
}

func (m *Model) renderCommandLine(w int) string {
	inner := max(1, w-4)
	m.commandLine.input.Width = max(1, inner-2)
	var lines []string
	visible := min(len(m.commandLine.completion.Items), max(0, min(8, m.height-12)))
	if m.commandLine.completion.Visible && visible > 0 {
		start := 0
		if m.commandLine.completion.Selected >= visible {
			start = m.commandLine.completion.Selected - visible + 1
		}
		for i := start; i < len(m.commandLine.completion.Items) && i < start+visible; i++ {
			item := m.commandLine.completion.Items[i]
			label := fmt.Sprintf("  %-14s %s", item.Display, truncate(item.Description, max(0, inner-18)))
			if i == m.commandLine.completion.Selected {
				lines = append(lines, styleSelected.Width(inner).Render(label))
			} else {
				lines = append(lines, styleDim.Render(label))
			}
		}
	}
	input := styleKey.Render(":") + m.commandLine.input.View()
	lines = append(lines, styleStatusBar.Width(max(0, w-2)).Render(input))
	return strings.Join(lines, "\n")
}

func (m *Model) renderCommandHelp() string {
	boxW := max(1, min(78, m.width-4))
	contentW := max(1, boxW-4)
	visible := max(1, m.height-6)
	lines := m.commandHelp.lines
	if m.commandHelp.filtering {
		lines = append([]string{"Filter: " + m.commandHelp.filter.View(), ""}, lines...)
	}
	maxOffset := max(0, len(lines)-visible)
	if m.commandHelp.offset > maxOffset {
		m.commandHelp.offset = maxOffset
	}
	end := min(len(lines), m.commandHelp.offset+visible)
	var rendered []string
	for i := m.commandHelp.offset; i < end; i++ {
		line := truncate(lines[i], contentW)
		switch {
		case i == m.commandHelp.offset && strings.HasPrefix(line, "::"):
			rendered = append(rendered, styleTitle.Render("  "+strings.TrimPrefix(line, "::")))
		case line == "pmusic Command Help":
			rendered = append(rendered, styleTitle.Render("  "+line))
		case line == "Playback" || line == "Audio" || line == "Library" || line == "Queue" || line == "Application" || line == "Usage" || line == "Arguments" || line == "Subcommands" || line == "Examples" || line == "Aliases" || line == "Related":
			rendered = append(rendered, stylePlaying.Render(line))
		default:
			rendered = append(rendered, styleDim.Render(line))
		}
	}
	box := stylePanelActive.Width(boxW).Render(strings.Join(rendered, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
