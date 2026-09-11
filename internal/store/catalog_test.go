package store

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testManifest(url string, body []byte) Manifest {
	hash := sha256.Sum256(body)
	return Manifest{
		Version: 1,
		Release: "test-release",
		Files: []ManifestFile{{
			Name: "example", Kind: "plugin", URL: url,
			SHA256: fmt.Sprintf("%x", hash[:]),
		}},
	}
}

func syncFixture(t *testing.T, handler http.HandlerFunc, expected []byte) (string, Manifest, *http.Client, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	dir := t.TempDir()
	dest := filepath.Join(dir, "lua", "plugins", "example.lua")
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, testManifest(server.URL, expected), server.Client(), server.Close
}

func assertExistingPlugin(t *testing.T, dir string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, "lua", "plugins", "example.lua"))
	if err != nil || string(got) != "existing" {
		t.Fatalf("existing plugin = %q, err=%v", got, err)
	}
}

func TestStoreSyncRejectsHashMismatch(t *testing.T) {
	dir, manifest, client, closeServer := syncFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("tampered"))
	}, []byte("expected"))
	defer closeServer()
	if err := syncManifest(context.Background(), dir, manifest, client); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("err = %v", err)
	}
	assertExistingPlugin(t, dir)
}

func TestStoreSyncTimeoutPreservesExistingPlugin(t *testing.T) {
	dir, manifest, _, closeServer := syncFixture(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}, []byte("expected"))
	defer closeServer()
	client := &http.Client{Timeout: 20 * time.Millisecond}
	if err := syncManifest(context.Background(), dir, manifest, client); err == nil {
		t.Fatal("expected timeout")
	}
	assertExistingPlugin(t, dir)
}

func TestStoreSyncOversizePreservesExistingPlugin(t *testing.T) {
	dir, manifest, client, closeServer := syncFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, maxFileSize+1))
	}, []byte("expected"))
	defer closeServer()
	if err := syncManifest(context.Background(), dir, manifest, client); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err = %v", err)
	}
	assertExistingPlugin(t, dir)
}

func TestStoreSyncPartialBodyPreservesExistingPlugin(t *testing.T) {
	dir, manifest, client, closeServer := syncFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("partial"))
	}, []byte("complete expected body"))
	defer closeServer()
	if err := syncManifest(context.Background(), dir, manifest, client); err == nil {
		t.Fatal("expected partial-body failure")
	}
	assertExistingPlugin(t, dir)
}

func TestStoreSyncNon2xxPreservesExistingPlugin(t *testing.T) {
	dir, manifest, client, closeServer := syncFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}, []byte("expected"))
	defer closeServer()
	if err := syncManifest(context.Background(), dir, manifest, client); err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("err = %v", err)
	}
	assertExistingPlugin(t, dir)
}

func TestStoreSyncInstallsVerifiedFileAtomically(t *testing.T) {
	body := []byte("return { verified = true }\n")
	dir, manifest, client, closeServer := syncFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}, body)
	defer closeServer()
	if err := syncManifest(context.Background(), dir, manifest, client); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "lua", "plugins", "example.lua")
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != string(body) {
		t.Fatalf("installed = %q, err=%v", got, err)
	}
	info, err := os.Stat(dest)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestStoreSyncInstallsCatalogAndSeedsLibrary(t *testing.T) {
	body := []byte("cins T\n")
	hash := sha256.Sum256(body)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	dir := t.TempDir()
	manifest := Manifest{
		Version: 1,
		Release: "test-release",
		Files: []ManifestFile{{
			Name: "library", Kind: "catalog", URL: server.URL,
			SHA256: fmt.Sprintf("%x", hash[:]),
		}},
	}
	if err := syncManifest(context.Background(), dir, manifest, server.Client()); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{filepath.Join("gl", "library.gl"), "library.gl"} {
		got, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil || string(got) != string(body) {
			t.Fatalf("%s = %q err=%v", rel, got, err)
		}
	}
}

func TestStoreSyncPreservesExistingLibraryGL(t *testing.T) {
	body := []byte("cins T\n")
	hash := sha256.Sum256(body)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	dir := t.TempDir()
	existing := filepath.Join(dir, "library.gl")
	if err := os.WriteFile(existing, []byte("veri mine { x = 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		Version: 1,
		Release: "test-release",
		Files: []ManifestFile{{
			Name: "library", Kind: "catalog", URL: server.URL,
			SHA256: fmt.Sprintf("%x", hash[:]),
		}},
	}
	if err := syncManifest(context.Background(), dir, manifest, server.Client()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(existing)
	if err != nil || string(got) != "veri mine { x = 1 }\n" {
		t.Fatalf("library.gl overwritten: %q err=%v", got, err)
	}
	pkg, err := os.ReadFile(filepath.Join(dir, "gl", "library.gl"))
	if err != nil || string(pkg) != string(body) {
		t.Fatalf("package = %q err=%v", pkg, err)
	}
}

func TestStoreSyncRejectsUnknownKind(t *testing.T) {
	dir := t.TempDir()
	err := syncManifest(context.Background(), dir, Manifest{
		Version: 1,
		Release: "test-release",
		Files:   []ManifestFile{{Name: "x", Kind: "exploit", URL: "http://example.invalid", SHA256: strings.Repeat("ab", 32)}},
	}, http.DefaultClient)
	if err == nil || !strings.Contains(err.Error(), "invalid kind") {
		t.Fatalf("err = %v", err)
	}
}

func TestDestinationForCatalog(t *testing.T) {
	got, err := destinationFor(ManifestFile{Name: "library", Kind: "catalog"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("gl", "library.gl")
	if got != want {
		t.Fatalf("destination = %q, want %q", got, want)
	}
}
