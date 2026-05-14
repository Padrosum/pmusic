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

type Player struct {
	mu       sync.Mutex
	state    State
	ctrl     *beep.Ctrl
	streamer beep.StreamSeekCloser
	srcRate  beep.SampleRate
	total    time.Duration
	onDone   func()
}

func New() *Player {
	return &Player{}
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
	p.state = Stopped
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
		return nil
	}
	if err != nil {
		f.Close()
		return err
	}

	var final beep.Streamer
	if format.SampleRate != outputRate {
		final = beep.Resample(4, format.SampleRate, outputRate, stream)
	} else {
		final = stream
	}

	ctrl := &beep.Ctrl{Streamer: final, Paused: false}

	p.mu.Lock()
	p.ctrl = ctrl
	p.streamer = stream
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
	p.state = Stopped
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
