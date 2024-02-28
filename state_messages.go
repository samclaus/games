package games

import (
	"encoding/binary"

	"github.com/google/uuid"
)

// This file contains constants and serialization code for every kind of
// message a room will send to clients to update their state.
//
// All integer types will be encoded big endian.
// UUIDs are always 16-bytes (no string encoding), which makes things easy.
// All strings are assumed to be UTF-8 and will be prefixed with 1 byte to
// encode the length, meaning their UTF-8 data must not exceed 255 bytes.
const (
	// Critical information for a client that has just joined the room.
	//
	// 1. uint32 room ID
	// 2. UUID client ID
	// 3. string room name
	// 4. string current game (may be empty string if no game booted)
	roomStateInit byte = iota
	// Tells clients to UPDATE their information regarding the given members,
	// i.e., do not delete information for members not included in the payload.
	//
	// 1 or more of:
	//		1. UUID client ID
	//		2. string client name
	roomStateSetMembers
	// Tells clients that the given members have left the room, i.e., disconnected.
	//
	// 1 or more of:
	//		1. UUID client ID
	roomStateDeleteMembers
	// Chat history, meaning how many messages have been sent total plus whatever
	// messages are still stored by the server.
	//
	// 1. uint16 total messages sent during life of room
	// 2. 0 or more of:
	//		1. UUID client ID that sent the message
	//		2. string message contents
	roomStateAllChatMessages
	// Tells clients that a new message was just appended to the chat.
	//
	// 1. UUID client ID that sent the message
	// 2. string message contents
	roomStateNewChatMessage
	// Tells clients that the game just changed (someone booted or killed game).
	//
	// 1. string current game ID (may be empty string if no game)
	roomStateSetGame
)

func appendStr(msg []byte, str string) []byte {
	msg = append(msg, byte(len(str)))
	return append(msg, str...)
}

func encodeInitState(r *room, clientID uuid.UUID) []byte {
	msg := make([]byte, 0, 2+4+16+1+len(r.Name)+1+len(r.currentGameID))
	msg = append(msg, scopeRoom, roomStateInit)
	msg = binary.BigEndian.AppendUint32(msg, r.ID)
	msg = append(msg, clientID[:]...)
	msg = appendStr(msg, r.Name)
	return appendStr(msg, r.currentGameID)
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
		msg = appendStr(msg, c.Name)
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
	msg := make([]byte, 0, 2+16+1+len(content))
	msg = append(msg, scopeRoom, roomStateNewChatMessage)
	msg = append(msg, src[:]...)
	msg = append(msg, byte(len(content)))
	return append(msg, content...)
}

func encodeSetGameState(gameID string) []byte {
	msg := make([]byte, 0, 2+len(gameID))
	msg = append(msg, scopeRoom, roomStateSetGame)
	return appendStr(msg, gameID)
}
