package blackjack

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Nord-compatible palette — hardcoded so cards always look like real cards.
const (
	colorCardBg     = "#ECEFF4" // white card face
	colorCardFg     = "#2E3440" // dark text on card
	colorCardRed    = "#BF616A" // hearts & diamonds
	colorCardHidden = "#3B4252" // back of hidden card
	colorHiddenFg   = "#4C566A" // pattern on hidden card
	colorPanelBg    = "#2E3440"
	colorTitle      = "#88C0D0"
	colorAccent     = "#A3BE8C"
	colorGold       = "#EBCB8B"
	colorRed        = "#BF616A"
	colorDim        = "#4C566A"
	colorNormal     = "#D8DEE9"
	colorBorder     = "#5E81AC"
	colorWin        = "#A3BE8C"
	colorLose       = "#BF616A"
	colorPush       = "#EBCB8B"
)

// renderCard returns 7 lines representing one playing card (9 chars wide each).
func renderCard(c Card) []string {
	if c.Hidden {
		bg := lipgloss.NewStyle().Background(lipgloss.Color(colorCardHidden)).Foreground(lipgloss.Color(colorHiddenFg))
		inner := bg.Render("▓▓▓▓▓▓▓")
		top := "╭───────╮"
		bot := "╰───────╯"
		return []string{top, "│" + inner + "│", "│" + inner + "│", "│" + inner + "│", "│" + inner + "│", "│" + inner + "│", bot}
	}

	rank := c.Rank.String()
	suit := c.Suit.Symbol()

	fgColor := colorCardFg
	if c.Suit.IsRed() {
		fgColor = colorCardRed
	}

	textStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(colorCardBg)).
		Foreground(lipgloss.Color(fgColor))

	// Build 7-display-column inner lines for the card face.
	padR := strings.Repeat(" ", 7-len(rank))
	padL := strings.Repeat(" ", 7-len(rank))

	line1 := textStyle.Render(rank + padR)             // "A      " / "10     "
	line2 := textStyle.Render(suit + "      ")          // "♠      "
	line3 := textStyle.Render("   " + suit + "   ")    // "   ♠   "
	line4 := textStyle.Render("      " + suit)          // "      ♠"
	line5 := textStyle.Render(padL + rank)             // "      A" / "     10"

	top := "╭───────╮"
	bot := "╰───────╯"

	return []string{
		top,
		"│" + line1 + "│",
		"│" + line2 + "│",
		"│" + line3 + "│",
		"│" + line4 + "│",
		"│" + line5 + "│",
		bot,
	}
}

// renderHand joins multiple cards side-by-side, returns the combined string.
func renderHand(hand []Card) string {
	if len(hand) == 0 {
		return ""
	}
	cardLines := make([][]string, len(hand))
	for i, c := range hand {
		cardLines[i] = renderCard(c)
	}

	var rows [7]string
	for row := 0; row < 7; row++ {
		var parts []string
		for _, cl := range cardLines {
			parts = append(parts, cl[row])
		}
		rows[row] = strings.Join(parts, " ")
	}
	return strings.Join(rows[:], "\n")
}

