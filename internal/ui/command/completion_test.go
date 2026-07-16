package command

import "testing"

func TestCommandAndSubcommandCompletion(t *testing.T) {
	r, _ := NewRegistry(validCommand("volume", "vol"), Command{Name: "queue", Category: "Queue", Summary: "queue", Description: "queue", Usage: ":queue", Examples: []string{":queue"}, Subcommands: []Subcommand{{Name: "show"}, {Name: "clear"}}}, Command{Name: "reload", Category: "Application", Summary: "reload", Description: "reload", Usage: ":reload", Examples: []string{":reload"}, Subcommands: []Subcommand{{Name: "lua"}, {Name: "library"}}})
	for _, tt := range []struct{ in, want string }{{"vo", "volume"}, {":vo", "volume"}, {"que", "queue"}, {"queue c", "clear"}} {
		items, _ := r.Complete(nil, tt.in, len([]rune(tt.in)), 10)
		if !containsCompletion(items, tt.want) {
			t.Fatalf("Complete(%q)=%v", tt.in, items)
		}
	}
	items, _ := r.Complete(nil, "reload l", 8, 10)
	if !containsCompletion(items, "lua") || !containsCompletion(items, "library") {
		t.Fatalf("reload=%v", items)
	}
}
func TestReplaceActiveAndQuotedToken(t *testing.T) {
	rng, _ := ActiveToken("play old value", 8)
	got, pos := ReplaceToken("play old value", rng, "Fade to Black")
	if got != `play "Fade to Black" value` || pos != 20 {
		t.Fatalf("got=%q pos=%d", got, pos)
	}
}
func TestActiveTokenInsideQuotedArgument(t *testing.T) {
	input := `play "Fade to Black" next`
	rng, index := ActiveToken(input, 10)
	if index != 1 || rng.Value != "Fade to Black" || !rng.Quoted {
		t.Fatalf("range=%#v index=%d", rng, index)
	}
	got, _ := ReplaceToken(input, rng, "Metallica One")
	if got != `play "Metallica One" next` {
		t.Fatalf("replacement=%q", got)
	}
}
func TestCompletionCycle(t *testing.T) {
	s := CompletionState{Items: []CompletionItem{{Value: "a"}, {Value: "b"}}}
	s.Next()
	if s.Selected != 1 {
		t.Fatal(s.Selected)
	}
	s.Next()
	if s.Selected != 0 {
		t.Fatal(s.Selected)
	}
	s.Previous()
	if s.Selected != 1 {
		t.Fatal(s.Selected)
	}
}
func containsCompletion(items []CompletionItem, want string) bool {
	for _, i := range items {
		if i.Value == want {
			return true
		}
	}
	return false
}
