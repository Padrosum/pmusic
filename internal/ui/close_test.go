package ui

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Padrosum/pmusic/internal/listening"
	"github.com/Padrosum/pmusic/internal/player"
)

func TestModelCloseIsIdempotent(t *testing.T) {
	m := commandTestModel(t)
	var closeCalls atomic.Int32
	p, err := player.NewWithBackend(uiTestAudioBackend{closeCalls: &closeCalls}, &uiTestDecoder{})
	if err != nil {
		t.Fatal(err)
	}
	m.player = p

	statsPath := filepath.Join(t.TempDir(), "stats.json")
	stats, err := listening.Load(statsPath)
	if err != nil {
		t.Fatal(err)
	}
	stats.Start(listening.Track{Path: "/song.mp3", Name: "Song"}, time.Now())
	m.listening = stats

	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if closeCalls.Load() != 1 {
		t.Fatalf("player backend close calls = %d", closeCalls.Load())
	}
	if _, err := os.Stat(statsPath); err != nil {
		t.Fatalf("listening stats were not saved: %v", err)
	}
}
