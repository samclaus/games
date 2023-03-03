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
