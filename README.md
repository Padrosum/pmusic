# pmusic

A terminal-based (TUI) local music player written in Go.

```text
┌── Folders ──────────────┬── Jazz ────────────────────────────────────┐
│  Classic Rock           │    1.  ▶ Kind of Blue - Miles Davis        │
│  Electronic             │    2.    So What                           │
│> Jazz                   │    3.    Freddie Freeloader                │
│  Lo-fi                  │    4.    Blue in Green                     │
└─────────────────────────┴────────────────────────────────────────────┘
  ▶ Kind of Blue - Miles Davis ↺                           2:14 / 9:22
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━────────────────────────────────────────
  j/k:move  h/l:panel  enter:play  spc:pause  n/p:next/prev  r:loop  ^r:lua  q:quit
```

## Features

- **Two-panel interface** — folders on the left, tracks on the right
- Supports **MP3, FLAC, and WAV** formats
- **Progress bar** with elapsed time / total duration display
- **Loop mode** — repeat the current track
- **Automatic track switching** — plays the next track when the current one ends
- **Live directory watching** — automatically refreshes when new files are added
- **Persistent configuration** — selected music directory is saved automatically
- **Lua scripting** — theme, keybindings, and hooks configurable without recompiling

## Installation

### With ppd (recommended)

```sh
ppd install pmusic
```

> ppd: https://github.com/Padrosum/ppd

### With Go

```sh
go install github.com/Padrosum/pmusic@latest
```

### Build from source

```sh
git clone https://github.com/Padrosum/pmusic
cd pmusic
go build -o pmusic .
```

## Usage

```sh
# On first launch, pmusic asks for your music directory and saves it
pmusic

# Specify a directory directly
pmusic ~/Music
```

On first startup a setup screen appears asking for your music folder path. This is saved to `~/.config/pmusic/config.json` and won't be asked again.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Switch to folders panel |
| `l` / `→` | Switch to tracks panel |
| `Enter` | Play selected track |
| `Space` | Pause / Resume |
| `n` | Next track |
| `p` | Previous track |
| `r` | Toggle loop mode |
| `Ctrl+R` | Reload Lua config (hot-reload) |
| `q` / `Ctrl+C` | Quit |

## Lua Scripting

pmusic is extensible via Lua scripts placed in `~/.config/pmusic/lua/`.

> For the full API reference, theme color map, plugin authoring guide, and execution model — see **[lua/info.md](lua/info.md)** (available in English and Turkish).

### Quick start

```sh
mkdir -p ~/.config/pmusic/lua/themes ~/.config/pmusic/lua/plugins

# copy the example config
cp lua/init.lua ~/.config/pmusic/lua/init.lua

# optionally copy a theme
cp lua/themes/gruvbox.lua ~/.config/pmusic/lua/themes/
```

Edit `~/.config/pmusic/lua/init.lua`, then press `Ctrl+R` inside pmusic to apply changes without restarting.

### Directory layout

```
~/.config/pmusic/lua/
├── init.lua        ← entry point (loaded on startup and on Ctrl+R)
├── themes/
│   ├── gruvbox.lua
│   ├── catppuccin.lua
│   └── tokyo-night.lua
└── plugins/
    ├── logger.lua
    └── keymaps.lua
```

### API reference

| Function | Description |
|----------|-------------|
| `pmusic.set_theme(t)` | Override UI colors — any subset of keys is accepted |
| `pmusic.get_theme()` | Return the current theme as a Lua table |
| `pmusic.register_keymap(key, action)` | Bind a key string to a built-in action |
| `pmusic.on_song_change(fn)` | Hook called with `{name, path, folder}` when a track starts |
| `pmusic.on_state_change(fn)` | Hook called with `"playing"`, `"paused"`, or `"stopped"` |
| `pmusic.notify(msg)` | Show a 5-second message in the status bar |
| `pmusic.config_dir()` | Returns the path to `~/.config/pmusic/lua/` |
| `pmusic.version` | Current API version string |

**Actions for `register_keymap`:** `toggle_pause` · `next` · `prev` · `loop` · `focus_folders` · `focus_tracks` · `reload_lua` · `quit`

### Themes

Three themes are included in `lua/themes/`. Activate one by adding a single line to `init.lua`:

```lua
require("themes/gruvbox")     -- Gruvbox Dark
require("themes/catppuccin")  -- Catppuccin Mocha
require("themes/tokyo-night") -- Tokyo Night Storm
```

Or define colors inline — any subset of the keys below can be overridden:

```lua
pmusic.set_theme({
    accent        = "#89b4fa",  -- active border, progress bar, title
    dim           = "#45475a",  -- inactive text, empty progress track
    selected_bg   = "#313244",  -- cursor row background
    now_playing   = "#a6e3a1",  -- currently playing track name
    border        = "#45475a",  -- inactive panel border
    border_active = "#89b4fa",  -- active panel border
    title         = "#cba6f7",  -- panel title text
    status_bg     = "#181825",  -- status bar background
    panel_bg      = "#1e1e2e",  -- canvas / panel background
    key           = "#fab387",  -- key-hint label color
})
```

### Plugins

**logger** (`lua/plugins/logger.lua`) — appends every played track with a timestamp to `~/.local/share/pmusic/plays.log`:

```lua
require("plugins/logger")
```

**stats** (`lua/plugins/stats.lua`) — tracks how many times each song was played in the current session and shows a notification on repeat plays:

```lua
require("plugins/stats")
```

**keymaps** (`lua/plugins/keymaps.lua`) — a commented preset of extra bindings (MPD-style, media keys, etc.):

```lua
require("plugins/keymaps")
```

### Example: notify on every song change

```lua
pmusic.on_song_change(function(track)
    pmusic.notify("▶  " .. track.folder .. " / " .. track.name)
end)
```

### Example: custom key binding

```lua
pmusic.register_keymap("f", "next")   -- f → next track
pmusic.register_keymap("b", "prev")   -- b → previous track
```

## Requirements

- Go 1.21+
- System audio driver (ALSA on Linux, CoreAudio on macOS, DirectSound on Windows)

## Why pmusic?

pmusic is designed for people who want to listen to music without leaving the terminal. It's lightweight, requires no graphical interface, and uses Vim-like keyboard shortcuts for fast navigation. No metadata database or external services required — just a music directory.
