package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	pfs "github.com/Padrosum/pmusic/internal/fs"
	tea "github.com/charmbracelet/bubbletea"
)

func playbackFixture(t *testing.T, decoder *uiTestDecoder) (*Model, pfs.Track) {
	t.Helper()
	m := commandTestModelWithDecoder(t, decoder)
	track := pfs.Track{Name: "Broken Song", Path: "/music/broken.mp3", Ext: ".mp3"}
	m.folders = []*pfs.Folder{{Name: "Music", Path: "/music", Tracks: []pfs.Track{track}}}
	m.root = m.folders[0]
	m.focused = panelTracks
	return m, track
}

func TestPlaybackFailureMessageIsShown(t *testing.T) {
	decoder := &uiTestDecoder{err: errors.New("invalid frame header")}
	m, track := playbackFixture(t, decoder)
	msg := m.playSelected()()
	m.Update(msg)
	if !strings.Contains(m.notification, track.Name) || !strings.Contains(m.notification, track.Path) || !strings.Contains(m.notification, "invalid frame header") {
		t.Fatalf("notification = %q", m.notification)
	}
}

func TestPlaybackFailurePreservesValidSelection(t *testing.T) {
	decoder := &uiTestDecoder{err: errors.New("decode failed")}
	m, _ := playbackFixture(t, decoder)
	m.folderIdx, m.trackIdx = 0, 0
	msg := m.playSelected()()
	m.Update(msg)
	if m.folderIdx != 0 || m.trackIdx != 0 {
		t.Fatalf("selection changed to folder=%d track=%d", m.folderIdx, m.trackIdx)
	}
}

func TestPlaybackFailureDoesNotLoopForever(t *testing.T) {
	decoder := &uiTestDecoder{err: errors.New("decode failed")}
	m, _ := playbackFixture(t, decoder)
	msg := m.playSelected()()
	m.Update(msg)
	m.Update(tickMsg(time.Now()))
	if got := decoder.calls.Load(); got != 1 {
		t.Fatalf("decode calls = %d, want 1", got)
	}
}

func TestPlaybackStartedUpdatesNowPlaying(t *testing.T) {
	decoder := &uiTestDecoder{}
	m, track := playbackFixture(t, decoder)
	msg := m.playSelected()()
	m.Update(msg)
	if m.nowPlaying == nil || m.nowPlaying.Path != track.Path {
		t.Fatalf("nowPlaying = %#v", m.nowPlaying)
	}
	if m.nowFolder != 0 || m.nowTrack != 0 {
		t.Fatalf("playback indices = %d/%d", m.nowFolder, m.nowTrack)
	}
}

func TestAllPlaybackOriginsUseSharedStartPath(t *testing.T) {
	origins := []playbackOrigin{playbackSelected, playbackNext, playbackPrevious, playbackLoop, playbackQueue, playbackMouse}
	for _, origin := range origins {
		t.Run(string(origin), func(t *testing.T) {
			decoder := &uiTestDecoder{}
			m, track := playbackFixture(t, decoder)
			msg := m.startPlayback(track, origin, 0, 0)()
			started, ok := msg.(playbackStartedMsg)
			if !ok {
				t.Fatalf("message type = %T", msg)
			}
			if started.origin != origin {
				t.Fatalf("origin = %q, want %q", started.origin, origin)
			}
			m.Update(msg)
			if decoder.calls.Load() != 1 || m.nowPlaying == nil {
				t.Fatalf("calls=%d nowPlaying=%#v", decoder.calls.Load(), m.nowPlaying)
			}
		})
	}
}

func assertOverlayDoesNotStopAutoAdvance(t *testing.T, open func(*Model)) {
	t.Helper()
	m := commandTestModel(t)
	tracks := []pfs.Track{
		{Name: "First", Path: "/music/first.mp3", Ext: ".mp3"},
		{Name: "Second", Path: "/music/second.mp3", Ext: ".mp3"},
	}
	m.folders = []*pfs.Folder{{Name: "Music", Path: "/music", Tracks: tracks}}
	m.root = m.folders[0]
	m.nowPlaying = &tracks[0]
	m.nowFolder, m.nowTrack = 0, 0
	open(m)

	m.Update(tickMsg(time.Now()))
	if m.playbackRequestID != 1 {
		t.Fatalf("playback request ID = %d, want auto-advance request", m.playbackRequestID)
	}
}

func TestHelpOverlayDoesNotStopAutoAdvance(t *testing.T) {
	assertOverlayDoesNotStopAutoAdvance(t, func(m *Model) { m.commandHelp.show = true })
}

func TestCommandOverlayDoesNotStopAutoAdvance(t *testing.T) {
	assertOverlayDoesNotStopAutoAdvance(t, func(m *Model) { m.commandLine.active = true })
}

func TestMusicSearchOverlayDoesNotStopAutoAdvance(t *testing.T) {
	assertOverlayDoesNotStopAutoAdvance(t, func(m *Model) { m.showMusicSearch = true })
}

func TestCommandOverlayDoesNotDropLibraryReloadedMsg(t *testing.T) {
	m := commandTestModel(t)
	m.openCommandLine()
	folder := &pfs.Folder{Name: "Reloaded", Path: "/reloaded"}
	m.Update(libraryReloadedMsg{root: folder, folders: []*pfs.Folder{folder}})
	if len(m.folders) != 1 || m.folders[0].Name != "Reloaded" {
		t.Fatalf("folders = %#v", m.folders)
	}
}

func TestHelpOverlayDoesNotDropLuaReloadedMsg(t *testing.T) {
	m := commandTestModel(t)
	m.commandHelp.show = true
	m.Update(luaReloadedMsg{})
	if m.notification != "lua reloaded" {
		t.Fatalf("notification = %q", m.notification)
	}
}

func TestOverlayDoesNotDropPlaybackFailureMsg(t *testing.T) {
	m := commandTestModel(t)
	m.commandLine.active = true
	m.playbackRequestID = 7
	m.Update(playbackFailedMsg{
		track:     pfs.Track{Name: "Bad", Path: "/bad.mp3"},
		err:       errors.New("decode failed"),
		requestID: 7,
	})
	if !strings.Contains(m.notification, "decode failed") {
		t.Fatalf("notification = %q", m.notification)
	}
}

func TestAllOverlaysHandleWindowResize(t *testing.T) {
	overlays := map[string]func(*Model){
		"help":         func(m *Model) { m.commandHelp.show = true },
		"command":      func(m *Model) { m.commandLine.active = true },
		"music search": func(m *Model) { m.showMusicSearch = true },
		"local search": func(m *Model) { m.showSearch = true },
	}
	for name, open := range overlays {
		t.Run(name, func(t *testing.T) {
			m := commandTestModel(t)
			open(m)
			m.Update(tea.WindowSizeMsg{Width: 41, Height: 13})
			if m.width != 41 || m.height != 13 {
				t.Fatalf("size = %dx%d", m.width, m.height)
			}
		})
	}
}

func TestOverlayStillOwnsNormalKeyInput(t *testing.T) {
	m := commandTestModel(t)
	m.openCommandLine()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		if _, quitting := cmd().(tea.QuitMsg); quitting {
			t.Fatal("command input leaked q to global quit")
		}
	}
	if got := m.commandLine.input.Value(); got != "q" {
		t.Fatalf("command input = %q", got)
	}
}
