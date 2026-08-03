package cover

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFolderArt(t *testing.T) {
	dir := t.TempDir()
	track := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(track, []byte("fake audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cover.jpg"), []byte("JPEG"), 0o644); err != nil {
		t.Fatal(err)
	}
	art, err := Resolve(track, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if art.Source != SourceFolder {
		t.Fatalf("source = %v, want %v", art.Source, SourceFolder)
	}
}

func TestResolveNoArtAndNoMetadata(t *testing.T) {
	dir := t.TempDir()
	track := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(track, []byte("fake audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(track, "", "", ""); err != ErrNoMetadata {
		t.Fatalf("error = %v, want ErrNoMetadata", err)
	}
}

func TestOnlineFetchAndCache(t *testing.T) {
	var pngData []byte
	{
		var b bytes.Buffer
		if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
			t.Fatal(err)
		}
		pngData = b.Bytes()
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"results":[{"artworkUrl100": %q}]}`, srv.URL+"/art.jpg")
			return
		}
		_, _ = w.Write(pngData)
	}))
	defer srv.Close()

	old := itunesBaseURL
	itunesBaseURL = srv.URL
	defer func() { itunesBaseURL = old }()

	cacheDir := t.TempDir()
	art, err := online("Artist Name", "Album Name", cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if art.Source != SourceOnline {
		t.Fatalf("source = %v, want online", art.Source)
	}
	if len(art.Data) == 0 {
		t.Fatal("empty art data")
	}

	art2, err := online("Artist Name", "Album Name", cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if art2.Source != SourceCache {
		t.Fatalf("second fetch source = %v, want cache", art2.Source)
	}
}

func TestOnlineNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()

	old := itunesBaseURL
	itunesBaseURL = srv.URL
	defer func() { itunesBaseURL = old }()

	if _, err := online("Artist", "Album", t.TempDir()); err == nil {
		t.Fatal("expected error for empty results")
	}
}

func TestRenderWithoutChafa(t *testing.T) {
	// Invalid image must error whether or not chafa is installed.
	if _, err := Render(&Art{Data: []byte("not an image")}, 20, 10); err == nil {
		t.Fatal("expected render error")
	}
}
