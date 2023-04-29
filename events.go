package bravewength

// This file contains types and serialization code needed for every type of event
// *payload* the server can emit to players. Each of these payloads must be
// serialized to JSON and prefixed with a header that says the type of the
// event. This code is more tightly coupled to the room code than the
// client-to-server request code is, simply because there is not a clean way
// for me to abstract it as much without hurting performance.

type roomInfo struct {
	RoomID uint32 `json:"room_id"`
}

// playerInfo represents the in-room state corresponding to a particular player, which
// get associated with the player's client.
type playerInfo struct {
	Name string `json:"name"`
	Role role   `json:"role"`
}

type gameEventType byte

const (
	gameEventTypeGameStarted gameEventType = iota
	gameEventTypeGameEnded
	gameEventTypeClueGiven
	gameEventTypeCardRevealed
)

// gameEventInfo is a struct used for every type of game event. Fields will
// be populated with relevant values based on the kind of event being
// described. I chose to use only one struct to make things faster and
// simpler on the server, but clients can easily make use of, say,
// TypeScript discriminated unions to make the events easier to work with.
type gameEventInfo struct {
	Src      playerInfo    `json:"src"`
	Kind     gameEventType `json:"kind"`
	Clue     string        `json:"clue"`
	Word     string        `json:"word"`
	CardType cardType      `json:"card_type"`
}

type gameStateInfo struct {
	Words       []string        `json:"words"`
	Types       string          `json:"types"`
	CurrentTurn role            `json:"current_turn"`
	CurrentClue string          `json:"current_clue"`
	GameEnded   bool            `json:"game_ended"`
	Winner      team            `json:"winner"`
	Log         []gameEventInfo `json:"log"`
}
