package command

import (
	"strings"
	"testing"
)

func TestHelpListDetailFilterAndUnknown(t *testing.T) {
	r, err := NewRegistry(Command{Name: "seek", Aliases: []string{"sk"}, Category: "Playback", Summary: "Move within track", Description: "Detailed seek description", Usage: ":seek <position>", Examples: []string{":seek +30"}}, Command{Name: "volume", Aliases: []string{"vol"}, Category: "Audio", Summary: "Change volume", Description: "Detailed volume description", Usage: ":volume [value]", Examples: []string{":volume 50"}})
	if err != nil {
		t.Fatal(err)
	}
	lines, err := r.Help("commands", "")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Playback", ":seek <position>", "Audio", ":volume [value]"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("help missing %q:\n%s", want, joined)
		}
	}
	detail, _ := r.Help("sk", "")
	d := strings.Join(detail, "\n")
	if !strings.Contains(d, "Detailed seek") || !strings.Contains(d, ":sk") {
		t.Fatalf("detail=%s", d)
	}
	filtered, _ := r.Help("commands", "volume")
	f := strings.Join(filtered, "\n")
	if strings.Contains(f, ":seek") || !strings.Contains(f, ":volume") {
		t.Fatalf("filter=%s", f)
	}
	if _, err = r.Help("missing", ""); err == nil {
		t.Fatal("unknown topic accepted")
	}
}