// Render produces the full blackjack overlay string, centered on the terminal.
func Render(g *Game, termW, termH int) string {
	boxW := 64
	if termW < boxW+4 {
		boxW = termW - 4
	}
	if boxW < 32 {
		boxW = 32
	}
	innerW := boxW - 4 // text content width (border=2 + visual margin=2)

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorTitle)).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorNormal))
	goldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorGold)).Bold(true)
	winStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorWin)).Bold(true)
	loseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorLose)).Bold(true)
	pushStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPush)).Bold(true)

	var lines []string

	// Title
	title := titleStyle.Render("♠ ♥  BLACKJACK  ♦ ♣")
	lines = append(lines, center(title, innerW))
	lines = append(lines, dimStyle.Render(strings.Repeat("─", innerW)))

	// Balance & Bet
	balStr := goldStyle.Render(formatMoney(g.Balance))
	betStr := normalStyle.Render(formatMoney(g.Bet))
	infoLine := "Bakiye: " + balStr + "   Bahis: " + betStr
	lines = append(lines, infoLine)
	lines = append(lines, "")

	// Dealer section
	dealerLabel := titleStyle.Render("DEALER")
	dealerScore := ""
	if g.Phase == PhaseResult || g.Phase == PhaseDealer {
		dv := g.DealerValue()
		if dv > 21 {
			dealerScore = loseStyle.Render(fmt.Sprintf(" — Puan: %d (BUST)", dv))
		} else {
			dealerScore = dimStyle.Render(fmt.Sprintf(" — Puan: %d", dv))
		}
	} else if len(g.DealerHand) > 0 {
		dealerScore = dimStyle.Render(fmt.Sprintf(" — Puan: %d + ?", g.DealerVisibleValue()))
	}
	lines = append(lines, dealerLabel+dealerScore)
	lines = append(lines, "")

	if len(g.DealerHand) > 0 {
		handStr := renderHand(g.DealerHand)
		for _, l := range strings.Split(handStr, "\n") {
			lines = append(lines, "  "+l)
		}
	} else {
		lines = append(lines, dimStyle.Render("  [ kart bekleniyor ]"))
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(strings.Repeat("─", innerW)))
	lines = append(lines, "")

	// Player section
	playerLabel := titleStyle.Render("SEN")
	playerScore := ""
	if len(g.PlayerHand) > 0 {
		pv := g.PlayerValue()
		switch {
		case pv > 21:
			playerScore = loseStyle.Render(fmt.Sprintf(" — Puan: %d (BUST)", pv))
		case pv == 21:
			playerScore = winStyle.Render(fmt.Sprintf(" — Puan: %d ✓", pv))
		default:
			playerScore = dimStyle.Render(fmt.Sprintf(" — Puan: %d", pv))
		}
	}
	lines = append(lines, playerLabel+playerScore)
	lines = append(lines, "")

	if len(g.PlayerHand) > 0 {
		handStr := renderHand(g.PlayerHand)
		for _, l := range strings.Split(handStr, "\n") {
			lines = append(lines, "  "+l)
		}
	} else {
		lines = append(lines, dimStyle.Render("  [ kart bekleniyor ]"))
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(strings.Repeat("─", innerW)))

	// Result message
	msg := g.Message
	if msg != "" {
		var msgStyle lipgloss.Style
		switch g.Result {
		case ResultWin, ResultBlackjack, ResultDealerBust:
			msgStyle = winStyle
		case ResultLose, ResultBust:
			msgStyle = loseStyle
		case ResultPush:
			msgStyle = pushStyle
		default:
			msgStyle = normalStyle
		}
		lines = append(lines, center(msgStyle.Render(msg), innerW))
	}

	lines = append(lines, "")

	// Key hints
	var hints []string
	switch g.Phase {
	case PhaseMenu:
		hints = []string{"[n] dağıt", "[+] bahis+", "[-] bahis-", "[b] kapat"}
	case PhasePlaying:
		hints = []string{"[h] hit", "[s] stand", "[d] double", "[b] kapat"}
	case PhaseResult:
		if g.Balance <= 0 {
			hints = []string{"[n] yeniden başla", "[b] kapat"}
		} else {
			hints = []string{"[n] yeni el", "[+] bahis+", "[-] bahis-", "[b] kapat"}
		}
	default:
		hints = []string{"[b] kapat"}
	}
	hintLine := dimStyle.Render(strings.Join(hints, "  "))
	lines = append(lines, center(hintLine, innerW))

	content := strings.Join(lines, "\n")

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Width(boxW)

	box := borderStyle.Render(content)
	return lipgloss.Place(termW, termH, lipgloss.Center, lipgloss.Center, box)
}

// center pads s with spaces to be centered within width w (ANSI-aware).
func center(s string, w int) string {
	sw := lipgloss.Width(s)
	if sw >= w {
		return s
	}
	pad := (w - sw) / 2
	return strings.Repeat(" ", pad) + s
}
