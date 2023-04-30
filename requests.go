package bravewength

// This file contains types and deserialization code for every type of request a player can
// make to the server.

import (
	"bytes"
	"fmt"
	"strconv"
)

// reqSetOwnName is a request to update your player name. Two players may not have the
// same name.
type reqSetOwnName string

// reqSetOwnRole is a request to change roles, i.e., switch from purple team to teal
// team, seeker to knower, and vice versa. Knowers may not change their role
// if a game is in-progress because they have already seen the card layout.
type reqSetOwnRole role

// reqRandomizeTeams is a request to randomize the teams. Teams may not be randomized while
// a game is in-progress.
type reqRandomizeTeams struct{}

// reqNewGame is a request to start a new game, and will destroy any in-progress game state.
type reqNewGame struct{}

// reqEndGame is a request to end the current game without starting a new game. Doing so is
// useful so that teams can be completely re-arranged, because knowers are not allowed to
// become seekers/spectators while a game is in-progress.
type reqEndGame struct{}

// reqGiveClue is a request to give a clue, and will have no effect unless it is the
// client's turn to play and they are currently a knower.
type reqGiveClue string

// reqRevealCard is a request to indicate the user clicked on a card; the value of the
// underlying integer is the index of the card on the board.
type reqRevealCard int

// reqEndTurn is a request to end a seeker's turn. Their turn gets ended automatically if
// they click a card that does not belong to their team.
type reqEndTurn struct{}

// reqBodyDelim is the delimiter to mark where the request type ends and the request body
// (if any) begins.
var reqBodyDelim = []byte{'\n'}

// decodeRequest attempts to deserialize a request from a client. This is where all
// validation occurs EXCEPT for validation which depends on the current room state, which
// must be performed inside the room's main event-processing goroutine. If the request is
// invalid, nil will be returned. Error feedback is intentionally not given to clients
// because it would sacrifice simplicity/performance and the API is not complex--the only
// reason a well-written client would ever send a bad request is if the game state is out
// of sync, probably due to a slow connection, which WON'T be helped by sending more
// messages down the wire!
func decodeRequest(msg []byte) any {
	method, body, hasBody := bytes.Cut(msg, reqBodyDelim)

	switch string(method) {
	case "set-own-name":
		if len(body) == 0 {
			return nil
		}

		return reqSetOwnName(string(body))
	case "set-own-role":
		role, err := strconv.Atoi(string(body))
		if err != nil {
			return nil
		}

		if role < 0 || role > 4 {
			return nil
		}

		return reqSetOwnRole(role)
	case "randomize-teams":
		return reqRandomizeTeams{}
	case "new-game":
		fmt.Println("deserialized new-game request")
		return reqNewGame{}
	case "end-game":
		return reqEndGame{}
	case "give-clue":
		return reqGiveClue(string(body))
	case "reveal-card":
		if !hasBody {
			return nil
		}

		i, err := strconv.Atoi(string(body))
		if err != nil {
			return nil
		}

		if i < 0 || i >= boardSize {
			return nil
		}

		return reqRevealCard(i)
	case "end-turn":
		return reqEndTurn{}
	}

	return nil
}
