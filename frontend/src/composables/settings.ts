/**
 * Shared helpers for settings maps returned by the Go backend.
 */

/** Drop undefined values from a settings map. */
export function normalizeSettings(
  input: Record<string, string | undefined> | null | undefined,
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(input ?? {}).filter(
      (entry): entry is [string, string] => entry[1] !== undefined,
    ),
  );
}
