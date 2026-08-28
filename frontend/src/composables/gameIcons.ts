/**
 * Official game icons for library headings and workspace dropdown groups.
 */
import iconCk3 from "@assets/Icon_CK3.png?url";
import iconEu5 from "@assets/Icon_EUV.png?url";
import iconVic3 from "@assets/Icon_Vic3.png?url";
import type { GameId } from "../stores/workspace";

/** PNG src for each supported game. */
const GAME_ICON_SRC: Record<GameId, string> = {
  ck3: iconCk3,
  eu5: iconEu5,
  vic3: iconVic3,
};

/** Image URL for a game, or undefined when the id is unknown. */
export function gameIconSrc(gameId: string): string | undefined {
  switch (gameId) {
    case "ck3":
    case "eu5":
    case "vic3":
      return GAME_ICON_SRC[gameId];
    default:
      return undefined;
  }
}
