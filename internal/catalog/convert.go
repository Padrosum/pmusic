package catalog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultTimeout = 5 * time.Second
	maxJSONBytes   = 4 << 20
	maxErrorBytes  = 32 << 10
)

var (
	ErrGenLangMissing = errors.New("genlang is not installed or not available on PATH")
	ErrNotGenLang     = errors.New("catalog file must be a local .gl file")
)

// Converter turns a GenLang source file into interchange JSON.
type Converter interface {
	ConvertJSON(ctx context.Context, path string) ([]byte, error)
}

// CLIConverter runs `genlang convert <file> --json`.
type CLIConverter struct {
	Binary  string
	Timeout time.Duration
	MaxJSON int64
}

func DefaultConverter() *CLIConverter {
	return &CLIConverter{Binary: "genlang", Timeout: defaultTimeout, MaxJSON: maxJSONBytes}
}

func (c *CLIConverter) ConvertJSON(ctx context.Context, path string) ([]byte, error) {
	if err := validateGLPath(path); err != nil {
		return nil, err
	}
	binary := c.Binary
	if binary == "" {
		binary = "genlang"
	}
	resolved, err := exec.LookPath(binary)
	if err != nil {
		return nil, ErrGenLangMissing
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	maxBytes := c.MaxJSON
	if maxBytes <= 0 {
		maxBytes = maxJSONBytes
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, resolved, "convert", path, "--json")
	var stdout bytes.Buffer
	var stderr limitedBuffer
	cmd.Stdout = &limitedWriter{buf: &stdout, remain: maxBytes}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("genlang timed out after %s", timeout)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		detail := strings.TrimSpace(stderr.String())
		if line, _, ok := strings.Cut(detail, "\n"); ok {
			detail = line
		}
		if detail != "" {
			return nil, fmt.Errorf("genlang: %s", detail)
		}
		return nil, fmt.Errorf("genlang convert failed: %w", err)
	}
	if int64(stdout.Len()) >= maxBytes {
		return nil, fmt.Errorf("genlang json exceeded %d bytes", maxBytes)
	}
	return stdout.Bytes(), nil
}

func validateGLPath(path string) error {
	cleaned := filepath.Clean(path)
	if cleaned == "" || strings.Contains(cleaned, "://") {
		return ErrNotGenLang
	}
	if !strings.EqualFold(filepath.Ext(cleaned), ".gl") {
		return ErrNotGenLang
	}
	if !filepath.IsAbs(cleaned) {
		return fmt.Errorf("catalog path must be absolute")
	}
	return nil
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remain := maxErrorBytes - b.Len()
	if remain > 0 {
		if len(p) > remain {
			p = p[:remain]
		}
		_, _ = b.Buffer.Write(p)
	}
	return original, nil
}

type limitedWriter struct {
	buf    *bytes.Buffer
	remain int64
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.remain <= 0 {
		return len(p), nil
	}
	if int64(len(p)) > w.remain {
		_, _ = w.buf.Write(p[:w.remain])
		w.remain = 0
		return len(p), nil
	}
	n, err := w.buf.Write(p)
	w.remain -= int64(n)
	if err != nil {
		return n, err
	}
	return n, nil
}

var _ io.Writer = (*limitedWriter)(nil)
