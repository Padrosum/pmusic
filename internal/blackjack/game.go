package blackjack

import (
	"math/rand"
)

type Suit int

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

func (s Suit) Symbol() string {
	switch s {
	case Spades:
		return "♠"
	case Hearts:
		return "♥"
	case Diamonds:
		return "♦"
	default:
		return "♣"
	}
}

func (s Suit) IsRed() bool {
	return s == Hearts || s == Diamonds
}

type Rank int

const (
	Ace   Rank = 1
	Two   Rank = 2
	Three Rank = 3
	Four  Rank = 4
	Five  Rank = 5
	Six   Rank = 6
	Seven Rank = 7
	Eight Rank = 8
	Nine  Rank = 9
	Ten   Rank = 10
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
)

func (r Rank) String() string {
	switch r {
	case Ace:
		return "A"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	default:
		if r == Ten {
			return "10"
		}
		return string(rune('0' + int(r)))
	}
}

func (r Rank) Value() int {
	if r >= Ten {
		return 10
	}
	return int(r)
}

type Card struct {
	Suit   Suit
	Rank   Rank
	Hidden bool
}

type Phase int

const (
	PhaseMenu    Phase = iota
	PhasePlaying       // player's turn
	PhaseDealer        // dealer reveals and draws
	PhaseResult        // round over
)

type Result int

const (
	ResultNone       Result = iota
	ResultWin               // player beats dealer
	ResultLose              // player loses
	ResultPush              // tie
	ResultBlackjack         // player natural blackjack (pays 1.5x)
	ResultBust              // player bust
	ResultDealerBust        // dealer bust, player wins
)

type Game struct {
	deck        []Card
	PlayerHand  []Card
	DealerHand  []Card
	Balance     int
	Bet         int
	Phase       Phase
	Result      Result
	Message     string
	DoubledDown bool
}

func New() *Game {
	g := &Game{
		Balance: 1000,
		Bet:     50,
		Phase:   PhaseMenu,
		Message: "Bahsini ayarla [+/-] ve dağıt [n]",
	}
	g.buildDeck()
	return g
}

func (g *Game) buildDeck() {
	g.deck = g.deck[:0]
	for _, suit := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		for rank := Ace; rank <= King; rank++ {
			g.deck = append(g.deck, Card{Suit: suit, Rank: rank})
		}
	}
	rand.Shuffle(len(g.deck), func(i, j int) {
		g.deck[i], g.deck[j] = g.deck[j], g.deck[i]
	})
}

func (g *Game) drawCard() Card {
	if len(g.deck) == 0 {
		g.buildDeck()
	}
	c := g.deck[0]
	g.deck = g.deck[1:]
	return c
}

func handValue(cards []Card) int {
	total := 0
	aces := 0
	for _, c := range cards {
		if c.Hidden {
			continue
		}
		v := c.Rank.Value()
		if c.Rank == Ace {
			aces++
		}
		total += v
	}
	for aces > 0 && total+10 <= 21 {
		total += 10
		aces--
	}
	return total
}

func (g *Game) PlayerValue() int {
	return handValue(g.PlayerHand)
}

func (g *Game) DealerValue() int {
	return handValue(g.DealerHand)
}

func (g *Game) DealerVisibleValue() int {
	visible := make([]Card, 0, len(g.DealerHand))
	for _, c := range g.DealerHand {
		if !c.Hidden {
			visible = append(visible, c)
		}
	}
	return handValue(visible)
}

func isBlackjack(cards []Card) bool {
	if len(cards) != 2 {
		return false
	}
	return handValue(cards) == 21
}

func (g *Game) Deal() {
	if g.Bet > g.Balance {
		g.Bet = g.Balance
	}
	if g.Bet < 10 {
		g.Bet = 10
	}

	if len(g.deck) < 10 {
		g.buildDeck()
	}

	g.PlayerHand = []Card{g.drawCard(), g.drawCard()}
	dealerVisible := g.drawCard()
	dealerHole := g.drawCard()
	dealerHole.Hidden = true
	g.DealerHand = []Card{dealerVisible, dealerHole}
	g.DoubledDown = false
	g.Result = ResultNone

	pBJ := isBlackjack(g.PlayerHand)
	dBJ := isBlackjack(g.revealedDealerHand())

	if pBJ && dBJ {
		g.revealDealer()
		g.Result = ResultPush
		g.Message = "İkisi de Blackjack! Berabere."
		g.Phase = PhaseResult
		return
	}
	if pBJ {
		g.revealDealer()
		g.Result = ResultBlackjack
		winnings := g.Bet + g.Bet*3/2
		g.Balance += winnings
		g.Message = "♠ BLACKJACK! +" + formatMoney(g.Bet*3/2) + " kazandın!"
		g.Phase = PhaseResult
		return
	}

	g.Phase = PhasePlaying
	g.Message = "[h]it  [s]tand  [d]ouble"
}

