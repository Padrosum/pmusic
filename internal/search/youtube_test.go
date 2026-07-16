package search

import (
	"slices"
	"strings"
	"testing"
)

func TestSearchArgumentsDoNotUseShell(t *testing.T) {
	query := `metallica; $(touch should-not-run)`
	args := buildSearchArgs(query, 10)
	if got := args[len(args)-1]; got != "ytsearch10:"+query {
		t.Fatalf("search target = %q", got)
	}
	for _, forbidden := range []string{"sh", "bash", "-c"} {
		if slices.Contains(args, forbidden) {
			t.Fatalf("shell argument %q found in %#v", forbidden, args)
		}
	}
}

func TestParseJSONLines(t *testing.T) {
	input := `{"id":"abc123","title":"Fade to Black","uploader":"Metallica","duration":417}` + "\n"
	results, malformed, err := parseJSONLines(strings.NewReader(input), "youtube", 10)
	if err != nil {
		t.Fatal(err)
	}
	if malformed != 0 || len(results) != 1 {
		t.Fatalf("got %d results, %d malformed", len(results), malformed)
	}
	got := results[0]
	if got.Title != "Fade to Black" || got.Uploader != "Metallica" || got.Duration != 417 || !got.DurationKnown {
		t.Fatalf("unexpected result: %#v", got)
	}
	if got.URL != "https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("URL = %q", got.URL)
	}
}

func TestParseJSONLinesMissingOptionalFields(t *testing.T) {
	results, _, err := parseJSONLines(strings.NewReader(`{"id":"abc","title":"Unknown song"}`+"\n"), "youtube", 10)
	if err != nil {
		t.Fatal(err)
	}
	got := results[0]
	if got.DurationKnown || got.Duration != 0 {
		t.Fatalf("missing duration was treated as known: %#v", got)
	}
	if got.Uploader != "Unknown uploader" {
		t.Fatalf("uploader = %q", got.Uploader)
	}
}

func TestWebpageURLPreferred(t *testing.T) {
	results, _, err := parseJSONLines(strings.NewReader(`{"id":"abc","title":"Song","webpage_url":"https://example.com/preferred","url":"https://example.com/fallback"}`+"\n"), "youtube", 10)
	if err != nil {
		t.Fatal(err)
	}
	if got := results[0].URL; got != "https://example.com/preferred" {
		t.Fatalf("URL = %q", got)
	}
}

func TestMalformedLineDoesNotDiscardValidResults(t *testing.T) {
	input := "not-json\n" + `{"id":"ok","title":"Still usable","channel":"Topic"}` + "\n"
	results, malformed, err := parseJSONLines(strings.NewReader(input), "youtube", 10)
	if err != nil {
		t.Fatal(err)
	}
	if malformed != 1 || len(results) != 1 || results[0].Title != "Still usable" {
		t.Fatalf("results=%#v malformed=%d", results, malformed)
	}
}

func TestParseJSONLinesNoResults(t *testing.T) {
	results, malformed, err := parseJSONLines(strings.NewReader("\n"), "youtube", 10)
	if err != nil || malformed != 0 || len(results) != 0 {
		t.Fatalf("results=%#v malformed=%d err=%v", results, malformed, err)
	}
}
