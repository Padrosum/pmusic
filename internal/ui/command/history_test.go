package command

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryMissingAddPersistClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "history")
	h, err := LoadHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Entries()) != 0 {
		t.Fatal(h.Entries())
	}
	if err = h.Add(":play"); err != nil {
		t.Fatal(err)
	}
	if err = h.Add("play"); err != nil {
		t.Fatal(err)
	}
	if len(h.Entries()) != 1 {
		t.Fatalf("duplicate retained: %v", h.Entries())
	}
	loaded, err := LoadHistory(path)
	if err != nil || len(loaded.Entries()) != 1 {
		t.Fatalf("reload=%v %v", loaded.Entries(), err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
	if err = loaded.Clear(); err != nil {
		t.Fatal(err)
	}
	again, _ := LoadHistory(path)
	if len(again.Entries()) != 0 {
		t.Fatal(again.Entries())
	}
}
func TestHistoryMaxAndDraftNavigation(t *testing.T) {
	h, _ := LoadHistory("")
	for i := 0; i < MaxHistory+20; i++ {
		h.Add(string(rune('a'+i%26)) + itoaTest(i))
	}
	if len(h.Entries()) != MaxHistory {
		t.Fatal(len(h.Entries()))
	}
	h.Reset("")
	last := h.Previous("draft")
	if last == "draft" {
		t.Fatal("did not navigate")
	}
	if got := h.Next(); got != "draft" {
		t.Fatalf("draft=%q", got)
	}
}
func TestHistoryWriteErrorDoesNotPanic(t *testing.T) {
	h, _ := LoadHistory("")
	h.path = t.TempDir()
	if err := h.Add("play"); err == nil {
		t.Fatal("expected write error")
	}
}
func itoaTest(v int) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 8)
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
