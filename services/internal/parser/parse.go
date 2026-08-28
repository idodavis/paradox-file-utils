// parse.go is the error-tolerant recursive-descent parser (Parse/ParseFile). It
// never panics and always returns a usable CST plus any recovered structural errors.

package parser

import "os"

// Result is an immutable parse result. Src is the exact text that was parsed;
// all node offsets are byte indices into it.
type Result struct {
	Root     *Root
	Errors   []ParseError
	Comments []Comment
	Src      string
	lines    *LineIndex
}

// Lines returns the LineIndex for Src (built at parse time).
func (r Result) Lines() *LineIndex {
	if r.lines != nil {
		return r.lines
	}
	return NewLineIndex(r.Src)
}

// Parse parses script text into a CST. It never panics and always returns a
// usable (possibly empty) tree plus any recovered structural errors.
func Parse(src string) Result {
	li := NewLineIndex(src)
	p := &parser{text: src, tokens: tokenize(src), li: li}
	root := p.parse()
	return Result{
		Root:     root,
		Errors:   p.errors,
		Comments: p.comments,
		Src:      src,
		lines:    li,
	}
}

// ParseFile reads and decodes a file, then parses it. The BOM is stripped by
// Decode, so offsets are byte indices into the decoded text stored in Result.Src.
func ParseFile(path string) (Result, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	text, _ := Decode(raw)
	return Parse(text), nil
}

// parser holds recursive-descent state over a token stream. Comments are skipped
// transparently and recorded as they are passed.
type parser struct {
	text     string
	tokens   []token
	pos      int
	errors   []ParseError
	comments []Comment
	li       *LineIndex
}

func (p *parser) parse() *Root {
	var statements []Statement
	for !p.atEOF() {
		tok := p.peek()
		if tok.kind == tokRBrace {
			p.errors = append(p.errors, ParseError{
				Code:    ErrStrayClose,
				Message: "Unexpected '}' with no matching open brace.",
				Range:   Range{Start: tok.start, End: tok.end},
			})
			p.advance()
			continue
		}
		if st := p.parseStatement(); st != nil {
			statements = append(statements, st)
		} else if !p.atEOF() {
			p.advance()
		}
	}
	return &Root{Statements: statements, Range: Range{Start: 0, End: len(p.text)}}
}

func (p *parser) skipComments() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].kind == tokComment {
		c := p.tokens[p.pos]
		p.comments = append(p.comments, Comment{
			Text:  p.text[c.start:c.end],
			Range: Range{Start: c.start, End: c.end},
			Line:  p.li.PositionAt(c.start).Line,
		})
		p.pos++
	}
}

func (p *parser) peek() token {
	p.skipComments()
	return p.tokens[p.pos]
}

func (p *parser) peekAhead() token {
	p.skipComments()
	j := p.pos + 1
	for j < len(p.tokens) && p.tokens[j].kind == tokComment {
		j++
	}
	if j >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[j]
}

