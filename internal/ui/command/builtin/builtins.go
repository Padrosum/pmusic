package builtin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Padrosum/pmusic/internal/ui/command"
	tea "github.com/charmbracelet/bubbletea"
)

func Commands() []command.Command {
	return []command.Command{
		playback("play", []string{"pl"}, "Play selected or specified track", ":play [track|index]", play, trackComplete, []string{":play", `:play "Fade to Black"`}),
		playback("pause", []string{"pa"}, "Pause playback", ":pause", pause, nil, []string{":pause"}),
		playback("toggle", []string{"t"}, "Toggle play/pause", ":toggle", toggle, nil, []string{":toggle"}),
		playback("next", []string{"n"}, "Play next track", ":next", next, nil, []string{":next"}),
		playback("prev", []string{"previous", "p"}, "Play previous track", ":prev", prev, nil, []string{":prev"}),
		{Name: "seek", Aliases: []string{"sk"}, Category: "Playback", Summary: "Move within the current track", Description: "Seek relatively or jump to a time, percentage, start, or end.", Usage: ":seek <+seconds|-seconds|seconds|mm:ss|percent|start|end>", Examples: []string{":seek +30", ":seek 2:15", ":seek 50%"}, Related: []string{"play"}, Arguments: []command.ArgumentSpec{{Name: "position", Description: "Relative seconds, timestamp, percentage, start, or end", Required: true}}, Complete: static("+5", "+30", "-5", "-30", "start", "50%", "end"), Execute: seek},
		{Name: "volume", Aliases: []string{"vol", "v"}, Category: "Audio", Summary: "Inspect or change volume", Description: "Show, set, adjust, mute, or restore playback volume.", Usage: ":volume [0-100|+N|-N|mute|unmute|toggle]", Examples: []string{":volume", ":volume 60", ":volume +10", ":volume mute"}, Complete: static("25", "50", "75", "100", "mute", "unmute", "toggle"), Execute: volume},
		{Name: "loop", Aliases: []string{"repeat"}, Category: "Audio", Summary: "Inspect or change looping", Description: "Show or change the repeat-current-track state used by the r shortcut.", Usage: ":loop [on|off|toggle]", Examples: []string{":loop", ":loop toggle"}, Complete: static("on", "off", "toggle"), Execute: loop},
		{Name: "queue", Category: "Queue", Summary: "Open or clear the play queue", Description: "Show the existing session queue or remove all queued tracks.", Usage: ":queue [show|clear]", Examples: []string{":queue", ":queue clear"}, Subcommands: []command.Subcommand{{Name: "show", Description: "Open the queue overlay"}, {Name: "clear", Description: "Remove all queued tracks"}}, Execute: queue},
		{Name: "search", Aliases: []string{"find"}, Category: "Library", Summary: "Search the local music library", Description: "Open the existing local library search and optionally prefill its query.", Usage: ":search [query]", Examples: []string{":search", `:search "Duman Seni Kendime Sakladım"`}, Execute: search},
		{Name: "online", Aliases: []string{"yt"}, Category: "Library", Summary: "Search YouTube for music", Description: "Open the existing online music search. Text search uses YouTube in this version.", Usage: ":online [query]", Examples: []string{":online Metallica"}, Execute: online},
		{Name: "download", Aliases: []string{"dl"}, Category: "Library", Summary: "Open the safe search/download flow", Description: "Search and let the user choose a result, or preview a direct URL before downloading.", Usage: ":download <query|url>", Examples: []string{`:download "Metallica Fade to Black"`, ":download https://youtube.com/watch?v=..."}, Execute: download},
		{Name: "reload", Aliases: []string{"source"}, Category: "Application", Summary: "Reload configuration or library", Description: "Reload Lua configuration, or rescan the local music directory in the background.", Usage: ":reload [lua|library]", Examples: []string{":reload", ":reload library"}, Subcommands: []command.Subcommand{{Name: "lua", Description: "Hot-reload Lua configuration"}, {Name: "library", Description: "Rescan the music directory"}}, Execute: reload},
		{Name: "quit", Aliases: []string{"q"}, Category: "Application", Summary: "Quit pmusic", Description: "Exit through pmusic's normal cleanup path. Add ! to force-cancel active work.", Usage: ":quit[!]", Examples: []string{":quit", ":q!"}, Execute: quit},
		{Name: "help", Aliases: []string{"h"}, Category: "Application", Summary: "Browse command help", Description: "Open searchable, scrollable help generated from the command registry.", Usage: ":help [commands|keys|command]", Examples: []string{":help", ":help seek"}, Execute: help},
		{Name: "history", Aliases: []string{"hist"}, Category: "Application", Summary: "View or clear command history", Description: "Open recent command history, limit it to N entries, or clear persistent history.", Usage: ":history [N|clear]", Examples: []string{":history", ":history 20", ":history clear"}, Subcommands: []command.Subcommand{{Name: "clear", Description: "Clear session and persistent command history"}}, Execute: history},
		{Name: "stats", Aliases: []string{"statistics"}, Category: "Application", Summary: "Show listening activity", Description: "Inspect plays, skips, completions, listening time, and top tracks for today, this week, all time, or an artist.", Usage: ":stats [today|week|all|artist <name>]", Examples: []string{":stats", ":stats week", ":stats artist Metallica"}, Subcommands: []command.Subcommand{{Name: "today", Description: "Today's listening activity"}, {Name: "week", Description: "The last seven days"}, {Name: "all", Description: "All recorded activity"}, {Name: "artist", Description: "Activity for a matching artist"}}, Execute: stats},
	}
}

