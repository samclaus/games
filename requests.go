package bravewength

import (
	"bytes"
	"strconv"
)

// reqSetOwnRole is a request to change roles, i.e., switch from purple team to teal
// team, seeker to knower, and vice versa. Knowers may not change their role
// if a game is in-progress because they have already seen the card layout.
type reqSetOwnRole role

// reqGiveClue is a request to give a clue, and will have no effect unless it is the
// client's turn to play and they are currently a knower. A 0 count means the
// knower did not want to indicate the number of cards.
type reqGiveClue struct {
	clue  string
	count int
}

// reqCardClicked is a request to indicate the user clicked on a card; the value of the
// underlying integer is the index of the card on the board.
type reqCardClicked int

// reqNewGame is a request to start a new game, and will destroy any in-progress game state.
type reqNewGame struct{}

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
	case "set-own-role":
		role, err := strconv.Atoi(string(body))
		if err != nil {
			return nil
		}

		if role < 0 || role > 4 {
			return nil
		}

		return reqSetOwnRole(role)
	case "give-clue":
		if !hasBody {
			return nil
		}

		clue, countStr, _ := bytes.Cut(body, reqBodyDelim)
		if len(countStr) == 1 {
			countASCII := countStr[0]

			if countASCII >= 48 && countASCII <= 57 {
				return reqGiveClue{string(clue), int(countASCII - 48)}
			}
		}

		// Count was not provided, was incorrectly encoded, or was not in range [0, 9]
		return reqGiveClue{clue: string(clue)}
	case "card-clicked":
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

		return reqCardClicked(i)
	case "new-game":
		return reqNewGame{}
	}

	return nil
}
