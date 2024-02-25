package bravewength

import (
	"github.com/samclaus/games"
)

// This file contains constants and serialization code for every kind of
// message a room will send to clients to update their state.

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
	Words       []string        `json:"words"`
	DiscTypes   string          `json:"disc_types"`
	FullTypes   string          `json:"full_types"`
	CurrentTurn role            `json:"current_turn"`
	CurrentClue string          `json:"current_clue"`
	GameEnded   bool            `json:"game_ended"`
	Winner      team            `json:"winner"`
	Log         []gameEventInfo `json:"log"`
}

func (g *gameState) encodeBoardState(showFullLayout bool) []byte {
	var discTypesASCII [boardSize]byte

	for i, ct := range g.Board.DiscTypes {
		discTypesASCII[i] = ct.ascii()
	}

	var fullTypesASCII string

	if showFullLayout || g.gameEnded {
		var ascii [boardSize]byte

		for i, ct := range g.Board.FullTypes {
			ascii[i] = ct.ascii()
		}

		fullTypesASCII = string(ascii[:])
	} else {
		// All hidden
		fullTypesASCII = "4444444444444444444444444"
	}

	body := mustEncodeJSON(
		boardStateBody{
			Words:       g.Board.Words[:],
			DiscTypes:   string(discTypesASCII[:]),
			FullTypes:   fullTypesASCII,
			CurrentTurn: g.currentTurn,
			CurrentClue: g.currentClue,
			GameEnded:   g.gameEnded,
			Winner:      g.winner,
			Log:         g.gameLog,
		},
	)

	return append(
		append(games.AllocGameMessage(1+len(body)), stateBoard),
		body...,
	)
}

func (g *gameState) encodeRolesState() []byte {
	body := mustEncodeJSON(g.roles)
	msg := append(games.AllocGameMessage(1+len(body)), stateRoles)
	return append(msg, body...)
}
