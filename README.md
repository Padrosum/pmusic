
# pmusic


A fast, keyboard-driven music player for the terminal.

▶ Search  
▶ Queue  
▶ Playlists  
▶ YouTube support  
▶ Downloads  
▶ Themes  
▶ Plugins  
▶ Lua scripting  
▶ Statistics  

## Latest update (2026-09-11)

Optional **[GenLang](https://github.com/Padrosum/GenLANG)** catalog: playlists
and tags without moving files.

- A `library.gl` overlay sits on top of the local library. Folders stay folders;
  GenLang **types** say what a record *is*, **sets** are playlists and tags.
- pmusic loads `~/.config/pmusic/library.gl` (or `<music-dir>/library.gl`) by
  running `genlang convert … --json`, the same optional-tool pattern as
  `yt-dlp`. There is no CGO. No file and no `genlang` → the player is unchanged.
- Sets appear under folders in the left panel (`♫`). `j`/`k` move through both;
  `a` queues a list. `:playlist`, `:type`, and `:inspect` (`:dyaz`) join command
  mode. `:reload library` rescans files and reloads the overlay.
- `pmusic catalog` writes `library.gl` from your local files (`path`, tags, albums).
  The bundled sample is replaced; a hand-edited catalog needs `--force`.
- Playlists need the **`genlang` CLI** on `PATH` (see
  [Optional GenLang catalog](#optional-genlang-catalog)). `pmusic -s` only
  downloads the sample `.gl` file; it does not install GenLang.
- `pmusic -s` also downloads the sample catalog to `~/.config/pmusic/gl/library.gl`
  and seeds `library.gl` when that file is missing (existing catalogs are left
  alone).
- Album set membership expands to tracks that reference that album. Unmatched
  `path` values are reported instead of silently dropped.

Sample file: [`examples/library.gl`](examples/library.gl). Full usage:
[Playlists and tags (GenLang)](#playlists-and-tags-genlang).

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
- **GenLang catalog** — optional `library.gl` overlay for playlists, tags, and type browse without moving files
- **Listening statistics** — inspect listening time, starts, completions, skips, artists, and top tracks
- **Vim-style command mode** — searchable help, completion, aliases, suggestions, and persistent history
- **Lua scripting** — theme, keybindings, and event hooks configurable without recompiling
- **Cover art** — view the current track's album art in the terminal (`c` or `:art`)
- **Blackjack mini-game** — play from inside the TUI while your music continues

## Installation

### Quick install (Linux x86-64)

The currently published binary is the automatically updated `edge` build for
**64-bit x86 Linux**. Check that `uname -m` prints `x86_64`, then run:

```sh
curl -fL https://github.com/Padrosum/pmusic/releases/download/edge/pmusic-linux-amd64 \
  -o pmusic-linux-amd64
sudo install -m 0755 pmusic-linux-amd64 /usr/local/bin/pmusic
pmusic --version
```

`edge` is rebuilt after every successful commit to `main`, so it may contain
new or unfinished changes. The `-f` flag makes `curl` fail instead of saving a
GitHub error page as an executable.

To update later, run the same three commands again.

### With ppd

If [ppd](https://github.com/Padrosum/ppd) is already installed:

```sh
ppd install pmusic
```

ppd installs the repository-root binary into `/usr/local/bin`. Use `ppd update`
to update packages managed by ppd.

### With Go

This method works on other operating systems and architectures supported by
pmusic. It requires the Go version declared in `go.mod`; Linux source builds
also require ALSA development headers. On Debian/Ubuntu:

```sh
sudo apt-get update
sudo apt-get install -y libasound2-dev pkg-config
go install github.com/Padrosum/pmusic@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`, then verify the installation
with `pmusic --version`.

### Build from source

On Debian/Ubuntu, install the build requirements first:

```sh
sudo apt-get update
sudo apt-get install -y git libasound2-dev pkg-config
```

Install the Go version declared in `go.mod` or newer, then clone, build, and
install pmusic:

```sh
git clone https://github.com/Padrosum/pmusic.git
cd pmusic
make release
sudo install -m 0755 dist/pmusic /usr/local/bin/pmusic
pmusic --version
```

`make release` creates a stripped binary at `dist/pmusic` and a ppd-compatible
copy at the repository root.

### Optional download support

Local playback has no external command dependency. Online search and downloads
additionally require `yt-dlp` and FFmpeg to be available in `PATH`:

```sh
yt-dlp --version
ffmpeg -version
```

### Optional cover art support

Album art is rendered with [chafa](https://hpjansson.org/chafa/), so cover art
requires `chafa` in `PATH` (only for the `c` / `:art` feature — playback works
without it):

```sh
chafa --version
```

Arch Linux: `sudo pacman -S chafa` · Debian/Ubuntu: `sudo apt-get install chafa`

### Optional GenLang catalog

Playlists and tags live in a `library.gl` overlay. Local playback does not
need it. When the file exists, pmusic runs `genlang convert … --json`.
That **requires the [GenLang](https://github.com/Padrosum/GenLANG) CLI** on
`PATH`. Build your own overlay with `pmusic catalog`. `pmusic -s` only
downloads the sample `.gl` catalog; it does not install `genlang`.

```sh
git clone https://github.com/Padrosum/GenLANG.git
cd GenLANG
cmake -S . -B build
cmake --build build
sudo cmake --install build
genlang --version
```

Needs CMake 3.16+ and a C17 compiler. Without `genlang`, pmusic keeps playing
local files and shows a status-bar notice if `library.gl` is present.

## Usage

```sh
# On first launch, pmusic asks for your music directory and saves it
pmusic

# Specify a directory directly
pmusic ~/Music

# Download bundled plugins, themes, and the sample GenLang catalog
pmusic -s

# Write ~/.config/pmusic/library.gl from your local music files
pmusic catalog

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
| `g` | Open plugin / theme / catalog store |
| `b` | Open or close the Blackjack mini-game |
| `c` | Open or close the cover art overlay |
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
- `:playlist Gece`
- `:playlist queue Favoriler`
- `:type Parca`
- `:inspect`
- `:online Metallica`
- `:download Duman Seni Kendime Sakladım`
- `:art`
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

## Playlists and tags (GenLang)

Folders stay folders. An optional [GenLang](https://github.com/Padrosum/GenLANG)
`library.gl` file adds playlists and tags on top of the same files, without
moving anything.

pmusic looks for the overlay in this order:

1. `~/.config/pmusic/library.gl`
2. `<music-dir>/library.gl`

If neither file exists, the player is unchanged. If a file exists, the
`genlang` CLI must be on `PATH`
([install it](#optional-genlang-catalog)); otherwise a status-bar notice
explains that and local playback continues. `pmusic -s` does not install
GenLang.

The usual way to fill it is to scan your music directory:

```sh
pmusic catalog
pmusic catalog --force   # replace a hand-edited library.gl
```

That writes `~/.config/pmusic/library.gl` with one `Parca` per file, `path`
set to the real relative path, and `Album` rows from tags. The bundled sample
and a previously generated file are replaced; anything else needs `--force`.

`pmusic -s` only downloads the commented sample into `gl/library.gl` (and
seeds `library.gl` when it is missing). You can also copy the sample yourself:

```sh
mkdir -p ~/.config/pmusic
cp examples/library.gl ~/.config/pmusic/library.gl
```

In the file, **types** (`cins` / `tur`) say what a record *is* (`Parca`,
`Album`). **Sets** (`kume` + `uye`) are playlists and tags. Being a `Parca`
does not put a track on a playlist.

```gl
kume Favoriler
kume Gece

veri kind_of_blue : Parca {
    baslik = "Kind of Blue"
    path = "Jazz/Kind of Blue.flac"
}

uye kind_of_blue -> Favoriler
uye kind_of_blue -> Gece
```

`path` may be absolute or relative to the music directory. Do not use `yol`
for the file; it is a GenLang keyword and the catalog will not parse. If
`path` is omitted, pmusic matches a unique local filename/title. A set member
that is an album expands to the tracks that reference it.

Loaded sets appear at the bottom of the left panel (prefixed with `♫`).
`j` / `k` move through folders and lists together. `a` on a list queues every
resolved member. `:reload library` rescans files and reloads `library.gl`.

| Command | Effect |
| --- | --- |
| `:playlist` | List every set |
| `:playlist Gece` | Open that set in the tracks panel |
| `:playlist queue Favoriler` | Append its tracks to the queue |
| `:type Parca` | Show tracks whose type is `Parca` or a descendant |
| `:inspect` | Print the selected track's types and sets as two lists |

`:tag` and `:lists` are aliases for `:playlist`. `:tur` aliases `:type`.
`:dyaz` aliases `:inspect`.

## Cover Art

When a track starts playing, pmusic looks up its album art in the background
and shows a **small thumbnail in the bottom bar beside the now-playing info** —
no action needed. Press `c` (or run `:art`) to view the same art in a larger
centered overlay. The art is searched in this order:

1. **Embedded tags** — the picture stored inside the audio file (ID3 APIC /
   Vorbis PICTURE)
2. **Folder files** — `cover.jpg`, `folder.jpg`, `front.jpg`, `albumart.jpg`,
   or `album.jpg` next to the track
3. **Online fallback** — if neither exists and the file has artist/album tags,
   the cover is looked up via the iTunes Search API and cached under
   `~/.cache/pmusic/covers/`, so it is only fetched once per album

Rendering requires [chafa](https://hpjansson.org/chafa/) in `PATH`; without it
the overlay explains how to install it. Rendered art is cached in memory per
track, so re-opening the overlay is instant. Track changes refresh both the
bottom-bar thumbnail and the overlay automatically.

## Plugin Store

pmusic has a built-in plugin manager. Run `pmusic -s` once to download all bundled plugins, themes, and the sample GenLang catalog, then press `g` inside pmusic to enable or disable Lua items without editing any files.

Sync downloads are pinned to an immutable repository commit, size-limited, and
SHA-256 verified before atomically replacing an installed file. Lua extensions
are trusted code rather than a sandbox; review [the security model](docs/security.md)
before enabling third-party code. Catalog `.gl` files are data, not code.

```sh
pmusic -s        # download plugins, themes, and GenLang catalogs
```

Lua files go to `~/.config/pmusic/lua/`. Catalog packages go to
`~/.config/pmusic/gl/`. If `~/.config/pmusic/library.gl` does not exist, sync
copies the sample catalog there so playlists can load immediately. An existing
`library.gl` is never overwritten.

Inside pmusic press `g` to open the store overlay:

```
╭── Plugin Store ──────────────────────────────────╮
│                                                  │
│  [Plugins]  Themes  Catalogs   pmusic -s ile indir│
│                                                  │
│  ✓  logger               Log played tracks...    │
│  ✓  listen-time          Session listening...    │
│  ○  stats                Session play-count...   │
│  ✗  notify-send          [kurulu değil]          │
│  ...                                             │
│                                                  │
│  Space:toggle/info  h/l:sekme  g/q:kapat         │
╰──────────────────────────────────────────────────╯
```

| Icon | Meaning |
|------|---------|
| `✓` | Lua: installed and **enabled**. Catalog: package file is present |
| `○` | Lua: installed but disabled |
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
- `chafa` only for cover art rendering
- `genlang` CLI ([GenLang](https://github.com/Padrosum/GenLANG)) only for the
  optional `library.gl` playlist/tag overlay; install separately, not via
  `pmusic -s`

## Why pmusic?

pmusic is designed for people who want to listen to music without leaving the terminal. It's lightweight, requires no graphical interface, and uses Vim-like keyboard shortcuts for fast navigation. No metadata database or external services required — just a music directory.
