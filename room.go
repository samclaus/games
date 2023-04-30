package bravewength

import (
	"fmt"
	"strconv"
)

// TODO: give credit in README for excellent WebSocket examples in github.com/gorilla/websocket
// which basically spelled out efficient room/client implementation.

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

	serializedInfo []byte

	// Set of clients currently connected to the room. I thought about making this a slice,
	// but a room might have a ton of spectators connected so I'd rather focus on worst-case
	// performance.
	clients map[*client]playerInfo

	// Incoming client connections
	register chan *client

	// Dead client connections which need to be removed from the room
	unregister chan *client

	// Incoming requests from connected clients; requests are deserialized (and invalid requests
	// are rejected) in each client's read goroutine so that the work can be done in parallel
	requests chan request

	// defaultPlayerNameCounter is used to generate random initial names for players
	defaultPlayerNameCounter uint64

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

func (r *room) log(format string, args ...any) {
	fmt.Printf(fmt.Sprintf("[Room %d] ", r.ID)+format, args...)
}

func (r *room) serializeFullGameStateEvent(showFullLayout bool) []byte {
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

func (r *room) broadcastPlayerState() {
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

func (r *room) broadcastGameState() {
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
		select {
		case c := <-r.register:
			r.log("Registering client...")
			r.defaultPlayerNameCounter++

			// NOTE: this is the first time anything will be pushed on the new client's send
			// channel, so the '<-' operations below literally cannot fail (channel is buffered)
			ps := playerInfo{Name: "Player " + strconv.FormatUint(r.defaultPlayerNameCounter, 10)}
			c.Send <- r.serializedInfo
			c.Send <- append(
				[]byte("own-player-info\n"),
				mustEncodeJSON(ps)...,
			)
			c.Send <- r.serializeFullGameStateEvent(false)

			r.clients[c] = ps
			r.broadcastPlayerState()
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
					r.broadcastPlayerState()
				}
			} else {
				fmt.Println("client wasn't in room!")
			}
		case req := <-r.requests:
			r.handleRequest(req)
		}
	}
}

func (ct cardType) ascii() byte {
	return byte(ct + 48)
}

func (r *room) newGame() {
	r.Board.reset()
	r.currentTurn = roleTealKnower
	r.currentClue = ""
	r.gameEnded = false
	r.winner = teamNone
	r.gameLog = make([]gameEventInfo, 0, 10)
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

	switch payload := req.payload.(type) {
	case reqSetOwnName:
		{
			// TODO: block duplicate names?
			// Golang does not let you assign struct fields through a map entry, so we must
			// update the local copy of the player information and then reassign the whole
			// thing to the map
			player.Name = string(payload)
			r.clients[req.origin] = player
			r.tryEmitOrKickUnresponsiveClient(req.origin, append(
				[]byte("own-player-info\n"),
				mustEncodeJSON(player)...,
			))
			r.broadcastPlayerState()
			return
		}
	case reqSetOwnRole:
		{
			newRole := role(payload)
			isKnower := player.Role.IsKnower()
			willBeKnower := newRole.IsKnower()

			// If a game is in-progress, knowers may change teams but may not change to
			// seekers or spectators because they have seen the card layout; we also do not
			// want to issue state changes if nothing got changed
			if newRole == player.Role || (!r.gameEnded && isKnower && !willBeKnower) {
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
			r.broadcastPlayerState()

			// If card visibility changed, we must send them freshly tailored game state
			if isKnower != willBeKnower {
				r.tryEmitOrKickUnresponsiveClient(
					req.origin,
					r.serializeFullGameStateEvent(willBeKnower),
				)
			}

			return
		}
	case reqNewGame:
		{
			if player.Role == roleSpectator {
				return
			}

			r.newGame()
			r.gameLog = append(r.gameLog, gameEventInfo{
				Src:  player,
				Kind: gameEventTypeGameStarted,
			})
			r.broadcastGameState()
			return
		}
	case reqEndGame:
		{
			if player.Role == roleSpectator {
				return
			}

			r.gameEnded = true
			r.winner = teamNone
			r.gameLog = append(r.gameLog, gameEventInfo{
				Src:  player,
				Kind: gameEventTypeGameEnded,
			})
			r.broadcastGameState()
			return
		}
	case reqRandomizeTeams:
		// TODO
	case reqGiveClue:
		{
			// Cannot give a clue if:
			// - The game is over
			// - It is not the requester's turn
			// - The requester is not a knower
			if r.gameEnded || player.Role != turn || !player.Role.IsKnower() {
				return
			}

			r.currentTurn = turn.NextTurn()
			r.currentClue = string(payload)
			r.gameLog = append(r.gameLog, gameEventInfo{
				Src:  player,
				Kind: gameEventTypeClueGiven,
				Clue: string(payload),
			})
			r.broadcastGameState()
			return
		}
	case reqRevealCard:
		{
			// Cannot reveal a card if:
			// - The game is over
			// - It is not the requester's turn
			// - The requester is not a seeker
			// - The card has already been revealed
			if r.gameEnded ||
				player.Role != turn ||
				!player.Role.IsSeeker() ||
				r.Board.DiscTypes[payload] != cardTypeHidden {
				return
			}

			revealedType := r.Board.FullTypes[payload]
			r.Board.DiscTypes[payload] = revealedType

			r.gameLog = append(r.gameLog, gameEventInfo{
				Src:      player,
				Kind:     gameEventTypeCardRevealed,
				Word:     r.Board.Words[payload],
				CardType: revealedType,
			})

			tealPlayer := player.Role == roleTealKnower || player.Role == roleTealSeeker

			if revealedType == cardTypeBlack {
				r.gameEnded = true

				if tealPlayer {
					r.winner = teamPurple
				} else {
					r.winner = teamTeal
				}

				r.gameLog = append(r.gameLog, gameEventInfo{
					Src:  player,
					Kind: gameEventTypeGameEnded,
				})
			} else if revealedType == cardTypeNeutral {
				r.currentTurn = turn.NextTurn()
			} else if winner := r.Board.winner(); winner != teamNone {
				r.gameEnded = true
				r.winner = winner
				r.gameLog = append(r.gameLog, gameEventInfo{
					Src:  player,
					Kind: gameEventTypeGameEnded,
				})
			} else {
				tealCard := revealedType == cardTypeTeal

				if tealPlayer != tealCard {
					r.currentTurn = turn.NextTurn()
				}
			}

			r.broadcastGameState()
			return
		}
	case reqEndTurn:
		{
			// Cannot end turn if:
			// - The game is over
			// - It is not the requester's turn
			// - The requester is not a seeker
			if r.gameEnded ||
				player.Role != turn ||
				!player.Role.IsSeeker() {
				return
			}

			r.currentTurn = turn.NextTurn()
			r.gameLog = append(r.gameLog, gameEventInfo{
				Src:  player,
				Kind: gameEventTypeTurnEnded,
			})
			r.broadcastGameState()
		}
	}
}
