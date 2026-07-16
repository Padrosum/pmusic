package command

import (
	"fmt"
	"strings"
	"unicode"
)

type token struct {
	value string
}

func Parse(input string) (ParsedCommand, error) {
	raw := input
	s := strings.TrimSpace(input)
	if strings.HasPrefix(s, ":") {
		s = strings.TrimSpace(strings.TrimPrefix(s, ":"))
	}
	if s == "" {
		return ParsedCommand{Raw: raw, Flags: map[string]string{}, BoolFlags: map[string]bool{}}, nil
	}
	tokens, err := lex(s)
	if err != nil {
		return ParsedCommand{}, err
	}
	p := ParsedCommand{Raw: raw, Flags: map[string]string{}, BoolFlags: map[string]bool{}}
	p.Name = tokens[0].value
	if strings.HasSuffix(p.Name, "!") {
		p.Name = strings.TrimSuffix(p.Name, "!")
		p.Bang = true
	}
	positional := false
	for _, tok := range tokens[1:] {
		v := tok.value
		if !positional && v == "--" {
			positional = true
			continue
		}
		if !positional && strings.HasPrefix(v, "--") && len(v) > 2 {
			flag := strings.TrimPrefix(v, "--")
			if name, value, ok := strings.Cut(flag, "="); ok {
				if name == "" {
					return ParsedCommand{}, &ParseError{Message: "flag name cannot be empty"}
				}
				p.Flags[name] = value
			} else {
				p.BoolFlags[flag] = true
			}
			continue
		}
		p.Args = append(p.Args, v)
	}
	return p, nil
}

func lex(s string) ([]token, error) {
	var out []token
	var b strings.Builder
	var quote rune
	escaped := false
	inToken := false
	flush := func() {
		if inToken {
			out = append(out, token{value: b.String()})
			b.Reset()
			inToken = false
		}
	}
	for _, r := range s {
		if escaped {
			// Backslash quotes/backslashes everywhere and otherwise preserves the
			// escaped rune. This makes paths and Unicode input predictable.
			b.WriteRune(r)
			escaped = false
			inToken = true
			continue
		}
		if r == '\\' {
			escaped = true
			inToken = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			inToken = true
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			inToken = true
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		b.WriteRune(r)
		inToken = true
	}
	if escaped {
		b.WriteRune('\\')
	}
	if quote != 0 {
		return nil, &ParseError{Message: fmt.Sprintf("unterminated %c quote", quote)}
	}
	flush()
	return out, nil
}
