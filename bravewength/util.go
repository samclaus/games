package bravewength

import (
	"encoding/json"
	"fmt"
)

// mustEncodeJSON calls json.Marshal and panics if it returns an error. This is a
// convenience function because we have many data types that need to be encoded and
// failing to marshal them as JSON (due to a cyclic field or something) is a game-
// breaking problem anyways.
func mustEncodeJSON(val any) []byte {
	utf8, err := json.Marshal(val)
	if err != nil {
		panic(fmt.Sprintf("mustEncodeJSON: failed to encode: %v", err))
	}
	return utf8
}
