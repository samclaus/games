package games

import (
	"encoding/binary"

	"github.com/google/uuid"
)

// This file contains constants and serialization code for every kind of
// message a room will send to clients to update their state.

const (
	roomStateConnection      byte = iota // 4-byte room ID (big endian uint32) then 16-byte UUID
	roomStateSetMembers                  // emits 0 or more member UUID/name pairs
	roomStateDeleteMembers               // emits 0 or more member UUIDs that disconnected
	roomStateAllChatMessages             // uint16 length of message history, followed by still available messages (each is 16-byte client UUID, 1-byte message length, <message length>-byte contents)
	roomStateNewChatMessage              // 16-byte client UUID, then rest is message contents
	roomStateCurrentGame                 // emits UTF-8 encoded game ID, which may be empty (0 bytes)
)

func encodeConnectionState(roomID uint32, clientID uuid.UUID) []byte {
	msg := make([]byte, 0, 2+4+16)
	msg = append(msg, scopeRoom, roomStateConnection)
	msg = binary.BigEndian.AppendUint32(msg, roomID)
	return append(msg, clientID[:]...)
}

func encodeSetMembersState(members []*Client) []byte {
	// 2 header bytes; each member has 16-byte UUID, 1-byte name length, then name value
	msgLen := 2 + len(members)*17
	for _, c := range members {
		msgLen += len(c.Name)
	}

	msg := make([]byte, 0, msgLen)
	msg = append(msg, scopeRoom, roomStateSetMembers)

	for _, c := range members {
		msg = append(msg, c.ID[:]...)
		msg = append(msg, byte(len(c.Name)))
		msg = append(msg, c.Name...)
	}

	return msg
}

func encodeDeleteMemberState(clientID uuid.UUID) []byte {
	msg := make([]byte, 0, 2+16)
	msg = append(msg, scopeRoom, roomStateDeleteMembers)
	return append(msg, clientID[:]...)
}

func encodeAllChatMessagesState(chat *chatBuffer) []byte {
	msg := make([]byte, 0, 2+chat.encodedHistoryLen())
	msg = append(msg, scopeRoom, roomStateAllChatMessages)
	return chat.appendHistory(msg)
}

func encodeNewChatMessageState(src uuid.UUID, content []byte) []byte {
	msg := make([]byte, 0, 2+16+len(content))
	msg = append(msg, scopeRoom, roomStateNewChatMessage)
	msg = append(msg, src[:]...)
	return append(msg, content...)
}

func encodeCurrentGameState(gameID string) []byte {
	msg := make([]byte, 0, 2+len(gameID))
	msg = append(msg, scopeRoom, roomStateCurrentGame)
	return append(msg, gameID...)
}
