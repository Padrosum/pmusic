package player

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/faiface/beep"
)

type fakeBackend struct {
	mu         sync.Mutex
	initErr    error
	stream     beep.Streamer
	closeCount atomic.Int32
}

func (b *fakeBackend) Init(beep.SampleRate, int) error { return b.initErr }
func (b *fakeBackend) Lock()                           { b.mu.Lock() }
func (b *fakeBackend) Unlock()                         { b.mu.Unlock() }
func (b *fakeBackend) Clear()                          { b.mu.Lock(); b.stream = nil; b.mu.Unlock() }
func (b *fakeBackend) Play(stream beep.Streamer)       { b.mu.Lock(); b.stream = stream; b.mu.Unlock() }
func (b *fakeBackend) Close() error                    { b.closeCount.Add(1); return nil }

func (b *fakeBackend) complete(started, proceed chan struct{}) {
	b.mu.Lock()
	stream := b.stream
	close(started)
	<-proceed
	if stream != nil {
		stream.Stream(make([][2]float64, 1))
	}
	b.mu.Unlock()
}

type fakeDecoder struct {
	mu      sync.Mutex
	streams []*fakeStream
	err     error
}

func (d *fakeDecoder) Decode(string) (beep.StreamSeekCloser, beep.Format, error) {
	if d.err != nil {
		return nil, beep.Format{}, d.err
	}
	s := &fakeStream{length: 1}
	d.mu.Lock()
	d.streams = append(d.streams, s)
	d.mu.Unlock()
	return s, beep.Format{SampleRate: outputRate, NumChannels: 2, Precision: 2}, nil
}

type fakeStream struct {
	mu         sync.Mutex
	position   int
	length     int
	closeCount int
}

func (s *fakeStream) Stream([][2]float64) (int, bool) { return 0, false }
func (s *fakeStream) Err() error                      { return nil }
func (s *fakeStream) Len() int                        { return s.length }
func (s *fakeStream) Position() int                   { return s.position }
func (s *fakeStream) Seek(position int) error         { s.position = position; return nil }
func (s *fakeStream) Close() error {
	s.mu.Lock()
	s.closeCount++
	s.mu.Unlock()
	return nil
}
func (s *fakeStream) closes() int { s.mu.Lock(); defer s.mu.Unlock(); return s.closeCount }

func newTestPlayer(t *testing.T) (*Player, *fakeBackend, *fakeDecoder) {
	t.Helper()
	backend, decoder := &fakeBackend{}, &fakeDecoder{}
	p, err := NewWithBackend(backend, decoder)
	if err != nil {
		t.Fatal(err)
	}
	return p, backend, decoder
}

func testCompletionDoesNotDeadlock(t *testing.T, action func(*Player)) {
	t.Helper()
	p, backend, _ := newTestPlayer(t)
	if err := p.Play("track.mp3"); err != nil {
		t.Fatal(err)
	}
	started, proceed := make(chan struct{}), make(chan struct{})
	completionDone := make(chan struct{})
	go func() { backend.complete(started, proceed); close(completionDone) }()
	<-started
	actionDone := make(chan struct{})
	go func() { action(p); close(actionDone) }()
	close(proceed)
	select {
	case <-completionDone:
	case <-time.After(time.Second):
		t.Fatal("completion callback deadlocked")
	}
	select {
	case <-actionDone:
	case <-time.After(time.Second):
		t.Fatal("player action deadlocked")
	}
	if state := p.State(); state != Stopped {
		t.Fatalf("state after completion = %v, want stopped", state)
	}
}

func TestPlayerPauseAtCompletionDoesNotDeadlock(t *testing.T) {
	testCompletionDoesNotDeadlock(t, func(p *Player) { p.Pause() })
}

func TestPlayerTogglePauseAtCompletionDoesNotDeadlock(t *testing.T) {
	testCompletionDoesNotDeadlock(t, func(p *Player) { p.TogglePause() })
}

