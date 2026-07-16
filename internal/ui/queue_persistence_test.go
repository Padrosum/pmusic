package ui

import (
	"errors"
	"strings"
	"testing"

	pfs "github.com/Padrosum/pmusic/internal/fs"
	tea "github.com/charmbracelet/bubbletea"
)

func queuePersistenceModel(t *testing.T) (*Model, *int) {
	t.Helper()
	m := commandTestModel(t)
	m.queue = []pfs.Track{
		{Name: "One", Path: "/one.mp3"},
		{Name: "Two", Path: "/two.mp3"},
	}
	m.showQueue = true
	calls := new(int)
	m.saveQueueFn = func([]pfs.Track) error {
		*calls++
		return nil
	}
	return m, calls
}

func queueKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestQueueNavigationDoesNotPersist(t *testing.T) {
	m, calls := queuePersistenceModel(t)
	m.handleQueue(queueKey('j'))
	if *calls != 0 {
		t.Fatalf("save calls = %d", *calls)
	}
}

func TestQueueCloseDoesNotPersistWithoutChanges(t *testing.T) {
	m, calls := queuePersistenceModel(t)
	m.handleQueue(queueKey('q'))
	if *calls != 0 {
		t.Fatalf("save calls = %d", *calls)
	}
}

func TestQueueMutationPersistsExactlyOnce(t *testing.T) {
	m, calls := queuePersistenceModel(t)
	m.handleQueue(queueKey('d'))
	if *calls != 1 {
		t.Fatalf("save calls = %d", *calls)
	}
}

func TestQueueSaveFailureIsReported(t *testing.T) {
	m, _ := queuePersistenceModel(t)
	m.saveQueueFn = func([]pfs.Track) error { return errors.New("disk full") }
	m.handleQueue(queueKey('d'))
	if !strings.Contains(m.notification, "queue save") || !strings.Contains(m.notification, "disk full") {
		t.Fatalf("notification = %q", m.notification)
	}
}
