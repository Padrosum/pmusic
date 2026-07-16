package ui

import (
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	pfs "github.com/Padrosum/pmusic/internal/fs"
	"github.com/Padrosum/pmusic/internal/listening"
	luaeng "github.com/Padrosum/pmusic/internal/lua"
	"github.com/Padrosum/pmusic/internal/player"
	"github.com/Padrosum/pmusic/internal/ui/command"
	"github.com/Padrosum/pmusic/internal/ui/command/builtin"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/faiface/beep"
)

func commandTestModel(t *testing.T) *Model {
	return commandTestModelWithDecoder(t, &uiTestDecoder{})
}

func commandTestModelWithDecoder(t *testing.T, decoder player.Decoder) *Model {
	t.Helper()
	r, err := command.NewRegistry(builtin.Commands()...)
	if err != nil {
		t.Fatal(err)
	}
	h, err := command.LoadHistory(filepath.Join(t.TempDir(), "history"))
	if err != nil {
		t.Fatal(err)
	}
	cl := commandLineModel{input: newCommandInput(), registry: r, history: h}
	p, err := player.NewWithBackend(uiTestAudioBackend{}, decoder)
	if err != nil {
		t.Fatal(err)
	}
	return &Model{width: 90, height: 30, player: p, luaEngine: luaeng.New(), searchInput: newSearchInput(), musicSearch: newMusicSearchModel(), commandLine: cl, commandHelp: newCommandHelpModel()}
}

type uiTestAudioBackend struct {
	closeCalls *atomic.Int32
}

func (uiTestAudioBackend) Init(beep.SampleRate, int) error { return nil }
func (uiTestAudioBackend) Lock()                           {}
func (uiTestAudioBackend) Unlock()                         {}
func (uiTestAudioBackend) Clear()                          {}
func (uiTestAudioBackend) Play(beep.Streamer)              {}
func (b uiTestAudioBackend) Close() error {
	if b.closeCalls != nil {
		b.closeCalls.Add(1)
	}
	return nil
}

type uiTestDecoder struct {
	err   error
	calls atomic.Int32
}

func (d *uiTestDecoder) Decode(string) (beep.StreamSeekCloser, beep.Format, error) {
	d.calls.Add(1)
	if d.err != nil {
		return nil, beep.Format{}, d.err
	}
	return &uiTestStream{length: int(beep.SampleRate(44100))}, beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2}, nil
}

type uiTestStream struct {
	position int
	length   int
}

func (*uiTestStream) Stream([][2]float64) (int, bool) { return 0, true }
func (*uiTestStream) Err() error                      { return nil }
func (s *uiTestStream) Len() int                      { return s.length }
func (s *uiTestStream) Position() int                 { return s.position }
func (s *uiTestStream) Seek(position int) error       { s.position = position; return nil }
func (*uiTestStream) Close() error                    { return nil }
func newCommandInput() textinput.Model {
	ti := textinput.New()
	ti.CharLimit = 2000
	ti.Prompt = ""
	return ti
}
func newSearchInput() textinput.Model { ti := textinput.New(); return ti }