func (p *parser) advance() token {
	p.skipComments()
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

func (p *parser) atEOF() bool {
	p.skipComments()
	return p.tokens[p.pos].kind == tokEOF
}

// parseStatement := key (op value | block)? | value
func (p *parser) parseStatement() Statement {
	tok := p.peek()
	if tok.kind == tokEOF || tok.kind == tokRBrace {
		return nil
	}

	// A block starting here is an anonymous list element.
	if tok.kind == tokLBrace {
		block := p.parseBlock()
		return &ValueStmt{Value: block, Range: block.Range}
	}

	if tok.kind == tokWord || tok.kind == tokString {
		key := p.makeScalar(p.advance())
		next := p.peek()

		if next.kind == tokOp {
			opTok := p.advance()
			value := p.parseValueAfterOperator()
			if value == nil {
				p.errors = append(p.errors, ParseError{
					Code:    ErrMissingValue,
					Message: "Missing value after '" + opTok.value + "'.",
					Range:   Range{Start: opTok.start, End: opTok.end},
				})
			}
			end := opTok.end
			if value != nil {
				end = value.NodeRange().End
			}
			return &Assignment{Key: *key, Op: opTok.value, Value: value, Range: Range{Start: key.Range.Start, End: end}}
		}

		// GUI-style `key { ... }` with no operator.
		if next.kind == tokLBrace {
			block := p.parseBlock()
			return &Assignment{Key: *key, Op: "", Value: block, Range: Range{Start: key.Range.Start, End: block.Range.End}}
		}

		// Bare scalar — list element.
		return &ValueStmt{Value: key, Range: key.Range}
	}

	p.advance()
	return nil
}

// parseValueAfterOperator := scalar | block | tagged-block | operator-as-scalar.
// Returns nil if no value is parseable.
func (p *parser) parseValueAfterOperator() Value {
	tok := p.peek()

	if tok.kind == tokLBrace {
		return p.parseBlock()
	}

	// A comparison operator used as a value (list/any triggers).
	if tok.kind == tokOp {
		opTok := p.advance()
		return &Scalar{Text: p.text[opTok.start:opTok.end], Quoted: false, Range: Range{Start: opTok.start, End: opTok.end}}
	}

	if tok.kind == tokWord || tok.kind == tokString {
		// `key =\n nextKey = ...`: if this scalar is immediately followed by an
		// operator, it is the next statement's key, not this value.
		if p.peekAhead().kind == tokOp {
			return nil
		}
		scalar := p.makeScalar(p.advance())
		if p.peek().kind == tokLBrace {
			block := p.parseBlock()
			return &TaggedBlock{Tag: *scalar, Block: *block, Range: Range{Start: scalar.Range.Start, End: block.Range.End}}
		}
		return scalar
	}

	return nil
}

// parseBlock := `{` statement* `}`. On EOF before `}`, records unclosed-brace at
// the opening brace and swallows the rest of the input.
func (p *parser) parseBlock() *Block {
	open := p.advance()
	openBrace := open.start
	var statements []Statement
	closeBrace := -1
	end := open.end

	for {
		tok := p.peek()
		if tok.kind == tokEOF {
			p.errors = append(p.errors, ParseError{
				Code:    ErrUnclosedBrace,
				Message: "Unclosed '{': the rest of the file is swallowed by this block.",
				Range:   Range{Start: openBrace, End: openBrace + 1},
			})
			end = tok.start
			break
		}
		if tok.kind == tokRBrace {
			closeBrace = tok.start
			end = tok.end
			p.advance()
			break
		}
		if st := p.parseStatement(); st != nil {
			statements = append(statements, st)
		} else if k := p.peek().kind; k != tokRBrace && k != tokEOF {
			p.advance()
		}
	}

	return &Block{
		Statements: statements,
		Range:      Range{Start: openBrace, End: end},
		OpenBrace:  openBrace,
		CloseBrace: closeBrace,
	}
}

// makeScalar builds a Scalar from a word/string token, stripping quotes from
// strings (Text) while keeping the range over the quotes. Unterminated strings
// record a diagnostic and take everything after the opening quote.
func (p *parser) makeScalar(tok token) *Scalar {
	if tok.kind == tokString {
		var inner string
		if tok.unterminated {
			p.errors = append(p.errors, ParseError{
				Code:    ErrUnterminatedString,
				Message: "Unterminated string; recovered at end of line.",
				Range:   Range{Start: tok.start, End: tok.end},
			})
			inner = p.text[tok.start+1 : tok.end]
		} else {
			inner = p.text[tok.start+1 : tok.end-1]
		}
		return &Scalar{Text: inner, Quoted: true, Range: Range{Start: tok.start, End: tok.end}}
	}
	return &Scalar{Text: p.text[tok.start:tok.end], Quoted: false, Range: Range{Start: tok.start, End: tok.end}}
}
