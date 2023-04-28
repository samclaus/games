package bravewength

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

// Server is the main state for the HTTP/WebSocket Server, which currently just
// consists of a global set of rooms and synchronization primitives.
//
// Server only has one method, ConnectClientToRoom, which is used to upgrade an HTTP
// connection to a WebSocket and then hand it off to the correct room.
type Server struct {
	Upgrader websocket.Upgrader

	rooms sync.Map
}

// ServeHTTP is Server's only public method. First, it expects a "room" query
// parameter on the given HTTP request, which should either correspond to an existing
// room or have the special value "new" (otherwise an error response will be sent).
// If the room is successfully found/created, the request's underlying TCP connection
// will be upgraded to a WebSocket and it will be handed off to the room, which will
// serve the connection in another goroutine until the client closes the WebSocket or
// an error occurs.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	newRoom := roomCode == "new"

	var rm *room

	if newRoom {
		// Allocate a new room struct
		rm = &room{}

		// Loop just in case we generate a room ID that already exists
		createdRoom := false
		for try := 0; try < 10; try++ {
			rm.ID = rand.Uint32()

			if _, roomExists := s.rooms.LoadOrStore(rm.ID, rm); !roomExists {
				createdRoom = true
				break
			}
		}

		// If we exceeded the number of allowed retries and were still unable to come up with
		// a NEW pseudorandom room ID, it probably means there are a ton of rooms open and the
		// client should just wait before trying again
		if !createdRoom {
			http.Error(w, "failed to generate unique room ID", http.StatusServiceUnavailable)
			return
		}

		// Might as well only allocate the room's machinery once we know it has a spot
		rm.Board.reset()
		rm.clients = make(map[*client]playerState)
		rm.register = make(chan *client)
		rm.unregister = make(chan *client)
		rm.requests = make(chan request, 100)
		rm.currentTurn = roleTealKnower

		// This is where the magic begins
		go func() {
			rm.processEventsUntilClosed()
			s.rooms.Delete(rm.ID)
		}()
	} else if roomID, err := strconv.ParseUint(roomCode, 10, 32); err == nil {
		if rmUntyped, foundRoom := s.rooms.Load(uint32(roomID)); foundRoom {
			rm = rmUntyped.(*room)
		} else {
			http.Error(w, "room does not exist", http.StatusNotFound)
			return
		}
	} else {
		http.Error(w, "bad room ID: expected 32-bit unsigned integer as decimal string", http.StatusBadRequest)
		return
	}

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		// If we were creating a brand new room, we can go ahead and delete it since
		// the client doesn't even know the room ID yet
		if newRoom {
			s.rooms.Delete(rm.ID)
		}

		// No need to send HTTP error reply because the .Upgrade() call will send
		// an error response before it returns an error to our code
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// TODO: need to make sure we don't have race condition where room was closed
	// and we give it a connection right before it gets deleted from the main map (s.rooms)

	cli := &client{Conn: conn, Room: rm, Send: make(chan []byte, 100)}
	rm.register <- cli

	// Start read/write in new goroutine so we can return from this HTTP handler and let the
	// request and response writer (etc.) get cleaned up
	go cli.readPump()
	go cli.writePump()
}