func TestColonOpensCommandModeAndConsumesNormalKeys(t *testing.T) {
	m := commandTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if !m.commandLine.active || cmd == nil {
		t.Fatalf("active=%v cmd=%v", m.commandLine.active, cmd)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !m.commandLine.active || m.commandLine.input.Value() != "q" {
		t.Fatalf("q leaked: active=%v value=%q", m.commandLine.active, m.commandLine.input.Value())
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.commandLine.active {
		t.Fatal("Esc did not close command mode")
	}
}
func TestCommandEnterCompletionAndFocus(t *testing.T) {
	m := commandTestModel(t)
	m.openCommandLine()
	m.commandLine.input.SetValue("vo")
	m.commandLine.input.CursorEnd()
	m.refreshCompletions()
	m.handleCommandLine(tea.KeyMsg{Type: tea.KeyTab})
	if m.commandLine.input.Value() != "volume" {
		t.Fatalf("completion=%q", m.commandLine.input.Value())
	}
	m.commandLine.input.SetValue("volume 60")
	m.commandLine.input.CursorEnd()
	m.handleCommandLine(tea.KeyMsg{Type: tea.KeyEnter})
	if m.commandLine.active || m.Volume() != 60 {
		t.Fatalf("active=%v volume=%d", m.commandLine.active, m.Volume())
	}
}

func TestTabCyclesMultipleCompletions(t *testing.T) {
	m := commandTestModel(t)
	m.openCommandLine()
	m.commandLine.input.SetValue("reload l")
	m.commandLine.input.CursorEnd()
	m.refreshCompletions()
	m.handleCommandLine(tea.KeyMsg{Type: tea.KeyTab})
	if got := m.commandLine.input.Value(); got != "reload lua" {
		t.Fatalf("first completion=%q", got)
	}
	m.handleCommandLine(tea.KeyMsg{Type: tea.KeyTab})
	if got := m.commandLine.input.Value(); got != "reload library" {
		t.Fatalf("second completion=%q", got)
	}
	m.handleCommandLine(tea.KeyMsg{Type: tea.KeyShiftTab})
	if got := m.commandLine.input.Value(); got != "reload lua" {
		t.Fatalf("reverse completion=%q", got)
	}
}
func TestCommandSearchDownloadHelpAndQuit(t *testing.T) {
	m := commandTestModel(t)
	m.folders = []*pfs.Folder{{Name: "Rock", Tracks: []pfs.Track{{Name: "Metallica", Path: "/missing.mp3"}}}}
	executeTestCommand(m, "search foo")
	if !m.showSearch || m.searchInput.Value() != "foo" || !m.searchInput.Focused() {
		t.Fatalf("search state: show=%v value=%q focus=%v", m.showSearch, m.searchInput.Value(), m.searchInput.Focused())
	}
	m.showSearch = false
	m.searchInput.Blur()
	executeTestCommand(m, "download foo")
	if !m.showMusicSearch || m.musicSearch.query != "foo" || m.musicSearch.state != musicSearchLoading {
		t.Fatalf("download state=%v query=%q show=%v", m.musicSearch.state, m.musicSearch.query, m.showMusicSearch)
	}
	m.showMusicSearch = false
	executeTestCommand(m, "help seek")
	if !m.commandHelp.show || !strings.Contains(strings.Join(m.commandHelp.lines, "\n"), ":seek") {
		t.Fatalf("help=%v", m.commandHelp.lines)
	}
	m.commandHelp.show = false
	m.openCommandLine()
	m.commandLine.input.SetValue("q")
	cmd := m.executeCommandLine()
	if cmd == nil {
		t.Fatal("quit returned nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("quit msg=%T", cmd())
	}
}
func TestCommandLineSmallViewDoesNotPanic(t *testing.T) {
	m := commandTestModel(t)
	m.width, m.height = 20, 5
	m.openCommandLine()
	m.commandLine.input.SetValue("seek")
	m.refreshCompletions()
	if view := m.View(); view == "" {
		t.Fatal("empty view")
	}
}
func TestStatsCommandOpensRecordedActivity(t *testing.T) {
	m := commandTestModel(t)
	m.listening, _ = listening.Load("")
	now := time.Now()
	m.listening.Start(listening.Track{Path: "/one", Name: "One", Artist: "Metallica"}, now)
	m.listening.Listen(65*time.Second, now)
	m.listening.Finish(true, now)
	executeTestCommand(m, "stats artist Metallica")
	view := strings.Join(m.commandHelp.lines, "\n")
	if !m.commandHelp.show || !strings.Contains(view, "Artist · Metallica") || !strings.Contains(view, "Tracks started   1") {
		t.Fatalf("stats overlay:\n%s", view)
	}
}
func TestAdaptiveTickIntervals(t *testing.T) {
	m := commandTestModel(t)
	if got := m.tickInterval(); got != time.Second {
		t.Fatalf("idle tick=%v", got)
	}
	m.player.MarkPending()
	if got := m.tickInterval(); got != time.Second/4 {
		t.Fatalf("playing tick=%v", got)
	}
	m.player.Stop()
	m.showMusicSearch = true
	m.musicSearch.state = musicSearchLoading
	if got := m.tickInterval(); got != time.Second/4 {
		t.Fatalf("loading tick=%v", got)
	}
}
func TestPlaybackLifecycleRecordsListeningCompletionAndSkip(t *testing.T) {
	m := commandTestModel(t)
	m.listening, _ = listening.Load("")
	now := time.Now()
	m.nowPlaying = &pfs.Track{Name: "One", Path: "/one"}
	m.player.MarkPending()
	m.Update(tickMsg(now))
	m.Update(tickMsg(now.Add(time.Second)))
	m.nowPlaying = &pfs.Track{Name: "Two", Path: "/two"}
	m.Update(tickMsg(now.Add(2 * time.Second)))
	m.player.Stop()
	m.Update(tickMsg(now.Add(3 * time.Second)))
	summary := m.listening.Period(1, now.Add(3*time.Second))
	if summary.Plays != 2 || summary.Skips != 1 || summary.Completions != 1 || summary.ListeningSeconds != 1 {
		t.Fatalf("lifecycle summary=%#v", summary)
	}
}
func executeTestCommand(m *Model, value string) {
	m.openCommandLine()
	m.commandLine.input.SetValue(value)
	m.commandLine.input.CursorEnd()
	m.executeCommandLine()
}
