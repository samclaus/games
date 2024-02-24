package games

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// NOTE: these constants and almost all of the readPump/writePump code are ripped
// straight from https://github.com/gorilla/websocket/blob/master/examples/chat/client.go
// (credit and much gratitude to the Gorilla toolkit authors for elegant design)
const (
	// Time allowed to write a message to a client
	sendToClientWait = 10 * time.Second

	// Time allowed to read a pong message from a client (after sending a ping);
	// ping-pong is used to make sure the client is still responsive even when
	// game-related messages aren't being sent back and forth, so a dead connection
	// can be detected (so the OS can clean up the TCP connection and thus trigger
	// our WebSocket close handling code) sooner rather than later
	pongWait = 60 * time.Second

	// Interval at which to send pings to a client; must be less than pongWait, for
	// reasons I don't fully understand
	pingInterval = 50 * time.Second

	// Most requests from clients should not be very large; this should be enough to
	// accomodate a paragraph of Chinese (or another language with large UTF-8 encoding)
	// in the chat
	maxMessageSize = 512
)

// Client corresponds to a single WebSocket connection. UUIDs are used for very
// barebones identity management, so that if a player disconnects, they can
// reconnect as the "same person".
type Client struct {

	// ID is the UUID of the player the connection is associated with. Must be
	// threadsafe, i.e., this field should be treated as read-only and mutating
	// it will introduce race conditions.
	ID uuid.UUID

	// Name is the player's display name that they provided when opening the
	// WebSocket. Must be threadsafe, i.e., this field should be treated as
	// read-only and mutating it will introduce race conditions.
	Name string

	conn *websocket.Conn
	room *room       // The room this connection belongs to
	send chan []byte // Buffered channel of outgoing messages

}

// Send attempts to send a message to the client, kicking the client from the
// room if the client's send channel is full/blocked. THIS IS ONLY SAFE TO CALL
// FROM THE ROOM'S PROCESSING GOROUTINE!
func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		// If this client's send channel, which uses a sizeable buffer,
		// is blocked, it means this client is being way too slow to
		// receive events and needs to be disconnected so we can reclaim
		// resources (the game would literally be unplayable for the user)
		c.room.removeMember(c)
	}
}

func (c *Client) readPump() {
	defer func() {
		c.room.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(timestamp string) error {
		then := int64(binary.BigEndian.Uint64([]byte(timestamp)))
		now := time.Now()
		fmt.Printf("Ping is %dms\n", now.UnixMilli()-then)
		c.conn.SetReadDeadline(now.Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		c.room.requests <- request{c, msg}
	}
}

func (c *Client) writePump() {
	pingTicker := time.NewTicker(pingInterval)

	defer func() {
		pingTicker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, chanStillOpen := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(sendToClientWait))

			// The room can decide to kill this connection by closing our send channel,
			// which is potentially useful for situations where the server is overloaded
			// or a client is behaving weirdly.
			//
			// Calling Close() on the WebSocket does NOT send a 'proper' close message
			// to the client, so we do it here (otherwise the client would see it as
			// an abnormal closure because the connection would just die without warning)
			if !chanStillOpen {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}

			if err := c.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return
			}
		case <-pingTicker.C:
			now := time.Now()

			var timestampBuff [8]byte
			binary.BigEndian.PutUint64(timestampBuff[:], uint64(now.UnixMilli()))

			if err := c.conn.WriteControl(websocket.PingMessage, timestampBuff[:], now.Add(sendToClientWait)); err != nil {
				return
			}
		}
	}
}
