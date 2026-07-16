package ui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Padrosum/pmusic/internal/ui/command"
	"github.com/Padrosum/pmusic/internal/ui/command/builtin"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type commandLineModel struct {
	active      bool
	input       textinput.Model
	registry    *command.Registry
	history     *command.History
	completion  command.CompletionState
	rangeInfo   command.TokenRange
	initWarning error
	cycleItems  []command.CompletionItem
	cycleInput  string
	cycleRange  command.TokenRange
	cycleIndex  int
}

func newCommandLine() (commandLineModel, error) {
	r, err := command.NewRegistry(builtin.Commands()...)
	if err != nil {
		return commandLineModel{}, err
	}
	path, pathErr := command.DefaultHistoryPath()
	var h *command.History
	if pathErr == nil {
		h, err = command.LoadHistory(path)
	}
	if h == nil {
		h, err = command.LoadHistory("")
		if err != nil {
			return commandLineModel{}, fmt.Errorf("initialize in-memory history: %w", err)
		}
	}
	ti := textinput.New()
	ti.CharLimit = 2000
	ti.Prompt = ""
	ti.Placeholder = "command"
	if pathErr != nil {
		err = pathErr
	}
	return commandLineModel{input: ti, registry: r, history: h, initWarning: err}, nil
}

func (m *Model) openCommandLine() tea.Cmd {
	m.commandLine.active = true
	m.commandLine.input.SetValue("")
	m.commandLine.input.SetCursor(0)
	m.commandLine.history.Reset("")
	m.resetCompletionCycle()
	m.refreshCompletions()
	return m.commandLine.input.Focus()
}
func (m *Model) closeCommandLine() {
	m.commandLine.active = false
	m.commandLine.input.Blur()
	m.commandLine.completion = command.CompletionState{}
	m.resetCompletionCycle()
}

func (m *Model) refreshCompletions() {
	if strings.TrimSpace(m.commandLine.input.Value()) == "" {
		m.commandLine.completion = command.CompletionState{}
		return
	}
	items, rng := m.commandLine.registry.Complete(m, m.commandLine.input.Value(), m.commandLine.input.Position(), 10)
	m.commandLine.rangeInfo = rng
	m.commandLine.completion.Items = items
	m.commandLine.completion.Visible = len(items) > 0
	if m.commandLine.completion.Selected >= len(items) {
		m.commandLine.completion.Selected = 0
	}
}

func (m *Model) cycleCompletion(direction int) {
	if len(m.commandLine.cycleItems) == 0 {
		m.commandLine.cycleItems = append([]command.CompletionItem(nil), m.commandLine.completion.Items...)
		m.commandLine.cycleInput = m.commandLine.input.Value()
		m.commandLine.cycleRange = m.commandLine.rangeInfo
		m.commandLine.cycleIndex = m.commandLine.completion.Selected
		if direction < 0 {
			m.commandLine.cycleIndex = (m.commandLine.cycleIndex - 1 + len(m.commandLine.cycleItems)) % len(m.commandLine.cycleItems)
		}
	} else {
		m.commandLine.cycleIndex = (m.commandLine.cycleIndex + direction + len(m.commandLine.cycleItems)) % len(m.commandLine.cycleItems)
	}
	if len(m.commandLine.cycleItems) == 0 {
		return
	}
	value, pos := command.ReplaceToken(m.commandLine.cycleInput, m.commandLine.cycleRange, m.commandLine.cycleItems[m.commandLine.cycleIndex].Value)
	m.commandLine.input.SetValue(value)
	m.commandLine.input.SetCursor(pos)
	m.commandLine.completion = command.CompletionState{Items: m.commandLine.cycleItems, Selected: m.commandLine.cycleIndex, Visible: true}
}

func (m *Model) resetCompletionCycle() {
	m.commandLine.cycleItems = nil
	m.commandLine.cycleInput = ""
	m.commandLine.cycleIndex = 0
}

