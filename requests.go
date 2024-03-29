package games

const (
	reqBootGame byte = iota
	reqKillGame
	reqMessageChat
)

// handleRequest should only ever be called by the room's event-processing goroutine;
// it will branch based on the request type, decide whether the given client is allowed
// to make the request (also depending on the current room state), and will then update
// room state and emit an event to all connected clients accordingly.
//
// Invalid requests are simply ignored, without sending error feedback to the client.
// Please see decodeRequest() for my explanation of why.
func (r *room) handleRequest(req request) {
	// TODO: disconnect any client that sends an invalid request structure?

	if len(req.msg) < 2 || req.msg[0] > scopeGame {
		return
	}
	if req.msg[0] == scopeGame {
		if r.currentGame != nil {
			r.currentGame.HandleRequest(r.members, req.src, req.msg[1:])
		}
		return
	}

	body := req.msg[2:]

	switch req.msg[1] {
	case reqBootGame:
		if r.currentGame != nil || len(body) == 0 {
			return
		}

		// TODO: can we just use bytes directly? Should consider places where
		// allocations can be avoided
		gameID := string(body)

		if factory := r.gameRegistry[gameID]; factory != nil {
			r.currentGameID = gameID
			r.broadcast(encodeSetGameState(gameID))
			r.currentGame = factory.NewInstance()
			r.currentGame.Init(r.members)
		}

	case reqKillGame:
		if r.currentGame == nil {
			return
		}

		r.currentGame.Deinit()
		r.currentGameID = ""
		r.currentGame = nil
		r.broadcast(encodeSetGameState(""))

	case reqMessageChat:
		if r.chat.addMessage(req.src.ID, body) {
			r.broadcast(encodeNewChatMessageState(req.src.ID, body))
		}

	}
}
