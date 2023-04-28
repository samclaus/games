package bravewength

type team byte

const (
	teamNone team = iota
	teamTeal
	teamPurple
)

func (r role) Team() team {
	// TODO: optimize the enums so we can just use simple math or a table
	switch r {
	case roleSpectator:
		return teamNone
	case roleTealKnower:
		return teamTeal
	case roleTealSeeker:
		return teamTeal
	case rolePurpleKnower:
		return teamPurple
	case rolePurpleSeeker:
		return teamPurple
	}
	return teamNone
}

func (ct cardType) Team() team {
	// TODO: optimize the enums so we can just use simple math or a table
	switch ct {
	case cardTypeTeal:
		return teamTeal
	case cardTypePurple:
		return teamPurple
	}
	return teamNone
}
