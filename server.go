package games

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const idCookieName = "id"

type server struct {
	upgrader websocket.Upgrader
	games    map[string]Game
	rooms    map[uint32]*room
	roomCtr  uint32
	roomsMtx sync.RWMutex
}

// ServeHTTP is Server's only public method. First, it expects a "room" query
// parameter on the given HTTP request, which should either correspond to an existing
// room or have the special value "new" (otherwise an error response will be sent).
// If the room is successfully found/created, the request's underlying TCP connection
// will be upgraded to a WebSocket and it will be handed off to the room, which will
// serve the connection in another goroutine until the client closes the WebSocket or
// an error occurs.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var clientID uuid.UUID

	if ck, err := r.Cookie(idCookieName); err != nil {
		clientID = uuid.New()
		http.SetCookie(w, &http.Cookie{
			Name:     idCookieName,
			Value:    clientID.String(),
			SameSite: http.SameSiteStrictMode,
			Secure:   true,
		})
	} else if clientID, err = uuid.Parse(ck.Value); err != nil {
		http.Error(w, "Invalid client ID cookie", http.StatusBadRequest)
		return
	}

	roomCode := r.URL.Query().Get("room")
	newRoom := roomCode == "new"
	playerName := r.URL.Query().Get("name")

	if playerName == "" {
		http.Error(w, "Must specify a player name with 'name' URL query parameter", http.StatusBadRequest)
		return
	}

	var rm *room

	if newRoom {
		roomName := r.URL.Query().Get("room-name")
		if roomName == "" {
			http.Error(w, "Must specify a name for the room with 'room-name' URL query parameter", http.StatusBadRequest)
			return
		}

		s.roomsMtx.Lock()

		rm = &room{
			gameRegistry: s.games,
			ID:           s.roomCtr,
			Name:         roomName,
			members:      make([]Client, 0, 15), // TODO: enforce max 15 members
			register:     make(chan *client),
			unregister:   make(chan *client),
			requests:     make(chan request, 100),
			chat:         &chatBuffer{},
		}

		s.roomCtr++
		s.rooms[rm.ID] = rm
		s.roomsMtx.Unlock()

		// This is where the magic begins
		go func() {
			rm.processEventsUntilClosed()

			s.roomsMtx.Lock()
			delete(s.rooms, rm.ID)
			s.roomsMtx.Unlock()
		}()
	} else {
		if roomID, err := strconv.ParseUint(roomCode, 10, 32); err == nil {
			s.roomsMtx.RLock()
			rm = s.rooms[uint32(roomID)]
			s.roomsMtx.RUnlock()
		}
		if rm == nil {
			return // handles both invalid ID format and nonexistent cases
		}
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// If we were creating a brand new room, we can go ahead and delete it since
		// the client doesn't even know the room ID yet
		if newRoom {
			s.roomsMtx.Lock()
			delete(s.rooms, rm.ID)
			s.roomsMtx.Unlock()
		}

		// No need to send HTTP error reply because the .Upgrade() call will send
		// an error response before it returns an error to our code
		debug("Failed to upgrade connection: %v", err)
		return
	}

	cli := &client{conn, clientID, playerName, rm, make(chan []byte, 100)}
	rm.register <- cli

	// Start read/write in new goroutine so we can return from this HTTP handler and let the
	// request and response writer (etc.) get cleaned up
	go cli.readPump()
	go cli.writePump()
}
