package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Padrosum/pmusic/internal/catalog"
	"github.com/Padrosum/pmusic/internal/config"
	"github.com/Padrosum/pmusic/internal/store"
	"github.com/Padrosum/pmusic/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-s", "--sync", "sync":
			runSync()
			return
		case "-c", "--catalog", "catalog":
			runCatalog(os.Args[2:])
			return
		case "-v", "--version", "version":
			fmt.Printf("pmusic %s (commit %s)\n", version, commit)
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
	_, runErr := p.Run()
	closeErr := m.Close()
	if runErr != nil || closeErr != nil {
		fatalf("%v", errors.Join(runErr, closeErr))
	}
}

func runSync() {
	base, err := os.UserConfigDir()
	if err != nil {
		fatalf("config dir: %v", err)
	}
	fmt.Println("Syncing pmusic plugins, themes, and GenLang catalogs...")
	if err := store.Sync(filepath.Join(base, "pmusic")); err != nil {
		fatalf("sync: %v", err)
	}
	fmt.Println("Done. Open pmusic and press g to manage.")
}

func runCatalog(args []string) {
	opt := catalog.WriteOptions{}
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help":
			fmt.Print(`Write a GenLang catalog from your local music files.

  pmusic catalog
  pmusic catalog ~/Music
  pmusic catalog --force

Writes ~/.config/pmusic/library.gl. The bundled sample and previously
generated catalogs are replaced. A hand-edited file needs --force.
Requires genlang on PATH only when you next open pmusic.
`)
			return
		case "-f", "--force":
			opt.Force = true
		default:
			if strings.HasPrefix(arg, "-") {
				fatalf("unknown flag %s (try pmusic catalog --help)", arg)
			}
			if opt.MusicDir != "" {
				fatalf("unexpected extra argument %q", arg)
			}
			opt.MusicDir = arg
		}
	}
	if opt.MusicDir == "" {
		cfg, err := config.Load()
		if err != nil {
			fatalf("config: %v", err)
		}
		opt.MusicDir = cfg.MusicDir
	}
	if opt.MusicDir == "" {
		fatalf("no music directory; run pmusic once or pass a path")
	}
	fmt.Printf("Scanning %s …\n", opt.MusicDir)
	stats, err := catalog.WriteFromScan(opt)
	if err != nil {
		fatalf("catalog: %v", err)
	}
	fmt.Printf("Wrote %d tracks, %d albums to %s\n", stats.Tracks, stats.Albums, stats.Dest)
	fmt.Println("Open pmusic (genlang on PATH) or run :reload library.")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "pmusic: "+format+"\n", args...)
	os.Exit(1)
}
