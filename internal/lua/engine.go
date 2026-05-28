package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	glua "github.com/yuin/gopher-lua"
)

// Theme holds lipgloss-compatible hex color strings for UI styling.
type Theme struct {
	Accent       string
	Dim          string
	SelectedBg   string
	NowPlaying   string
	Border       string
	BorderActive string
	Title        string
	StatusBg     string
	PanelBg      string
	Key          string
}

// DefaultTheme returns the built-in Nord color palette.
func DefaultTheme() Theme {
	return Theme{
		Accent:       "#88C0D0",
		Dim:          "#4C566A",
		SelectedBg:   "#434C5E",
		NowPlaying:   "#A3BE8C",
		Border:       "#4C566A",
		BorderActive: "#88C0D0",
		Title:        "#88C0D0",
		StatusBg:     "#3B4252",
		PanelBg:      "#2E3440",
		Key:          "#81A1C1",
	}
}

// Engine manages the gopher-lua VM lifecycle.
// All exported methods are safe for concurrent use.
type Engine struct {
	mu sync.Mutex
	L  *glua.LState

	theme         Theme
	keymaps       map[string]string // raw key string → action name
	pendingNotify string            // consumed by PopNotification

	onSongChange  *glua.LFunction
	onStateChange *glua.LFunction
}

// New creates an Engine with the default theme and empty keymaps.
func New() *Engine {
	return &Engine{
		theme:   DefaultTheme(),
		keymaps: make(map[string]string),
	}
}

// Load (re)initializes the Lua VM and runs ~/.config/pmusic/lua/init.lua.
// Resets all Lua-managed state first. Safe to call multiple times (hot-reload).
// Returns nil if init.lua does not exist.
func (e *Engine) Load() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.L != nil {
		e.L.Close()
		e.L = nil
	}
	e.theme = DefaultTheme()
	e.keymaps = make(map[string]string)
	e.onSongChange = nil
	e.onStateChange = nil
	e.pendingNotify = ""

	initPath, err := configInitPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return nil
	}

	L := glua.NewState()
	e.registerAPI(L, filepath.Dir(initPath))

	if err := L.DoFile(initPath); err != nil {
		L.Close()
		return fmt.Errorf("lua: %w", err)
	}
	e.L = L
	return nil
}

// Close shuts down the Lua VM and releases resources.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.L != nil {
		e.L.Close()
		e.L = nil
	}
}

// Theme returns a copy of the current theme (possibly user-overridden).
func (e *Engine) Theme() Theme {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.theme
}

// Keymap returns the action name bound to a key string, or "" if unbound.
func (e *Engine) Keymap(key string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.keymaps[key]
}

// PopNotification returns and clears any pending message from pmusic.notify().
func (e *Engine) PopNotification() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := e.pendingNotify
	e.pendingNotify = ""
	return n
}

// CallOnSongChange fires the on_song_change hook with track metadata.
func (e *Engine) CallOnSongChange(name, path, folder string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.onSongChange == nil || e.L == nil {
		return
	}
	t := e.L.NewTable()
	e.L.SetField(t, "name", glua.LString(name))
	e.L.SetField(t, "path", glua.LString(path))
	e.L.SetField(t, "folder", glua.LString(folder))
	if err := e.L.CallByParam(glua.P{Fn: e.onSongChange, NRet: 0, Protect: true}, t); err != nil {
		e.pendingNotify = "lua on_song_change: " + err.Error()
	}
}

// CallOnStateChange fires the on_state_change hook.
// state is one of: "playing", "paused", "stopped".
func (e *Engine) CallOnStateChange(state string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.onStateChange == nil || e.L == nil {
		return
	}
	if err := e.L.CallByParam(glua.P{Fn: e.onStateChange, NRet: 0, Protect: true}, glua.LString(state)); err != nil {
		e.pendingNotify = "lua on_state_change: " + err.Error()
	}
}

func configInitPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pmusic", "lua", "init.lua"), nil
}
