package skull

import (
	"math/rand"

	"github.com/google/uuid"
)

type handStatus = uint8

const (
	statusUnclaimed handStatus = iota
	statusClaimed
	statusLeft
)

type skullStatus = uint8

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

func (h *hand) resetCardsAndScore() {
	h.hcards = 4
	h.pcards = 0
	h.skullStatus = skullInHand
	h.skullPos = uint8(rand.Intn(4))
	h.score = 0
}

func (h *hand) hasCards() bool {
	return (h.hcards + h.pcards) > 0
}

func (h *hand) reclaimPlayedCards() {
	// TODO: return played cards, in order (including skull), to end of held cards
}
