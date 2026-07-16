package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestWatcherReportsErrors(t *testing.T) {
	wt := &Watcher{}
	wt.report(errors.New("inotify queue overflow"))
	if err := wt.TakeError(); err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("error = %v", err)
	}
	if err := wt.TakeError(); err != nil {
		t.Fatalf("error was not consumed: %v", err)
	}
}

func TestWatcherAddsNewSubtreeRecursively(t *testing.T) {
	root := t.TempDir()
	wt, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer wt.Close()
	subtree := filepath.Join(root, "new", "album", "disc")
	if err := os.MkdirAll(subtree, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := wt.addTree(filepath.Join(root, "new")); err != nil {
		t.Fatal(err)
	}
	watches := wt.w.WatchList()
	for _, path := range []string{filepath.Join(root, "new"), filepath.Join(root, "new", "album"), subtree} {
		if !slices.Contains(watches, path) {
			t.Fatalf("watch %q missing from %#v", path, watches)
		}
	}
}
