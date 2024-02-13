package bravewength

import (
	"github.com/google/uuid"
)

type Bravewength struct {
	Board board

	// Set of clients currently connected to the room. I thought about making this a slice,
	// but a room might have a ton of spectators connected so I'd rather focus on worst-case
	// performance.
	roles     map[uuid.UUID]role
	rolesSize uint

	// currentTurn marks which kind of players are currently active (rolePurpleSeeker, etc.).
	// The roleSpectator (0) value indicates that no game/match/round is in progress, i.e.,
	// the players must set up a game.
	currentTurn role

	// currentClue is the clue given by the current team's knower if the current turn is
	// one of the seekers, and is the empty string otherwise.
	currentClue string

	gameEnded bool

	// winner is whichever team won the game, but only applicable if gameEnded is true. teamNone
	// indicates that someone canceled the current game, meaning no team won.
	winner team

	gameLog []gameEventInfo
}

func (r *Bravewength) serializeFullGameStateEvent(showFullLayout bool) []byte {
	var discTypesASCII [boardSize]byte

	for i, ct := range r.Board.DiscTypes {
		discTypesASCII[i] = ct.ascii()
	}

	var fullTypesASCII string

	if showFullLayout || r.gameEnded {
		var ascii [boardSize]byte

		for i, ct := range r.Board.FullTypes {
			ascii[i] = ct.ascii()
		}

		fullTypesASCII = string(ascii[:])
	} else {
		// All hidden
		fullTypesASCII = "4444444444444444444444444"
	}

	return append(
		[]byte("game-state\n"),
		mustEncodeJSON(
			gameStateInfo{
				Words:       r.Board.Words[:],
				DiscTypes:   string(discTypesASCII[:]),
				FullTypes:   fullTypesASCII,
				CurrentTurn: r.currentTurn,
				CurrentClue: r.currentClue,
				GameEnded:   r.gameEnded,
				Winner:      r.winner,
				Log:         r.gameLog,
			},
		)...,
	)
}

func (r *Bravewength) broadcastPlayerState() {
	nclients := len(r.clients)
	clientArr := make([]*client, 0, nclients)
	playerArr := make([]playerInfo, 0, nclients)

	for client, player := range r.clients {
		clientArr = append(clientArr, client)
		playerArr = append(playerArr, player)
	}

	msg := append(
		[]byte("all-player-info\n"),
		mustEncodeJSON(playerArr)...,
	)

	for _, client := range clientArr {
		r.tryEmitOrKickUnresponsiveClient(client, msg)
	}
}

func (r *Bravewength) broadcastGameState() {
	gameStateKnower := r.serializeFullGameStateEvent(true)
	gameStateSeeker := r.serializeFullGameStateEvent(false)

	for client, player := range r.clients {
		var gameState []byte

		if player.Role.IsKnower() {
			gameState = gameStateKnower
		} else {
			gameState = gameStateSeeker
		}

		r.tryEmitOrKickUnresponsiveClient(client, gameState)
	}
}

func (ct cardType) ascii() byte {
	return byte(ct + 48)
}

func (r *Bravewength) newGame() {
	r.Board.reset()
	r.currentTurn = roleTealKnower
	r.currentClue = ""
	r.gameEnded = false
	r.winner = teamNone
	r.gameLog = make([]gameEventInfo, 0, 10)
}
