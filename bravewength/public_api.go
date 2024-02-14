package bravewength

import (
	"github.com/google/uuid"
	"github.com/samclaus/games"
)

type game struct{}

func (g game) ID() string {
	return "samclaus/bravewength"
}

func (g game) Version() int {
	return 0
}

func (g game) NewInstance() games.GameState {
	instance := &gameState{
		roles: make(map[uuid.UUID]role),
	}
	instance.newGame()

	return instance
}

var Game games.Game = game{}
