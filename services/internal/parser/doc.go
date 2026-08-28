// Package parser is the game-agnostic Paradox/Jomini script parser.
//
// It answers one question: "what is the syntax tree of this script text?" It
// produces an error-tolerant concrete syntax tree (CST) and never panics on any
// input. All offsets are UTF-8 byte indices into the source string (unlike the
// toolkit it is ported from, which used UTF-16 code units); the UTF-16 seam
// lives only at the Monaco boundary in the frontend.
//
// Files in this package:
//   - encoding.go — decode raw bytes to text (UTF-8 BOM strip, latin1 fallback)
//   - lexer.go    — single-pass, allocation-light tokenizer
//   - cst.go      — node types, LineIndex, walk/lookup helpers, TopAssignments
//   - parse.go    — recursive-descent parser (Parse / ParseFile)
//
// The parser has no dependency on game or workspace state; game-specific key
// identity (EU5 MODE: prefixes) lives in the game package, not here.
package parser
