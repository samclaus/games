package bravewength

import "github.com/google/uuid"

// This file contains types and serialization code needed for every type of event
// *payload* the server can emit to players. Each of these payloads must be
// serialized to JSON and prefixed with a header that says the type of the
// event. This code is more tightly coupled to the room code than the
// client-to-server request code is, simply because there is not a clean way
// for me to abstract it as much without hurting performance.

// TODO: optimized binary format and we definitely don't need to send the full
// state (especially the potentially big player UUID->role mapping) every time
// something happens

const (
	stateBoard byte = iota
	stateRoles
)

type gameEventType byte

const (
	gameEventTypeGameStarted gameEventType = iota
	gameEventTypeGameEnded
	gameEventTypeClueGiven
	gameEventTypeCardRevealed
	gameEventTypeTurnEnded
)

// gameEventInfo is a struct used for every type of game event. Fields will
// be populated with relevant values based on the kind of event being
// described. I chose to use only one struct to make things faster and
// simpler on the server, but clients can easily make use of, say,
// TypeScript discriminated unions to make the events easier to work with.
type gameEventInfo struct {
	Src      string        `json:"src"`
	Role     role          `json:"role"`
	Kind     gameEventType `json:"kind"`
	Clue     string        `json:"clue"`
	Word     string        `json:"word"`
	CardType cardType      `json:"card_type"`
}

type boardStateBody struct {
	Roles       map[uuid.UUID]role `json:"roles"`
	Words       []string           `json:"words"`
	DiscTypes   string             `json:"disc_types"`
	FullTypes   string             `json:"full_types"`
	CurrentTurn role               `json:"current_turn"`
	CurrentClue string             `json:"current_clue"`
	GameEnded   bool               `json:"game_ended"`
	Winner      team               `json:"winner"`
	Log         []gameEventInfo    `json:"log"`
}