func playback(name string, aliases []string, summary, usage string, h command.Handler, c command.Completer, examples []string) command.Command {
	return command.Command{Name: name, Aliases: aliases, Category: "Playback", Summary: summary, Description: summary + " using pmusic's existing playback state.", Usage: usage, Examples: examples, Execute: h, Complete: c}
}
func static(values ...string) command.Completer {
	return func(_ command.Runtime, _ string) []command.CompletionItem {
		out := make([]command.CompletionItem, len(values))
		for i, v := range values {
			out[i] = command.CompletionItem{Value: v, Display: v, Kind: command.CompletionArgument}
		}
		return out
	}
}
func trackComplete(rt command.Runtime, q string) []command.CompletionItem {
	return rt.TrackCompletions(q, 10)
}
func joined(p command.ParsedCommand) string { return strings.Join(p.Args, " ") }
func requireAtMost(p command.ParsedCommand, n int) error {
	if len(p.Args) > n {
		return &command.InvalidArgumentError{Message: "Too many arguments.", Usage: ":" + p.Name}
	}
	return nil
}

func play(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error)  { return rt.Play(joined(p)) }
func pause(rt command.Runtime, _ command.ParsedCommand) (tea.Cmd, error) { return nil, rt.Pause() }
func toggle(rt command.Runtime, _ command.ParsedCommand) (tea.Cmd, error) {
	return nil, rt.TogglePause()
}
func next(rt command.Runtime, _ command.ParsedCommand) (tea.Cmd, error) { return rt.Next() }
func prev(rt command.Runtime, _ command.ParsedCommand) (tea.Cmd, error) { return rt.Previous() }

