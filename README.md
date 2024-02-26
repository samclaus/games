This Go package implements a WebSocket-based
server system for turn-based games. It
provides some simple interfaces so that the
server code for various game implementations
can focus on game logic while this package
takes care of all the asynchronous
connection management ans provides some
niceties like game _rooms_ and chat.