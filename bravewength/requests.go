package bravewength

// This file contains types and deserialization code for every type of request a player can
// make to the server.

import (
	"github.com/samclaus/games"
)

const (
	// reqSetOwnRole is a request to change roles, i.e., switch from purple team to teal
	// team, seeker to knower, and vice versa. Knowers may not change their role
	// if a game is in-progress because they have already seen the card layout.
	reqSetRole byte = iota

	// reqRandomizeTeams is a request to randomize the teams. Teams may not be randomized while
	// a game is in-progress.
	reqRandomizeTeams

	// reqNewGame is a request to start a new game, and will destroy any in-progress game state.
	reqNewGame

	// reqEndGame is a request to end the current game without starting a new game. Doing so is
	// useful so that teams can be completely re-arranged, because knowers are not allowed to
	// become seekers/spectators while a game is in-progress.
	reqEndGame

	// reqGiveClue is a request to give a clue, and will have no effect unless it is the
	// client's turn to play and they are currently a knower.
	reqGiveClue

	// reqRevealCard is a request to indicate the user clicked on a card; the value of the
	// underlying integer is the index of the card on the board.
	reqRevealCard

	// reqEndTurn is a request to end a seeker's turn. Their turn gets ended automatically if
	// they click a card that does not belong to their team.
	reqEndTurn
)

// HandleRequest is required to satisfy the (github.com/samclaus/games).Game interface and
// implements all turn-based game logic for Bravewength.
//
// TODO: tell room to disconnect client for sending invalid request structure?
func (g *gameState) HandleRequest(players []games.Client, src games.Client, payload []byte) {
	if len(payload) == 0 {
		return
	}

	body := payload[1:]
	turn := g.currentTurn
	srcID := src.ID()
	srcRole := g.roles[srcID]

	switch payload[0] {
	case reqSetRole:
		if len(body) != 1 || body[0] > 4 {
			return
		}

		newRole := role(body[0])
		isKnower := srcRole.IsKnower()
		willBeKnower := newRole.IsKnower()

		// If a game is in-progress, knowers may change teams but may not change to
		// seekers or spectators because they have seen the card layout; we also do not
		// want to issue state changes if nothing got changed
		if newRole == srcRole || (!g.gameEnded && isKnower && !willBeKnower) {
			return
		}

		// Spectator is the default role
		if newRole == roleSpectator {
			delete(g.roles, srcID)
		} else {
			// TODO: prevent map from growing too large if people keep connecting,
			// setting role, and disconnecting
			g.roles[srcID] = newRole
		}

		g.broadcastRolesState(players)

		// If card visibility changed, we must send them freshly tailored game state
		if isKnower != willBeKnower {
			src.Send(g.encodeBoardState(willBeKnower))
		}

	case reqNewGame:
		g.newGame()
		g.gameLog = append(g.gameLog, gameEventInfo{
			Src:  srcID.String(),
			Role: srcRole,
			Kind: gameEventTypeGameStarted,
		})
		g.broadcastBoardState(players)

	case reqEndGame:
		// Cannot end game if no game is in-progress
		if !g.gameEnded {
			g.gameEnded = true
			g.winner = teamNone
			g.gameLog = append(g.gameLog, gameEventInfo{
				Src:  srcID.String(),
				Role: srcRole,
				Kind: gameEventTypeGameEnded,
			})
			g.broadcastBoardState(players)
		}

	case reqRandomizeTeams:
		// TODO
	case reqGiveClue:
		if len(body) == 0 {
			return
		}

		// Cannot give a clue if:
		// - The game is over
		// - It is not the requester's turn
		// - The requester is not a knower
		if g.gameEnded || srcRole != turn || !srcRole.IsKnower() {
			return
		}

		clue := string(body)

		g.currentTurn = turn.NextTurn()
		g.currentClue = clue
		g.gameLog = append(g.gameLog, gameEventInfo{
			Src:  srcID.String(),
			Role: srcRole,
			Kind: gameEventTypeClueGiven,
			Clue: clue,
		})
		g.broadcastBoardState(players)

	case reqRevealCard:
		if len(body) != 1 || body[0] >= boardSize {
			return
		}
		cardIndex := body[0]

		// Cannot reveal a card if:
		// - The game is over
		// - It is not the requester's turn
		// - The requester is not a seeker
		// - The card has already been revealed
		if g.gameEnded ||
			srcRole != turn ||
			!srcRole.IsSeeker() ||
			g.Board.DiscTypes[cardIndex] != cardTypeHidden {
			return
		}

		revealedType := g.Board.FullTypes[cardIndex]
		g.Board.DiscTypes[cardIndex] = revealedType

		g.gameLog = append(g.gameLog, gameEventInfo{
			Src:      srcID.String(),
			Role:     srcRole,
			Kind:     gameEventTypeCardRevealed,
			Word:     g.Board.Words[cardIndex],
			CardType: revealedType,
		})

		tealPlayer := srcRole == roleTealKnower || srcRole == roleTealSeeker

		if revealedType == cardTypeBlack {
			g.gameEnded = true

			if tealPlayer {
				g.winner = teamPurple
			} else {
				g.winner = teamTeal
			}

			g.gameLog = append(g.gameLog, gameEventInfo{
				Src:  srcID.String(),
				Role: srcRole,
				Kind: gameEventTypeGameEnded,
			})
		} else if revealedType == cardTypeNeutral {
			g.currentTurn = turn.NextTurn()
		} else if winner := g.Board.winner(); winner != teamNone {
			g.gameEnded = true
			g.winner = winner
			g.gameLog = append(g.gameLog, gameEventInfo{
				Src:  srcID.String(),
				Role: srcRole,
				Kind: gameEventTypeGameEnded,
			})
		} else {
			tealCard := revealedType == cardTypeTeal

			if tealPlayer != tealCard {
				g.currentTurn = turn.NextTurn()
			}
		}

		g.broadcastBoardState(players)

	case reqEndTurn:
		// Cannot end turn if:
		// - The game is over
		// - It is not the requester's turn
		// - The requester is not a seeker
		if g.gameEnded ||
			srcRole != turn ||
			!srcRole.IsSeeker() {
			return
		}

		g.currentTurn = turn.NextTurn()
		g.gameLog = append(g.gameLog, gameEventInfo{
			Src:  srcID.String(),
			Role: srcRole,
			Kind: gameEventTypeTurnEnded,
		})
		g.broadcastBoardState(players)

	}
}
