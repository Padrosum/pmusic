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
	return ParseCommand(input, nil)
}

// ParseCommand parses input like Parse but additionally lets value-taking flags
// consume a separate following token, so both `-f "value"` and `--folder
// "value"` are supported in addition to `--flag=value`. valueFlags maps flag
// names (without dashes) to whether the flag expects a value.
func ParseCommand(input string, valueFlags map[string]bool) (ParsedCommand, error) {
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
	for i := 1; i < len(tokens); i++ {
		v := tokens[i].value
		if !positional && v == "--" {
			positional = true
			continue
		}
		if !positional {
			if name, isFlag, hasEq, eqValue := splitFlag(v); isFlag {
				if hasEq {
					if name == "" {
						return ParsedCommand{}, &ParseError{Message: "flag name cannot be empty"}
					}
					p.Flags[name] = eqValue
					continue
				}
				if valueFlags != nil && valueFlags[name] {
					if i+1 >= len(tokens) {
						return ParsedCommand{}, &ParseError{Message: "flag -" + name + " requires a value"}
					}
					i++
					p.Flags[name] = tokens[i].value
					continue
				}
				p.BoolFlags[name] = true
				continue
			}
		}
		p.Args = append(p.Args, v)
	}
	return p, nil
}

// splitFlag classifies a token as a long (--) or short (-x) flag and extracts
// its name and optional `name=value` payload. Short flags are only recognized
// for a single letter so negative numbers like -30 remain positional args.
func splitFlag(v string) (name string, isFlag, hasEq bool, eqValue string) {
	if strings.HasPrefix(v, "--") && len(v) > 2 {
		body := strings.TrimPrefix(v, "--")
		if n, val, ok := strings.Cut(body, "="); ok {
			return n, true, true, val
		}
		return body, true, false, ""
	}
	if len(v) >= 2 && v[0] == '-' && v[1] != '-' {
		body := strings.TrimPrefix(v, "-")
		if body == "" {
			return "", false, false, ""
		}
		if len(body) == 1 && isAlpha(body[0]) {
			return body, true, false, ""
		}
		if n, val, ok := strings.Cut(body, "="); ok && len(n) == 1 && isAlpha(n[0]) {
			return n, true, true, val
		}
	}
	return "", false, false, ""
}

func isAlpha(r byte) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
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
