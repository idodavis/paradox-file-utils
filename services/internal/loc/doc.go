// Package loc parses Paradox's localization "YAML" dialect (which is not real
// YAML). It answers "what keys/values are in this loc file?" and is game-agnostic
// with no dependency on other engine packages.
//
// Shape:
//
//	l_english:
//	 key:0 "value with $vars$, [GetName], #bold#!, £gold£"
//	 other_key: "no version number"   # trailing comment
//
// Files: parse.go (this dialect parser), properties.go (STRICT/BROAD loc-key
// property sets), lang.go (language id from filename/header). Offsets are UTF-8
// byte indices into the decoded text (the BOM is stripped by Decode/ParseFile).
package loc
