package command

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Registry struct {
	commands map[string]Command
	aliases  map[string]string
	order    []string
}

func NewRegistry(commands ...Command) (*Registry, error) {
	r := &Registry{commands: map[string]Command{}, aliases: map[string]string{}}
	for _, c := range commands {
		if err := r.Register(c); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Register(c Command) error {
	name := normalize(c.Name)
	if name == "" {
		return fmt.Errorf("command name cannot be empty")
	}
	if c.Summary == "" || c.Usage == "" || c.Category == "" {
		return fmt.Errorf("command %q has incomplete metadata", c.Name)
	}
	if len(c.Examples) == 0 {
		return fmt.Errorf("command %q must have an example", c.Name)
	}
	if _, ok := r.commands[name]; ok {
		return fmt.Errorf("duplicate command %q", c.Name)
	}
	if owner, ok := r.aliases[name]; ok {
		return fmt.Errorf("command %q collides with alias for %q", c.Name, owner)
	}
	seen := map[string]bool{}
	for _, a := range c.Aliases {
		alias := normalize(a)
		if alias == "" {
			return fmt.Errorf("command %q has empty alias", c.Name)
		}
		if seen[alias] {
			return fmt.Errorf("duplicate alias %q", a)
		}
		seen[alias] = true
		if _, ok := r.commands[alias]; ok {
			return fmt.Errorf("alias %q collides with command", a)
		}
		if owner, ok := r.aliases[alias]; ok {
			return fmt.Errorf("alias %q already belongs to %q", a, owner)
		}
	}
	c.Name = name
	r.commands[name] = c
	r.order = append(r.order, name)
	for _, a := range c.Aliases {
		r.aliases[normalize(a)] = name
	}
	return nil
}

func (r *Registry) Resolve(name string) (Command, bool) {
	n := normalize(name)
	if canonical, ok := r.aliases[n]; ok {
		n = canonical
	}
	c, ok := r.commands[n]
	return c, ok
}

func (r *Registry) Execute(rt Runtime, parsed ParsedCommand) (tea.Cmd, error) {
	c, ok := r.Resolve(parsed.Name)
	if !ok {
		return nil, &UnknownCommandError{Name: parsed.Name, Suggestions: r.Suggest(parsed.Name, 3)}
	}
	if c.Execute == nil {
		return nil, fmt.Errorf("command :%s cannot be executed", c.Name)
	}
	cmd, err := c.Execute(rt, parsed)
	return cmd, err
}

func (r *Registry) Commands() []Command {
	out := make([]Command, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.commands[name])
	}
	return out
}

func (r *Registry) Suggest(input string, limit int) []string {
	type scored struct {
		name  string
		score int
	}
	q := normalize(input)
	var found []scored
	for _, c := range r.commands {
		d := levenshtein(q, c.Name)
		score := d * 10
		if strings.HasPrefix(c.Name, q) || strings.HasPrefix(q, c.Name) {
			score -= 20
		}
		for _, a := range c.Aliases {
			ad := levenshtein(q, normalize(a))*10 + 2
			if ad < score {
				score = ad
			}
		}
		threshold := max(20, len([]rune(q))*6)
		if score <= threshold {
			found = append(found, scored{c.Name, score})
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].score == found[j].score {
			return found[i].name < found[j].name
		}
		return found[i].score < found[j].score
	})
	if limit > len(found) {
		limit = len(found)
	}
	out := make([]string, limit)
	for i := range out {
		out[i] = found[i].name
	}
	return out
}

func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ra := range ar {
		cur := make([]int, len(br)+1)
		cur[0] = i + 1
		for j, rb := range br {
			cost := 0
			if ra != rb {
				cost = 1
			}
			cur[j+1] = min(cur[j]+1, prev[j+1]+1, prev[j]+cost)
		}
		prev = cur
	}
	return prev[len(br)]
}
