package fs

import (
	"os"
	"path/filepath"
	"strings"
)

var supportedExts = map[string]bool{
	".mp3":  true,
	".flac": true,
	".wav":  true,
}

type Track struct {
	Name string
	Path string
	Ext  string
}

type Folder struct {
	Name     string
	Path     string
	Tracks   []Track
	Children []*Folder
}

func Scan(root string) (*Folder, error) {
	// Resolve symlinks on the root so the visited set works correctly.
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, err
	}
	folder := &Folder{
		Name: info.Name(),
		Path: real,
	}
	scanDir(folder, map[string]bool{real: true})
	return folder, nil
}

func scanDir(f *Folder, visited map[string]bool) {
	entries, err := os.ReadDir(f.Path)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(f.Path, e.Name())
		if e.IsDir() {
			// Resolve symlinks to detect cycles before recursing.
			real, err := filepath.EvalSymlinks(full)
			if err != nil || visited[real] {
				continue
			}
			visited[real] = true
			child := &Folder{Name: e.Name(), Path: full}
			scanDir(child, visited)
			if child.hasContent() {
				f.Children = append(f.Children, child)
			}
		} else {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if supportedExts[ext] {
				f.Tracks = append(f.Tracks, Track{
					Name: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())),
					Path: full,
					Ext:  ext,
				})
			}
		}
	}
}

func (f *Folder) hasContent() bool {
	if len(f.Tracks) > 0 {
		return true
	}
	for _, c := range f.Children {
		if c.hasContent() {
			return true
		}
	}
	return false
}

// FlatFolders returns all folders that contain tracks, depth-first.
func FlatFolders(root *Folder) []*Folder {
	var result []*Folder
	walk(root, &result)
	return result
}

func walk(f *Folder, out *[]*Folder) {
	if len(f.Tracks) > 0 {
		*out = append(*out, f)
	}
	for _, c := range f.Children {
		walk(c, out)
	}
}
