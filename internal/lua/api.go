package lua

import (
	"path/filepath"

	glua "github.com/yuin/gopher-lua"
)

// registerAPI installs the pmusic global table and extends package.path.
// Must be called with e.mu held (invoked only from Load).
func (e *Engine) registerAPI(L *glua.LState, dir string) {
	// Let require() find scripts in the user's lua config dir.
	if pkg, ok := L.GetGlobal("package").(*glua.LTable); ok {
		cur := L.GetField(pkg, "path").String()
		L.SetField(pkg, "path", glua.LString(cur+
			";"+filepath.Join(dir, "?.lua")+
			";"+filepath.Join(dir, "?", "init.lua")))
	}

	api := L.NewTable()
	L.SetField(api, "version", glua.LString("0.2.0"))
	L.SetField(api, "set_theme", L.NewFunction(e.luaSetTheme))
	L.SetField(api, "register_keymap", L.NewFunction(e.luaRegisterKeymap))
	L.SetField(api, "on_song_change", L.NewFunction(e.luaOnSongChange))
	L.SetField(api, "on_state_change", L.NewFunction(e.luaOnStateChange))
	L.SetField(api, "notify", L.NewFunction(e.luaNotify))
	L.SetField(api, "config_dir", L.NewFunction(e.luaConfigDir))
	L.SetField(api, "music_dir", L.NewFunction(e.luaMusicDir))
	L.SetField(api, "get_theme", L.NewFunction(e.luaGetTheme))
	L.SetGlobal("pmusic", api)
}

// pmusic.set_theme({ accent="#88C0D0", dim="#4C566A", ... })
// Fields: accent, dim, selected_bg, now_playing, border, border_active,
//         title, status_bg, panel_bg, key
func (e *Engine) luaSetTheme(L *glua.LState) int {
	t := L.CheckTable(1)
	set := func(field string, dst *string) {
		if v := L.GetField(t, field); v != glua.LNil {
			*dst = v.String()
		}
	}
	set("accent", &e.theme.Accent)
	set("dim", &e.theme.Dim)
	set("selected_bg", &e.theme.SelectedBg)
	set("now_playing", &e.theme.NowPlaying)
	set("border", &e.theme.Border)
	set("border_active", &e.theme.BorderActive)
	set("title", &e.theme.Title)
	set("status_bg", &e.theme.StatusBg)
	set("panel_bg", &e.theme.PanelBg)
	set("key", &e.theme.Key)
	return 0
}

// pmusic.get_theme() → table  -- returns current theme as a Lua table
func (e *Engine) luaGetTheme(L *glua.LState) int {
	t := L.NewTable()
	L.SetField(t, "accent", glua.LString(e.theme.Accent))
	L.SetField(t, "dim", glua.LString(e.theme.Dim))
	L.SetField(t, "selected_bg", glua.LString(e.theme.SelectedBg))
	L.SetField(t, "now_playing", glua.LString(e.theme.NowPlaying))
	L.SetField(t, "border", glua.LString(e.theme.Border))
	L.SetField(t, "border_active", glua.LString(e.theme.BorderActive))
	L.SetField(t, "title", glua.LString(e.theme.Title))
	L.SetField(t, "status_bg", glua.LString(e.theme.StatusBg))
	L.SetField(t, "panel_bg", glua.LString(e.theme.PanelBg))
	L.SetField(t, "key", glua.LString(e.theme.Key))
	L.Push(t)
	return 1
}

// validActions is the whitelist for register_keymap. Unknown names are rejected
// with a pmusic.notify() message so the user can catch typos immediately.
var validActions = map[string]bool{
	"toggle_pause":  true,
	"next":          true,
	"prev":          true,
	"loop":          true,
	"focus_folders": true,
	"focus_tracks":  true,
	"reload_lua":    true,
	"quit":          true,
	"vol_up":        true,
	"vol_down":      true,
	"seek_back5":    true,
	"seek_fwd5":     true,
	"seek_back30":   true,
	"seek_fwd30":    true,
}

// pmusic.register_keymap("f", "next")          -- bind to built-in action
// pmusic.register_keymap("Y", function() end)  -- bind to Lua function
func (e *Engine) luaRegisterKeymap(L *glua.LState) int {
	k := L.CheckString(1)
	switch L.Get(2).Type() {
	case glua.LTString:
		action := L.CheckString(2)
		if !validActions[action] {
			e.pendingNotify = "unknown keymap action: " + action
			return 0
		}
		e.keymaps[k] = action
	case glua.LTFunction:
		e.keyFuncs[k] = L.CheckFunction(2)
	default:
		e.pendingNotify = "register_keymap: arg2 must be a string action or a function"
	}
	return 0
}

// pmusic.on_song_change(function(track) ... end)
// track = { name: string, path: string, folder: string }
func (e *Engine) luaOnSongChange(L *glua.LState) int {
	e.onSongChange = L.CheckFunction(1)
	return 0
}

// pmusic.on_state_change(function(state) ... end)
// state = "playing" | "paused" | "stopped"
func (e *Engine) luaOnStateChange(L *glua.LState) int {
	e.onStateChange = L.CheckFunction(1)
	return 0
}

// pmusic.notify("message")  -- displays in the status bar for ~5 seconds
func (e *Engine) luaNotify(L *glua.LState) int {
	e.pendingNotify = L.CheckString(1)
	return 0
}

// pmusic.config_dir() → string  -- returns ~/.config/pmusic/lua/
func (e *Engine) luaConfigDir(L *glua.LState) int {
	p, err := configInitPath()
	if err != nil {
		L.Push(glua.LNil)
		return 1
	}
	L.Push(glua.LString(filepath.Dir(p)))
	return 1
}

// pmusic.music_dir() → string  -- returns the configured music directory
// Called while e.mu is already held (from CallKeyFunc / hook calls), so no lock.
func (e *Engine) luaMusicDir(L *glua.LState) int {
	L.Push(glua.LString(e.musicDir))
	return 1
}
