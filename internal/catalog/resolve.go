package catalog

import (
	"path/filepath"
	"strings"

	pfs "github.com/Padrosum/pmusic/internal/fs"
)

type libraryIndex struct {
	byPath   map[string]pfs.Track
	byBase   map[string][]pfs.Track
	byTitle  map[string][]pfs.Track
	musicDir string
}

func indexFolders(musicDir string, folders []*pfs.Folder) libraryIndex {
	idx := libraryIndex{
		byPath:   map[string]pfs.Track{},
		byBase:   map[string][]pfs.Track{},
		byTitle:  map[string][]pfs.Track{},
		musicDir: musicDir,
	}
	for _, folder := range folders {
		if folder == nil {
			continue
		}
		for _, track := range folder.Tracks {
			idx.add(track)
		}
	}
	return idx
}

func (idx libraryIndex) add(track pfs.Track) {
	idx.byPath[track.Path] = track
	if abs, err := filepath.Abs(track.Path); err == nil {
		idx.byPath[abs] = track
	}
	base := strings.ToLower(filepath.Base(track.Path))
	idx.byBase[base] = append(idx.byBase[base], track)
	idx.byTitle[strings.ToLower(track.Name)] = append(idx.byTitle[strings.ToLower(track.Name)], track)
}

func (idx libraryIndex) match(declaredPath, title string) (pfs.Track, bool) {
	if track, ok := idx.matchPath(declaredPath); ok {
		return track, true
	}
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return pfs.Track{}, false
	}
	matches := idx.byTitle[title]
	if len(matches) == 1 {
		return matches[0], true
	}
	return pfs.Track{}, false
}

func (idx libraryIndex) matchPath(declared string) (pfs.Track, bool) {
	declared = strings.TrimSpace(declared)
	if declared == "" {
		return pfs.Track{}, false
	}
	candidates := []string{declared}
	if !filepath.IsAbs(declared) && idx.musicDir != "" {
		candidates = append(candidates, filepath.Join(idx.musicDir, declared))
	}
	for _, path := range candidates {
		clean := filepath.Clean(path)
		if track, ok := idx.byPath[clean]; ok {
			return track, true
		}
		if abs, err := filepath.Abs(clean); err == nil {
			if track, ok := idx.byPath[abs]; ok {
				return track, true
			}
		}
		base := strings.ToLower(filepath.Base(clean))
		if matches := idx.byBase[base]; len(matches) == 1 {
			return matches[0], true
		}
	}
	return pfs.Track{}, false
}

func Build(path string, doc Document, musicDir string, folders []*pfs.Folder) *Overlay {
	return buildOverlay(path, doc, musicDir, folders)
}

func buildOverlay(path string, doc Document, musicDir string, folders []*pfs.Folder) *Overlay {
	idx := indexFolders(musicDir, folders)
	o := &Overlay{
		Path:     path,
		Doc:      doc,
		Entities: make([]EntityInfo, 0, len(doc.Entities)),
		byPath:   map[string]*EntityInfo{},
		byName:   map[string]*EntityInfo{},
	}
	ents := doc.entityMap()
	for _, ent := range doc.Entities {
		obj := objectValue(ent.Value)
		info := EntityInfo{
			Name:  ent.Name,
			Title: entityTitle(obj, ent.Name),
			Sets:  doc.SetsOf(ent.Name),
		}
		if ent.Type != nil {
			info.Type = *ent.Type
			info.Types = doc.TypesOf(ent.Name)
		}
		info.Artist = entityArtist(obj)
		info.AlbumRef = refName(obj["album"])
		if info.AlbumRef != "" {
			info.Album = info.AlbumRef
			if target, ok := ents[info.AlbumRef]; ok {
				info.Album = entityTitle(objectValue(target.Value), info.AlbumRef)
			}
		} else if name := stringField(obj, "album_adi", "album_name"); name != "" {
			info.Album = name
		}
		declared := entityPath(obj)
		info.Path = declared
		if track, ok := idx.match(declared, info.Title); ok {
			copy := track
			info.Track = &copy
			info.Path = track.Path
		}
		o.Entities = append(o.Entities, info)
	}
	for i := range o.Entities {
		info := &o.Entities[i]
		o.byName[info.Name] = info
		if info.Track != nil {
			o.byPath[info.Track.Path] = info
			if abs, err := filepath.Abs(info.Track.Path); err == nil {
				o.byPath[abs] = info
			}
		}
	}
	for _, set := range doc.Sets {
		pl := Playlist{Name: set.Name}
		seen := map[string]bool{}
		for _, member := range doc.Members(set.Name) {
			tracks := o.tracksForEntity(member)
			if len(tracks) == 0 {
				pl.Missing = append(pl.Missing, member)
				continue
			}
			for _, track := range tracks {
				if seen[track.Path] {
					continue
				}
				seen[track.Path] = true
				pl.Tracks = append(pl.Tracks, track)
			}
		}
		o.Playlists = append(o.Playlists, pl)
	}
	return o
}
