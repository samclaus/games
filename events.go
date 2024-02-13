package games

import (
	"encoding/binary"

	"github.com/google/uuid"
)

// This file contains types and serialization code needed for every type of event
// *payload* the server can emit to players. Each of these payloads must be
// serialized to JSON and prefixed with a header that says the type of the
// event. This code is more tightly coupled to the room code than the
// client-to-server request code is, simply because there is not a clean way
// for me to abstract it as much without hurting performance.

const (
	roomStateConnection      byte = iota // 4-byte room ID (big endian uint32) then 16-byte UUID
	roomStateAllMembers                  // emits mapping of UUID->name as JSON
	roomStateAllChatMessages             // uint16 length of message history, followed by still available messages (each is 16-byte client UUID, 1-byte message length - 1, <message length>-byte contents)
	roomStateNewChatMessage              // 16-byte client UUID, then rest is message contents
	roomStateCurrentGame                 // emits UTF-8 encoded game ID, which may be empty (0 bytes)
)

func encodeConnectionInfoEvent(c *client, roomID uint32) []byte {
	msg := make([]byte, 0, 2+4+16)
	msg = append(msg, targetRoom, roomStateConnection)
	msg = binary.BigEndian.AppendUint32(msg, roomID)
	return append(msg, c.id[:]...)
}

func (r *room) encodeAllChatMessagesEvent() []byte {
	msg := make([]byte, 0, 2+r.chat.encodedHistoryLen())
	msg = append(msg, targetRoom, roomStateAllChatMessages)
	return r.chat.appendHistory(msg)
}

func (r *room) broadcast(msg []byte) {
	for _, c := range r.members {
		c.Send(msg)
	}
}

func (r *room) broadcastPlayerState() {
	if len(r.members) == 0 {
		return
	}

	jsonLen := 2 + 2 + len(r.members)*(2+36+1+2+1) - 1

	for _, c := range r.members {
		jsonLen += len(c.Name())
	}

	msg := make([]byte, 0, 2+jsonLen)
	msg = append(msg, targetRoom, roomStateAllMembers, '{')

	for _, c := range r.members {
		msg = append(msg, '"')
		msg = append(msg, c.ID().String()...)
		msg = append(msg, "\":\""...)
		msg = append(msg, c.Name()...)
		msg = append(msg, '"', ',')
	}

	msg = append(msg, '}')

	r.broadcast(msg)
}

func (r *room) broadcastCurrentGame() {
	msg := make([]byte, 0, 2+len(r.currentGameID))
	msg = append(msg, targetRoom, roomStateCurrentGame)
	msg = append(msg, r.currentGameID...)

	r.broadcast(msg)
}

func (r *room) broadcastNewChatMessage(src uuid.UUID, content []byte) {
	msg := make([]byte, 0, 2+16+len(content))
	msg = append(msg, targetRoom, roomStateNewChatMessage)
	msg = append(msg, src[:]...)
	msg = append(msg, content...)

	r.broadcast(msg)
}
