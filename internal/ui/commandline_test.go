package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	pfs "github.com/padros/pmusic/internal/fs"
	luaeng "github.com/padros/pmusic/internal/lua"
	"github.com/padros/pmusic/internal/player"
	"github.com/padros/pmusic/internal/ui/command"
	"github.com/padros/pmusic/internal/ui/command/builtin"
)

func commandTestModel(t *testing.T) *Model {
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
	return &Model{width: 90, height: 30, player: player.New(), luaEngine: luaeng.New(), searchInput: newSearchInput(), musicSearch: newMusicSearchModel(), commandLine: cl, commandHelp: newCommandHelpModel()}
}
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
func executeTestCommand(m *Model, value string) {
	m.openCommandLine()
	m.commandLine.input.SetValue(value)
	m.commandLine.input.CursorEnd()
	m.executeCommandLine()
}
