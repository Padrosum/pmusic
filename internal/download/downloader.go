package download

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrYTDLP       = errors.New("yt-dlp is not installed or not available on PATH")
	ErrInvalidURL  = errors.New("the selected result has no valid URL")
	ErrInvalidDest = errors.New("music directory does not exist or is not a directory")
	ErrNotWritable = errors.New("music directory is not writable")
)

const maxErrorOutput = 32 << 10

type Downloader struct {
	Binary string
}

func New() *Downloader { return &Downloader{Binary: "yt-dlp"} }

func BuildArgs(musicDir, rawURL string) ([]string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, ErrInvalidURL
	}
	return []string{
		"--no-warnings",
		"-x", "--audio-format", "mp3", "--audio-quality", "0",
		"--embed-metadata", "--embed-thumbnail",
		"-o", filepath.Join(musicDir, "%(title)s.%(ext)s"),
		rawURL,
	}, nil
}

func (d *Downloader) Download(ctx context.Context, musicDir, rawURL string) error {
	info, err := os.Stat(musicDir)
	if err != nil || !info.IsDir() {
		return ErrInvalidDest
	}
	if info.Mode().Perm()&0222 == 0 {
		return ErrNotWritable
	}
	binary := d.Binary
	if binary == "" {
		binary = "yt-dlp"
	}
	if _, err := exec.LookPath(binary); err != nil {
		return ErrYTDLP
	}
	args, err := BuildArgs(musicDir, rawURL)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = io.Discard
	var stderr limitedBuffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		detail := strings.TrimSpace(stderr.String())
		if line, _, ok := strings.Cut(detail, "\n"); ok {
			detail = line
		}
		if detail != "" {
			return fmt.Errorf("yt-dlp failed: %s", detail)
		}
		return fmt.Errorf("yt-dlp failed: %w", err)
	}
	return nil
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := maxErrorOutput - b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return original, nil
}
