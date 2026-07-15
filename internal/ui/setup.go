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
	input  textinput.Model
	err    string
	w, h   int
	frame  int
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
	return tea.Batch(textinput.Blink, tickCmd())
}

func (s SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.w, s.h = msg.Width, msg.Height
		return s, nil

	case tickMsg:
		s.frame = (s.frame + 1) % 120
		return s, tickCmd()

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
	boxW := 62
	if s.w > 0 {
		boxW = min(boxW, max(36, s.w-6))
	}
	s.input.Width = max(20, boxW-10)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#8AADF4")).
		Background(lipgloss.Color("#24273A")).
		Padding(1, 3).
		Width(boxW)

	brand := styleLogo.Render("♪ PMUSIC") + "  " + styleVisualizer.Render(visualizer(s.frame, 9))
	title := "\n" + styleTitle.Render("LET'S BUILD YOUR LIBRARY")
	prompt := styleDim.Render("\nChoose the folder where your music lives.\n\n") +
		styleHeaderMeta.Render("MUSIC DIRECTORY\n") + s.input.View()

	errLine := ""
	if s.err != "" {
		errLine = "\n\n" + styleError.Render("● "+strings.TrimSpace(s.err))
	}

	hint := "\n\n" + styleKey.Render("ENTER") + styleDim.Render(" confirm    ") +
		styleKey.Render("ESC") + styleDim.Render(" quit")
	content := brand + title + prompt + errLine + hint

	dialog := box.Render(content)

	if s.w == 0 {
		return dialog
	}
	return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#1E2030")))
}