func volume(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	if err := requireAtMost(p, 1); err != nil {
		return nil, err
	}
	if len(p.Args) == 0 {
		rt.Notify(fmt.Sprintf("Volume: %d%%", rt.Volume()))
		return nil, nil
	}
	v := strings.ToLower(p.Args[0])
	if v == "mute" || v == "unmute" || v == "toggle" {
		if err := rt.Mute(v); err != nil {
			return nil, err
		}
		rt.Notify(fmt.Sprintf("Volume: %d%%", rt.Volume()))
		return nil, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil, &command.InvalidArgumentError{Message: "Invalid volume value: " + p.Args[0], Usage: ":volume <0-100|+N|-N|mute|unmute|toggle>"}
	}
	target := n
	if strings.HasPrefix(v, "+") || strings.HasPrefix(v, "-") {
		target = rt.Volume() + n
	}
	if target < 0 {
		target = 0
	}
	if target > 100 {
		target = 100
	}
	if err := rt.SetVolume(target); err != nil {
		return nil, err
	}
	rt.Notify(fmt.Sprintf("Volume: %d%%", rt.Volume()))
	return nil, nil
}

type SeekTarget struct {
	Relative bool
	Position time.Duration
	Percent  *float64
	End      bool
}

func ParseSeek(value string, duration time.Duration) (SeekTarget, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return SeekTarget{}, &command.MissingArgumentError{Message: "Missing seek position.", Usage: ":seek <+seconds|-seconds|seconds|mm:ss|percent|start|end>"}
	}
	if v == "start" {
		return SeekTarget{}, nil
	}
	if v == "end" {
		if duration <= 0 {
			return SeekTarget{}, &command.InvalidArgumentError{Message: "Track duration is unknown; :seek end is unavailable."}
		}
		return SeekTarget{Position: max(0, duration-time.Millisecond), End: true}, nil
	}
	if strings.HasSuffix(v, "%") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64)
		if err != nil || n < 0 || n > 100 {
			return SeekTarget{}, &command.InvalidArgumentError{Message: "Invalid percentage: " + value + "\nExpected a value between 0% and 100%."}
		}
		if duration <= 0 {
			return SeekTarget{}, &command.InvalidArgumentError{Message: "Track duration is unknown; percentage seeking is unavailable."}
		}
		pct := n
		return SeekTarget{Position: time.Duration(float64(duration) * n / 100), Percent: &pct}, nil
	}
	if strings.Contains(v, ":") {
		parts := strings.Split(v, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return SeekTarget{}, invalidSeek(value)
		}
		var sec int64
		for i, s := range parts {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil || n < 0 || (i > 0 && n >= 60) {
				return SeekTarget{}, invalidSeek(value)
			}
			sec = sec*60 + n
		}
		return SeekTarget{Position: time.Duration(sec) * time.Second}, nil
	}
	rel := strings.HasPrefix(v, "+") || strings.HasPrefix(v, "-")
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return SeekTarget{}, invalidSeek(value)
	}
	if !rel && n < 0 {
		return SeekTarget{}, invalidSeek(value)
	}
	return SeekTarget{Relative: rel, Position: time.Duration(n) * time.Second}, nil
}
func invalidSeek(v string) error {
	return &command.InvalidArgumentError{Message: "Invalid seek position: " + v, Usage: ":seek <+seconds|-seconds|seconds|mm:ss|percent|start|end>"}
}
func seek(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	if len(p.Args) != 1 {
		return nil, &command.MissingArgumentError{Message: "Missing seek position.", Usage: ":seek <+seconds|-seconds|seconds|mm:ss|percent|start|end>"}
	}
	elapsed, total, loaded := rt.Position()
	if !loaded {
		return nil, &command.RuntimeCommandError{Message: "No track is currently loaded."}
	}
	target, err := ParseSeek(p.Args[0], total)
	if err != nil {
		return nil, err
	}
	pos := target.Position
	if target.Relative {
		pos = elapsed + target.Position
	}
	if pos < 0 {
		pos = 0
	}
	if total > 0 && pos >= total {
		pos = total - time.Millisecond
	}
	if err := rt.SeekAbsolute(pos); err != nil {
		return nil, err
	}
	rt.Notify("Position: " + formatDuration(pos))
	return nil, nil
}
func formatDuration(d time.Duration) string {
	d = max(0, d)
	s := int(d.Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, (s%3600)/60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func loop(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	if len(p.Args) == 0 {
		rt.Notify(fmt.Sprintf("Loop: %s", onOff(rt.Loop())))
		return nil, nil
	}
	if len(p.Args) > 1 {
		return nil, &command.InvalidArgumentError{Message: "Invalid loop state.", Usage: ":loop [on|off|toggle]"}
	}
	switch strings.ToLower(p.Args[0]) {
	case "on":
		rt.SetLoop(true)
	case "off":
		rt.SetLoop(false)
	case "toggle":
		rt.SetLoop(!rt.Loop())
	default:
		return nil, &command.InvalidArgumentError{Message: "Invalid loop state: " + p.Args[0], Usage: ":loop [on|off|toggle]"}
	}
	rt.Notify("Loop: " + onOff(rt.Loop()))
	return nil, nil
}
func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
func queue(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	sub := "show"
	if len(p.Args) > 0 {
		sub = strings.ToLower(p.Args[0])
	}
	switch sub {
	case "show":
		rt.OpenQueue()
	case "clear":
		n, err := rt.ClearQueue()
		if err != nil {
			return nil, err
		}
		if n == 0 {
			rt.Notify("Queue is already empty.")
		} else {
			rt.Notify(fmt.Sprintf("Cleared %d queued tracks.", n))
		}
	default:
		return nil, &command.InvalidArgumentError{Message: "Unknown queue action: " + sub, Usage: ":queue [show|clear]"}
	}
	return nil, nil
}
func search(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	rt.OpenLocalSearch(joined(p))
	return nil, nil
}
func online(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	q := joined(p)
	return rt.OpenOnlineSearch(q, q != ""), nil
}
func download(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	q := joined(p)
	if q == "" {
		return nil, &command.MissingArgumentError{Message: "Missing download query or URL.", Usage: ":download <query|url>"}
	}
	return rt.OpenOnlineSearch(q, true), nil
}
func reload(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	target := "lua"
	if len(p.Args) > 0 {
		target = strings.ToLower(p.Args[0])
	}
	switch target {
	case "lua":
		return rt.ReloadLua(), nil
	case "library":
		return rt.ReloadLibrary(), nil
	default:
		return nil, &command.InvalidArgumentError{Message: "Unknown reload target: " + target, Usage: ":reload [lua|library]"}
	}
}
func quit(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) { return rt.Quit(p.Bang) }
func help(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	topic := ""
	if len(p.Args) > 0 {
		topic = p.Args[0]
	}
	return nil, rt.OpenHelp(topic)
}
func history(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	if len(p.Args) == 0 {
		rt.OpenHistory(0)
		return nil, nil
	}
	if strings.EqualFold(p.Args[0], "clear") {
		return nil, rt.ClearHistory()
	}
	n, err := strconv.Atoi(p.Args[0])
	if err != nil || n < 1 {
		return nil, &command.InvalidArgumentError{Message: "Invalid history count: " + p.Args[0], Usage: ":history [N|clear]"}
	}
	rt.OpenHistory(n)
	return nil, nil
}
func stats(rt command.Runtime, p command.ParsedCommand) (tea.Cmd, error) {
	scope := "today"
	if len(p.Args) > 0 {
		scope = strings.ToLower(p.Args[0])
	}
	query := ""
	if len(p.Args) > 1 {
		query = strings.Join(p.Args[1:], " ")
	}
	if scope == "artist" && strings.TrimSpace(query) == "" {
		return nil, &command.MissingArgumentError{Message: "Missing artist name.", Usage: ":stats artist <name>"}
	}
	if scope != "today" && scope != "week" && scope != "all" && scope != "artist" {
		return nil, &command.InvalidArgumentError{Message: "Unknown statistics scope: " + scope, Usage: ":stats [today|week|all|artist <name>]"}
	}
	return nil, rt.OpenStats(scope, query)
}
