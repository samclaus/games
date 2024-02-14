package bravewength

import (
	"github.com/google/uuid"
	"github.com/samclaus/games"
)

type game struct {
	deck []string
}

func (g game) ID() string {
	return "samclaus/bravewength"
}

func (g game) Version() int {
	return 0
}

func (g game) NewInstance() games.GameState {
	instance := &gameState{
		Board: board{
			Deck: g.deck,
		},
		roles: make(map[uuid.UUID]role),
	}
	instance.newGame()

	return instance
}

func Game(deck []string) games.Game {
	const minDeckSize = 200

	if len(deck) == 0 {
		// No words provided, just use massive default deck as-is.
		deck = defaultDeck[:]
	} else {
		deckSize := len(deck)
		if deckSize < minDeckSize {
			deckSize = minDeckSize
		}

		// They passed us at least one word, but we need 200 for a decent
		// deck. Fill in the remainder with words from the default deck.
		tmp := make([]string, deckSize)
		copy(tmp[copy(tmp, deck):], defaultDeck[:])
		deck = tmp
	}

	return game{deck}
}
