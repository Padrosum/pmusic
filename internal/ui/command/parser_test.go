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

func TestParseCommandValueFlags(t *testing.T) {
	valueFlags := map[string]bool{"f": true, "folder": true}
	tests := []struct {
		input  string
		args   []string
		flags  map[string]string
		bools  map[string]bool
		hasErr bool
	}{
		{`:download "Fade to Black" -f "Metallica"`, []string{"Fade to Black"}, map[string]string{"f": "Metallica"}, nil, false},
		{`:download "Fade to Black" --folder "Classic Rock"`, []string{"Fade to Black"}, map[string]string{"folder": "Classic Rock"}, nil, false},
		{`:download "Fade to Black" -f=Metallica`, []string{"Fade to Black"}, map[string]string{"f": "Metallica"}, nil, false},
		{`:download "Fade to Black" --folder=Rock`, []string{"Fade to Black"}, map[string]string{"folder": "Rock"}, nil, false},
		{`:download "Fade to Black" --folder "A B" --verbose`, []string{"Fade to Black"}, map[string]string{"folder": "A B"}, map[string]bool{"verbose": true}, false},
		{`:download "Fade to Black" -f`, nil, nil, nil, true},
		{`:download "Fade to Black" --folder "A B" -- "literal -f arg"`, []string{"Fade to Black", "literal -f arg"}, map[string]string{"folder": "A B"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := ParseCommand(tt.input, valueFlags)
			if (err != nil) != tt.hasErr {
				t.Fatalf("error = %v, hasErr = %v", err, tt.hasErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(p.Args, tt.args) {
				t.Fatalf("args = %q, want %q", p.Args, tt.args)
			}
			for k, v := range tt.flags {
				if p.Flags[k] != v {
					t.Fatalf("Flags[%q] = %q, want %q (all flags %v)", k, p.Flags[k], v, p.Flags)
				}
			}
			for k, v := range tt.bools {
				if p.BoolFlags[k] != v {
					t.Fatalf("BoolFlags[%q] = %v, want %v", k, p.BoolFlags[k], v)
				}
			}
		})
	}
}

func TestParseNegativeNumbersRemainArgs(t *testing.T) {
	for in, want := range map[string]string{":seek -30": "-30", ":volume -10": "-10"} {
		p, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if len(p.Args) != 1 || p.Args[0] != want {
			t.Fatalf("Parse(%q) args = %v, want [%q]", in, p.Args, want)
		}
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
