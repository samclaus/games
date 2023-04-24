package bravewength

import (
	"encoding/binary"
	"fmt"
	"time"

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

// A client is basically a WebSocket connection with some added metadata
// (such as the player name) and a link to the room the connection belongs
// to.
type client struct {
	// We are "extending" a WebSocket connection
	*websocket.Conn

	// The room this connection belongs to
	Room *room

	// Buffered channel of outgoing messages
	Send chan []byte
}

func (c *client) readPump() {
	defer func() {
		c.Room.unregister <- c
		c.Close()
	}()

	c.SetReadLimit(maxMessageSize)
	c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(timestamp string) error {
		then := int64(binary.BigEndian.Uint64([]byte(timestamp)))
		now := time.Now()
		fmt.Printf("Ping is %dms\n", now.UnixMilli()-then)
		c.SetReadDeadline(now.Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		if req := decodeRequest(msg); req != nil {
			c.Room.requests <- request{c, req}
		} else {
			break
		}
	}
}

func (c *client) writePump() {
	pingTicker := time.NewTicker(pingInterval)

	defer func() {
		pingTicker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, chanStillOpen := <-c.Send:
			c.SetWriteDeadline(time.Now().Add(sendToClientWait))

			// The room can decide to kill this connection by closing our send channel,
			// which is potentially useful for situations where the server is overloaded
			// or a client is behaving weirdly.
			//
			// Calling Close() on the WebSocket does NOT send a 'proper' close message
			// to the client, so we do it here (otherwise the client would see it as
			// an abnormal closure because the connection would just die without warning)
			if !chanStillOpen {
				c.WriteMessage(websocket.CloseMessage, nil)
				return
			}

			if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-pingTicker.C:
			now := time.Now()

			var timestampBuff [8]byte
			binary.BigEndian.PutUint64(timestampBuff[:], uint64(now.UnixMilli()))

			if err := c.WriteControl(websocket.PingMessage, timestampBuff[:], now.Add(sendToClientWait)); err != nil {
				return
			}
		}
	}
}
