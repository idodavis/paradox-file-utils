package parser

import (
	"fmt"
	"os"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var (
	paradoxLexer = lexer.MustSimple([]lexer.SimpleRule{
		{Name: "BOM", Pattern: "\uFEFF"},

		{Name: "EQ", Pattern: `=`},
		{Name: "LT", Pattern: `<`},
		{Name: "GT", Pattern: `>`},
		{Name: "QEQ", Pattern: `\?=`},
		{Name: "LTE", Pattern: `<=`},
		{Name: "GTE", Pattern: `>=`},
		{Name: "NEQ", Pattern: `!=`},

		{Name: "Ident", Pattern: `[\p{L}\p{N}_.$:'-]+`},
		{Name: "Number", Pattern: `[-+]?(\d*\.)?\d+`},
		{Name: "String", Pattern: `"(\\"|[^"])*"`},

		{Name: "Comment", Pattern: `#+[^\r\n]*`},
		{Name: "WS", Pattern: `[ \t\r\n]+`},
	})

	paradoxParser = participle.MustBuild[ParadoxFile](
		participle.Lexer(paradoxLexer),
		participle.Unquote("String"),
		participle.UseLookahead(2),
	)
)

type Positioned struct {
	Pos lexer.Position
}

type Value struct {
	Positioned

	HSVColor   *string  `parser:"'hsv' '{' @Number @Number @Number '}'"`
	RGBColor   *string  `parser:"| '{' @Number @Number @Number (@Number)? '}'"`
	Variable   *string  `parser:"| @'@' @Ident"`
	Boolean    *string  `parser:"| @('yes'|'no')"`
	Identifier *string  `parser:"| @Ident ( @Ident )?"`
	String     *string  `parser:"| @String"`
	Number     *float64 `parser:"| @Number"`
	Array      []*Value `parser:"| '{' ( @@ )* '}'"`
}

type Comment struct {
	Positioned
	Text string
}

func (c *Comment) Capture(values []string) error {
	if len(values) > 0 {
		c.Text = values[0]
	}
	return nil
}

type Entry struct {
	Positioned
	Comment  *Comment `parser:"@Comment |"`
	Key      string   `parser:"@Ident"`
	Operator string   `parser:"@(EQ|LT|GT|LTE|GTE|NEQ|QEQ)"`
	Value    *Value   `parser:"( @@"`
	Block    *Block   `parser:"| @@ )"`
}

type Block struct {
	Positioned
	Entries []*Entry `parser:"'{' @@* '}'"`
}

type ParadoxFile struct {
	Positioned
	BOM     string   `parser:"@BOM?"`
	Entries []*Entry `parser:"@@*"`
}

func ParseFile(filename string) (*ParadoxFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return ParseString(string(data))
}

func ParseString(input string) (*ParadoxFile, error) {
	return paradoxParser.ParseString("", input)
}
