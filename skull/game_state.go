package skull

import (
	"github.com/google/uuid"
	"github.com/samclaus/games"
)

// Must not exceed 16 because we use uint16 bitset to flag players who "passed" a bid
const maxPlayers = 6

type gamePhase uint

const (
	// NOTE: careful when changing order, some code uses <> comparisons
	phaseNoGame        gamePhase = iota // Fresh game lobby, have not played yet
	phaseWinner                         // Game was played and someone won
	phaseAborted                        // Game was aborted without a winner
	phasePlay                           // Go around circle, playing cards until someone bids
	phaseBid                            // Go around circle until someone WINS the bid
	phasePick                           // Someone won bid, now they try to pick that many roses
	phaseBidderShuffle                  // Bidder picked skull, allow them to shuffle before they get a card taken
	phaseTakeCard                       // Bidder failed, ended shuffling phase, now skull player gets to take a card from them
)

func (p gamePhase) Active() bool {
	return p > phaseAborted
}

type gameState struct {
	hands    [maxPlayers]hand
	phase    gamePhase // currently playing cards? bidding? attempting to pick cards for bid?
	nplayers int       // how many players are there (from left; other hands unclaimed)
	turn     int       // index of hand/player whose turn it is
	pcards   uint8     // total cards played, for validating bid amounts
	bid      uint8     // current bid
	bidder   int       // index of last hand/player who raised the bid
	passed   uint16    // bitset of hand/player indices that passed and cannot bid this time
	taker    uint8     // index of hand/player whose skull got picked by bidder; takes card from bidder
	winner   uuid.UUID // ID of client that won game; only valid for phaseWinner
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

func (g *gameState) getHand(clientID uuid.UUID) (int, *hand) {
	for i := range g.hands {
		if g.hands[i].status == statusClaimed && g.hands[i].id == clientID {
			return i, &g.hands[i]
		}
	}
	return -1, nil
}

// Finds the next player, in order (and handling loop-around), who still has
// at least one card, which could be held or played. I.e., ignores players
// who have been eliminated by losing all of their cards.
func (g *gameState) nextTurn() {
	for i := 0; i < g.nplayers; i++ {
		g.turn = (g.turn + 1) % g.nplayers

		// TODO: can I refactor this somehow so that it's not always running
		// the bidding phase logic where it needs to skip over players who
		// have already chosen to pass up the bid
		if (g.phase != phaseBid || (1<<g.turn)&g.passed == 0) &&
			g.hands[g.turn].hasCards() {
			break
		}
	}
}

func (g *gameState) reclaimPlayedCards() {
	g.pcards = 0

	for i := 0; i < g.nplayers; i++ {
		g.hands[i].reclaimPlayedCards()
	}
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
