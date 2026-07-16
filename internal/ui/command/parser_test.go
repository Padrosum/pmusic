package command

import (
	"reflect"
	"testing"
)

func TestParseCommands(t *testing.T) {
	tests := []struct {
		input, name string
		args        []string
		bang        bool
	}{
		{":play", "play", nil, false}, {"play", "play", nil, false}, {":volume 60", "volume", []string{"60"}, false}, {":volume +10", "volume", []string{"+10"}, false}, {":seek -30", "seek", []string{"-30"}, false},
		{`:play "Fade to Black"`, "play", []string{"Fade to Black"}, false}, {`:play 'Fade to Black'`, "play", []string{"Fade to Black"}, false}, {`:download --folder "Classic Rock" "Kansas Carry On"`, "download", []string{"Classic Rock", "Kansas Carry On"}, false},
		{":q!", "q", nil, true}, {":quit!", "quit", nil, true}, {"  :play   Duman   Seni Kendime Sakladım  ", "play", []string{"Duman", "Seni", "Kendime", "Sakladım"}, false}, {`:play "İstanbul 東京"`, "play", []string{"İstanbul 東京"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.name || got.Bang != tt.bang || !reflect.DeepEqual(got.Args, tt.args) {
				t.Fatalf("Parse=%#v want name=%q args=%#v bang=%v", got, tt.name, tt.args, tt.bang)
			}
		})
	}
}

func TestParseFlagsEscapesAndSeparator(t *testing.T) {
	p, err := Parse(`download --folder="Classic Rock" --dry-run -- "A \\"quoted\\" song" C:\\Music`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags["folder"] != "Classic Rock" || !p.BoolFlags["dry-run"] {
		t.Fatalf("flags=%v bool=%v", p.Flags, p.BoolFlags)
	}
	if !reflect.DeepEqual(p.Args, []string{`A \quoted\ song`, `C:\Music`}) {
		t.Fatalf("args=%q", p.Args)
	}
}

func TestParseEmpty(t *testing.T) {
	for _, s := range []string{"", ":", "  :  "} {
		p, err := Parse(s)
		if err != nil || p.Name != "" {
			t.Fatalf("Parse(%q)=%#v,%v", s, p, err)
		}
	}
}
func TestParseUnterminatedQuotes(t *testing.T) {
	for _, s := range []string{`:play "oops`, `:play 'oops`} {
		if _, err := Parse(s); err == nil {
			t.Fatalf("Parse(%q) succeeded", s)
		}
	}
}
func TestParseEscapedQuoteAndBackslash(t *testing.T) {
	p, err := Parse(`play "Fade \\"Black\\"" path\\name`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Args, []string{`Fade \Black\`, `path\name`}) {
		t.Fatalf("args=%q", p.Args)
	}
}
