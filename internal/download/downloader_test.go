package download

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDownloadCreatesMissingSubfolder(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "subfolder")
	d := &Downloader{Binary: "definitely-not-a-real-binary-xyz"}
	err := d.Download(t.Context(), sub, "https://example.com/watch?v=one")
	if err != ErrYTDLP {
		t.Fatalf("error = %v, want ErrYTDLP (before binary lookup the folder should exist)", err)
	}
	info, statErr := os.Stat(sub)
	if statErr != nil || !info.IsDir() {
		t.Fatalf("subfolder was not created: %v", statErr)
	}
}

func TestDownloadRejectsNonDirectory(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "occupied")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &Downloader{Binary: "definitely-not-a-real-binary-xyz"}
	if err := d.Download(t.Context(), file, "https://example.com/watch?v=one"); err != ErrInvalidDest {
		t.Fatalf("error = %v, want ErrInvalidDest", err)
	}
}

func TestBuildArgsUsesDirectProcessArguments(t *testing.T) {
	dir := filepath.Join("tmp", "music with spaces")
	rawURL := "https://example.com/watch?v=one&list=two"
	args, err := BuildArgs(dir, rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if args[len(args)-1] != rawURL {
		t.Fatalf("URL argument = %q", args[len(args)-1])
	}
	if !slices.Contains(args, filepath.Join(dir, "%(title)s.%(ext)s")) {
		t.Fatalf("output template missing from args: %#v", args)
	}
	for _, arg := range args {
		if arg == "sh" || arg == "bash" || arg == "-c" {
			t.Fatalf("shell argument found: %#v", args)
		}
	}
	for _, want := range []string{"-x", "--audio-format", "mp3", "--embed-metadata", "--embed-thumbnail"} {
		if !slices.Contains(args, want) {
			t.Fatalf("missing argument %q in %#v", want, args)
		}
	}
}

func TestBuildArgsRejectsEmptyURL(t *testing.T) {
	if _, err := BuildArgs("/music", "  "); err != ErrInvalidURL {
		t.Fatalf("error = %v, want ErrInvalidURL", err)
	}
}

func TestDownloaderTerminatesOptionsBeforeURL(t *testing.T) {
	args, err := BuildArgs("/music", "https://example.com/-playlist")
	if err != nil {
		t.Fatal(err)
	}
	if len(args) < 2 || args[len(args)-2] != "--" || args[len(args)-1] != "https://example.com/-playlist" {
		t.Fatalf("argument tail = %#v", args)
	}
}

func TestDownloaderRejectsInvalidScheme(t *testing.T) {
	if _, err := BuildArgs("/music", "file:///etc/passwd"); err != ErrInvalidURL {
		t.Fatalf("error = %v", err)
	}
}

func TestDownloaderRejectsControlCharacters(t *testing.T) {
	if _, err := BuildArgs("/music", "https://example.com/song\n--exec=bad"); err != ErrInvalidURL {
		t.Fatalf("error = %v", err)
	}
}
