package bravewength

import "math/rand"

type cardType byte

const boardSize = 25
const (
	cardTypeNeutral cardType = iota
	cardTypeTeal
	cardTypePurple
	cardTypeBlack
	cardTypeHidden
)

type board struct {
	Words     [boardSize]string
	FullTypes [boardSize]cardType
	DiscTypes [boardSize]cardType
}

// allCardsBlank is an array of card types with every element set to cardTypeNeutral;
// it is simply an array of zeros
var allCardsBlank [boardSize]cardType

// allCardsHidden is an array of card types with every element set to cardTypeHidden.
var allCardsHidden = func() [boardSize]cardType {
	var types [boardSize]cardType

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

	for numTeal := 0; numTeal < 9; {
		tealCardPos := rand.Intn(boardSize)

		if !hasColor[tealCardPos] {
			b.FullTypes[tealCardPos] = cardTypeTeal
			hasColor[tealCardPos] = true
			numTeal += 1
		}
	}

	for numPurple := 0; numPurple < 8; {
		purpleCardPos := rand.Intn(boardSize)

		if !hasColor[purpleCardPos] {
			b.FullTypes[purpleCardPos] = cardTypePurple
			hasColor[purpleCardPos] = true
			numPurple += 1
		}
	}
}

func (b *board) winner() team {
	numTeal := 0
	numPurple := 0

	// TODO: use table somehow so that Go does not insert bounds checks?
	// i.e., have an array containing as many integer elements as there
	// are card types, and for each loop iteration, just increment the
	// integer corresponding to the current card type (using it as an index)
	for _, ct := range b.DiscTypes {
		if ct == cardTypeTeal {
			numTeal++
		} else if ct == cardTypePurple {
			numPurple++
		}
	}

	if numTeal >= 9 {
		return teamTeal
	}
	if numPurple >= 8 {
		return teamPurple
	}
	return teamNone
}
