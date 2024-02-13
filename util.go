package games

import "bytes"

// TakeNullTerminatedString searches a slice for the first 0 (aka NULL) byte,
// returning a subslice containing everything before it, followed by a subslice
// containing everything after/remaining. If the given slice does not contain
// the value 0, returns nil followed by the entire unmodified slice.
func TakeNullTerminatedString(payload []byte) (str []byte, remaining []byte) {
	if i := bytes.IndexByte(payload, 0); i >= 0 {
		return payload[:i], payload[i+1:]
	}
	return nil, payload
}
