package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/samclaus/games"
	"github.com/samclaus/games/bravewength"
)

func main() {
	http.ListenAndServe(":8080", games.NewServer(
		websocket.Upgrader{},
		bravewength.Game,
	))
}
