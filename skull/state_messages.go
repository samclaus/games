package skull

// This file contains constants and serialization code for every kind of
// message a room will send to clients to update their state.

// TODO: optimized binary format and we definitely don't need to send the full
// state (especially the potentially big player UUID->role mapping) every time
// something happens

const ()
