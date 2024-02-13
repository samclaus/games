package games

import (
	"encoding/binary"

	"github.com/google/uuid"
)

// This file contains constants and serialization code for every kind of
// message a room will send to clients to update their state.

const (
	roomStateConnection      byte = iota // 4-byte room ID (big endian uint32) then 16-byte UUID
	roomStateAllMembers                  // emits mapping of UUID->name as JSON
	roomStateSetMember                   // emits name for single member to indicate join or rename
	roomStateDeleteMember                // emits UUID of player that disconnected
	roomStateAllChatMessages             // uint16 length of message history, followed by still available messages (each is 16-byte client UUID, 1-byte message length - 1, <message length>-byte contents)
	roomStateNewChatMessage              // 16-byte client UUID, then rest is message contents
	roomStateCurrentGame                 // emits UTF-8 encoded game ID, which may be empty (0 bytes)
)

func encodeConnectionState(roomID uint32, clientID uuid.UUID) []byte {
	msg := make([]byte, 0, 2+4+16)
	msg = append(msg, targetRoom, roomStateConnection)
	msg = binary.BigEndian.AppendUint32(msg, roomID)
	return append(msg, clientID[:]...)
}

func encodeAllMembersState(members []Client) []byte {
	if len(members) == 0 {
		return []byte{targetRoom, roomStateAllMembers, '{', '}'}
	}

	jsonLen := 2 + 2 + len(members)*(2+36+1+2+1) - 1

	for _, c := range members {
		jsonLen += len(c.Name())
	}

	msg := make([]byte, 0, 2+jsonLen)
	msg = append(msg, targetRoom, roomStateAllMembers, '{')

	for _, c := range members {
		msg = append(msg, '"')
		msg = append(msg, c.ID().String()...)
		msg = append(msg, "\":\""...)
		msg = append(msg, c.Name()...)
		msg = append(msg, '"', ',')
	}

	// Overwrite the last comma
	msg[len(msg)-1] = '}'

	return msg
}

func encodeSetMemberState(clientID uuid.UUID, clientName []byte) []byte {
	msg := make([]byte, 0, 2+16+len(clientName))
	msg = append(msg, targetRoom, roomStateSetMember)
	msg = append(msg, clientID[:]...)
	return append(msg, clientName...)
}

func encodeDeleteMemberState(clientID uuid.UUID) []byte {
	msg := make([]byte, 0, 2+16)
	msg = append(msg, targetRoom, roomStateDeleteMember)
	return append(msg, clientID[:]...)
}

func encodeAllChatMessagesState(chat *chatBuffer) []byte {
	msg := make([]byte, 0, 2+chat.encodedHistoryLen())
	msg = append(msg, targetRoom, roomStateAllChatMessages)
	return chat.appendHistory(msg)
}

func encodeNewChatMessageState(src uuid.UUID, content []byte) []byte {
	msg := make([]byte, 0, 2+16+len(content))
	msg = append(msg, targetRoom, roomStateNewChatMessage)
	msg = append(msg, src[:]...)
	return append(msg, content...)
}

func encodeCurrentGameState(gameID string) []byte {
	msg := make([]byte, 0, 2+len(gameID))
	msg = append(msg, targetRoom, roomStateCurrentGame)
	return append(msg, gameID...)
}
