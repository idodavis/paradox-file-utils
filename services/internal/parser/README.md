# parser

Game-agnostic Paradox/Jomini **script** parser. Answers one question: *"what is
the syntax tree of this script text?"*

It is a leaf package in the layered engine (imports nothing from other engine
packages). `model`, `session`, `lsp`, `graph`, and the merge service all build on
it.

## Contracts

- **Never panics** on any input; it is error-tolerant and always returns a usable
  (possibly empty) CST plus a list of recovered `ParseError`s.
- **UTF-8 byte offsets** end-to-end. Every `Range` is a byte span into the source
  string. (The toolkit this is ported from used UTF-16; the UTF-16 seam lives only
  at the Monaco boundary in the frontend.)
- **Never reformats** — the CST preserves exact source; callers slice `Result.Src`.
- `TopAssignments` returns root-level `Assignment` nodes only (not bare list
  elements). GUI `template Foo { ... }` yields one entry keyed `Foo`; the leading
  `template` word is a skipped `ValueStmt` that callers capture via gap-slicing.

## Files

| File          | Responsibility |
| ------------- | -------------- |
| `doc.go`      | package doc comment (godoc) |
| `encoding.go` | decode raw bytes to text: UTF-8 BOM strip, latin1 fallback |
| `lexer.go`    | single-pass, allocation-light tokenizer |
| `cst.go`      | node types, `LineIndex`, walk/lookup helpers, `TopAssignments` |
| `parse.go`    | recursive-descent parser: `Parse`, `ParseFile` |

## Ported from

Toolkit `packages/server/src/parser/{cst,lexer,parser,encoding}.ts`, converted to
UTF-8 byte offsets and idiomatic Go.
