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
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━────────────────────────────────────────
  j/k:move  h/l:panel  enter:play  spc:pause  n/p:next/prev  r:loop  q:quit
```

## Features

- **Two-panel interface** — folders on the left, tracks on the right
- Supports **MP3, FLAC, and WAV** formats
- **Progress bar** with elapsed time / total duration display
- **Loop mode** — repeat the current track
- **Automatic track switching** — plays the next track when the current one ends
- **Live directory watching** — automatically refreshes when new files are added to the music folder
- **Persistent configuration** — selected music directory is saved and reused automatically

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

On first startup, a setup screen appears asking for your music folder path. This setting is saved to `~/.config/pmusic/config.json` and won’t be asked again.

## Keyboard Shortcuts

| Key | Action |
|------|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Switch to folders panel |
| `l` / `→` | Switch to tracks panel |
| `Enter` | Play selected track |
| `Space` | Pause / Resume |
| `n` | Next track |
| `p` | Previous track |
| `r` | Toggle loop mode |
| `q` / `Ctrl+C` | Quit |

## Requirements

- Go 1.21+
- System audio driver for sound output (ALSA / CoreAudio / DirectSound)

## Why pmusic?

pmusic is designed for people who want to listen to music without leaving the terminal. It’s lightweight, requires no graphical interface, and uses Vim-like keyboard shortcuts for fast navigation. No metadata database or external services are needed — just a music directory.
