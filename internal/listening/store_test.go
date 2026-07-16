package listening

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordPersistAndSummaries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "stats.json")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	track := Track{Path: "/music/one.flac", Name: "One", Artist: "Metallica", Album: "Justice"}
	s.Start(track, now)
	s.Listen(1500*time.Millisecond, now)
	s.Finish(true, now)
	s.Start(track, now.Add(time.Minute))
	s.Listen(2*time.Second, now)
	s.Finish(false, now)
	summary := s.Period(1, now)
	if summary.Plays != 2 || summary.Completions != 1 || summary.Skips != 1 || summary.ListeningSeconds != 3 {
		t.Fatalf("summary=%#v", summary)
	}
	artist := s.Artist("metal")
	if artist.Plays != 2 || len(artist.Top) != 1 {
		t.Fatalf("artist=%#v", artist)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Artist("Metallica"); got.Plays != 2 || got.ListeningSeconds != 3 {
		t.Fatalf("loaded=%#v", got)
	}
}
func TestStartSwitchCountsSkipAndListenClamp(t *testing.T) {
	s, _ := Load("")
	now := time.Now()
	s.Start(Track{Path: "a", Name: "A"}, now)
	s.Listen(20*time.Second, now)
	s.Start(Track{Path: "b", Name: "B"}, now)
	if got := s.Artist(""); got.Skips != 1 || got.ListeningSeconds != 2 || got.Plays != 2 {
		t.Fatalf("summary=%#v", got)
	}
}
func TestSaveAttemptsAreRateLimited(t *testing.T) {
	s, _ := Load("")
	now := time.Now()
	s.Start(Track{Path: "a", Name: "A"}, now)
	if s.ShouldSave(now.Add(30 * time.Second)) {
		t.Fatal("saved before interval")
	}
	if !s.ShouldSave(now.Add(time.Minute)) {
		t.Fatal("due save was not requested")
	}
	if s.ShouldSave(now.Add(time.Minute + time.Second)) {
		t.Fatal("save retry was not rate limited")
	}
}
func TestRestartSameTrackCountsNewPlay(t *testing.T) {
	s, _ := Load("")
	now := time.Now()
	track := Track{Path: "a", Name: "A"}
	s.Start(track, now)
	s.Restart(track, now.Add(time.Second))
	got := s.Artist("")
	if got.Plays != 2 || got.Skips != 1 {
		t.Fatalf("restart=%#v", got)
	}
}

func TestListeningPartialDecodeCannotCorruptLiveState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	partial := []byte(`{"version":1,"tracks":{"injected":{"path":"injected"}},"days":`)
	if err := os.WriteFile(path, partial, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Load(path)
	if err == nil {
		t.Fatal("expected malformed JSON error")
	}
	if got := store.Artist(""); got.Plays != 0 || len(got.Top) != 0 {
		t.Fatalf("partially decoded state became live: %#v", got)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil || string(data) != string(partial) {
		t.Fatalf("source changed: %q err=%v", data, readErr)
	}
}
