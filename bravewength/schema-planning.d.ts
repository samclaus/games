/**
 * REMEMBER:
 * 
 * Player information (name, has host permissions, online, team, role for each)
 * Players can change team, but only change role if they have not picked knower and seen cards
 * General a11y like nice keyboard navigation
 * Ability to view game rules
 * Full game log (every action: giving a clue, picking a card, X changed to Y team, new round, etc.)
 * Checkbox to say "I am giving clues externally" which enables "Give clue" button even if clue is empty
 * Reset game log when starting new round?
 * i18n for the UI
 */

/**
 * Settings related to the user interface. These can optionally be saved to the browser's local storage,
 * so that the game remembers your theme and accessibility preferences, and vice versa.
 */
interface ClientSettings {
    /**
     * True means dark, false means light, and undefined (the default) means to theme according to the
     * user's OS preferences (via the CSS prefers-color-scheme media query).
     */
    dark_mode: boolean | undefined;
    /**
     * Whether to show additional markers (such as shapes) rather than just colors to distinguish between
     * teams. Defaults to false.
     */
    colorblind: boolean;
    /**
     * Disables the checkbox that you normally have to click to enable giving an empty clue. This is just
     * for players who routinely use voice and tire of clicking the "I am giving my clues through voice"
     * confirmation every time a new lobby is started. Defaults to false.
     */
    assume_voice_clues: boolean;
}

/**
 * All of the state for an active game room. Usually, the full state will be sent to a client when it first
 * connects, and then incremental updates will be sent after that. For instance, we don't want to send the
 * entire chat history to every client each time someone types a message in the chat--we would just send
 * the new message and each client would add it to the UI, rather than re-rendering the chat from scratch.
 */
interface RoomState {
    /**
     * Should spectators be able to join a team, or current players be able to switch to the
     * other team, while a round is in-progress?
     * 
     * NOTE: reconnections (connections which authenticate as an existing player) are still
     * allowed, assuming the given player WAS disconnected. 
     * 
     * NOTE: knowers are never allowed to switch to seekers during a round. This is because
     * knowers see the cards (or rather, which cards are which); spectators, on the other
     * hand, do not see the cards.
     */
    lock_teams_during_game: boolean;
    spectators: string[];
    purple_team_seekers: string[];
    purple_team_knowers: string[];
    teal_team_seekers: string[];
    teal_team_knowers: string[];
    game?: GameState;
}

declare const enum Role {
    Spectator,
	PurpleSeeker,
	PurpleKnower,
	TealSeeker,
	TealKnower,
}

interface GameState {
    /**
     * Whose turn it is; will NEVER be Role.Spectator (0).
     */
    turn: Role;
    clue_text: string;
    clue_count: number;
}
