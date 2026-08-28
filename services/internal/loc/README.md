# loc

Parser for Paradox's localization "YAML" dialect (which is **not** real YAML).
Answers *"what keys/values are in this loc file?"*

Leaf package in the layered engine (no engine dependencies). Used by `model`
(loc-key defs, English strings), `lsp` (loc diagnostics/hover/complete), and
`graph` (loc coverage).

## Dialect shape

```
l_english:
 key:0 "value with $vars$, [GetName], #bold#!, £gold£"
 other_key: "no version number"   # trailing comment
```

## Contracts

- **Never panics.** Recovers per-line and records typed errors
  (`no-header`, `bad-entry`, `tab-indent`, `unterminated-value`,
  `content-before-header`).
- **Last-quote-wins:** a value runs from the opening quote to the *last* quote on
  the line; inner quotes are literal (vanilla wraps direct speech as `""…""`).
- **UTF-8 byte offsets** into the decoded text (BOM stripped by `ParseFile`).

## Files

| File            | Responsibility |
| --------------- | -------------- |
| `doc.go`        | package doc comment (godoc) |
| `parse.go`      | the loc dialect parser (`Parse`, `ParseFile`) |
| `properties.go` | STRICT/BROAD sets of script properties that hold loc keys |
| `lang.go`       | language id from a loc filename |

## Ported from

Toolkit `packages/server/src/parser/locParser.ts` and
`packages/protocol/src/locProperties.ts`, converted to UTF-8 byte offsets.
