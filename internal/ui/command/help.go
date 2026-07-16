package command

import (
	"fmt"
	"sort"
	"strings"
)

var categoryOrder = []string{"Playback", "Audio", "Library", "Queue", "Application"}

func (r *Registry) Help(topic, filter string) ([]string, error) {
	topic = normalize(topic)
	filter = strings.ToLower(strings.TrimSpace(filter))
	if topic == "keys" {
		return nil, nil
	} // UI reuses the existing shortcut overlay.
	if topic != "" && topic != "commands" {
		c, ok := r.Resolve(topic)
		if !ok {
			return nil, &UnknownCommandError{Name: topic, Suggestions: r.Suggest(topic, 3)}
		}
		return commandHelp(c), nil
	}
	groups := map[string][]Command{}
	for _, c := range r.Commands() {
		if matchesHelp(c, filter) {
			groups[c.Category] = append(groups[c.Category], c)
		}
	}
	lines := []string{"pmusic Command Help", ""}
	for _, category := range categoryOrder {
		cs := groups[category]
		if len(cs) == 0 {
			continue
		}
		lines = append(lines, category)
		for _, c := range cs {
			lines = append(lines, fmt.Sprintf("  %-24s %s", c.Usage, c.Summary))
		}
		lines = append(lines, "")
	}
	for category, cs := range groups {
		found := false
		for _, known := range categoryOrder {
			if category == known {
				found = true
			}
		}
		if found {
			continue
		}
		sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
		lines = append(lines, category)
		for _, c := range cs {
			lines = append(lines, fmt.Sprintf("  %-24s %s", c.Usage, c.Summary))
		}
		lines = append(lines, "")
	}
	lines = append(lines, "Type :help <command> for detailed help.", "/:filter  j/k:scroll  PgUp/PgDn  g/G  Esc/q:close")
	return lines, nil
}

func commandHelp(c Command) []string {
	lines := []string{"::" + c.Name, c.Summary, "", c.Description, "", "Usage", "  " + c.Usage}
	if len(c.Arguments) > 0 {
		lines = append(lines, "", "Arguments")
		for _, a := range c.Arguments {
			lines = append(lines, fmt.Sprintf("  %-14s %s", a.Name, a.Description))
		}
	}
	if len(c.Subcommands) > 0 {
		lines = append(lines, "", "Subcommands")
		for _, s := range c.Subcommands {
			lines = append(lines, fmt.Sprintf("  %-14s %s", s.Name, s.Description))
		}
	}
	if len(c.Examples) > 0 {
		lines = append(lines, "", "Examples")
		for _, e := range c.Examples {
			lines = append(lines, "  "+e)
		}
	}
	if len(c.Aliases) > 0 {
		lines = append(lines, "", "Aliases", "  :"+strings.Join(c.Aliases, "  :"))
	}
	if len(c.Related) > 0 {
		lines = append(lines, "", "Related", "  :"+strings.Join(c.Related, "  :"))
	}
	lines = append(lines, "", "j/k:scroll  PgUp/PgDn  g/G  Esc/q:close")
	return lines
}
func matchesHelp(c Command, q string) bool {
	if q == "" {
		return true
	}
	hay := []string{c.Name, c.Category, c.Summary, c.Description, strings.Join(c.Aliases, " ")}
	for _, s := range hay {
		if strings.Contains(strings.ToLower(s), q) {
			return true
		}
	}
	return false
}
