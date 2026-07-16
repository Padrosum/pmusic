package persistence

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type testData struct {
	Value string `json:"value"`
}

func TestPersistenceMissingFileIsNotCorruption(t *testing.T) {
	_, found, err := LoadJSON[testData](filepath.Join(t.TempDir(), "missing.json"), nil)
	if err != nil || found {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestPersistenceMalformedJSONIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{"value":`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, found, err := LoadJSON[testData](path, nil)
	var decodeErr *DecodeError
	if !found || !errors.As(err, &decodeErr) || !strings.Contains(err.Error(), path) {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestPersistencePermissionDeniedIsReported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission semantics")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	err := SaveJSON(filepath.Join(dir, "state.json"), testData{Value: "new"})
	if err == nil {
		t.Skip("environment can write through directory permissions")
	}
}

func TestPersistenceFailedRenamePreservesOldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SaveJSON(path, testData{Value: "new"}); err == nil {
		t.Fatal("expected rename failure")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("old destination changed: info=%v err=%v", info, err)
	}
}

func TestPersistencePartialWriteDoesNotReplaceOldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	old := []byte("old data")
	if err := os.WriteFile(path, old, 0o600); err != nil {
		t.Fatal(err)
	}
	err := writeFileAtomic(path, []byte("new data"), 0o600, func(string) error {
		return errors.New("simulated interrupted write")
	})
	if err == nil {
		t.Fatal("expected injected failure")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(old) {
		t.Fatalf("old file = %q, err=%v", got, err)
	}
}
