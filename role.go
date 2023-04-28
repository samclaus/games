package bravewength

// role is role of a player in a lobby.
type role byte

const (
	roleSpectator role = iota
	rolePurpleSeeker
	rolePurpleKnower
	roleTealSeeker
	roleTealKnower
)

func (r role) IsSeeker() bool {
	return r == rolePurpleSeeker || r == roleTealSeeker
}

func (r role) IsKnower() bool {
	return r == rolePurpleKnower || r == roleTealKnower
}

// NextRole yields the role whose turn comes after this role's.
func (r role) NextTurn() role {
	if r > rolePurpleSeeker {
		return r - 1
	}
	return roleTealKnower
}
