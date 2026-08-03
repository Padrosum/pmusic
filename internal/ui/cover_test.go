package ui

import (
	"strings"
	"testing"

	"github.com/Padrosum/pmusic/internal/cover"
	pfs "github.com/Padrosum/pmusic/internal/fs"
	"github.com/Padrosum/pmusic/internal/meta"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCoverOverlayToggle(t *testing.T) {
	m := commandTestModel(t)
	m.nowPlaying = &pfs.Track{Name: "One", Path: "/one"}
	cmd := m.toggleCover()
	if !m.showCover || cmd == nil {
		t.Fatalf("open: showCover=%v cmd=%v", m.showCover, cmd)
	}
	m.toggleCover()
	if m.showCover {
		t.Fatal("cover did not close")
	}
}

func TestCoverReadyMsgPopulatesLines(t *testing.T) {
	m := commandTestModel(t)
	m.nowPlaying = &pfs.Track{Name: "One", Path: "/one"}
	m.Update(coverReadyMsg{path: "/one", lines: []string{"a", "b"}, source: cover.SourceFolder})
	if len(m.coverLines) != 2 || m.coverSource != cover.SourceFolder || m.coverError != "" {
		t.Fatalf("lines=%v source=%v err=%q", m.coverLines, m.coverSource, m.coverError)
	}
	m.Update(coverReadyMsg{path: "/two", lines: []string{"x"}, source: cover.SourceOnline})
	if len(m.coverLines) != 2 {
		t.Fatalf("stale update overwrote lines: %v", m.coverLines)
	}
	m.coverLines = nil
	cmd := m.requestCover()
	if cmd != nil || len(m.coverLines) != 2 || m.coverError != "" {
		t.Fatalf("cache miss: cmd=%v lines=%v err=%q", cmd, m.coverLines, m.coverError)
	}
}

func TestArtCommandOpensCoverOverlay(t *testing.T) {
	m := commandTestModel(t)
	executeTestCommand(m, "art")
	if !m.showCover {
		t.Fatal(":art did not open cover overlay")
	}
}

func TestCoverOverlayKeyBinding(t *testing.T) {
	m := commandTestModel(t)
	m.nowPlaying = &pfs.Track{Name: "One", Path: "/one"}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if !m.showCover {
		t.Fatal("c did not open cover overlay")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.showCover {
		t.Fatal("Esc did not close cover overlay")
	}
}

func TestCoverReadyMsgPopulatesInline(t *testing.T) {
	m := commandTestModel(t)
	m.nowPlaying = &pfs.Track{Name: "One", Path: "/one"}
	m.Update(coverReadyMsg{path: "/one", lines: []string{"ov"}, inline: []string{"in1", "in2"}, source: cover.SourceEmbedded})
	if len(m.coverInline) != 2 || m.coverInline[0] != "in1" {
		t.Fatalf("inline = %v", m.coverInline)
	}
	if m.coverCache["/one"].inline == nil || len(m.coverCache["/one"].inline) != 2 {
		t.Fatalf("cache inline = %v", m.coverCache["/one"].inline)
	}
}

func TestRenderBottomShowsCoverThumbnail(t *testing.T) {
	m := commandTestModel(t)
	m.width, m.height = 100, 30
	m.folders = []*pfs.Folder{{Name: "Rock", Tracks: []pfs.Track{{Name: "One", Path: "/one"}, {Name: "Two", Path: "/two"}}}}
	m.nowPlaying = &pfs.Track{Name: "One", Path: "/one"}
	m.nowMeta = meta.Meta{Title: "One", Artist: "Metallica", Album: "Album"}
	m.coverInline = []string{"ART1", "ART2", "ART3"}
	view := m.renderBottom(m.width)
	if !strings.Contains(view, "ART1") || !strings.Contains(view, "ART2") || !strings.Contains(view, "ART3") {
		t.Fatalf("cover thumbnail missing from bottom bar:\n%s", view)
	}
	if !strings.Contains(view, "Metallica — One") {
		t.Fatalf("now-playing title missing:\n%s", view)
	}
}

func TestPlaybackStartedPreloadsCover(t *testing.T) {
	m := commandTestModel(t)
	m.playbackRequestID = 1
	_, cmd := m.Update(playbackStartedMsg{track: pfs.Track{Name: "One", Path: "/one"}, requestID: 1})
	if cmd == nil {
		t.Fatal("playback start did not preload cover art")
	}
}