func (g *Game) revealedDealerHand() []Card {
	revealed := make([]Card, len(g.DealerHand))
	copy(revealed, g.DealerHand)
	for i := range revealed {
		revealed[i].Hidden = false
	}
	return revealed
}

func (g *Game) revealDealer() {
	for i := range g.DealerHand {
		g.DealerHand[i].Hidden = false
	}
}

func (g *Game) Hit() {
	if g.Phase != PhasePlaying {
		return
	}
	g.PlayerHand = append(g.PlayerHand, g.drawCard())
	if g.PlayerValue() > 21 {
		g.revealDealer()
		g.Result = ResultBust
		g.Balance -= g.Bet
		g.Phase = PhaseResult
		if g.Balance <= 0 {
			g.Balance = 0
			g.Message = "Battın! Yeni oyun için [n]"
		} else {
			g.Message = "Battın! -" + formatMoney(g.Bet) + "  [n] yeni el"
		}
	}
}

func (g *Game) Stand() {
	if g.Phase != PhasePlaying {
		return
	}
	g.revealDealer()
	g.runDealer()
	g.settle()
}

func (g *Game) Double() {
	if g.Phase != PhasePlaying || len(g.PlayerHand) != 2 {
		return
	}
	actualBet := g.Bet
	if g.Bet*2 <= g.Balance {
		actualBet = g.Bet * 2
	} else {
		actualBet = g.Balance
	}
	g.Bet = actualBet
	g.DoubledDown = true
	g.PlayerHand = append(g.PlayerHand, g.drawCard())
	if g.PlayerValue() > 21 {
		g.revealDealer()
		g.Result = ResultBust
		g.Balance -= g.Bet
		g.Phase = PhaseResult
		if g.Balance <= 0 {
			g.Balance = 0
			g.Message = "Double — Battın! [n] yeni el"
		} else {
			g.Message = "Double — Battın! -" + formatMoney(g.Bet) + "  [n] yeni el"
		}
		return
	}
	g.revealDealer()
	g.runDealer()
	g.settle()
}

func (g *Game) runDealer() {
	for handValue(g.DealerHand) < 17 {
		g.DealerHand = append(g.DealerHand, g.drawCard())
	}
}

func (g *Game) settle() {
	pv := g.PlayerValue()
	dv := g.DealerValue()

	if dv > 21 {
		g.Result = ResultDealerBust
		g.Balance += g.Bet
		g.Message = "Dealer battı! +" + formatMoney(g.Bet) + "  [n] yeni el"
	} else if pv > dv {
		g.Result = ResultWin
		g.Balance += g.Bet
		g.Message = "Kazandın! +" + formatMoney(g.Bet) + "  [n] yeni el"
	} else if dv > pv {
		g.Result = ResultLose
		g.Balance -= g.Bet
		if g.Balance <= 0 {
			g.Balance = 0
			g.Message = "Kaybettin! Battın — [n] ile yeniden başla"
		} else {
			g.Message = "Kaybettin! -" + formatMoney(g.Bet) + "  [n] yeni el"
		}
	} else {
		g.Result = ResultPush
		g.Message = "Berabere! Bahis iade edildi.  [n] yeni el"
	}
	g.Phase = PhaseResult
}

func (g *Game) AdjustBet(delta int) {
	if g.Phase != PhaseMenu && g.Phase != PhaseResult {
		return
	}
	g.Bet += delta
	if g.Bet < 10 {
		g.Bet = 10
	}
	if g.Bet > g.Balance {
		g.Bet = g.Balance
	}
	if g.Balance <= 0 {
		g.Bet = 0
	}
}

func (g *Game) NewRound() {
	if g.Balance <= 0 {
		g.Balance = 1000
		g.Bet = 50
		g.Message = "Yeniden başlıyorsun! Bahsini ayarla [+/-] ve dağıt [n]"
		g.Phase = PhaseMenu
		g.PlayerHand = nil
		g.DealerHand = nil
		g.Result = ResultNone
		g.buildDeck()
		return
	}
	if g.Bet > g.Balance {
		g.Bet = g.Balance
	}
	g.Phase = PhaseMenu
	g.PlayerHand = nil
	g.DealerHand = nil
	g.Result = ResultNone
	g.Message = "Bahsini ayarla [+/-] ve dağıt [n]"
}

func formatMoney(n int) string {
	if n < 0 {
		n = -n
	}
	s := ""
	for n > 0 {
		if s != "" {
			s = "," + s
		}
		rem := n % 1000
		n /= 1000
		if n > 0 {
			s = pad3(rem) + s
		} else {
			s = itoa(rem) + s
		}
	}
	if s == "" {
		s = "0"
	}
	return "$" + s
}

func pad3(n int) string {
	s := itoa(n)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
