# CLAUDE.md

Guidance for working in this repository.

## Overview

**pmusic** is a terminal (TUI) music player written in Go, built on
[Bubble Tea](https://github.com/charmbracelet/bubbletea) (UI) and
[beep](https://github.com/faiface/beep) (audio). It scans a local music
library, plays `.mp3` / `.flac` / `.wav`, reads embedded tags, and adds:

- A **Lua plugin & theme engine** (gopher-lua) with hot-reload.
- A built-in **plugin "store"** that downloads plugins/themes from GitHub.
- **yt-dlp** integration to search YouTube, preview results, and download a selected URL as local audio.
- A **play queue** (session-only).
- A **Blackjack** mini-game.

All user-facing text is in **English**.

## Commands

```sh
go build ./...          # build all packages
go vet ./...            # static checks
go run . [<music-dir>]  # run (no arg → use saved config or first-run setup)
go run . sync           # or -s / --sync: download all store plugins+themes
./pmusic                # run the prebuilt binary (note: it is committed but may be stale)
```

There are **no tests**; a successful build/vet is the bar. After changing
dependencies, run `go mod tidy`.

## Architecture

Entry point `main.go` routes args (`sync` vs a directory), loads config, and
runs first-run setup if no music dir is saved. Everything else lives under
`internal/`:

| Package | Responsibility |
|---|---|
| `internal/ui` | Bubble Tea model & view. `model.go` (state + Update/View), `keys.go` (key bindings), `setup.go` (first-run dir prompt), `styles.go` (styles regenerated from the active theme). |
| `internal/player` | beep-based playback. Fixed 44100 output rate, `volumeStreamer`, `State` (Stopped/Playing/Paused), seek/volume/progress. |
| `internal/fs` | Recursive library scanner with symlink-cycle protection; `FlatFolders` returns the playable folder list. |
| `internal/meta` | Reads embedded tags via `dhowden/tag`. |
| `internal/watcher` | fsnotify wrapper; watches the root **and all subdirectories**, exposes a `Changed()` flag polled on tick. |
| `internal/lua` | gopher-lua VM lifecycle (`engine.go`) and the `pmusic` Lua API (`api.go`). |
| `internal/config` | `config.json` (`music_dir`) and `enabled.json` (enabled plugin/theme names). |
| `internal/store` | Hardcoded catalog of plugins/themes; `Sync` downloads them from the GitHub raw repo. |
| `internal/blackjack` | Self-contained game state (`game.go`) and card rendering (`render.go`). |

## Key patterns (follow these when extending)

- **Overlays** generally use a `show<X> bool` field plus a `handle<X>(msg)` / `render<X>()`
  pair, intercepted at the top of `Update`'s `KeyMsg` branch and short-circuited
  in `View()`. Existing overlays: store, help, blackjack, queue, and music search.
  Music search additionally has an explicit input/loading/results/downloading/
  success/error state machine in `music_search.go`.
- **Starting playback** always follows: `m.player.MarkPending()` (sets state to
  Playing so the tick loop doesn't auto-advance during the goroutine window),
  then return a `tea.Cmd` that calls `m.player.Play(path)` and returns
  `tickMsg`. On failure `Play` calls `markStopped()` so auto-advance skips a
  dead track instead of freezing.
- **Auto-advance priority** (in the `tickMsg` handler when a track ends):
  `loop` → `queue` → sequential `playNext`.
- **Notifications**: `m.notify(msg)` shows text in the status bar for ~5s.
- **Themes**: `applyTheme(theme)` regenerates all package-level `style*` vars;
  called on startup and after every Lua reload.

## Lua scripting

User config lives at `~/.config/pmusic/lua/` (`init.lua`, `plugins/`, `themes/`).
The repo's `lua/` directory holds the source/sample versions that the store
downloads. `pmusic` API (see `internal/lua/api.go`): `set_theme`, `get_theme`,
`register_keymap` (string action or function), `on_song_change`,
`on_state_change` (both support **multiple chained** callbacks), `notify`,
`config_dir`, `music_dir`, `current_track`. Press **Ctrl+R** in-app to hot-reload.

## Key bindings (defaults, see `internal/ui/keys.go`)

`j/k` move · `h/l` folders/tracks · `Enter` play · `Space` pause · `n/p`
next/prev · `r` loop · `+/-` volume · `[ ]` ±5s · `{ }` ±30s · `a` queue add ·
`u` queue overlay · `Y` music search/download · `g` store · `b` blackjack · `?` help ·
`Ctrl+R` reload Lua · `q` quit. Lua keymaps are checked before built-ins
(except the Download/`Y` panel).

## Gotchas

- The compiled `pmusic` binary is committed and can be stale — rebuild rather
  than trusting it.
- Audio needs a working output device (oto/ALSA on Linux).
- `yt-dlp` is an optional external dependency, looked up on `PATH` at use time.
