# pmusic

A terminal-based (TUI) local music player written in Go.

```text
 ♪ PMUSIC   LIBRARY  4 folders · 38 tracks                 ▂▅█▃ LIVE
┏━━━━━━━━━━━━━━━━━━━━━━━━┓┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  COLLECTIONS  04      ┃┃  TRACKS  Jazz                /\_/\  ┃
┃  Classic Rock         ┃┃   1. ▶ Kind of Blue          (^.^)  ┃
┃  Electronic           ┃┃   2.   So What                >♪ <  ┃
┃› Jazz                 ┃┃   3.   Freddie Freeloader           ┃
┃  Lo-fi                ┃┃   4.   Blue in Green                ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━┛┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
  ▂▆█▃▇▅▁  ▶ Miles Davis — Kind of Blue ↺              2:14 / 9:22
  Kind of Blue
  ━━━━━━━━━━━━━━━━━━━━━●──────────────────────────────────────────
  j/k:move  h/l:panel  enter:play  spc:pause  n/p:skip  ?:help  q:quit
```

## Features

- **Two-panel interface** — folders on the left, tracks on the right
- Supports **MP3, FLAC, and WAV** formats
- **ID3 / Vorbis metadata** — artist, album, and title read from file tags; falls back to filename
- **Volume control** — `+` / `-` keys in 10% steps, persists across tracks
- **Song seeking** — `[` / `]` for ±5 seconds, `{` / `}` for ±30 seconds
- **Mouse support** — click to select and play, scroll wheel to navigate
- **Animated ASCII mascot** — a cat in the corner of the tracks panel that reacts to play / pause / stop state
- **Help overlay** — press `?` to show all shortcuts in a centered popup
- **Progress bar** with elapsed time / total duration display
- **Loop mode** — repeat the current track (`r`)
- **Automatic track switching** — plays the next track when the current one ends
- **Adaptive refresh rate** — smooth 4 Hz playback animation with a lower-power 1 Hz idle loop
- **Live directory watching** — automatically refreshes when files are added or removed
- **Persistent configuration** — music directory is saved to `~/.config/pmusic/config.json`
- **Persistent play queue** — queue tracks or whole folders, reorder them, and continue across restarts
- **Local library search** — find tracks without leaving the player
- **Listening statistics** — inspect listening time, starts, completions, skips, artists, and top tracks
- **Vim-style command mode** — searchable help, completion, aliases, suggestions, and persistent history
- **Lua scripting** — theme, keybindings, and event hooks configurable without recompiling
- **Blackjack mini-game** — play from inside the TUI while your music continues

## Installation

The published GitHub binary currently targets **Linux x86-64/AMD64**. Users on
another architecture or operating system should use `go install` or build from
source.

### With ppd (recommended)

```sh
ppd install pmusic
```

> ppd: https://github.com/Padrosum/ppd

### From GitHub Releases

There are two release channels:

| Channel | Purpose | Updated when |
|---------|---------|--------------|
| Stable | Versioned, permanent releases | A `v*` version tag is published |
| Edge | Latest successful `main` build; may contain unfinished changes | Every commit pushed to `main` |

Install the latest stable release:

```sh
curl -fL -o pmusic https://github.com/Padrosum/pmusic/releases/latest/download/pmusic-linux-amd64
chmod +x pmusic
sudo install pmusic /usr/local/bin/pmusic
```

Or install the continuously updated edge build:

```sh
curl -fL -o pmusic https://github.com/Padrosum/pmusic/releases/download/edge/pmusic-linux-amd64
chmod +x pmusic
sudo install pmusic /usr/local/bin/pmusic
```

`curl -fL` makes the command fail instead of saving a GitHub error page as the
`pmusic` executable.

### With Go

On Debian/Ubuntu, install the Linux audio development dependencies first:

```sh
sudo apt-get update
sudo apt-get install -y libasound2-dev pkg-config
```

```sh
go install github.com/Padrosum/pmusic@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`. This method requires the Go
version declared in `go.mod` or newer.

### Build from source

```sh
git clone https://github.com/Padrosum/pmusic
cd pmusic
go build -o pmusic .
sudo install pmusic /usr/local/bin/pmusic
```

For a smaller binary with embedded version information, debug tables, and local
source paths removed:

```sh
make release
sudo install dist/pmusic /usr/local/bin/pmusic
```

`make release` writes the ppd-compatible binary to `pmusic` and mirrors it to
`dist/pmusic`.

### Optional download support

Local playback has no external command dependency. Online search and downloads
additionally require `yt-dlp` and FFmpeg to be available in `PATH`:

