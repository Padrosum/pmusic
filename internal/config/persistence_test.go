package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueueCorruptionDoesNotSilentlyBecomeEmpty(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	path := filepath.Join(base, "pmusic", "queue.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	bad := []byte(`[{"name":"partially decoded"}`)
	if err := os.WriteFile(path, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	queue, err := LoadQueue()
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("queue=%#v err=%v", queue, err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil || string(got) != string(bad) {
		t.Fatalf("corrupt source was changed: %q err=%v", got, readErr)
	}
}

func TestPrivatePersistenceUsesOwnerOnlyPermissions(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	if err := Save(&Config{MusicDir: "/music"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, "pmusic", "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o", got)
	}
}
