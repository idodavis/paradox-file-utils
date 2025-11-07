package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// ///////////////////////////////////////
// Lexer (Used for parsing the input file)
// ///////////////////////////////////////
var (
	paradoxLexer = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "BOM", Pattern: "\uFEFF"},
		{Name: "Comment", Pattern: `#[^\r\n]*`},
		{Name: "WS", Pattern: `[\s\r\n]+`},

		{Name: "LBRACE", Pattern: `{`},
		{Name: "RBRACE", Pattern: `}`},
		{Name: "HSV", Pattern: `hsv`},
		{Name: "RGB", Pattern: `rgb`},

		{Name: "Boolean", Pattern: `(yes|no)`},

		{Name: "LTE", Pattern: `<=`},
		{Name: "GTE", Pattern: `>=`},
		{Name: "NEQ", Pattern: `!=`},
		{Name: "QEQ", Pattern: `\?=`},

		{Name: "Ident", Pattern: `@?[\p{L}\p{N}_.$:'-]+`},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		{Name: "String", Pattern: `"(\\"|[^"])*"`},

		{Name: "EQ", Pattern: `=`},
		{Name: "LT", Pattern: `<`},
		{Name: "GT", Pattern: `>`},
	})

	paradoxParser = participle.MustBuild[ParadoxFile](
		participle.Lexer(paradoxLexer),
		participle.Unquote("String"),
		participle.UseLookahead(participle.MaxLookahead),
	)
)

// ////////////////////////////////////////////////////
// Parser Structs (Used for parsing into different types)
// ////////////////////////////////////////////////////

// Base struct for all parser nodes
type Node struct {
	Pos    lexer.Position
	Tokens []lexer.Token
}

type Literal struct {
	Node

	Boolean    *string    `parser:"('yes'|'no')"`
	Identifier *string    `parser:"| @Ident"`
	String     *string    `parser:"| @String"`
	Number     *float64   `parser:"| @Number"`
	Array      []*Literal `parser:"| '{' WS? (@@ WS)+ WS? '}'"`
	Color      *string    `parser:"| ('rgb'|'hsv')? WS? '{' WS @Number WS @Number WS @Number ( WS @Number)? WS? '}'"`
}

type ObjectEntry struct {
	Node

	Assignment *Assignment `parser:"@@"`
	Literal    *Literal    `parser:"| @@"`
	Object     *Object     `parser:"| @@"`
	Comment    string      `parser:"| @Comment"`
}

type Object struct {
	Node

	Entries []*ObjectEntry `parser:"'{' WS* (@@ WS*)* WS* '}'"`
}

type Assignment struct {
	Node

	Key      string   `parser:"@Ident WS* (@Ident)?"`
	Operator *string  `parser:"WS* @(LTE|GTE|NEQ|QEQ|EQ|LT|GT) WS*"`
	Literal  *Literal `parser:"( @@"`
	Object   *Object  `parser:"| @@ )"`
}

type Namespace struct {
	Node

	Key   string `parser:"'namespace' WS? @EQ"`
	Value string `parser:"WS? @Ident"`
}

type Entry struct {
	Node

	Namespace  *Namespace  `parser:"@@"`
	Assignment *Assignment `parser:"| @@"`
	Comment    string      `parser:"| @Comment"`
	Whitespace string      `parser:"| @WS"`
}

type ParadoxFile struct {
	Node

	BOM     string   `parser:"@BOM?"`
	Entries []*Entry `parser:"@@*"`
}

// ////////////////////////////////////////////////////
// Raw text method - extract original text from source
// ////////////////////////////////////////////////////
func (node *Node) GetRawText() string {
	var sb strings.Builder
	for _, tok := range node.Tokens {
		sb.WriteString(tok.Value)
	}
	return sb.String()
}

// /////////////////////////////////////
// Pretty formatted methods (JSON-like)
// /////////////////////////////////////
func (l *Literal) ToPrettyString() string {
	switch {
	case l.Boolean != nil:
		return *l.Boolean
	case l.Identifier != nil:
		return *l.Identifier
	case l.String != nil:
		return fmt.Sprintf("%q", *l.String)
	case l.Number != nil:
		return fmt.Sprintf("%v", *l.Number)
	case l.Array != nil:
		parts := make([]string, len(l.Array))
		for i, e := range l.Array {
			parts[i] = e.ToPrettyString()
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return "null"
}

func (o *Object) ToPrettyString() string {
	if o == nil {
		return "{}"
	}
	if len(o.Entries) == 0 {
		return "{}"
	}

	nextIndent := "	"
	parts := make([]string, 0, len(o.Entries))

	for _, e := range o.Entries {
		if e.Assignment != nil {
			parts = append(parts, nextIndent+e.Assignment.ToPrettyString())
		} else if e.Literal != nil {
			parts = append(parts, nextIndent+e.Literal.ToPrettyString())
		} else if e.Object != nil {
			parts = append(parts, nextIndent+e.Object.ToPrettyString())
		} else if e.Comment != "" {
			parts = append(parts, nextIndent+"# "+strings.TrimPrefix(e.Comment, "#"))
		}
	}

	return "{\n" + strings.Join(parts, ",\n") + "\n" + "	" + "}"
}

func (a *Assignment) ToPrettyString() string {
	if a == nil {
		return ""
	}
	op := "="
	if a.Operator != nil {
		op = *a.Operator
	}

	key := a.Key
	value := ""
	if a.Literal != nil {
		value = a.Literal.ToPrettyString()
	} else if a.Object != nil {
		value = a.Object.ToPrettyString()
	}

	return fmt.Sprintf("%s %s %s", key, op, value)
}

func (n *Namespace) ToPrettyString() string {
	return fmt.Sprintf("namespace = %s", n.Value)
}

func (e *Entry) ToPrettyString() string {
	if e.Assignment != nil {
		return e.Assignment.ToPrettyString()
	}
	if e.Namespace != nil {
		return e.Namespace.ToPrettyString()
	}
	if e.Comment != "" {
		return "# " + strings.TrimPrefix(e.Comment, "#")
	}
	return ""
}

// ///////////////
// Parser Methods
// ///////////////
func ParseFile(filename string) (*ParadoxFile, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	paradoxScript, err := paradoxParser.Parse("", r)
	if err != nil {
		return nil, err
	}
	return paradoxScript, nil
}

func ParseString(input string) (*ParadoxFile, error) {
	return paradoxParser.ParseString("", input)
}
