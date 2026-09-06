// Package game is the only game-aware package in the backend. It answers "what
// does this title call things, and where do they live?" for CK3, Victoria 3 and
// EU5, and nothing else needs a game id to read Paradox script.
//
// It contains NO bundled schema or data dumps — the type system comes from the
// game's own script_docs and the object databases from the install walk, both
// read by the catalog package.
//
// Two files:
//
//   - registry.go — the GameInfo table plus the rules that read it: folder-to-
//     extract mapping, FIOS, entry-mode prefixes, key identity.
//   - detect.go — install discovery, user-data folders and install mtime, all
//     keyed off the same table.
//
// The membership test is behavioural, not syntactic: a function belongs here
// only if it acts differently on CK3, Victoria 3 and EU5. Everything that
// merely takes a gameID without branching on it is shared Clausewitz/Jomini
// grammar and lives in parser/jomini (kinds, script slots, typed prefixes,
// ephemeral names) or parser/loc, and mod packaging lives in modfile.
package game