func (m *Model) handleCommandLine(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		return m, m.tickCmd()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.closeCommandLine()
			return m, nil
		case tea.KeyEnter:
			return m, m.executeCommandLine()
		case tea.KeyTab:
			if len(m.commandLine.completion.Items) > 0 {
				m.cycleCompletion(1)
			}
			return m, nil
		case tea.KeyShiftTab:
			m.cycleCompletion(-1)
			return m, nil
		case tea.KeyCtrlN:
			m.commandLine.completion.Next()
			return m, nil
		case tea.KeyCtrlP:
			m.commandLine.completion.Previous()
			return m, nil
		case tea.KeyUp:
			if m.commandLine.completion.Visible && len(m.commandLine.completion.Items) > 1 {
				m.commandLine.completion.Previous()
			} else {
				m.commandLine.input.SetValue(m.commandLine.history.Previous(m.commandLine.input.Value()))
				m.commandLine.input.CursorEnd()
			}
			return m, nil
		case tea.KeyDown:
			if m.commandLine.completion.Visible && len(m.commandLine.completion.Items) > 1 {
				m.commandLine.completion.Next()
			} else {
				m.commandLine.input.SetValue(m.commandLine.history.Next())
				m.commandLine.input.CursorEnd()
			}
			return m, nil
		case tea.KeyCtrlU:
			m.resetCompletionCycle()
			m.commandLine.input.SetValue("")
			m.commandLine.input.SetCursor(0)
			m.refreshCompletions()
			return m, nil
		case tea.KeyCtrlW:
			m.resetCompletionCycle()
			m.deleteCommandWord()
			m.refreshCompletions()
			return m, nil
		}
	}
	before := m.commandLine.input.Value()
	var cmd tea.Cmd
	m.commandLine.input, cmd = m.commandLine.input.Update(msg)
	if before != m.commandLine.input.Value() {
		m.resetCompletionCycle()
		m.refreshCompletions()
	}
	return m, cmd
}

func (m *Model) deleteCommandWord() {
	r := []rune(m.commandLine.input.Value())
	pos := m.commandLine.input.Position()
	start := pos
	for start > 0 && unicode.IsSpace(r[start-1]) {
		start--
	}
	for start > 0 && !unicode.IsSpace(r[start-1]) {
		start--
	}
	m.commandLine.input.SetValue(string(r[:start]) + string(r[pos:]))
	m.commandLine.input.SetCursor(start)
}

func (m *Model) executeCommandLine() tea.Cmd {
	raw := m.commandLine.input.Value()
	trimmed := strings.TrimSpace(raw)
	m.closeCommandLine()
	if trimmed == "" {
		return nil
	}
	p, err := command.Parse(raw)
	// history clear intentionally does not re-add itself after clearing.
	isClear := err == nil && (strings.EqualFold(p.Name, "history") || strings.EqualFold(p.Name, "hist")) && len(p.Args) > 0 && strings.EqualFold(p.Args[0], "clear")
	if !isClear {
		if historyErr := m.commandLine.history.Add(raw); historyErr != nil {
			m.notify("history: " + historyErr.Error())
		}
	}
	if err != nil {
		m.notifyMultiline(err.Error())
		return nil
	}
	c, ok := m.commandLine.registry.Resolve(p.Name)
	if !ok {
		m.notifyMultiline((&command.UnknownCommandError{Name: p.Name, Suggestions: m.commandLine.registry.Suggest(p.Name, 3)}).Error())
		return nil
	}
	cmd, err := c.Execute(m, p)
	if err != nil {
		m.notifyMultiline(err.Error())
		return cmd
	}
	return cmd
}

func (m *Model) notifyMultiline(value string) { m.notify(strings.ReplaceAll(value, "\n", " · ")) }

func (m *Model) commandHistoryEntries(limit int) []string {
	values := m.commandLine.history.Entries()
	if limit > 0 && len(values) > limit {
		values = values[len(values)-limit:]
	}
	return values
}
func (m *Model) commandRegistry() *command.Registry { return m.commandLine.registry }
