package command

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type ParsedCommand struct {
	Raw       string
	Name      string
	Args      []string
	Flags     map[string]string
	BoolFlags map[string]bool
	Bang      bool
}

type ArgumentSpec struct {
	Name        string
	Description string
	Required    bool
}

type FlagSpec struct {
	Name        string
	Description string
	TakesValue  bool
}

type Subcommand struct {
	Name        string
	Description string
}

type CompletionKind int

const (
	CompletionCommand CompletionKind = iota
	CompletionAlias
	CompletionSubcommand
	CompletionArgument
)

type CompletionItem struct {
	Value       string
	Display     string
	Description string
	Kind        CompletionKind
}

type CompletionState struct {
	Items    []CompletionItem
	Selected int
	Visible  bool
}

func (s *CompletionState) Next() {
	if len(s.Items) == 0 {
		return
	}
	s.Selected = (s.Selected + 1) % len(s.Items)
}

func (s *CompletionState) Previous() {
	if len(s.Items) == 0 {
		return
	}
	s.Selected--
	if s.Selected < 0 {
		s.Selected = len(s.Items) - 1
	}
}

// Runtime is the deliberately small bridge between reusable command handlers
// and pmusic's Bubble Tea model. Implementations must mutate UI state only from
// the Update goroutine; slow work is returned as tea.Cmd.
type Runtime interface {
	Play(query string) (tea.Cmd, error)
	Pause() error
	TogglePause() error
	Next() (tea.Cmd, error)
	Previous() (tea.Cmd, error)
	Volume() int
	SetVolume(value int) error
	Muted() bool
	Mute(mode string) error
	Position() (elapsed, total time.Duration, loaded bool)
	SeekAbsolute(position time.Duration) error
	Loop() bool
	SetLoop(enabled bool) error
	OpenQueue()
	ClearQueue() (int, error)
	OpenLocalSearch(query string)
	OpenOnlineSearch(query string, start bool) tea.Cmd
	ReloadLua() tea.Cmd
	ReloadLibrary() tea.Cmd
	OpenHelp(topic string) error
	OpenHistory(limit int)
	ClearHistory() error
	Notify(message string)
	Quit(force bool) (tea.Cmd, error)
	TrackCompletions(query string, limit int) []CompletionItem
}

type Handler func(Runtime, ParsedCommand) (tea.Cmd, error)
type Completer func(Runtime, string) []CompletionItem

type Command struct {
	Name        string
	Aliases     []string
	Category    string
	Summary     string
	Description string
	Usage       string
	Examples    []string
	Related     []string
	Subcommands []Subcommand
	Arguments   []ArgumentSpec
	Flags       []FlagSpec
	Complete    Completer
	Execute     Handler
}
