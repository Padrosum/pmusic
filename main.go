package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/padros/pmusic/internal/config"
	"github.com/padros/pmusic/internal/store"
	"github.com/padros/pmusic/internal/ui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-s", "--sync", "sync":
			runSync()
			return
		default:
			runPlayer(os.Args[1])
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fatalf("config: %v", err)
	}

	if cfg.MusicDir == "" {
		dir := runSetup()
		if dir == "" {
			return // user pressed esc
		}
		cfg.MusicDir = dir
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save config: %v\n", err)
		}
	}

	runPlayer(cfg.MusicDir)
}

func runSetup() string {
	p := tea.NewProgram(ui.NewSetup(), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fatalf("setup: %v", err)
	}
	if sm, ok := finalModel.(ui.SetupModel); ok {
		return sm.Result
	}
	return ""
}

func runPlayer(dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		fatalf("%q is not a valid directory", dir)
	}

	m, err := ui.New(dir)
	if err != nil {
		fatalf("%v", err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fatalf("%v", err)
	}
}

func runSync() {
	base, err := os.UserConfigDir()
	if err != nil {
		fatalf("config dir: %v", err)
	}
	luaDir := filepath.Join(base, "pmusic", "lua")
	fmt.Println("Syncing pmusic plugins and themes...")
	if err := store.Sync(luaDir); err != nil {
		fatalf("sync: %v", err)
	}
	fmt.Println("Done. Open pmusic and press g to manage.")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "pmusic: "+format+"\n", args...)
	os.Exit(1)
}
