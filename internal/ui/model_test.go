package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestVisualizerKeepsRequestedWidth(t *testing.T) {
	for frame := 0; frame < 20; frame++ {
		if got := lipgloss.Width(visualizer(frame, 7)); got != 7 {
			t.Fatalf("frame %d: visualizer width = %d, want 7", frame, got)
		}
	}
}

func TestMarquee(t *testing.T) {
	const title = "A very long track title"
	if got := lipgloss.Width(marquee(title, 10, 0)); got != 10 {
		t.Fatalf("marquee width = %d, want 10", got)
	}
	if got := marquee("short", 10, 4); got != "short" {
		t.Fatalf("short title changed to %q", got)
	}
}

func TestProgressWidth(t *testing.T) {
	for _, ratio := range []float64{0, .25, 1} {
		got := renderProgress(24, ratio)
		if width := lipgloss.Width(got); width != 26 { // two-cell left inset
			t.Fatalf("ratio %.2f: progress width = %d, want 26", ratio, width)
		}
	}
}

func TestSmallTerminalFallback(t *testing.T) {
	m := &Model{width: 40, height: 8}
	view := m.View()
	if !strings.Contains(view, "52 × 12") {
		t.Fatalf("small-terminal guidance missing from view: %q", view)
	}
}

func TestHelpIncludesYtDlpShortcut(t *testing.T) {
	m := &Model{width: 80, height: 40}
	view := m.renderHelp()
	if !strings.Contains(view, "Y          YouTube download (yt-dlp)") {
		t.Fatalf("yt-dlp shortcut missing from help")
	}
}
