package command

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func validCommand(name string, aliases ...string) Command {
	return Command{Name: name, Aliases: aliases, Category: "Test", Summary: "summary", Description: "description", Usage: ":" + name, Examples: []string{":" + name}, Execute: func(Runtime, ParsedCommand) (tea.Cmd, error) { return nil, nil }}
}

func TestRegistryResolveAndCase(t *testing.T) {
	r, err := NewRegistry(validCommand("volume", "vol", "v"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"volume", "VOLUME", "vol", "V"} {
		c, ok := r.Resolve(name)
		if !ok || c.Name != "volume" {
			t.Fatalf("Resolve(%q)=%#v,%v", name, c, ok)
		}
	}
}
func TestRegistryValidation(t *testing.T) {
	tests := []struct {
		name     string
		commands []Command
	}{{"empty", []Command{validCommand("")}}, {"duplicate", []Command{validCommand("one"), validCommand("ONE")}}, {"duplicate alias", []Command{validCommand("one", "x"), validCommand("two", "X")}}, {"alias command collision", []Command{validCommand("one", "two"), validCommand("two")}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewRegistry(tt.commands...); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
func TestRegistryMetadataValidation(t *testing.T) {
	c := validCommand("bad")
	c.Summary = ""
	if _, err := NewRegistry(c); err == nil {
		t.Fatal("missing metadata accepted")
	}
}
func TestSuggestTypo(t *testing.T) {
	r, _ := NewRegistry(validCommand("volume"), validCommand("seek"), validCommand("search"))
	got := r.Suggest("voleume", 3)
	if len(got) == 0 || got[0] != "volume" {
		t.Fatalf("suggestions=%v", got)
	}
}
