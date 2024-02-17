package skull

// This file contains types and deserialization code for every type of request a player can
// make to the server.

import (
	"github.com/samclaus/games"
)

const (
	reqJoinGame byte = iota
	reqLeaveGame
	reqRestartGame
	reqAbortGame
	reqPlay
	reqBid
	reqPass
	reqPick
)

// HandleRequest is required to satisfy the (github.com/samclaus/games).GameState interface and
// implements all turn-based game logic for Skull.
//
// TODO: tell room to disconnect client for sending invalid request structure?
func (g *gameState) HandleRequest(players []games.Client, src games.Client, payload []byte) {
	if len(payload) == 0 {
		return
	}

	body := payload[1:]
	srcID := src.ID()

	switch payload[0] {
	case reqJoinGame:
		if len(body) != 1 || body[0] > 5 {
			return
		}

		requestPos := body[0] // guaranteed range [0, 5] by check above
		existingPos := -1

		for i, hand := range g.hands {
			if hand.status != statusUnclaimed && hand.id == srcID {
				existingPos = i
				break
			}
		}

		hand := &g.hands[requestPos]

		// 1. Rejoining as same position does nothing
		// 2. Cannot take a hand if we have one and game is active (even if we left it)
		// 2. Cannot insert new player into active game
		// 3. Cannot take over existing player's hand unless they surrender it
		if (existingPos >= 0 && byte(existingPos) == requestPos) ||
			(existingPos >= 0 && g.phase.Active()) ||
			(g.phase.Active() && hand.status == statusUnclaimed) ||
			hand.status == statusClaimed {
			return
		}

		hand.status = statusClaimed
		hand.id = srcID

		if existingPos >= 0 {
			// We know the game must not be active from checks above, so this basically
			// means we are just switching our play order before the game begins
			g.hands[existingPos].status = statusUnclaimed
		}

		// TODO: broadcast new game state

	case reqLeaveGame:
		pos := -1
		for i, hand := range g.hands {
			if hand.status != statusUnclaimed && hand.id == srcID {
				pos = i
				break
			}
		}

		if pos < 0 {
			return
		}

		if g.phase.Active() {
			g.hands[pos].status = statusLeft
		} else {
			g.hands[pos].status = statusUnclaimed
		}

		// TODO: broadcast new game state

	case reqRestartGame:
		g.phase = phasePlay
		g.turn = 0
		g.lockInPlayers()

		// TODO: broadcast new game state

	case reqAbortGame:
		if !g.phase.Active() {
			return
		}

		g.phase = phaseAborted

		// TODO: broadcast new game state

	case reqPlay:
		pos, hand := g.getHand(srcID)

		// Ignore request if:
		// - Game is not in play phase
		// - It is not their turn (also handles -1 position case meaning they don't have a hand)
		// - They did provide a valid card index to play
		if g.phase != phasePlay ||
			g.turn != pos ||
			len(body) != 1 ||
			body[0] >= byte(hand.hcards) {
			return
		}

		cardIndex := uint8(body[0])

		if hand.skullStatus == skullInHand {
			if cardIndex == hand.skullPos {
				hand.skullStatus = skullPlayed // they are playing the skull
				hand.skullPos = hand.pcards    // now skull position is in played cards
			} else if cardIndex < hand.skullPos {
				hand.skullPos-- // shift position to account for card removal
			}
		}

		hand.pcards++
		hand.hcards--

		g.nextTurn()

		// TODO: broadcast new game state

	case reqBid:
		pos, _ := g.getHand(srcID)

		// Ignore request if:
		// 1. Game is not in play OR bid phase
		// 2. It is not their turn (also handles -1 position case meaning they don't have a hand)
		// 3. They did not provide a bid (invalid request)
		// 4. They did not bid higher than current bid (also handles case where they start bidding)
		if !(g.phase == phasePlay || g.phase == phaseBid) ||
			g.turn != pos ||
			len(body) != 1 ||
			body[0] <= g.bid {
			return
		}

		g.phase = phaseBid // in case they are starting the bid
		g.bid = body[0]
		g.bidder = pos
		g.nextTurn()

		// TODO: broadcast state update

	case reqPass:
		pos, _ := g.getHand(srcID)

		// Ignore request if:
		// 1. Game is not in bid phase
		// 2. It is not their turn (also handles -1 position case meaning they don't have a hand)
		if g.phase != phaseBid || g.turn != pos {
			return
		}

		g.nextTurn()

		// If we have cycled all the way back to the most recent bidder,
		// the game enters the picking phase where that bidder must
		// successfully pick the number of cards they bid!
		if g.turn == g.bidder {
			g.phase = phasePick
		}

		// TODO: broadcast state update

	case reqPick:
		pos, _ := g.getHand(srcID)

		// Ignore request if:
		// 1. Game is not in pick phase
		// 2. It is not their turn (also handles -1 position case meaning they don't have a hand)
		// 3. They did not provide a hand index to take a card from (invalid request)
		// 4. They provided an invalid hand index (too high)
		if g.phase != phasePick ||
			g.turn != pos ||
			len(body) != 1 ||
			int(body[0]) >= g.nplayers {
			return
		}

		pickedHand := g.hands[body[0]]

		if pickedHand.pcards == 0 {
			return
		}

		if pickedHand.skullStatus == skullPlayed &&
			pickedHand.skullPos == (pickedHand.pcards-1) {
			// TODO: they just picked someone's skull
		}

		// TODO

		// TODO: broadcast state update

	}
}
