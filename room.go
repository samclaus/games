package bravewength

import (
	"fmt"
	"strconv"
)

// TODO: give credit in README for excellent WebSocket examples in github.com/gorilla/websocket
// which basically spelled out efficient room/client implementation.

// playerState represents the in-room state corresponding to a particular player, which
// get associated with the player's client.
type playerState struct {
	Name string `json:"name"`
	Role role   `json:"role"`
}

// request contains a request payload and the client it originated from.
type request struct {
	origin  *client
	payload any
}

// room represents a room which may be (1) pending, meaning the game has not started
// and new players can connect (and switch roles, and vice versa), or (2) in-game,
// meaning the words have been revealed and players are locked into their chosen roles.
//
// If the game has started, new connections will only be accepted if they correctly
// provide a player name and the given player exists and does not have an active
// connection; this is so that clients can reconnect without having to start a brand
// new game thanks to someone's spotty internet. However, the new connection is still
// locked to the same role so that someone can't, for example, start as a knower and
// then reconnect as a seeker to cheat.
//
// A room will be cleaned up as soon as every member disconnects from it.
type room struct {
	ID    uint32
	Board board

	// defaultPlayerNameCounter is used to generate random initial names for players
	defaultPlayerNameCounter uint64

	// currentTurn marks which kind of players are currently active (rolePurpleSeeker, etc.).
	// The roleSpectator (0) value indicates that no game/match/round is in progress, i.e.,
	// the players must set up a game.
	currentTurn role

	// currentClue is the clue given by the current team's knower if the current turn is
	// one of the seekers, and is the empty string otherwise.
	currentClue      string
	currentClueCount int

	// Set of clients currently connected to the room. I thought about making this a slice,
	// but a room might have a ton of spectators connected so I'd rather focus on worst-case
	// performance.
	clients map[*client]playerState

	// Incoming client connections
	register chan *client

	// Dead client connections which need to be removed from the room
	unregister chan *client

	// Incoming requests from connected clients; requests are deserialized (and invalid requests
	// are rejected) in each client's read goroutine so that the work can be done in parallel
	requests chan request
}

func (r *room) log(format string, args ...any) {
	fmt.Printf(fmt.Sprintf("[Room %d] ", r.ID)+format, args...)
}

func (r *room) serializeFullGameStateEvent(showFullCardLayout bool) []byte {
	type FullStateReloadPayload struct {
		RoomID           uint32        `json:"room_id"`
		Players          []playerState `json:"players"`
		Words            []string      `json:"words"`
		Types            string        `json:"types"`
		CurrentTurn      role          `json:"current_turn"`
		CurrentClue      string        `json:"current_clue"`
		CurrentClueCount int           `json:"current_clue_count"`
	}

	players := make([]playerState, 0, len(r.clients))
	for _, player := range r.clients {
		players = append(players, player)
	}

	var types [boardSize]byte
	var typesASCII [boardSize]byte

	if showFullCardLayout {
		types = r.Board.FullTypes
	} else {
		types = r.Board.DiscTypes
	}

	for i, cardType := range types {
		typesASCII[i] = cardTypeToASCII(cardType)
	}

	return append(
		[]byte("full-state-reload\n"),
		mustEncodeJSON(
			FullStateReloadPayload{
				RoomID:           r.ID,
				Players:          players,
				Words:            r.Board.Words[:],
				Types:            string(typesASCII[:]),
				CurrentTurn:      r.currentTurn,
				CurrentClue:      r.currentClue,
				CurrentClueCount: r.currentClueCount,
			},
		)...,
	)
}

func (r *room) tryEmitOrKickUnresponsiveClient(c *client, msg []byte) {
	select {
	case c.Send <- msg:
	default:
		// If this client's send channel, which uses a sizeable buffer,
		// is blocked, it means this client is being way too slow to
		// receive events and needs to be disconnected so we can reclaim
		// resources (the game would literally be unplayable for the user)
		close(c.Send)
		delete(r.clients, c)
	}
}

