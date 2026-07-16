package watcher

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	w        *fsnotify.Watcher
	changed  atomic.Bool
	errMu    sync.Mutex
	err      error
	close    sync.Once
	done     chan struct{}
	closeErr error
}

func New(root string, _ func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	wt := &Watcher{w: fw, done: make(chan struct{})}
	if err := wt.addTree(root); err != nil {
		return nil, errors.Join(err, fw.Close())
	}
	go wt.loop()
	return wt, nil
}

func (wt *Watcher) loop() {
	defer close(wt.done)
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
					if err := wt.addTree(ev.Name); err != nil {
						wt.report(err)
					}
				}
			}
			wt.changed.Store(true)
		case err, ok := <-wt.w.Errors:
			if !ok {
				return
			}
			wt.report(err)
		}
	}
}

func (wt *Watcher) addTree(root string) error {
	var addErrors []error
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			addErrors = append(addErrors, fmt.Errorf("walk %s: %w", path, walkErr))
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if err := wt.w.Add(path); err != nil {
			addErrors = append(addErrors, fmt.Errorf("watch %s: %w", path, err))
		}
		return nil
	})
	if err != nil {
		addErrors = append(addErrors, err)
	}
	return errors.Join(addErrors...)
}

func (wt *Watcher) report(err error) {
	if err == nil {
		return
	}
	wt.errMu.Lock()
	wt.err = errors.Join(wt.err, err)
	wt.errMu.Unlock()
}

// TakeError returns accumulated asynchronous watcher errors exactly once.
func (wt *Watcher) TakeError() error {
	wt.errMu.Lock()
	defer wt.errMu.Unlock()
	err := wt.err
	wt.err = nil
	return err
}

// Changed returns true (and resets the flag) if any event was seen.
func (wt *Watcher) Changed() bool {
	return wt.changed.Swap(false)
}

func (wt *Watcher) Close() error {
	wt.close.Do(func() {
		wt.closeErr = wt.w.Close()
		<-wt.done
	})
	return wt.closeErr
}
