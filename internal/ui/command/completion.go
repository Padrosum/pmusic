package command

import (
	"sort"
	"strings"
	"unicode"
)

type TokenRange struct {
	Start, End int
	Value      string
	Quoted     bool
}

// Complete returns context-sensitive items for the token at cursor. Cursor is
// a rune offset, matching bubbles/textinput's cursor representation.
func (r *Registry) Complete(rt Runtime, input string, cursor, limit int) ([]CompletionItem, TokenRange) {
	runes := []rune(input)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	rng, index := ActiveToken(input, cursor)
	if index == 0 {
		q := strings.ToLower(rng.Value)
		type scored struct {
			item  CompletionItem
			score int
		}
		seen := map[string]bool{}
		var values []scored
		for _, c := range r.Commands() {
			best, kind, display := matchScore(q, c.Name), CompletionCommand, c.Name
			for _, a := range c.Aliases {
				if s := matchScore(q, a); s < best {
					best, kind, display = s, CompletionAlias, a
				}
			}
			if best < 1000 && !seen[c.Name] {
				seen[c.Name] = true
				description := c.Summary
				if kind == CompletionAlias {
					description = "Alias for :" + c.Name + " — " + c.Summary
				}
				values = append(values, scored{CompletionItem{Value: c.Name, Display: display, Description: description, Kind: kind}, best})
			}
		}
		sort.SliceStable(values, func(i, j int) bool { return values[i].score < values[j].score })
		if limit > len(values) {
			limit = len(values)
		}
		items := make([]CompletionItem, limit)
		for i := range items {
			items[i] = values[i].item
		}
		return items, rng
	}
	first, _ := firstToken(input)
	c, ok := r.Resolve(first)
	if !ok {
		return nil, rng
	}
	q := strings.ToLower(rng.Value)
	var items []CompletionItem
	if index == 1 {
		for _, sub := range c.Subcommands {
			if strings.HasPrefix(strings.ToLower(sub.Name), q) {
				items = append(items, CompletionItem{Value: sub.Name, Display: sub.Name, Description: sub.Description, Kind: CompletionSubcommand})
			}
		}
	}
	if c.Complete != nil {
		for _, item := range c.Complete(rt, rng.Value) {
			if q == "" || strings.Contains(strings.ToLower(item.Value), q) {
				items = appendUnique(items, item)
			}
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, rng
}

func ActiveToken(input string, cursor int) (TokenRange, int) {
	r := []rune(input)
	if cursor > len(r) {
		cursor = len(r)
	}
	start := 0
	quote := rune(0)
	escaped := false
	// Identify token start while respecting quoted spaces.
	for i, ch := range r[:cursor] {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if unicode.IsSpace(ch) {
			start = i + 1
		}
	}
	if start == 0 && len(r) > 0 && r[0] == ':' {
		start = 1
	}
	end := cursor
	escaped = false
	for end < len(r) {
		ch := r[end]
		if escaped {
			escaped = false
			end++
			continue
		}
		if ch == '\\' {
			escaped = true
			end++
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			end++
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			end++
			continue
		}
		if unicode.IsSpace(ch) {
			break
		}
		end++
	}
	value := string(r[start:end])
	quoted := false
	if len(value) >= 1 && (value[0] == '\'' || value[0] == '"') {
		quoted = true
		value = strings.Trim(value, "\"'")
	}
	prefix := strings.TrimSpace(strings.TrimPrefix(string(r[:start]), ":"))
	index := 0
	if prefix != "" {
		fields, _ := lex(prefix)
		index = len(fields)
	}
	return TokenRange{Start: start, End: end, Value: value, Quoted: quoted}, index
}

func ReplaceToken(input string, rng TokenRange, value string) (string, int) {
	if strings.ContainsFunc(value, unicode.IsSpace) {
		value = "\"" + strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "\"", "\\\"") + "\""
	}
	r := []rune(input)
	replacement := []rune(value)
	out := string(r[:rng.Start]) + value + string(r[rng.End:])
	return out, rng.Start + len(replacement)
}

func firstToken(input string) (string, bool) {
	ts, err := lex(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), ":")))
	if err != nil || len(ts) == 0 {
		return "", false
	}
	return ts[0].value, true
}
func appendUnique(items []CompletionItem, v CompletionItem) []CompletionItem {
	for _, i := range items {
		if i.Value == v.Value {
			return items
		}
	}
	return append(items, v)
}
func matchScore(q, value string) int {
	value = strings.ToLower(value)
	if q == "" {
		return 10
	}
	if strings.HasPrefix(value, q) {
		return len(value) - len(q)
	}
	if strings.Contains(value, q) {
		return 100 + len(value) - len(q)
	}
	d := levenshtein(q, value)
	if d <= 2 {
		return 200 + d
	}
	return 1000
}