```sh
yt-dlp --version
ffmpeg -version
```

If an older ppd installation fails with `syntax error` followed by
`<!DOCTYPE html>`, it installed GitHub's old 404 page rather than an executable.
Update after the repository-root binary is published, or repair it locally:

```sh
make release
sudo install dist/pmusic /usr/local/bin/pmusic
```

## Usage

```sh
# On first launch, pmusic asks for your music directory and saves it
pmusic

# Specify a directory directly
pmusic ~/Music

# Download all bundled plugins and themes to ~/.config/pmusic/lua/
pmusic -s

# Print build version and commit information
pmusic --version
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
| `[` / `]` | Seek ±5 seconds |
| `{` / `}` | Seek ±30 seconds |
| `+` / `=` | Volume up (+10%) |
| `-` | Volume down (−10%) |
| `:` | Open Vim-style command mode |
| `?` | Show / hide help overlay |
| `Y` | Open music search and download screen |
| `/` | Search the local music library |
| `a` | Add the selected track or folder to the queue |
| `u` | Open or close the play queue |
| `g` | Open plugin / theme store |
| `b` | Open or close the Blackjack mini-game |
| `Ctrl+R` | Reload Lua config (hot-reload) |
| `q` / `Ctrl+C` | Quit |

## Command Mode

Press `:` to open the Vim-style command line. Command execution, completion,
aliases, suggestions, and help all use the same central command registry.

Examples:

- `:play`
- `:pause`
- `:volume 60`
- `:seek +30`
- `:loop toggle`
- `:queue clear`
- `:search Metallica`
- `:online Metallica`
- `:download Duman Seni Kendime Sakladım`
- `:reload lua`
- `:help seek`
- `:stats week`
- `:stats artist Metallica`
- `:quit`

Use `Tab` and `Shift+Tab` for completion, `↑`/`↓` for suggestions or command
history, and `:help` for the complete searchable command reference. `Ctrl+U`
clears the line, `Ctrl+W` deletes the previous word, and `Esc` or `Ctrl+C`
returns to normal mode. Command history is kept in
`$XDG_STATE_HOME/pmusic/command-history` (or
`~/.local/state/pmusic/command-history`).

Listening activity is stored locally in
`$XDG_STATE_HOME/pmusic/listening-stats.json`. Use `:stats`, `:stats week`,
`:stats all`, or `:stats artist <name>` to inspect listening time, started
tracks, completions, skips, and top tracks. Statistics are written atomically
with user-only file permissions.

## Mouse

| Input | Action |
|-------|--------|
| Left click on track | Select and play immediately |
| Left click on folder | Select folder |
| Scroll wheel | Navigate up / down |

## Music Search and Download

Press `Y` to open the music search screen. Enter a song or artist name to search YouTube and inspect up to 10 results before downloading anything.

```
╭── ♫ Music Search ─────────────────────────────────────╮
│                                                       │
│  Search: metallica fade to black                      │
│  Source: YouTube (text search)                        │
│                                                       │
│  › Metallica - Fade to Black                          │
│      Metallica · 6:57 · YouTube                       │
│                                                       │
│  j/k:select  Enter:download  /:new search  Esc/q:close│
╰───────────────────────────────────────────────────────╯
```

- **Search text** — lists YouTube results; use `j` / `k` or the arrow keys to select one, then press `Enter` to download it
- **URL** (starts with `http://` or `https://`) — previews and downloads that URL directly through yt-dlp
- **New search** — press `/` from the result screen to edit the query again
- **Close** — press `Esc` or `q`; an active download continues safely in the background

Text search is YouTube-only in this version. Direct URLs may point to YouTube, SoundCloud, or any other source supported by yt-dlp; pMusic does not claim that those sites support text search.

