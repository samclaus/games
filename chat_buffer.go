package games

import (
	"encoding/binary"

	"github.com/google/uuid"
)

const (
	maxScrollback = 50                     // must be 255 or less because we only use 1 byte for message count
	maxMessageLen = 100                    // must be 255 or less because we only use 1 byte for message length
	lineLen       = 16 + 1 + maxMessageLen // 16-byte client UUID, message length, message capacity
)

type chatBuffer struct {
	buff [maxScrollback * lineLen]byte
	hist uint16
}

func (cb *chatBuffer) numMessages() int {
	if cb.hist > maxScrollback {
		return maxScrollback
	}
	return int(cb.hist)
}

func (cb *chatBuffer) addMessage(clientID uuid.UUID, msg []byte) bool {
	if len(msg) < 1 || len(msg) > maxMessageLen {
		return false
	}

	pos := (cb.hist % maxScrollback) * lineLen
	copy(cb.buff[pos:pos+16], clientID[:])
	cb.buff[pos+16] = byte(len(msg))
	copy(cb.buff[pos+17:pos+17+uint16(len(msg))], msg)
	cb.hist++

	return true
}

// Calculate size of encoded history (number of bytes required) up-front to
// avoid wasteful memory allocations due to automatic slice growth
func (cb *chatBuffer) encodedHistoryLen() int {
	numMessages := cb.numMessages()

	// 2 bytes for uint16 length of message HISTORY (not scrollback) +
	// <number of retained messages> * (16 bytes for client UUID + 1 byte for message length)
	encLen := 2 + numMessages*(16+1)

	for i := 0; i < numMessages; i++ {
		encLen += int(cb.buff[i*lineLen+16])
	}

	return encLen
}

func (cb *chatBuffer) appendHistory(dst []byte) []byte {
	dst = binary.BigEndian.AppendUint16(dst, cb.hist)

	currentLine := int(cb.hist % maxScrollback)
	numMessages := cb.numMessages()

	// Start from the current offset
	for i := currentLine; i < numMessages; i++ {
		pos := i * lineLen
		msgLen := cb.buff[pos+16]

		dst = append(dst, cb.buff[pos:pos+16]...)                // client UUID
		dst = append(dst, msgLen)                                // message length
		dst = append(dst, cb.buff[pos+17:pos+17+int(msgLen)]...) // message contents
	}

	// In case we have more than <maxScrollback> messages, we need to start
	// from beginning of buffer and work our way to the current line to get
	// the newest messages
	for i := 0; i < currentLine; i++ {
		pos := i * lineLen
		msgLen := cb.buff[pos+16]

		dst = append(dst, cb.buff[pos:pos+16]...)                // client UUID
		dst = append(dst, msgLen)                                // message length
		dst = append(dst, cb.buff[pos+17:pos+17+int(msgLen)]...) // message contents
	}

	return dst
}
