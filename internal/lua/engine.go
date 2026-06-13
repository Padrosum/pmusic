package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	pmcfg "github.com/padros/pmusic/internal/config"
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

// DefaultTheme returns a modern vibrant color palette (Catppuccin Macchiato inspired).
func DefaultTheme() Theme {
	return Theme{
		Accent:       "#8AADF4", // Macchiato Blue
		Dim:          "#5B6078", // Surface 2
		SelectedBg:   "#363A4F", // Surface 0
		NowPlaying:   "#A6DA95", // Green
		Border:       "#494D64", // Surface 1
		BorderActive: "#8AADF4", // Blue
		Title:        "#F5A97F", // Peach
		StatusBg:     "#1E2030", // Mantle
		PanelBg:      "#24273A", // Base
		Key:          "#C6A0F6", // Mauve
	}
}

// Engine manages the gopher-lua VM lifecycle.
// All exported methods are safe for concurrent use.
type Engine struct {
	mu sync.Mutex
	L  *glua.LState

	theme         Theme
	keymaps       map[string]string           // raw key string → action name
	keyFuncs      map[string]*glua.LFunction  // raw key string → Lua function
	musicDir      string                      // configured music directory
	pendingNotify string                      // consumed by PopNotification

	// current track — updated by CallOnSongChange so pmusic.current_track()
	// always returns live data regardless of when a plugin was loaded.
	currentName   string
	currentPath   string
	currentFolder string

	// Multiple plugins (and init.lua) may each register a hook; all are called
	// in registration order rather than the last one overwriting the others.
	onSongChange  []*glua.LFunction
	onStateChange []*glua.LFunction
}

// New creates an Engine with the default theme and empty keymaps.
func New() *Engine {
	return &Engine{
		theme:    DefaultTheme(),
		keymaps:  make(map[string]string),
		keyFuncs: make(map[string]*glua.LFunction),
	}
}

// SetMusicDir stores the configured music directory so Lua plugins can query it.
func (e *Engine) SetMusicDir(dir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.musicDir = dir
}

// HasKeyFunc reports whether a Lua function is bound to the given key string.
func (e *Engine) HasKeyFunc(key string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.keyFuncs[key]
	return ok
}

// CallKeyFunc calls the Lua function bound to key and returns any error.
func (e *Engine) CallKeyFunc(key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	fn, ok := e.keyFuncs[key]
	if !ok || e.L == nil {
		return nil
	}
	return e.L.CallByParam(glua.P{Fn: fn, NRet: 0, Protect: true})
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
	e.keyFuncs = make(map[string]*glua.LFunction)
	e.onSongChange = nil
	e.onStateChange = nil
	e.pendingNotify = ""
	// currentName/Path/Folder intentionally NOT reset — track is still playing

	initPath, err := configInitPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return nil
	}

	L := glua.NewState()
	luaDir := filepath.Dir(initPath)
	e.registerAPI(L, luaDir)

	enabled, _ := pmcfg.LoadEnabled()
	for _, name := range enabled.Plugins {
		p := filepath.Join(luaDir, "plugins", name+".lua")
		if _, err := os.Stat(p); err == nil {
			if err := L.DoFile(p); err != nil {
				e.pendingNotify = "plugin " + name + ": " + err.Error()
			}
		}
	}
	for _, name := range enabled.Themes {
		p := filepath.Join(luaDir, "themes", name+".lua")
		if _, err := os.Stat(p); err == nil {
			if err := L.DoFile(p); err != nil {
				e.pendingNotify = "theme " + name + ": " + err.Error()
			}
		}
	}

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
	e.currentName = name
	e.currentPath = path
	e.currentFolder = folder
	if len(e.onSongChange) == 0 || e.L == nil {
		return
	}
	t := e.L.NewTable()
	e.L.SetField(t, "name", glua.LString(name))
	e.L.SetField(t, "path", glua.LString(path))
	e.L.SetField(t, "folder", glua.LString(folder))
	for _, fn := range e.onSongChange {
		if err := e.L.CallByParam(glua.P{Fn: fn, NRet: 0, Protect: true}, t); err != nil {
			e.pendingNotify = "lua on_song_change: " + err.Error()
		}
	}
}

// CallOnStateChange fires the on_state_change hook.
// state is one of: "playing", "paused", "stopped".
func (e *Engine) CallOnStateChange(state string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.onStateChange) == 0 || e.L == nil {
		return
	}
	for _, fn := range e.onStateChange {
		if err := e.L.CallByParam(glua.P{Fn: fn, NRet: 0, Protect: true}, glua.LString(state)); err != nil {
			e.pendingNotify = "lua on_state_change: " + err.Error()
		}
	}
}

func configInitPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pmusic", "lua", "init.lua"), nil
}