Requires [yt-dlp](https://github.com/yt-dlp/yt-dlp) in `$PATH` (and its normal audio conversion dependencies). Downloads run in the background and are written to the configured local music folder. The filesystem watcher adds the resulting MP3 to the library, and pMusic plays it as a local file—it never streams the remote result.

## Plugin Store

pmusic has a built-in plugin manager. Run `pmusic -s` once to download all bundled plugins and themes, then press `g` inside pmusic to enable or disable them without editing any files.

Sync downloads are pinned to an immutable repository commit, size-limited, and
SHA-256 verified before atomically replacing an installed file. Lua extensions
are trusted code rather than a sandbox; review [the security model](docs/security.md)
before enabling third-party code.

```sh
pmusic -s        # download plugins + themes to ~/.config/pmusic/lua/
```

Inside pmusic press `g` to open the store overlay:

```
╭── Plugin Store ──────────────────────────────────╮
│                                                  │
│  [Plugins]  Themes   pmusic -s ile indir         │
│                                                  │
│  ✓  logger               Log played tracks...    │
│  ✓  listen-time          Session listening...    │
│  ○  stats                Session play-count...   │
│  ✗  notify-send          [kurulu değil]          │
│  ...                                             │
│                                                  │
│  Space:toggle  h/l:sekme  g/q:kapat              │
╰──────────────────────────────────────────────────╯
```

| Icon | Meaning |
|------|---------|
| `✓` | Installed and **enabled** |
| `○` | Installed but disabled |
| `✗` | Not installed — run `pmusic -s` first |

Enable state is saved to `~/.config/pmusic/enabled.json`. Enabled plugins and themes are loaded automatically on startup and after every `Ctrl+R` hot-reload.

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
    ├── stats.lua
    ├── keymaps.lua
    ├── listen-time.lua
    ├── notify-send.lua
    ├── statusline.lua
    └── theme-scheduler.lua
```

### API reference

| Function | Description |
|----------|-------------|
| `pmusic.set_theme(t)` | Override UI colors — any subset of keys is accepted |
| `pmusic.get_theme()` | Return the current theme as a Lua table |
| `pmusic.register_keymap(key, action_or_fn)` | Bind a key to a built-in action string **or** a Lua function — `pmusic.register_keymap("Y", function() … end)` |
| `pmusic.on_song_change(fn)` | Hook called with `{name, path, folder}` when a track starts |
| `pmusic.on_state_change(fn)` | Hook called with `"playing"`, `"paused"`, or `"stopped"` |
| `pmusic.notify(msg)` | Show a 5-second message in the status bar |
| `pmusic.config_dir()` | Returns the path to `~/.config/pmusic/lua/` |
| `pmusic.music_dir()` | Returns the configured music directory |
| `pmusic.current_track()` | Returns `{name, path, folder}` of the playing (or last-played) track |
| `pmusic.version` | Current API version string (`"0.2.0"`) |

**Actions for `register_keymap` (string form):**
`toggle_pause` · `next` · `prev` · `loop` · `focus_folders` · `focus_tracks` · `reload_lua` · `quit` · `vol_up` · `vol_down` · `seek_back5` · `seek_fwd5` · `seek_back30` · `seek_fwd30`

**Function form** — bind any Lua logic to a key:
```lua
pmusic.register_keymap("Y", function()
    pmusic.notify("custom action!")
    os.execute("some-command &")
end)
```

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

**stats** (`lua/plugins/stats.lua`) — tracks how many times each song was played in the current session and notifies on repeat plays:

```lua
require("plugins/stats")
```

**keymaps** (`lua/plugins/keymaps.lua`) — a commented preset of extra bindings (MPD-style, media keys, etc.):

```lua
require("plugins/keymaps")
```

**listen-time** (`lua/plugins/listen-time.lua`) — tracks total listening time for the session (pause-aware) and notifies on each track change:

```lua
require("plugins/listen-time")
```

**notify-send** (`lua/plugins/notify-send.lua`) — sends a desktop notification on every track change (auto-detects dunstify / notify-send / osascript):

```lua
require("plugins/notify-send")
```

**statusline** (`lua/plugins/statusline.lua`) — writes current playback state to `/tmp/pmusic-status.json` for waybar, polybar, eww, or tmux integration:

```lua
require("plugins/statusline")
```

**theme-scheduler** (`lua/plugins/theme-scheduler.lua`) — automatically switches theme based on the time of day (light in morning, dark at night):

```lua
require("plugins/theme-scheduler")
```



### Example: notify on every song change

```lua
pmusic.on_song_change(function(track)
    pmusic.notify("▶  " .. track.folder .. " / " .. track.name)
end)
```

### Example: custom key binding

```lua
pmusic.register_keymap("f", "next")      -- f → next track
pmusic.register_keymap("b", "prev")      -- b → previous track
pmusic.register_keymap("u", "vol_up")    -- u → volume up
pmusic.register_keymap("d", "vol_down")  -- d → volume down
```

## Requirements

- Go 1.26.3+ (for building from source; the minimum follows `go.mod`)
- System audio driver (ALSA on Linux, CoreAudio on macOS, DirectSound on Windows)
- ALSA development headers and `pkg-config` for Linux source builds
- `yt-dlp` and FFmpeg only for online search/download support

## Why pmusic?

pmusic is designed for people who want to listen to music without leaving the terminal. It's lightweight, requires no graphical interface, and uses Vim-like keyboard shortcuts for fast navigation. No metadata database or external services required — just a music directory.
