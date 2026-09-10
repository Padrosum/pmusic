package catalog

import (
	"context"
	"fmt"
	"os"

	pfs "github.com/Padrosum/pmusic/internal/fs"
)

// Load converts library.gl with genlang (when present) and matches entities
// to scanned tracks. A missing overlay file is not an error.
func Load(ctx context.Context, folders []*pfs.Folder, opt Options) (*Overlay, error) {
	path, err := FindPath(opt)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	conv := opt.Converter
	if conv == nil {
		conv = DefaultConverter()
	}
	data, err := conv.ConvertJSON(ctx, path)
	if err != nil {
		return nil, err
	}
	doc, err := ParseJSON(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	o := buildOverlay(path, doc, opt.MusicDir, folders)
	if info, err := os.Stat(path); err == nil {
		o.mtime = info.ModTime()
	}
	return o, nil
}

// Resolve rematches an already converted document against a new library scan
// without calling genlang again.
func (o *Overlay) Resolve(musicDir string, folders []*pfs.Folder) *Overlay {
	if o == nil {
		return nil
	}
	next := buildOverlay(o.Path, o.Doc, musicDir, folders)
	next.mtime = o.mtime
	return next
}

func (o *Overlay) FileChanged() bool {
	if o == nil || o.Path == "" {
		return false
	}
	info, err := os.Stat(o.Path)
	if err != nil {
		return true
	}
	return !info.ModTime().Equal(o.mtime)
}

func (o *Overlay) Missing() bool {
	if o == nil || o.Path == "" {
		return true
	}
	_, err := os.Stat(o.Path)
	return err != nil
}
