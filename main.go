package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/padros/pmusic/internal/config"
	"github.com/padros/pmusic/internal/ui"
)

func main() {
	// CLI argument overrides config (useful for one-off sessions).
	if len(os.Args) > 1 {
		runPlayer(os.Args[1])
		return
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

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "pmusic: "+format+"\n", args...)
	os.Exit(1)
}
