package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Space    key.Binding
	Next     key.Binding
	Prev     key.Binding
	Loop     key.Binding
	VolUp    key.Binding
	VolDown  key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	Left:    key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "folders")),
	Right:   key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "tracks")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play")),
	Space:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "pause")),
	Next:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next")),
	Prev:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev")),
	Loop:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "loop")),
	VolUp:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "vol+")),
	VolDown: key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "vol-")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
