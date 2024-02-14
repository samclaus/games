package bravewength

import (
	"github.com/google/uuid"
	"github.com/samclaus/games"
)

type gameState struct {
	Board board

	roles map[uuid.UUID]role

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

func (g *gameState) newGame() {
	g.Board.reset()
	g.currentTurn = roleTealKnower
	g.currentClue = ""
	g.gameEnded = false
	g.winner = teamNone
	g.gameLog = make([]gameEventInfo, 0, 10)
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

func (g *gameState) broadcastRolesState(players []games.Client) {
	body := mustEncodeJSON(g.roles)
	msg := append(games.AllocGameMessage(1+len(body)), stateRoles)
	msg = append(msg, body...)

	for _, p := range players {
		p.Send(msg)
	}
}

func (g *gameState) broadcastBoardState(players []games.Client) {
	boardStateKnower := g.encodeBoardState(true)
	boardStateSeeker := g.encodeBoardState(false)

	for _, p := range players {
		var boardState []byte

		if g.roles[p.ID()].IsKnower() {
			boardState = boardStateKnower
		} else {
			boardState = boardStateSeeker
		}

		p.Send(boardState)
	}
}

func (g *gameState) Init(players []games.Client) {
	// Don't need to broadcast roles because they all start out as
	// spectators, and spectator is the default role
	g.broadcastBoardState(players)
}

func (g *gameState) HandleNewPlayer(c games.Client) {
	// Don't need to broadcast roles because they all start out as
	// spectators, and spectator is the default role
	c.Send(g.encodeBoardState(g.roles[c.ID()].IsKnower()))
}

func (g *gameState) Deinit() {
	g.roles = nil
	g.gameLog = nil
}
