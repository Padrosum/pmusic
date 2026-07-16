package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type commandHelpModel struct {
	show         bool
	topic        string
	filter       textinput.Model
	filtering    bool
	offset       int
	lines        []string
	historyLines []string
}

func newCommandHelpModel() commandHelpModel {
	ti := textinput.New()
	ti.Placeholder = "filter commands"
	ti.CharLimit = 100
	return commandHelpModel{filter: ti}
}
func (m *Model) openRegistryHelp(topic string) error {
	if strings.EqualFold(topic, "keys") {
		m.showHelp = true
		return nil
	}
	lines, err := m.commandLine.registry.Help(topic, "")
	if err != nil {
		return err
	}
	m.commandHelp.show = true
	m.commandHelp.topic = topic
	m.commandHelp.lines = lines
	m.commandHelp.offset = 0
	m.commandHelp.filter.SetValue("")
	m.commandHelp.filtering = false
	m.commandHelp.historyLines = nil
	return nil
}
func (m *Model) openHistoryHelp(limit int) {
	entries := m.commandHistoryEntries(limit)
	lines := []string{"Command History", ""}
	if len(entries) == 0 {
		lines = append(lines, "No commands in history.")
	} else {
		start := len(m.commandLine.history.Entries()) - len(entries)
		for i, v := range entries {
			lines = append(lines, formatHistoryLine(start+i+1, v))
		}
	}
	lines = append(lines, "", "j/k:scroll  PgUp/PgDn  g/G  Esc/q:close")
	m.commandHelp.show = true
	m.commandHelp.lines = lines
	m.commandHelp.offset = 0
	m.commandHelp.historyLines = lines
}
func formatHistoryLine(i int, v string) string {
	return strings.Join([]string{" ", itoa(i), "  :", v}, "")
}
func itoa(v int) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = digits[v%10]
		v /= 10
	}
	return string(b[i:])
}

func (m *Model) handleCommandHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.commandHelp.filtering {
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
				m.commandHelp.filtering = false
				m.commandHelp.filter.Blur()
				return m, nil
			}
		}
		before := m.commandHelp.filter.Value()
		var cmd tea.Cmd
		m.commandHelp.filter, cmd = m.commandHelp.filter.Update(msg)
		if before != m.commandHelp.filter.Value() {
			lines, err := m.commandLine.registry.Help(m.commandHelp.topic, m.commandHelp.filter.Value())
			if err == nil {
				m.commandHelp.lines = lines
				m.commandHelp.offset = 0
			}
		}
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		return m, tickCmd()
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.commandHelp.show = false
		case "j", "down":
			m.commandHelp.offset = min(max(0, len(m.commandHelp.lines)-1), m.commandHelp.offset+1)
		case "k", "up":
			m.commandHelp.offset = max(0, m.commandHelp.offset-1)
		case "pgdown":
			m.commandHelp.offset = min(max(0, len(m.commandHelp.lines)-1), m.commandHelp.offset+max(1, m.height-8))
		case "pgup":
			m.commandHelp.offset = max(0, m.commandHelp.offset-max(1, m.height-8))
		case "g":
			m.commandHelp.offset = 0
		case "G":
			m.commandHelp.offset = max(0, len(m.commandHelp.lines)-max(1, m.height-6))
		case "/":
			if m.commandHelp.historyLines == nil {
				m.commandHelp.filtering = true
				return m, m.commandHelp.filter.Focus()
			}
		}
	}
	return m, nil
}
