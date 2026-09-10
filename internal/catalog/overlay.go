package catalog

import (
	"path/filepath"
	"strings"
	"time"

	pfs "github.com/Padrosum/pmusic/internal/fs"
)

// Playlist is a GenLang set resolved against the local library.
type Playlist struct {
	Name    string
	Tracks  []pfs.Track
	Missing []string
}

// EntityInfo is one veri row after matching it to files and walking types/sets.
type EntityInfo struct {
	Name     string
	Title    string
	Artist   string
	Album    string
	AlbumRef string
	Type     string
	Types    []string
	Sets     []string
	Path     string
	Track    *pfs.Track
}

// Overlay is a loaded library.gl after genlang convert and path matching.
type Overlay struct {
	Path      string
	Doc       Document
	Playlists []Playlist
	Entities  []EntityInfo
	mtime     time.Time
	byPath    map[string]*EntityInfo
	byName    map[string]*EntityInfo
}

func (o *Overlay) TypeNames() []string {
	if o == nil {
		return nil
	}
	out := make([]string, 0, len(o.Doc.Types))
	for _, t := range o.Doc.Types {
		out = append(out, t.Name)
	}
	return out
}

func (o *Overlay) SetNames() []string {
	if o == nil {
		return nil
	}
	out := make([]string, 0, len(o.Doc.Sets))
	for _, s := range o.Doc.Sets {
		out = append(out, s.Name)
	}
	return out
}

func (o *Overlay) Playlist(name string) (Playlist, bool) {
	if o == nil {
		return Playlist{}, false
	}
	for _, p := range o.Playlists {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Playlist{}, false
}

func (o *Overlay) EntityByPath(path string) (*EntityInfo, bool) {
	if o == nil || path == "" {
		return nil, false
	}
	if info, ok := o.byPath[path]; ok {
		return info, true
	}
	clean, err := filepath.Abs(path)
	if err == nil {
		if info, ok := o.byPath[clean]; ok {
			return info, true
		}
	}
	return nil, false
}

func (o *Overlay) SearchText(path string) string {
	info, ok := o.EntityByPath(path)
	if !ok {
		return ""
	}
	parts := []string{info.Name, info.Title, info.Artist, info.Album, info.Type}
	parts = append(parts, info.Types...)
	parts = append(parts, info.Sets...)
	return strings.ToLower(strings.Join(parts, " "))
}

func (o *Overlay) TracksOfType(name string) (typeName string, tracks []pfs.Track, missing []string, ok bool) {
	if o == nil {
		return "", nil, nil, false
	}
	t, found := o.Doc.findType(name)
	if !found {
		return "", nil, nil, false
	}
	wanted := map[string]bool{t.Name: true}
	for _, d := range o.Doc.Descendants(t.Name) {
		wanted[d] = true
	}
	var out []pfs.Track
	var miss []string
	seen := map[string]bool{}
	for _, info := range o.Entities {
		if info.Type == "" || !wanted[info.Type] {
			continue
		}
		resolved := o.tracksForEntity(info.Name)
		if len(resolved) == 0 {
			miss = append(miss, info.Name)
			continue
		}
		for _, track := range resolved {
			if seen[track.Path] {
				continue
			}
			seen[track.Path] = true
			out = append(out, track)
		}
	}
	return t.Name, out, miss, true
}

func (o *Overlay) tracksForEntity(name string) []pfs.Track {
	info, ok := o.byName[name]
	if !ok {
		return nil
	}
	if info.Track != nil {
		return []pfs.Track{*info.Track}
	}
	var out []pfs.Track
	seen := map[string]bool{}
	for _, other := range o.Entities {
		if other.Track == nil || other.AlbumRef != name {
			continue
		}
		if seen[other.Track.Path] {
			continue
		}
		seen[other.Track.Path] = true
		out = append(out, *other.Track)
	}
	return out
}
