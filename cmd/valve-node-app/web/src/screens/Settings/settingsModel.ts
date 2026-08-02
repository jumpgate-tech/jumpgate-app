// Pure derivation for the Settings screen (#/settings), extracted from
// settings.ts's keyNames()/providerKeyNote() so it can be unit-tested without
// mounting the screen.

// VALVE_API_KEY gets a row whether or not one is stored: it is the slot the
// app itself always wants filled, and a row that only appears once a key
// exists is a row nobody can use to enter their first one.
export const VALVE_KEY = "VALVE_API_KEY";

// keyNames is every provider-key row to draw: the placeholders the server
// says hold a key, plus VALVE_API_KEY whether or not it does — always first,
// with the rest sorted after it. `providerKeysSet` is coalesced to [] before
// use: a nil slice from an older binary serialises as JSON null, and a
// screen that called .filter on that directly would take the whole render
// down with it.
export function keyNames(providerKeysSet: string[] | null | undefined): string[] {
  const set = Array.isArray(providerKeysSet) ? providerKeysSet : [];
  const rest = set.filter((n) => n !== VALVE_KEY).sort();
  return [VALVE_KEY, ...rest];
}
