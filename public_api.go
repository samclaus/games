package games

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Server interface {
	HandleGetRooms(http.ResponseWriter, *http.Request)
	HandleJoinRoom(http.ResponseWriter, *http.Request)
}

// GameState is the interface implemented by individual game instances. Each
// game room does all of its processing in a single goroutine, and will only
// be running one instance of a game at a time. These methods will only be
// called in the same goroutine that created the instance via Game.NewInstance().
type GameState interface {
	// Init is a hook allowing the game to broadcast initial state to players as
	// necessary. Client references are NOT safe to retain and use after this
	// method returns!
	Init(players []*Client)
	// HandleRequest is a hook allowing the game to act on a request made by a
	// player. Client references are NOT safe to retain and use after this
	// method returns!
	HandleRequest(players []*Client, src *Client, payload []byte)
	// HandleNewPlayer is a hook allowing the game to emit initial state to a
	// new player that has just joined the room. The client reference is NOT safe
	// to retain and use after this method returns!
	HandleNewPlayer(player *Client)
	// Deinit is a hook allowing the game to clean up its memory and help out
	// the garbage collector.
	Deinit()
}

// Game is a turn-based game implementation designed to be run within
// the system provided by this library. All of its methods MUST be
// safe to call from multiple goroutines without additional synchronization.
type Game interface {
	// ID should return a static string identifying the game implementation
	// universally, e.g., "samclaus/bravewength".
	ID() string
	// Version is the version of the game implementation, and should be
	// incremented whenever changes are made to the request/state API of
	// the game, i.e., the structure of the messages that get exhanged with
	// connected clients via WebSockets.
	//
	// TODO: this will probably use semantic versioning rather than just a
	// single integer in the future.
	Version() int
	// NewInstance provides a standalone instance of the game that can be
	// run within a single game room's goroutine, concurrently with any
	// other room goroutines running separate instances. MUST NOT RETURN
	// NIL!
	//
	// If instances of a game need to talk to each other (which is highly
	// unlikely and probably just a hack), they must use some global state
	// with synchronization provided by the implementation author.
	NewInstance() GameState
}

// AllocGameMessage allocates a byte slice with a 1-byte header to tell
// client-side code that the remainder of the WebSocket message is only to
// be interpreted by the current game's client-side code. The slice is
// allocated with 1+cap bytes to factor in the header byte.
func AllocGameMessage(cap int) []byte {
	return append(make([]byte, 0, 1+cap), scopeGame)
}

func NewServer(u websocket.Upgrader, games ...Game) Server {
	gamesByID := make(map[string]Game)
	for _, g := range games {
		gamesByID[g.ID()] = g
	}

	return &server{
		upgrader: u,
		games:    gamesByID,
		rooms:    make(map[uint32]*room),
	}
}
