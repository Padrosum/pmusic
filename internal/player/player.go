package player

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

// Fixed output sample rate — speaker is initialized once at startup.
const outputRate beep.SampleRate = 44100

func init() {
	speaker.Init(outputRate, outputRate.N(time.Second/10))
}

type State int

const (
	Stopped State = iota
	Playing
	Paused
)

// volumeStreamer wraps a Streamer and scales sample amplitudes by volume (0.0–1.0).
// Volume must only be modified while holding speaker.Lock.
type volumeStreamer struct {
	s      beep.Streamer
	volume float64
}

func (v *volumeStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = v.s.Stream(samples)
	for i := 0; i < n; i++ {
		samples[i][0] *= v.volume
		samples[i][1] *= v.volume
	}
	return
}

func (v *volumeStreamer) Err() error { return v.s.Err() }

type Player struct {
	mu       sync.Mutex
	state    State
	ctrl     *beep.Ctrl
	streamer beep.StreamSeekCloser
	volCtrl  *volumeStreamer
	volume   float64 // 0.0–1.0; default 1.0
	srcRate  beep.SampleRate
	total    time.Duration
	onDone   func()
}

func New() *Player {
	return &Player{volume: 1.0}
}

func (p *Player) SetOnDone(fn func()) {
	p.mu.Lock()
	p.onDone = fn
	p.mu.Unlock()
}

func (p *Player) Play(path string) error {
	// Pause and clear before swapping — prevents concurrent reads on closed streamer.
	p.mu.Lock()
	oldCtrl := p.ctrl
	oldStream := p.streamer
	p.ctrl = nil
	p.streamer = nil
	p.volCtrl = nil
	p.mu.Unlock()

	if oldCtrl != nil {
		speaker.Lock()
		oldCtrl.Paused = true
		speaker.Unlock()
	}
	speaker.Clear()
	if oldStream != nil {
		oldStream.Close()
	}

	f, err := os.Open(path)
	if err != nil {
		// MarkPending may have set Playing; reset so auto-advance can move on.
		p.markStopped()
		return err
	}

	ext := strings.ToLower(filepath.Ext(path))
	var stream beep.StreamSeekCloser
	var format beep.Format

	switch ext {
	case ".mp3":
		stream, format, err = mp3.Decode(f)
	case ".flac":
		stream, format, err = flac.Decode(f)
	case ".wav":
		stream, format, err = wav.Decode(f)
	default:
		f.Close()
		p.markStopped()
		return nil
	}
	if err != nil {
		f.Close()
		p.markStopped()
		return err
	}

	var mid beep.Streamer
	if format.SampleRate != outputRate {
		mid = beep.Resample(4, format.SampleRate, outputRate, stream)
	} else {
		mid = stream
	}

	volNode := &volumeStreamer{s: mid}
	ctrl := &beep.Ctrl{Streamer: volNode, Paused: false}

	p.mu.Lock()
	p.ctrl = ctrl
	p.streamer = stream
	p.volCtrl = volNode
	volNode.volume = p.volume // preserve volume across track changes
	p.srcRate = format.SampleRate
	p.total = format.SampleRate.D(stream.Len())
	p.state = Playing
	onDone := p.onDone
	p.mu.Unlock()

	speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
		p.mu.Lock()
		// Only mark stopped if this is still the active ctrl (not already replaced).
		if p.ctrl == ctrl {
			p.state = Stopped
			p.streamer = nil // prevent Progress() from showing stale 100% position
			p.total = 0
		}
		p.mu.Unlock()
		if onDone != nil {
			onDone()
		}
	})))

	return nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	ctrl := p.ctrl
	stream := p.streamer
	p.ctrl = nil
	p.streamer = nil
	p.volCtrl = nil
	p.state = Stopped
	p.total = 0
	p.mu.Unlock()

	if ctrl != nil {
		speaker.Lock()
		ctrl.Paused = true
		speaker.Unlock()
	}
	speaker.Clear()
	if stream != nil {
		stream.Close()
	}
}

// MarkPending sets state to Playing immediately so tick-based auto-advance
// doesn't fire during the window between cmd dispatch and goroutine execution.
func (p *Player) MarkPending() {
	p.mu.Lock()
	p.state = Playing
	p.mu.Unlock()
}

// markStopped resets the state to Stopped. Used by Play when a track fails to
// open or decode, so the tick-based auto-advance moves past the dead track
// instead of freezing on a phantom "Playing" state set by MarkPending.
func (p *Player) markStopped() {
	p.mu.Lock()
	p.state = Stopped
	p.mu.Unlock()
}

func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return
	}
	speaker.Lock()
	p.ctrl.Paused = !p.ctrl.Paused
	speaker.Unlock()
	if p.ctrl.Paused {
		p.state = Paused
	} else {
		p.state = Playing
	}
}

// Pause is idempotent and reports whether a loaded stream exists.
func (p *Player) Pause() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return false
	}
	speaker.Lock()
	p.ctrl.Paused = true
	speaker.Unlock()
	p.state = Paused
	return true
}

// Seek moves playback position by delta. Clamps to [0, total). No-op if stopped.
func (p *Player) Seek(delta time.Duration) {
	p.mu.Lock()
	stream := p.streamer
	srcRate := p.srcRate
	tot := p.total
	p.mu.Unlock()

	if stream == nil || tot == 0 {
		return
	}

	speaker.Lock()
	pos := srcRate.D(stream.Position())
	target := pos + delta
	if target < 0 {
		target = 0
	}
	if target >= tot {
		target = tot - time.Millisecond
	}
	_ = stream.Seek(srcRate.N(target))
	speaker.Unlock()
}

// SeekTo jumps to an absolute playback position and clamps to [0,total).
// It returns false when no track is loaded.
func (p *Player) SeekTo(position time.Duration) bool {
	p.mu.Lock()
	stream := p.streamer
	srcRate := p.srcRate
	tot := p.total
	p.mu.Unlock()
	if stream == nil || tot <= 0 {
		return false
	}
	if position < 0 {
		position = 0
	}
	if position >= tot {
		position = tot - time.Millisecond
	}
	speaker.Lock()
	err := stream.Seek(srcRate.N(position))
	speaker.Unlock()
	return err == nil
}

// SetVolume sets playback volume in [0.0, 1.0]. Takes effect immediately.
func (p *Player) SetVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.mu.Lock()
	p.volume = v
	vc := p.volCtrl
	p.mu.Unlock()

	if vc != nil {
		speaker.Lock()
		vc.volume = v
		speaker.Unlock()
	}
}

// Volume returns the current volume in [0.0, 1.0].
func (p *Player) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) Progress() (ratio float64, elapsed, total time.Duration) {
	p.mu.Lock()
	stream := p.streamer
	srcRate := p.srcRate
	tot := p.total
	p.mu.Unlock()

	if stream == nil || tot == 0 {
		return 0, 0, 0
	}

	speaker.Lock()
	pos := srcRate.D(stream.Position())
	speaker.Unlock()

	ratio = float64(pos) / float64(tot)
	if ratio > 1 {
		ratio = 1
	}
	return ratio, pos, tot
}
