/**
 * Shared Vue app context for PMT state.
 */
import { inject, provide, ref, type Ref } from "vue";

export type GameId = "CK3" | "EU5";

const currentGameKey = Symbol("pmt-current-game");

export function provideCurrentGame(game: Ref<GameId>): void {
  provide(currentGameKey, game);
}

export function useCurrentGame(): Ref<GameId> {
  const game = inject<Ref<GameId>>(currentGameKey);
  if (!game) {
    return ref("CK3");
  }
  return game;
}
