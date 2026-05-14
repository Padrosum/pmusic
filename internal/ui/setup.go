package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SetupDoneMsg struct{ Dir string }

type SetupModel struct {
	input textinput.Model
	err   string
	w, h  int
	Result string // populated when the user confirms a valid directory
}

func NewSetup() SetupModel {
	ti := textinput.New()
	ti.Placeholder = "/home/user/Music"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	// Pre-fill with ~/Music if it exists.
	if home, err := os.UserHomeDir(); err == nil {
		candidate := home + "/Music"
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			ti.SetValue(candidate)
		}
	}

	return SetupModel{input: ti}
}

func (s SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (s SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.w, s.h = msg.Width, msg.Height
		return s, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return s, tea.Quit

		case tea.KeyEnter:
			dir := strings.TrimSpace(s.input.Value())
			if dir == "" {
				s.err = "  path cannot be empty"
				return s, nil
			}
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				s.err = "  not a valid directory: " + dir
				return s, nil
			}
			s.Result = dir
			return s, tea.Quit
		}
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s SetupModel) View() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(nord8).
		Background(nord0).
		Padding(1, 3).
		Width(60)

	title := styleTitle.Render("pmusic — first run setup")
	prompt := styleNormal.Render("\n  Enter your music directory:\n\n  ") + s.input.View()

	errLine := ""
	if s.err != "" {
		errLine = "\n" + lipgloss.NewStyle().Foreground(nord11).Render(s.err)
	}

	hint := "\n\n" + styleDim.Render("  enter: confirm   esc: quit")
	content := title + prompt + errLine + hint

	dialog := box.Render(content)

	if s.w == 0 {
		return dialog
	}
	return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceBackground(nord0))
}