func TestPlayerInitFailureIsReturned(t *testing.T) {
	want := errors.New("no audio device")
	_, err := NewWithBackend(&fakeBackend{initErr: want}, &fakeDecoder{})
	if !errors.Is(err, want) {
		t.Fatalf("NewWithBackend error = %v, want %v", err, want)
	}
}

func TestPlayerStopIsIdempotent(t *testing.T) {
	p, _, decoder := newTestPlayer(t)
	if err := p.Play("track.mp3"); err != nil {
		t.Fatal(err)
	}
	p.Stop()
	p.Stop()
	if got := decoder.streams[0].closes(); got != 1 {
		t.Fatalf("stream close count = %d, want 1", got)
	}
}

func TestPlayerReplacementClosesPreviousStream(t *testing.T) {
	p, _, decoder := newTestPlayer(t)
	if err := p.Play("one.mp3"); err != nil {
		t.Fatal(err)
	}
	if err := p.Play("two.mp3"); err != nil {
		t.Fatal(err)
	}
	if got := decoder.streams[0].closes(); got != 1 {
		t.Fatalf("replaced stream close count = %d, want 1", got)
	}
	if got := decoder.streams[1].closes(); got != 0 {
		t.Fatalf("active stream close count = %d, want 0", got)
	}
}

func TestPlayerClosesStreamOnNaturalEOF(t *testing.T) {
	p, backend, decoder := newTestPlayer(t)
	if err := p.Play("track.mp3"); err != nil {
		t.Fatal(err)
	}
	started, proceed := make(chan struct{}), make(chan struct{})
	go backend.complete(started, proceed)
	<-started
	close(proceed)
	for deadline := time.Now().Add(time.Second); p.State() != Stopped; {
		if time.Now().After(deadline) {
			t.Fatal("completion was not observed")
		}
	}
	if got := decoder.streams[0].closes(); got != 1 {
		t.Fatalf("stream close count = %d, want 1", got)
	}
}

func TestPlayerClosesStreamOnStop(t *testing.T) {
	p, _, decoder := newTestPlayer(t)
	if err := p.Play("track.mp3"); err != nil {
		t.Fatal(err)
	}
	p.Stop()
	if got := decoder.streams[0].closes(); got != 1 {
		t.Fatalf("stream close count = %d, want 1", got)
	}
}

func TestPlayerClosesStreamOnReplacement(t *testing.T) {
	p, _, decoder := newTestPlayer(t)
	if err := p.Play("one.mp3"); err != nil {
		t.Fatal(err)
	}
	if err := p.Play("two.mp3"); err != nil {
		t.Fatal(err)
	}
	if got := decoder.streams[0].closes(); got != 1 {
		t.Fatalf("replaced stream close count = %d, want 1", got)
	}
}

func TestPlayerCloseIsIdempotent(t *testing.T) {
	p, backend, decoder := newTestPlayer(t)
	if err := p.Play("track.mp3"); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if got := decoder.streams[0].closes(); got != 1 {
		t.Fatalf("stream close count = %d, want 1", got)
	}
	if got := backend.closeCount.Load(); got != 1 {
		t.Fatalf("backend close count = %d, want 1", got)
	}
}

type trackingReadCloser struct {
	io.Reader
	closed atomic.Int32
}

func (r *trackingReadCloser) Close() error { r.closed.Add(1); return nil }

func TestDecodeFailureClosesOpenedFile(t *testing.T) {
	contents := stringsReader("not an mp3")
	source := &trackingReadCloser{Reader: &contents}
	decoder := fileDecoder{open: func(string) (io.ReadCloser, error) { return source, nil }}
	if _, _, err := decoder.Decode("broken.mp3"); err == nil {
		t.Fatal("Decode returned nil error")
	}
	if got := source.closed.Load(); got != 1 {
		t.Fatalf("source close count = %d, want 1", got)
	}
}

type stringsReader string

func (r *stringsReader) Read(p []byte) (int, error) {
	if len(*r) == 0 {
		return 0, io.EOF
	}
	n := copy(p, string(*r))
	*r = (*r)[n:]
	return n, nil
}
