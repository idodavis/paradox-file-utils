// Package game holds the small, hardcoded engine grammar and per-title identity
// for the supported Paradox games (CK3, Vic3, EU5). It answers "what does this
// game call things, and where do they live?" It contains NO bundled schema/data
// dumps — all game data is derived from the install by the catalog package.
//
// Files: registry.go, detect.go, rules.go, prefix.go (ParsePrefixed / scope:),
// kind.go (KindLabel / KindHint).
package game
