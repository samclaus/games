package skull

import (
	"math/rand"

	"github.com/google/uuid"
	"github.com/samclaus/games"
)

type gamePhase uint

const (
	phaseNoGame gamePhase = iota
	phasePlay
	phaseBid
	phasePick
	phaseWinner
	phaseAborted
)

func (p gamePhase) Active() bool {
	return p == phasePlay || p == phaseBid || p == phasePick
}

type handStatus uint

const (
	statusUnclaimed handStatus = iota
	statusClaimed
	statusLeft
)

type skullStatus uint8

const (
	skullInHand skullStatus = iota
	skullPlayed
	skullGone
)

type hand struct {
	status      handStatus
	id          uuid.UUID   // ID of the client that owns this hand
	hcards      uint8       // num held cards (including skull if it is not played)
	pcards      uint8       // num played cards (including skull if it is played)
	skullStatus skullStatus // skull in hand, played in stack of cards, or taken by another player?
	skullPos    uint8       // position of skull in hand or played stack; invalid if skull gone
	score       uint8       // num successful bids, 2 wins the game
}

type gameState struct {
	hands    [6]hand
	phase    gamePhase
	nplayers int
	turn     int
	bid      uint8
	bidder   int
	winner   uuid.UUID // only valid if game complete
}

func (h *hand) resetCardsAndScore() {
	h.hcards = 4
	h.pcards = 0
	h.skullStatus = skullInHand
	h.skullPos = uint8(rand.Intn(4))
	h.score = 0
}

// Shifts all claimed hands to the left of the array, marks any
// "left" hands as unclaimed, and resets the card counts and
// randomizes skull positions.
func (g *gameState) lockInPlayers() {
	g.nplayers = 0

	for _, h := range g.hands {
		if h.status == statusClaimed {
			h.resetCardsAndScore()
			g.hands[g.nplayers] = h
			g.nplayers++
		}
	}

	for i := g.nplayers; i < len(g.hands); i++ {
		g.hands[i].status = statusUnclaimed
		g.hands[i].resetCardsAndScore()
	}
}

func (g *gameState) nextTurn() {
	// During, say, *play* phase, we ignore their played cards and only players
	// who have HELD cards still can do anything; during bidding phase, you only
	// need to have some cards somewhere, i.e., not eliminated by getting all
	// your cards taking due to failed bid/picks
	pcardMult := uint8(0)
	if g.phase == phaseBid {
		pcardMult = 1
	}

	// Find next player who can still play, which depends on their hand AND the
	// phase of the game
	for i := 0; i < g.nplayers; i++ {
		g.turn = (g.turn + 1) % g.nplayers

		if (g.hands[g.turn].hcards + g.hands[g.turn].pcards*pcardMult) > 0 {
			break
		}
	}
}

func (g *gameState) getHand(clientID uuid.UUID) (int, *hand) {
	for i := range g.hands {
		if g.hands[i].status == statusClaimed && g.hands[i].id == clientID {
			return i, &g.hands[i]
		}
	}
	return -1, nil
}

func (g *gameState) broadcastState(players []games.Client) {
	for _, p := range players {
		p.Send(nil) // TODO
	}
}

func (g *gameState) Init(players []games.Client) {
	// TODO
}

func (g *gameState) HandleNewPlayer(c games.Client) {
	// TODO
}

func (g *gameState) Deinit() {
	// TODO
}
