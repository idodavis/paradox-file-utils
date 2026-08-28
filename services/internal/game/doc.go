// Package game holds the small, hardcoded engine grammar and per-title identity
// for the supported Paradox games (CK3, Vic3, EU5). It answers "what does this
// game call things, and where do they live?" It contains NO bundled schema/data
// dumps — all game data is derived from the install by the model package. Only
// engine grammar (extract overrides, call kinds, FIOS, key identity, scope links)
// and static identity (paths, Steam ids, wiki endpoints) live here.
//
// Files: registry.go (GameInfo table + Get), detect.go (install/version/mod-root
// detection), rules.go (extract overrides, CALL_KINDS, FIOS, KeyIdentity).
package game
