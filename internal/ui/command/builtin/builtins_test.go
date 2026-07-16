package builtin

import (
	"errors"
	"testing"
	"time"

	"github.com/Padrosum/pmusic/internal/ui/command"
	tea "github.com/charmbracelet/bubbletea"
)

type mockRuntime struct {
	volume         int
	muted          bool
	elapsed, total time.Duration
	loaded         bool
	loop           bool
	notices        []string
}

func (*mockRuntime) Play(string) (tea.Cmd, error) { return nil, nil }
func (*mockRuntime) Pause() error                 { return nil }
func (*mockRuntime) TogglePause() error           { return nil }
func (*mockRuntime) Next() (tea.Cmd, error)       { return nil, nil }
func (*mockRuntime) Previous() (tea.Cmd, error)   { return nil, nil }
func (m *mockRuntime) Volume() int                { return m.volume }
func (m *mockRuntime) SetVolume(v int) error      { m.volume = v; return nil }
func (m *mockRuntime) Muted() bool                { return m.muted }
func (m *mockRuntime) Mute(mode string) error {
	switch mode {
	case "mute":
		m.volume = 0
		m.muted = true
	case "unmute":
		m.volume = 75
		m.muted = false
	case "toggle":
		if m.muted {
			m.volume = 75
			m.muted = false
		} else {
			m.volume = 0
			m.muted = true
		}
	}
	return nil
}
func (m *mockRuntime) Position() (time.Duration, time.Duration, bool) {
	return m.elapsed, m.total, m.loaded
}
func (m *mockRuntime) SeekAbsolute(v time.Duration) error  { m.elapsed = v; return nil }
func (m *mockRuntime) Loop() bool                          { return m.loop }
func (m *mockRuntime) SetLoop(v bool) error                { m.loop = v; return nil }
func (*mockRuntime) OpenQueue()                            {}
func (*mockRuntime) ClearQueue() (int, error)              { return 0, nil }
func (*mockRuntime) OpenLocalSearch(string)                {}
func (*mockRuntime) OpenOnlineSearch(string, bool) tea.Cmd { return nil }
func (*mockRuntime) ReloadLua() tea.Cmd                    { return nil }
func (*mockRuntime) ReloadLibrary() tea.Cmd                { return nil }
func (*mockRuntime) OpenHelp(string) error                 { return nil }
func (*mockRuntime) OpenHistory(int)                       {}
func (*mockRuntime) OpenStats(string, string) error        { return nil }
func (*mockRuntime) ClearHistory() error                   { return nil }
func (m *mockRuntime) Notify(v string)                     { m.notices = append(m.notices, v) }
func (*mockRuntime) Quit(bool) (tea.Cmd, error)            { return nil, nil }
func (*mockRuntime) TrackCompletions(string, int) []command.CompletionItem {
	return []command.CompletionItem{
		{Value: "Metallica — One", Display: "Metallica — One", Kind: command.CompletionArgument},
		{Value: "Duman — Seni Kendime Sakladım", Display: "Duman — Seni Kendime Sakladım", Kind: command.CompletionArgument},
	}
}

func TestParseSeek(t *testing.T) {
	duration := 2 * time.Hour
	tests := []struct {
		in       string
		want     time.Duration
		relative bool
	}{{"+30", 30 * time.Second, true}, {"-10", -10 * time.Second, true}, {"90", 90 * time.Second, false}, {"1:30", 90 * time.Second, false}, {"01:02:30", time.Hour + 2*time.Minute + 30*time.Second, false}, {"0%", 0, false}, {"50%", time.Hour, false}, {"100%", duration, false}, {"start", 0, false}, {"end", duration - time.Millisecond, false}}
	for _, tt := range tests {
		got, err := ParseSeek(tt.in, duration)
		if err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		if got.Position != tt.want || got.Relative != tt.relative {
			t.Fatalf("%s=%#v", tt.in, got)
		}
	}
}
func TestParseSeekErrors(t *testing.T) {
	for _, in := range []string{"140%", "1:99", "1:2:3:4", "banana", "-1:30"} {
		if _, err := ParseSeek(in, time.Minute); err == nil {
			t.Fatalf("%q accepted", in)
		}
	}
	if _, err := ParseSeek("50%", 0); err == nil {
		t.Fatal("unknown duration percentage accepted")
	}
	if _, err := ParseSeek("end", 0); err == nil {
		t.Fatal("unknown duration end accepted")
	}
}
func TestVolumeHandler(t *testing.T) {
	r := &mockRuntime{volume: 50}
	for _, tt := range []struct {
		arg  string
		want int
	}{{"60", 60}, {"+50", 100}, {"-150", 0}} {
		p := command.ParsedCommand{Name: "volume"}
		if tt.arg != "" {
			p.Args = []string{tt.arg}
		}
		_, err := volume(r, p)
		if err != nil {
			t.Fatal(err)
		}
		if r.volume != tt.want {
			t.Fatalf("%s volume=%d want=%d", tt.arg, r.volume, tt.want)
		}
	}
	if _, err := volume(r, command.ParsedCommand{Args: []string{"banana"}}); err == nil {
		t.Fatal("invalid volume accepted")
	}
	_, _ = volume(r, command.ParsedCommand{Args: []string{"mute"}})
	if r.volume != 0 {
		t.Fatal("mute")
	}
	_, _ = volume(r, command.ParsedCommand{Args: []string{"unmute"}})
	if r.volume == 0 {
		t.Fatal("unmute")
	}
}
func TestBuiltinRegistryMetadataAndAliases(t *testing.T) {
	r, err := command.NewRegistry(Commands()...)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pl", "pa", "t", "n", "previous", "p", "vol", "v", "sk", "repeat", "find", "yt", "dl", "source", "q", "h", "hist", "statistics"} {
		if _, ok := r.Resolve(name); !ok {
			t.Errorf("alias %s missing", name)
		}
	}
	if _, err = r.Execute(&mockRuntime{}, command.ParsedCommand{Name: "missing"}); err == nil {
		t.Fatal("unknown command accepted")
	} else {
		var unknown *command.UnknownCommandError
		if !errors.As(err, &unknown) {
			t.Fatalf("error=%T", err)
		}
	}
}

func TestBuiltinArgumentCompletions(t *testing.T) {
	r, err := command.NewRegistry(Commands()...)
	if err != nil {
		t.Fatal(err)
	}
	rt := &mockRuntime{}
	for _, tt := range []struct{ input, want string }{
		{"volume ", "mute"}, {"loop ", "toggle"}, {"seek ", "50%"},
		{"reload l", "library"}, {"queue c", "clear"},
		{"stats w", "week"},
		{"play Met", "Metallica — One"}, {"play Duman", "Duman — Seni Kendime Sakladım"},
	} {
		items, _ := r.Complete(rt, tt.input, len([]rune(tt.input)), 10)
		found := false
		for _, item := range items {
			if item.Value == tt.want {
				found = true
			}
		}
		if !found {
			t.Errorf("Complete(%q)=%v; want %q", tt.input, items, tt.want)
		}
	}
}
