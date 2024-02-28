package games

// TODO: give credit in README for excellent WebSocket examples in github.com/gorilla/websocket
// which basically spelled out efficient room/client implementation.

const (
	// scopeRoom means a request/event is intended for the room itself, not whatever
	// game (if any) is in progress.
	scopeRoom byte = iota
	// scopeGame means a request/event is intended for the current game.
	scopeGame
)

// request contains a request payload and the client it originated from.
type request struct {
	src *Client
	msg []byte
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
	gameRegistry map[string]Game

	ID   uint32
	Name string

	members []*Client

	// Incoming client connections
	register chan *Client

	// Dead client connections which need to be removed from the room
	unregister chan *Client

	// Incoming requests from connected clients; requests are deserialized (and invalid requests
	// are rejected) in each client's read goroutine so that the work can be done in parallel
	requests chan request

	// Room-global chat for members
	chat *chatBuffer

	// The in-progress game, which may be nil if a game is not in-progress
	currentGameID string
	currentGame   GameState
}

func (r *room) broadcast(msg []byte) {
	for _, c := range r.members {
		c.Send(msg)
	}
}

func (r *room) broadcastAllMembersState() {
	if len(r.members) > 0 {
		r.broadcast(encodeSetMembersState(r.members))
	}
}

func (r *room) removeMember(c *Client) {
	pos := -1
	for i := range r.members {
		if r.members[i] == c {
			pos = i
			break
		}
	}

	if pos < 0 {
		return
	}

	if lastIndex := len(r.members) - 1; pos == lastIndex {
		r.members = r.members[:pos]
	} else {
		r.members[pos] = r.members[lastIndex]
		r.members = r.members[:lastIndex]
	}

	// IMPORTANT: attempting to close an already-closed channel
	// causes Go to panic. removeMember() might be called twice,
	// so it should only close the Send channel the first time
	// when it actually removes the client.
	//
	// TODO: cleaner way to handle the whole dance between
	// client read/write goroutines and the room goroutine?
	close(c.send)

	r.broadcast(encodeDeleteMemberState(c.ID))
	r.debug("Unregistered client [ID: %s, Name: %q]", c.ID.String(), c.Name)
}

// processEvents should be started in a new goroutine as soon as a room is created. This
// function will continually process client requests and broadcasting state until the room
// is closed (when the last client disconnects).
func (r *room) processEventsUntilClosed() {
	r.debug("Room created")
	defer r.debug("Room destroyed")

	for {
		select {
		case c := <-r.register:
			r.debug("Registering client [ID: %s, Name: %q]", c.ID.String(), c.Name)

			if len(r.members) >= maxRoomMembers {
				close(c.send)
				continue
			}

			// NOTE: this is the first time anything will be pushed on the new client's send
			// channel, so the '<-' operations below literally cannot fail (channel is buffered)
			c.send <- encodeInitState(r, c.ID)
			c.send <- encodeAllChatMessagesState(r.chat)

			r.members = append(r.members, c)
			r.broadcastAllMembersState() // TODO: just set member? still need all members for new client

			if r.currentGameID != "" {
				r.currentGame.HandleNewPlayer(c)
			}
		case c := <-r.unregister:
			r.removeMember(c)
			c.room = nil

			if len(r.members) == 0 {
				// Last client disconnected so this room needs to get cleaned up
				r.members = nil
				close(r.requests)
				// TODO: more cleanup necessary here?
				return
			}
		case req := <-r.requests:
			r.handleRequest(req)
		}
	}
}
