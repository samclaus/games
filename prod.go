//go:build !debug

package games

func debug(format string, args ...any) {}

func (r *room) debug(format string, args ...any) {}
