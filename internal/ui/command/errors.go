package command

import (
	"fmt"
	"strings"
)

type ParseError struct{ Message string }

func (e *ParseError) Error() string { return "Could not parse command: " + e.Message }

type UnknownCommandError struct {
	Name        string
	Suggestions []string
}

func (e *UnknownCommandError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Unknown command: :%s", e.Name)
	if len(e.Suggestions) > 0 {
		b.WriteString("\n\nDid you mean:")
		for _, s := range e.Suggestions {
			fmt.Fprintf(&b, "\n  :%s", s)
		}
	}
	b.WriteString("\n\nRun :help commands to list all commands.")
	return b.String()
}

type MissingArgumentError struct{ Message, Usage string }

func (e *MissingArgumentError) Error() string { return e.Message + "\nUsage: " + e.Usage }

type InvalidArgumentError struct{ Message, Usage string }

func (e *InvalidArgumentError) Error() string {
	if e.Usage == "" {
		return e.Message
	}
	return e.Message + "\nUsage: " + e.Usage
}

type RuntimeCommandError struct{ Message string }

func (e *RuntimeCommandError) Error() string { return e.Message }

type AmbiguousMatchError struct {
	Query string
	Count int
}

func (e *AmbiguousMatchError) Error() string {
	return fmt.Sprintf("Found %d matching tracks.\nOpening local search for %q.", e.Count, e.Query)
}
