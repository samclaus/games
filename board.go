package bravewength

import "math/rand"

const boardSize = 25
const (
	cardTypeBlank = iota
	cardTypeTeal
	cardTypePurple
	cardTypeBlack
	cardTypeHidden
)

type board struct {
	Words     [boardSize]string
	FullTypes [boardSize]byte
	DiscTypes [boardSize]byte
}

// allCardsBlank is an array of card types with every element set to cardTypeBlank;
// it is simply an array of zeros
var allCardsBlank [boardSize]byte

// allCardsHidden is an array of card types with every element set to cardTypeHidden.
var allCardsHidden = func() [boardSize]byte {
	var types [boardSize]byte

	for i := range types {
		types[i] = cardTypeHidden
	}

	return types
}()

// reset randomizes the board and resets the discovered flags. Must be
// called to initialize the board the first time as well.
func (b *board) reset() {
	b.FullTypes = allCardsBlank
	b.DiscTypes = allCardsHidden

	randomDeckIndices := make(map[int]struct{})

	for len(randomDeckIndices) < boardSize {
		randomDeckIndices[rand.Intn(len(wordDeck))] = struct{}{}
	}

	boardIndex := 0

	for deckIndex := range randomDeckIndices {
		b.Words[boardIndex] = wordDeck[deckIndex]
		boardIndex += 1
	}

	var hasColor [25]bool

	blackCardPos := rand.Intn(boardSize)
	b.FullTypes[blackCardPos] = cardTypeBlack
	hasColor[blackCardPos] = true

	for numPurple := 0; numPurple < 8; {
		purpleCardPos := rand.Intn(boardSize)

		if !hasColor[purpleCardPos] {
			b.FullTypes[purpleCardPos] = cardTypePurple
			hasColor[purpleCardPos] = true
			numPurple += 1
		}
	}

	for numTeal := 0; numTeal < 9; {
		tealCardPos := rand.Intn(boardSize)

		if !hasColor[tealCardPos] {
			b.FullTypes[tealCardPos] = cardTypeTeal
			hasColor[tealCardPos] = true
			numTeal += 1
		}
	}
}
