package watcher

import (
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
	if err := fw.Add(root); err != nil {
		fw.Close()
		return nil, err
	}

	wt := &Watcher{w: fw}
	go wt.loop()
	return wt, nil
}

func (wt *Watcher) loop() {
	for {
		select {
		case _, ok := <-wt.w.Events:
			if !ok {
				return
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
