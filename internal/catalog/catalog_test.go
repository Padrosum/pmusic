package catalog

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pfs "github.com/Padrosum/pmusic/internal/fs"
)

const musicJSON = `{"types":[{"kind":"cins","name":"Medya","parent":null,"properties":{"baslik":""}},{"kind":"cins","name":"Ses","parent":"Medya","properties":null},{"kind":"tur","name":"Album","parent":"Ses","properties":null},{"kind":"tur","name":"Parca","parent":"Ses","properties":{"sure_sn":0}}],"sets":[{"name":"Favoriler","properties":null},{"name":"Gece","properties":null},{"name":"Enstruman","properties":null}],"entities":[{"name":"gece_yuruyusu","type":"Album","value":{"baslik":"Gece Yürüyüşü","sanatci":"Padros","yil":2026}},{"name":"parca_01","type":"Parca","value":{"baslik":"İlk Işık","sure_sn":214,"album":{"$ref":"gece_yuruyusu"},"yol":"/music/Gece/ilk-isik.flac"}},{"name":"parca_02","type":"Parca","value":{"baslik":"Yağmur","sure_sn":187,"album":{"$ref":"gece_yuruyusu"}}},{"name":"parca_03","type":"Parca","value":{"baslik":"Sessizlik","sure_sn":96,"album":{"$ref":"gece_yuruyusu"},"yol":"Gece/sessizlik.mp3"}}],"memberships":[{"entity":"parca_01","set":"Favoriler"},{"entity":"parca_01","set":"Gece"},{"entity":"parca_02","set":"Gece"},{"entity":"parca_03","set":"Enstruman"},{"entity":"parca_03","set":"Gece"},{"entity":"gece_yuruyusu","set":"Favoriler"}]}`

func testFolders() []*pfs.Folder {
	return []*pfs.Folder{{
		Name: "Gece",
		Path: "/music/Gece",
		Tracks: []pfs.Track{
			{Name: "ilk-isik", Path: "/music/Gece/ilk-isik.flac", Ext: ".flac"},
			{Name: "Yağmur", Path: "/music/Gece/yagmur.flac", Ext: ".flac"},
			{Name: "sessizlik", Path: "/music/Gece/sessizlik.mp3", Ext: ".mp3"},
		},
	}}
}

func TestParseAndQueries(t *testing.T) {
	doc, err := ParseJSON([]byte(musicJSON))
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Members("Gece"); len(got) != 3 {
		t.Fatalf("Gece members = %v", got)
	}
	types := doc.TypesOf("parca_01")
	if len(types) < 3 || types[0] != "Parca" {
		t.Fatalf("types of parca_01 = %v", types)
	}
	foundSes := false
	for _, name := range types {
		if name == "Ses" {
			foundSes = true
		}
	}
	if !foundSes {
		t.Fatalf("expected Ses ancestor, got %v", types)
	}
	desc := doc.Descendants("Ses")
	if len(desc) != 2 {
		t.Fatalf("descendants of Ses = %v", desc)
	}
}

func TestResolvePathTitleAndAlbumExpansion(t *testing.T) {
	doc, err := ParseJSON([]byte(musicJSON))
	if err != nil {
		t.Fatal(err)
	}
	o := buildOverlay("/tmp/library.gl", doc, "/music", testFolders())
	fav, ok := o.Playlist("Favoriler")
	if !ok {
		t.Fatal("Favoriler missing")
	}
	if len(fav.Tracks) != 3 {
		t.Fatalf("Favoriler tracks = %#v (album membership should expand)", fav.Tracks)
	}
	gece, _ := o.Playlist("Gece")
	if len(gece.Tracks) != 3 {
		t.Fatalf("Gece tracks = %#v missing=%v", gece.Tracks, gece.Missing)
	}
	info, ok := o.EntityByPath("/music/Gece/ilk-isik.flac")
	if !ok || info.Title != "İlk Işık" {
		t.Fatalf("entity by path = %#v ok=%v", info, ok)
	}
	name, tracks, missing, ok := o.TracksOfType("Parca")
	if !ok || name != "Parca" || len(tracks) != 3 || len(missing) != 0 {
		t.Fatalf("Parca tracks=%d missing=%v", len(tracks), missing)
	}
	if blob := o.SearchText("/music/Gece/ilk-isik.flac"); blob == "" || !strings.Contains(blob, "favoriler") || !strings.Contains(blob, "parca") {
		t.Fatalf("search text = %q", blob)
	}
}

func TestFindPathPrefersConfig(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	musicDir := filepath.Join(root, "music")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(musicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	musicFile := filepath.Join(musicDir, FileName)
	if err := os.WriteFile(musicFile, []byte("cins T\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := FindPath(Options{MusicDir: musicDir, ConfigDir: configDir})
	if err != nil {
		t.Fatal(err)
	}
	wantMusic, _ := filepath.Abs(musicFile)
	if path != wantMusic {
		t.Fatalf("path = %s want music overlay %s", path, wantMusic)
	}
	want := filepath.Join(configDir, FileName)
	if err := os.WriteFile(want, []byte("cins T\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err = FindPath(Options{MusicDir: musicDir, ConfigDir: configDir})
	if err != nil {
		t.Fatal(err)
	}
	absWant, _ := filepath.Abs(want)
	if path != absWant {
		t.Fatalf("path = %s want %s", path, absWant)
	}
}

func TestLoadMissingFile(t *testing.T) {
	o, err := Load(context.Background(), nil, Options{MusicDir: t.TempDir(), ConfigDir: t.TempDir()})
	if err != nil || o != nil {
		t.Fatalf("overlay=%v err=%v", o, err)
	}
}

func TestLoadUsesConverter(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, FileName), []byte("cins T\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	o, err := Load(context.Background(), testFolders(), Options{
		MusicDir:  "/music",
		ConfigDir: configDir,
		Converter: converterFunc(func(context.Context, string) ([]byte, error) { return []byte(musicJSON), nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if o == nil || len(o.Playlists) != 3 {
		t.Fatalf("overlay = %#v", o)
	}
}

func TestLoadMissingGenLang(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("cins T\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(context.Background(), nil, Options{
		ConfigDir: root,
		Converter: converterFunc(func(context.Context, string) ([]byte, error) { return nil, ErrGenLangMissing }),
	})
	if !errors.Is(err, ErrGenLangMissing) {
		t.Fatalf("err = %v", err)
	}
}

func TestConvertJSONWithGenLang(t *testing.T) {
	binary, err := exec.LookPath("genlang")
	if err != nil {
		t.Skip("genlang not on PATH")
	}
	root := t.TempDir()
	src := filepath.Join(root, FileName)
	if err := os.WriteFile(src, []byte("cins T\nveri n : T { x = 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &CLIConverter{Binary: binary, Timeout: 5 * time.Second}
	data, err := c.ConvertJSON(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ParseJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Types) != 1 || doc.Types[0].Name != "T" {
		t.Fatalf("doc = %#v", doc)
	}
}

type converterFunc func(context.Context, string) ([]byte, error)

func (f converterFunc) ConvertJSON(ctx context.Context, path string) ([]byte, error) {
	return f(ctx, path)
}
