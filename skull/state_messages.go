package skull

import (
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/samclaus/games"
)

// This file contains constants and serialization code for every kind of
// message a room will send to clients to update their state.

// TODO: optimized binary format and we definitely don't need to send the full
// state (especially the potentially big player UUID->role mapping) every time
// something happens

const (
	stateFull byte = iota
)

type handInfo struct {
	status handStatus
	id     uuid.UUID
	hcards uint8
	pcards uint8
	score  uint8
}

func (g *gameState) encodeFullStateMessage() []byte {
	var nhands int
	if g.phase.Active() {
		nhands = int(g.nplayers)
	} else {
		nhands = maxPlayers
	}

	msg := games.AllocGameMessage(27 + nhands*20)
	msg = append(msg, stateFull)
	msg = append(msg, byte(g.phase), g.turn, g.pcards, g.bid, g.bidder)
	msg = binary.BigEndian.AppendUint16(msg, g.passed)
	msg = append(msg, g.taker)
	msg = append(msg, g.winner[:]...)
	// TODO: need to include skull status/position on per-client basis

	for i := 0; i < nhands; i++ {
		h := &g.hands[i]

		msg = append(msg, h.id[:]...)
		msg = append(msg, h.status, h.hcards, h.pcards, h.score)
	}

	return msg
}
