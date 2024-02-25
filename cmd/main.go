package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/samclaus/games"
	"github.com/samclaus/games/bravewength"
	"github.com/samclaus/games/skull"
)

func main() {
	mux := http.NewServeMux()
	s := games.NewServer(
		websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		bravewength.Game(nil), // use default word deck
		skull.Game(),
	)

	mux.HandleFunc("/rooms", s.HandleGetRooms)
	mux.HandleFunc("/join", s.HandleJoinRoom)

	http.ListenAndServe(":8080", mux)
}
