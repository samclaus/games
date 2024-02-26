//go:build debug

package games

import "fmt"

func debug(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func (r *room) debug(format string, args ...any) {
	fmt.Printf(fmt.Sprintf("[Room %d] ", r.ID)+format+"\n", args...)
}
