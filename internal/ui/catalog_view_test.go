package ui

import (
	"strings"
	"testing"

	"github.com/Padrosum/pmusic/internal/catalog"
	pfs "github.com/Padrosum/pmusic/internal/fs"
)

func catalogFixture(t *testing.T) *Model {
	t.Helper()
	m := commandTestModel(t)
	m.saveQueueFn = func([]pfs.Track) error { return nil }
	trackA := pfs.Track{Name: "ilk-isik", Path: "/music/Gece/ilk-isik.flac", Ext: ".flac"}
	trackB := pfs.Track{Name: "Yağmur", Path: "/music/Gece/yagmur.flac", Ext: ".flac"}
	m.folders = []*pfs.Folder{{Name: "Gece", Path: "/music/Gece", Tracks: []pfs.Track{trackA, trackB}}}
	m.root = m.folders[0]
	doc, err := catalog.ParseJSON([]byte(`{"types":[{"kind":"tur","name":"Parca","parent":null,"properties":null},{"kind":"tur","name":"Album","parent":null,"properties":null}],"sets":[{"name":"Favoriler","properties":null},{"name":"Gece","properties":null}],"entities":[{"name":"parca_01","type":"Parca","value":{"baslik":"İlk Işık","yol":"/music/Gece/ilk-isik.flac"}},{"name":"parca_02","type":"Parca","value":{"baslik":"Yağmur"}}],"memberships":[{"entity":"parca_01","set":"Favoriler"},{"entity":"parca_01","set":"Gece"},{"entity":"parca_02","set":"Gece"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	m.catalog = catalog.Build("/tmp/library.gl", doc, "/music", m.folders)
	return m
}

func TestLeftPanelIncludesPlaylists(t *testing.T) {
	m := catalogFixture(t)
	if m.leftLen() != 3 {
		t.Fatalf("leftLen = %d", m.leftLen())
	}
	m.focused = panelFolders
	m.moveDown()
	if !m.viewingPlaylist {
		t.Fatal("expected playlist after moving past the folder")
	}
	tracks := m.currentTracks()
	if len(tracks) != 1 || tracks[0].Name != "ilk-isik" {
		t.Fatalf("Favoriler tracks = %#v", tracks)
	}
}

func TestOpenAndQueuePlaylist(t *testing.T) {
	m := catalogFixture(t)
	if err := m.OpenPlaylist("Gece"); err != nil {
		t.Fatal(err)
	}
	if !m.viewingPlaylist || m.catalogTitle != "Gece" {
		t.Fatalf("viewing=%v title=%q", m.viewingPlaylist, m.catalogTitle)
	}
	n, err := m.QueuePlaylist("Favoriler")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(m.queue) != 1 {
		t.Fatalf("queued %d queue=%d", n, len(m.queue))
	}
}

func TestInspectSeparatesTypesAndSets(t *testing.T) {
	m := catalogFixture(t)
	m.focused = panelTracks
	m.trackIdx = 0
	if err := m.InspectSelection(); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(m.commandHelp.lines, "\n")
	if !strings.Contains(joined, "Types") || !strings.Contains(joined, "Sets") || !strings.Contains(joined, "Favoriler") {
		t.Fatalf("inspect overlay:\n%s", joined)
	}
}

func TestTypeBrowse(t *testing.T) {
	m := catalogFixture(t)
	if err := m.OpenType("Parca"); err != nil {
		t.Fatal(err)
	}
	if !m.viewingCatalog || len(m.catalogTracks) != 2 {
		t.Fatalf("type tracks = %#v", m.catalogTracks)
	}
}

func TestFoldersWithoutCatalogUnchanged(t *testing.T) {
	m := commandTestModel(t)
	m.folders = []*pfs.Folder{{Name: "Jazz", Path: "/music/Jazz", Tracks: []pfs.Track{{Name: "So What", Path: "/music/Jazz/so-what.flac", Ext: ".flac"}}}}
	m.focused = panelFolders
	m.moveDown()
	if m.viewingPlaylist || m.folderIdx != 0 {
		t.Fatalf("folder navigation changed without catalogs: playlist=%v idx=%d", m.viewingPlaylist, m.folderIdx)
	}
	if got := m.currentTracks(); len(got) != 1 {
		t.Fatalf("tracks = %#v", got)
	}
}
