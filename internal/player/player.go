package player

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

const outputRate beep.SampleRate = 44100

var ErrClosed = errors.New("player is closed")

type State int

const (
	Stopped State = iota
	Playing
	Paused
)

// AudioBackend is the smallest boundary around beep's process-global speaker.
// Player never holds its state mutex while calling these methods.
type AudioBackend interface {
	Init(sampleRate beep.SampleRate, bufferSize int) error
	Lock()
	Unlock()
	Clear()
	Play(beep.Streamer)
	Close() error
}

// Decoder owns opening the source and must return a stream that closes it.
type Decoder interface {
	Decode(path string) (beep.StreamSeekCloser, beep.Format, error)
}

type speakerBackend struct{}

func (speakerBackend) Init(rate beep.SampleRate, size int) error { return speaker.Init(rate, size) }
func (speakerBackend) Lock()                                     { speaker.Lock() }
func (speakerBackend) Unlock()                                   { speaker.Unlock() }
func (speakerBackend) Clear()                                    { speaker.Clear() }
func (speakerBackend) Play(s beep.Streamer)                      { speaker.Play(s) }
func (speakerBackend) Close() error                              { speaker.Close(); return nil }

type fileDecoder struct {
	open func(string) (io.ReadCloser, error)
}

func (d fileDecoder) Decode(path string) (beep.StreamSeekCloser, beep.Format, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".mp3" && ext != ".flac" && ext != ".wav" {
		return nil, beep.Format{}, fmt.Errorf("unsupported audio format %q", ext)
	}
	opener := d.open
	if opener == nil {
		opener = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	}
	f, err := opener(path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	var stream beep.StreamSeekCloser
	var format beep.Format
	switch ext {
	case ".mp3":
		stream, format, err = mp3.Decode(f)
	case ".flac":
		stream, format, err = flac.Decode(f)
	case ".wav":
		stream, format, err = wav.Decode(f)
	}
	if err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return nil, beep.Format{}, errors.Join(err, closeErr)
		}
		return nil, beep.Format{}, err
	}
	return stream, format, nil
}

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
	return n, ok
}

func (v *volumeStreamer) Err() error { return v.s.Err() }

// ownedStream serializes stream access and makes source ownership exactly-once.
type ownedStream struct {
	mu        sync.Mutex
	stream    beep.StreamSeekCloser
	closeOnce sync.Once
	closeErr  error
}

func (s *ownedStream) Stream(samples [][2]float64) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stream.Stream(samples)
}

func (s *ownedStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stream.Err()
}

func (s *ownedStream) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stream.Len()
}

func (s *ownedStream) Position() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stream.Position()
}

func (s *ownedStream) Seek(position int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stream.Seek(position)
}

func (s *ownedStream) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.closeErr = s.stream.Close()
	})
	return s.closeErr
}

type Player struct {
	opMu sync.Mutex
	mu   sync.Mutex

	backend  AudioBackend
	decoder  Decoder
	closed   bool
	closeErr error

	state        State
	ctrl         *beep.Ctrl
	streamer     *ownedStream
	volCtrl      *volumeStreamer
	volume       float64
	srcRate      beep.SampleRate
	total        time.Duration
	onDone       func()
	generation   uint64
	completed    atomic.Uint64
	lifecycleErr error
}

func New() (*Player, error) {
	return NewWithBackend(speakerBackend{}, fileDecoder{})
}

func NewWithBackend(backend AudioBackend, decoder Decoder) (*Player, error) {
	if backend == nil {
		return nil, errors.New("audio backend is nil")
	}
	if decoder == nil {
		return nil, errors.New("audio decoder is nil")
	}
	if err := backend.Init(outputRate, outputRate.N(time.Second/10)); err != nil {
		return nil, fmt.Errorf("initialize audio backend: %w", err)
	}
	return &Player{backend: backend, decoder: decoder, volume: 1}, nil
}

func (p *Player) SetOnDone(fn func()) {
	p.mu.Lock()
	p.onDone = fn
	p.mu.Unlock()
}

func (p *Player) Play(path string) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	p.generation++
	generation := p.generation
	oldCtrl, oldStream := p.ctrl, p.streamer
	p.ctrl, p.streamer, p.volCtrl = nil, nil, nil
	p.state, p.total = Stopped, 0
	p.mu.Unlock()

	if err := p.stopAudio(oldCtrl, oldStream); err != nil {
		return fmt.Errorf("close previous stream: %w", err)
	}

	decoded, format, err := p.decoder.Decode(path)
	if err != nil {
		return err
	}
	stream := &ownedStream{stream: decoded}
	var middle beep.Streamer = stream
	if format.SampleRate != outputRate {
		middle = beep.Resample(4, format.SampleRate, outputRate, stream)
	}
	volNode := &volumeStreamer{s: middle}
	ctrl := &beep.Ctrl{Streamer: volNode}

	p.mu.Lock()
	if p.closed || p.generation != generation {
		p.mu.Unlock()
		return stream.Close()
	}
	volNode.volume = p.volume
	p.ctrl = ctrl
	p.streamer = stream
	p.volCtrl = volNode
	p.srcRate = format.SampleRate
	p.total = format.SampleRate.D(stream.Len())
	p.state = Playing
	p.mu.Unlock()

	p.backend.Play(beep.Seq(ctrl, beep.Callback(func() {
		// beep invokes callbacks while holding its speaker lock. Publishing the
		// generation atomically keeps that callback independent of Player.mu.
		for {
			previous := p.completed.Load()
			if generation <= previous || p.completed.CompareAndSwap(previous, generation) {
				return
			}
		}
	})))
	return nil
}

