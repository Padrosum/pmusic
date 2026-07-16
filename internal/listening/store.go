package listening

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Padrosum/pmusic/internal/persistence"
)

const Version = 1

type Track struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Artist string `json:"artist,omitempty"`
	Album  string `json:"album,omitempty"`
}

type TrackStats struct {
	Track
	Plays            int       `json:"plays"`
	Skips            int       `json:"skips"`
	Completions      int       `json:"completions"`
	ListeningSeconds int64     `json:"listening_seconds"`
	LastPlayed       time.Time `json:"last_played"`
}

type DayStats struct {
	ListeningSeconds int64            `json:"listening_seconds"`
	Plays            int              `json:"plays"`
	Completions      int              `json:"completions"`
	Skips            int              `json:"skips"`
	TrackSeconds     map[string]int64 `json:"track_seconds,omitempty"`
}

type Data struct {
	Version int                    `json:"version"`
	Tracks  map[string]*TrackStats `json:"tracks"`
	Days    map[string]*DayStats   `json:"days"`
}

type Store struct {
	path            string
	data            Data
	activePath      string
	dirty           bool
	lastSave        time.Time
	listenRemainder time.Duration
}

func DefaultPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "pmusic", "listening-stats.json"), nil
}

func Load(path string) (*Store, error) {
	s := &Store{path: path, lastSave: time.Now(), data: Data{Version: Version, Tracks: map[string]*TrackStats{}, Days: map[string]*DayStats{}}}
	if path == "" {
		return s, nil
	}
	data, found, err := persistence.LoadJSON[Data](path, func(data *Data) error {
		if data.Version != 0 && data.Version != Version {
			return fmt.Errorf("unsupported listening data version %d", data.Version)
		}
		return nil
	})
	if err != nil {
		return s, err
	}
	if !found {
		return s, nil
	}
	s.data = data
	if s.data.Tracks == nil {
		s.data.Tracks = map[string]*TrackStats{}
	}
	if s.data.Days == nil {
		s.data.Days = map[string]*DayStats{}
	}
	s.data.Version = Version
	s.pruneDays(time.Now(), 400)
	return s, nil
}

func (s *Store) Start(track Track, now time.Time) {
	if track.Path == "" || s.activePath == track.Path {
		return
	}
	if s.activePath != "" {
		s.Finish(false, now)
	}
	item := s.ensure(track)
	item.Plays++
	item.LastPlayed = now
	s.day(now).Plays++
	s.activePath = track.Path
	s.dirty = true
}

func (s *Store) Restart(track Track, now time.Time) {
	if track.Path == "" {
		return
	}
	if s.activePath != "" {
		s.Finish(false, now)
	}
	s.Start(track, now)
}

func (s *Store) Listen(delta time.Duration, now time.Time) {
	if s.activePath == "" || delta <= 0 {
		return
	}
	// Prevent suspend/resume or delayed event loops from inventing listening time.
	if delta > 2*time.Second {
		delta = 2 * time.Second
	}
	delta += s.listenRemainder
	seconds := int64(delta / time.Second)
	s.listenRemainder = delta - time.Duration(seconds)*time.Second
	if seconds == 0 {
		return
	}
	if item := s.data.Tracks[s.activePath]; item != nil {
		item.ListeningSeconds += seconds
	}
	day := s.day(now)
	day.ListeningSeconds += seconds
	day.TrackSeconds[s.activePath] += seconds
	s.dirty = true
}

func (s *Store) Finish(completed bool, now time.Time) {
	if s.activePath == "" {
		return
	}
	item := s.data.Tracks[s.activePath]
	if completed {
		if item != nil {
			item.Completions++
		}
		s.day(now).Completions++
	} else {
		if item != nil {
			item.Skips++
		}
		s.day(now).Skips++
	}
	s.activePath = ""
	s.listenRemainder = 0
	s.dirty = true
}

func (s *Store) FinishPath(path string, completed bool, now time.Time) {
	if path == "" || s.activePath != path {
		return
	}
	s.Finish(completed, now)
}

