/**
 * Shared Vue app context for PMT state.
 */
import { inject, provide, ref, type Ref } from "vue";

/** Supported Paradox game identifiers. */
export type GameId = "CK3" | "EU5";

const currentGameKey = Symbol("pmt-current-game");

/** Provide the selected game ref to descendant components. */
export function provideCurrentGame(game: Ref<GameId>): void {
  provide(currentGameKey, game);
}

/** Read the selected game ref, defaulting to CK3 outside the app tree. */
export function useCurrentGame(): Ref<GameId> {
  const game = inject<Ref<GameId>>(currentGameKey);
  if (!game) {
    return ref("CK3");
  }
  return game;
}
