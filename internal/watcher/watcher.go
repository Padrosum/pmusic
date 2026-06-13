package watcher

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	w       *fsnotify.Watcher
	changed atomic.Bool
}

func New(root string, _ func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// fsnotify is not recursive, so register every directory in the tree.
	// The root must be watchable; failures on individual subdirs are tolerated.
	if err := fw.Add(root); err != nil {
		fw.Close()
		return nil, err
	}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == root {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		_ = fw.Add(path)
		return nil
	})

	wt := &Watcher{w: fw}
	go wt.loop()
	return wt, nil
}

func (wt *Watcher) loop() {
	for {
		select {
		case ev, ok := <-wt.w.Events:
			if !ok {
				return
			}
			// A newly created subdirectory must be watched too, otherwise
			// tracks added inside it later go unnoticed.
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() &&
					!strings.HasPrefix(filepath.Base(ev.Name), ".") {
					_ = wt.w.Add(ev.Name)
				}
			}
			wt.changed.Store(true)
		case _, ok := <-wt.w.Errors:
			if !ok {
				return
			}
		}
	}
}

// Changed returns true (and resets the flag) if any event was seen.
func (wt *Watcher) Changed() bool {
	return wt.changed.Swap(false)
}

func (wt *Watcher) Close() {
	wt.w.Close()
}
