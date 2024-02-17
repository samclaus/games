package skull

import (
	"github.com/samclaus/games"
)

type game struct{}

func (g game) ID() string {
	return "samclaus/skull"
}

func (g game) Version() int {
	return 0
}

func (g game) NewInstance() games.GameState {
	return &gameState{}
}

func Game() games.Game {
	return game{}
}