func (s *Store) ShouldSave(now time.Time) bool {
	if !s.dirty || (!s.lastSave.IsZero() && now.Sub(s.lastSave) < time.Minute) {
		return false
	}
	// Record attempts as well as successes so a read-only/full filesystem does
	// not cause a write-and-notification storm on every animation tick.
	s.lastSave = now
	return true
}
func (s *Store) Save() error {
	if !s.dirty || s.path == "" {
		return nil
	}
	if err := persistence.SaveJSON(s.path, s.data); err != nil {
		return err
	}
	s.dirty = false
	s.lastSave = time.Now()
	return nil
}

type Summary struct {
	ListeningSeconds          int64
	Plays, Completions, Skips int
	Top                       []TrackStats
}

func (s *Store) Period(days int, now time.Time) Summary {
	if days < 1 {
		days = 1
	}
	var out Summary
	cutoff := now.AddDate(0, 0, -(days - 1))
	for key, d := range s.data.Days {
		day, err := time.ParseInLocation("2006-01-02", key, now.Location())
		if err != nil || day.Before(midnight(cutoff)) || day.After(now) {
			continue
		}
		out.ListeningSeconds += d.ListeningSeconds
		out.Plays += d.Plays
		out.Completions += d.Completions
		out.Skips += d.Skips
	}
	periodSeconds := map[string]int64{}
	for key, d := range s.data.Days {
		day, err := time.ParseInLocation("2006-01-02", key, now.Location())
		if err != nil || day.Before(midnight(cutoff)) || day.After(now) {
			continue
		}
		for path, seconds := range d.TrackSeconds {
			periodSeconds[path] += seconds
		}
	}
	for path, seconds := range periodSeconds {
		if item := s.data.Tracks[path]; item != nil {
			copy := *item
			copy.ListeningSeconds = seconds
			out.Top = append(out.Top, copy)
		}
	}
	sort.Slice(out.Top, func(i, j int) bool { return out.Top[i].ListeningSeconds > out.Top[j].ListeningSeconds })
	if len(out.Top) > 8 {
		out.Top = out.Top[:8]
	}
	return out
}

func (s *Store) Artist(query string) Summary {
	q := strings.ToLower(strings.TrimSpace(query))
	var out Summary
	for _, t := range s.data.Tracks {
		if q != "" && !strings.Contains(strings.ToLower(t.Artist), q) {
			continue
		}
		out.ListeningSeconds += t.ListeningSeconds
		out.Plays += t.Plays
		out.Completions += t.Completions
		out.Skips += t.Skips
		out.Top = append(out.Top, *t)
	}
	sort.Slice(out.Top, func(i, j int) bool {
		if out.Top[i].ListeningSeconds == out.Top[j].ListeningSeconds {
			return out.Top[i].Plays > out.Top[j].Plays
		}
		return out.Top[i].ListeningSeconds > out.Top[j].ListeningSeconds
	})
	if len(out.Top) > 8 {
		out.Top = out.Top[:8]
	}
	return out
}

func (s *Store) ensure(track Track) *TrackStats {
	item := s.data.Tracks[track.Path]
	if item == nil {
		item = &TrackStats{Track: track}
		s.data.Tracks[track.Path] = item
	} else {
		item.Track = track
	}
	return item
}
func (s *Store) day(now time.Time) *DayStats {
	key := now.Format("2006-01-02")
	d := s.data.Days[key]
	if d == nil {
		d = &DayStats{TrackSeconds: map[string]int64{}}
		s.data.Days[key] = d
	}
	if d.TrackSeconds == nil {
		d.TrackSeconds = map[string]int64{}
	}
	return d
}
func (s *Store) top(include func(*TrackStats) bool, limit int) []TrackStats {
	var out []TrackStats
	for _, t := range s.data.Tracks {
		if include(t) {
			out = append(out, *t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ListeningSeconds == out[j].ListeningSeconds {
			return out[i].Plays > out[j].Plays
		}
		return out[i].ListeningSeconds > out[j].ListeningSeconds
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
func (s *Store) pruneDays(now time.Time, keep int) {
	cutoff := midnight(now.AddDate(0, 0, -keep))
	for key := range s.data.Days {
		day, err := time.ParseInLocation("2006-01-02", key, now.Location())
		if err != nil || day.Before(cutoff) {
			delete(s.data.Days, key)
		}
	}
}