func (r *room) emitFullStateToPlayers() {
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

// processEvents should be started in a new goroutine as soon as a room is created. This
// function will continually process client requests and broadcasting state until the room
// is closed (when the last client disconnects).
func (r *room) processEventsUntilClosed() {
	r.log("Room created\n")
	defer r.log("Room destroyed\n")

	for {
		// TODO: actual handling code
		select {
		case c := <-r.register:
			r.log("Registering client...")
			r.defaultPlayerNameCounter++
			ps := playerState{Name: "Player " + strconv.FormatUint(r.defaultPlayerNameCounter, 10)}
			r.clients[c] = ps
			r.tryEmitOrKickUnresponsiveClient(c, append(
				[]byte("own-player-info\n"),
				mustEncodeJSON(ps)...,
			))
			r.emitFullStateToPlayers()
			fmt.Println("done.")
		case c := <-r.unregister:
			r.log("Unregistering client...")

			if _, ok := r.clients[c]; ok {
				delete(r.clients, c)
				close(c.Send)

				fmt.Println("done.")

				if len(r.clients) == 0 {
					// Last client disconnected so this room needs to get cleaned up
					close(r.requests)
					// TODO: more cleanup necessary here?
					return
				} else {
					r.emitFullStateToPlayers()
				}
			} else {
				fmt.Println("client wasn't in room!")
			}
		case req := <-r.requests:
			r.handleRequest(req)
		}
	}
}

func cardTypeToASCII(cardType byte) byte {
	return byte(cardType + 48)
}

// handleRequest should only ever be called by the room's event-processing goroutine;
// it will branch based on the request type, decide whether the given client is allowed
// to make the request (also depending on the current room state), and will then update
// room state and emit an event to all connected clients accordingly.
//
// Invalid requests are simply ignored, without sending error feedback to the client.
// Please see decodeRequest() for my explanation of why.
func (r *room) handleRequest(req request) {
	turn := r.currentTurn
	player := r.clients[req.origin]

	// TODO: don't let spectators make weird requests when in-between games (because currentTurn
	// will be equal to roleSpectator)

	switch payload := req.payload.(type) {
	case reqSetOwnRole:
		newRole := role(payload)

		// If a game is in-progress, knowers may change teams but may not change to
		// seekers or spectators because they have seen the card layout; we also do not
		// want to issue state changes if nothing got changed
		if newRole == player.Role ||
			(turn != roleSpectator && player.Role.IsKnower() && !newRole.IsKnower()) {
			return
		}

		// Golang does not let you assign struct fields through a map entry, so we must
		// update the local copy of the player information and then reassign the whole
		// thing to the map
		player.Role = newRole
		r.clients[req.origin] = player
		r.tryEmitOrKickUnresponsiveClient(req.origin, append(
			[]byte("own-player-info\n"),
			mustEncodeJSON(player)...,
		))
	case reqGiveClue:
		// If it is not a knower's turn OR it is not THIS player's turn,
		// they should not be giving a clue
		if turn != player.Role || !turn.IsKnower() {
			return
		}

		r.currentTurn = turn.NextTurn()
		r.currentClue = payload.clue
		r.currentClueCount = payload.count
	case reqCardClicked:
		if turn != player.Role || !turn.IsSeeker() || r.Board.DiscTypes[payload] != cardTypeHidden {
			return
		}

		revealedType := r.Board.FullTypes[payload]
		r.Board.DiscTypes[payload] = revealedType

		if revealedType == cardTypeBlack {
			// TODO: game over
		} else if revealedType == cardTypeBlank {
			r.currentTurn = turn.NextTurn()
		} else {
			tealPlayer := player.Role == roleTealKnower || player.Role == roleTealSeeker
			tealCard := revealedType == cardTypeTeal

			if tealPlayer != tealCard {
				r.currentTurn = turn.NextTurn()
			}
		}
	case reqResetBoard:
		r.Board.reset()
	}

	// TODO: optimize this (send partial state updates, don't send if nothing changed)
	r.emitFullStateToPlayers()
}