func (p *Player) stopAudio(ctrl *beep.Ctrl, stream *ownedStream) error {
	if ctrl != nil {
		p.backend.Lock()
		ctrl.Paused = true
		p.backend.Unlock()
	}
	p.backend.Clear()
	if stream != nil {
		return stream.Close()
	}
	return nil
}

func (p *Player) consumeCompletion() {
	generation := p.completed.Swap(0)
	if generation == 0 {
		return
	}
	p.opMu.Lock()
	defer p.opMu.Unlock()
	p.mu.Lock()
	if generation != p.generation || p.ctrl == nil {
		p.mu.Unlock()
		return
	}
	stream := p.streamer
	onDone := p.onDone
	p.ctrl, p.streamer, p.volCtrl = nil, nil, nil
	p.state, p.total = Stopped, 0
	p.mu.Unlock()
	if stream != nil {
		p.recordLifecycleError(stream.Close())
	}
	if onDone != nil {
		onDone()
	}
}

func (p *Player) Stop() error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	err := p.stopLocked()
	p.recordLifecycleError(err)
	return err
}

func (p *Player) stopLocked() error {
	p.mu.Lock()
	ctrl, stream := p.ctrl, p.streamer
	p.generation++
	p.ctrl, p.streamer, p.volCtrl = nil, nil, nil
	p.state, p.total = Stopped, 0
	p.mu.Unlock()
	return p.stopAudio(ctrl, stream)
}

func (p *Player) Close() error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	p.mu.Lock()
	if p.closed {
		err := p.closeErr
		p.mu.Unlock()
		return err
	}
	p.closed = true
	p.mu.Unlock()
	stopErr := p.stopLocked()
	backendErr := p.backend.Close()
	p.mu.Lock()
	p.closeErr = errors.Join(p.lifecycleErr, stopErr, backendErr)
	err := p.closeErr
	p.mu.Unlock()
	return err
}

func (p *Player) recordLifecycleError(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	p.lifecycleErr = errors.Join(p.lifecycleErr, err)
	p.mu.Unlock()
}

func (p *Player) MarkPending() {
	p.mu.Lock()
	if !p.closed {
		p.state = Playing
	}
	p.mu.Unlock()
}

func (p *Player) TogglePause() {
	p.consumeCompletion()
	p.mu.Lock()
	ctrl := p.ctrl
	p.mu.Unlock()
	if ctrl == nil {
		return
	}
	p.backend.Lock()
	ctrl.Paused = !ctrl.Paused
	paused := ctrl.Paused
	p.backend.Unlock()
	p.mu.Lock()
	if p.ctrl == ctrl {
		if paused {
			p.state = Paused
		} else {
			p.state = Playing
		}
	}
	p.mu.Unlock()
}

func (p *Player) Pause() bool {
	p.consumeCompletion()
	p.mu.Lock()
	ctrl := p.ctrl
	p.mu.Unlock()
	if ctrl == nil {
		return false
	}
	p.backend.Lock()
	ctrl.Paused = true
	p.backend.Unlock()
	p.mu.Lock()
	if p.ctrl == ctrl {
		p.state = Paused
	}
	p.mu.Unlock()
	return true
}

func (p *Player) Seek(delta time.Duration) bool {
	p.consumeCompletion()
	p.mu.Lock()
	stream, srcRate, total := p.streamer, p.srcRate, p.total
	p.mu.Unlock()
	if stream == nil || total <= 0 {
		return false
	}
	p.backend.Lock()
	position := srcRate.D(stream.Position()) + delta
	if position < 0 {
		position = 0
	}
	if position >= total {
		position = total - time.Millisecond
	}
	err := stream.Seek(srcRate.N(position))
	p.backend.Unlock()
	return err == nil
}

func (p *Player) SeekTo(position time.Duration) bool {
	p.consumeCompletion()
	p.mu.Lock()
	stream, srcRate, total := p.streamer, p.srcRate, p.total
	p.mu.Unlock()
	if stream == nil || total <= 0 {
		return false
	}
	if position < 0 {
		position = 0
	}
	if position >= total {
		position = total - time.Millisecond
	}
	p.backend.Lock()
	err := stream.Seek(srcRate.N(position))
	p.backend.Unlock()
	return err == nil
}

func (p *Player) SetVolume(value float64) {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	p.mu.Lock()
	p.volume = value
	volume := p.volCtrl
	p.mu.Unlock()
	if volume != nil {
		p.backend.Lock()
		volume.volume = value
		p.backend.Unlock()
	}
}

func (p *Player) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *Player) State() State {
	p.consumeCompletion()
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) Progress() (ratio float64, elapsed, total time.Duration) {
	p.consumeCompletion()
	p.mu.Lock()
	stream, srcRate, duration := p.streamer, p.srcRate, p.total
	p.mu.Unlock()
	if stream == nil || duration <= 0 {
		return 0, 0, 0
	}
	p.backend.Lock()
	position := srcRate.D(stream.Position())
	p.backend.Unlock()
	ratio = float64(position) / float64(duration)
	if ratio > 1 {
		ratio = 1
	}
	return ratio, position, duration
}
