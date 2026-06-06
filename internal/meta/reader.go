package meta

import (
	"os"

	"github.com/dhowden/tag"
)

// Meta holds the audio file's embedded tag metadata.
type Meta struct {
	Title  string
	Artist string
	Album  string
}

// Read opens path and parses its embedded tags. Returns zero-value Meta on any error.
func Read(path string) Meta {
	f, err := os.Open(path)
	if err != nil {
		return Meta{}
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return Meta{}
	}
	return Meta{
		Title:  m.Title(),
		Artist: m.Artist(),
		Album:  m.Album(),
	}
}
