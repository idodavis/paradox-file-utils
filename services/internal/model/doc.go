// Package model builds and persists the semantic model of a game + workspace. It
// answers "what objects/refs/edges exist?" by scanning the game install into a
// Cache and indexing a workspace into an Index, then saving both to user-data.
// All game knowledge is DERIVED here from the install — nothing is bundled. It
// imports the leaf packages parser, loc, and game.
//
// Files: types.go (Def/Cache/Index + format versions), scan.go (install ->
// Cache), index.go (workspace -> Index), override.go (FIOS/LIOS winner),
// persist.go (save/load, reject wrong format version).
package model
