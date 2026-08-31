// Package jomini is the game-agnostic Paradox/Jomini script parser.
//
// It answers one question: "what is the syntax tree of this script text?" It
// produces an error-tolerant concrete syntax tree (CST) and never panics on any
// input. All offsets are UTF-8 byte indices into the source string (unlike the
// toolkit it is ported from, which used UTF-16 code units); the UTF-16 seam
// lives only at the Monaco boundary in the frontend.
//
// Files: encoding.go (Decode), lexer.go, cst.go, parse.go (Parse / ParseFile).
// The parser has no dependency on game or workspace state; game-specific key
// identity lives in the game package. Parent parser/ is a folder, not a package.
package jomini
