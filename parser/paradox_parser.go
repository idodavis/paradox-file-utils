// Code generated from Paradox.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Paradox

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type ParadoxParser struct {
	*antlr.BaseParser
}

var ParadoxParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func paradoxParserInit() {
	staticData := &ParadoxParserStaticData
	staticData.LiteralNames = []string{
		"", "", "", "", "", "'='", "'<'", "'>'", "'<='", "'>='", "'!='", "'?='",
		"'{'", "'}'", "':'", "'.'", "'hsv'", "'\\uFEFF'",
	}
	staticData.SymbolicNames = []string{
		"", "ID", "NUMBER", "STRING", "VARIABLE", "EQ", "LT", "GT", "LTE", "GTE",
		"NEQ", "QEQ", "LBRACE", "RBRACE", "COLON", "DOT", "HSV", "BOM", "COMMENT",
		"WS",
	}
	staticData.RuleNames = []string{
		"file", "line", "statement", "assignment", "key", "value", "valueAtom",
		"color_block", "hsv_block", "rgb_or_rgba_block", "block", "blockContent",
		"operator",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 19, 197, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 1, 0, 3, 0, 28, 8, 0, 1, 0, 5, 0, 31, 8,
		0, 10, 0, 12, 0, 34, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 3, 1, 41, 8, 1,
		1, 2, 1, 2, 1, 3, 1, 3, 5, 3, 47, 8, 3, 10, 3, 12, 3, 50, 9, 3, 1, 3, 1,
		3, 5, 3, 54, 8, 3, 10, 3, 12, 3, 57, 9, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4,
		1, 4, 1, 4, 3, 4, 66, 8, 4, 1, 4, 1, 4, 3, 4, 70, 8, 4, 1, 4, 1, 4, 4,
		4, 74, 8, 4, 11, 4, 12, 4, 75, 1, 4, 3, 4, 79, 8, 4, 1, 5, 1, 5, 1, 5,
		1, 5, 5, 5, 85, 8, 5, 10, 5, 12, 5, 88, 9, 5, 1, 5, 1, 5, 5, 5, 92, 8,
		5, 10, 5, 12, 5, 95, 9, 5, 1, 5, 1, 5, 3, 5, 99, 8, 5, 3, 5, 101, 8, 5,
		1, 6, 1, 6, 1, 7, 1, 7, 3, 7, 107, 8, 7, 1, 8, 1, 8, 5, 8, 111, 8, 8, 10,
		8, 12, 8, 114, 9, 8, 1, 8, 1, 8, 5, 8, 118, 8, 8, 10, 8, 12, 8, 121, 9,
		8, 1, 8, 1, 8, 4, 8, 125, 8, 8, 11, 8, 12, 8, 126, 1, 8, 1, 8, 4, 8, 131,
		8, 8, 11, 8, 12, 8, 132, 1, 8, 1, 8, 5, 8, 137, 8, 8, 10, 8, 12, 8, 140,
		9, 8, 1, 8, 1, 8, 1, 9, 1, 9, 5, 9, 146, 8, 9, 10, 9, 12, 9, 149, 9, 9,
		1, 9, 1, 9, 4, 9, 153, 8, 9, 11, 9, 12, 9, 154, 1, 9, 1, 9, 4, 9, 159,
		8, 9, 11, 9, 12, 9, 160, 1, 9, 1, 9, 4, 9, 165, 8, 9, 11, 9, 12, 9, 166,
		1, 9, 3, 9, 170, 8, 9, 1, 9, 5, 9, 173, 8, 9, 10, 9, 12, 9, 176, 9, 9,
		1, 9, 1, 9, 1, 10, 1, 10, 5, 10, 182, 8, 10, 10, 10, 12, 10, 185, 9, 10,
		1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 11, 3, 11, 193, 8, 11, 1, 12, 1,
		12, 1, 12, 0, 0, 13, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 0,
		2, 1, 0, 1, 4, 1, 0, 5, 11, 216, 0, 27, 1, 0, 0, 0, 2, 40, 1, 0, 0, 0,
		4, 42, 1, 0, 0, 0, 6, 44, 1, 0, 0, 0, 8, 78, 1, 0, 0, 0, 10, 100, 1, 0,
		0, 0, 12, 102, 1, 0, 0, 0, 14, 106, 1, 0, 0, 0, 16, 108, 1, 0, 0, 0, 18,
		143, 1, 0, 0, 0, 20, 179, 1, 0, 0, 0, 22, 192, 1, 0, 0, 0, 24, 194, 1,
		0, 0, 0, 26, 28, 5, 17, 0, 0, 27, 26, 1, 0, 0, 0, 27, 28, 1, 0, 0, 0, 28,
		32, 1, 0, 0, 0, 29, 31, 3, 2, 1, 0, 30, 29, 1, 0, 0, 0, 31, 34, 1, 0, 0,
		0, 32, 30, 1, 0, 0, 0, 32, 33, 1, 0, 0, 0, 33, 35, 1, 0, 0, 0, 34, 32,
		1, 0, 0, 0, 35, 36, 5, 0, 0, 1, 36, 1, 1, 0, 0, 0, 37, 41, 3, 4, 2, 0,
		38, 41, 5, 18, 0, 0, 39, 41, 5, 19, 0, 0, 40, 37, 1, 0, 0, 0, 40, 38, 1,
		0, 0, 0, 40, 39, 1, 0, 0, 0, 41, 3, 1, 0, 0, 0, 42, 43, 3, 6, 3, 0, 43,
		5, 1, 0, 0, 0, 44, 48, 3, 8, 4, 0, 45, 47, 5, 19, 0, 0, 46, 45, 1, 0, 0,
		0, 47, 50, 1, 0, 0, 0, 48, 46, 1, 0, 0, 0, 48, 49, 1, 0, 0, 0, 49, 51,
		1, 0, 0, 0, 50, 48, 1, 0, 0, 0, 51, 55, 3, 24, 12, 0, 52, 54, 5, 19, 0,
		0, 53, 52, 1, 0, 0, 0, 54, 57, 1, 0, 0, 0, 55, 53, 1, 0, 0, 0, 55, 56,
		1, 0, 0, 0, 56, 58, 1, 0, 0, 0, 57, 55, 1, 0, 0, 0, 58, 59, 3, 10, 5, 0,
		59, 7, 1, 0, 0, 0, 60, 79, 5, 1, 0, 0, 61, 79, 5, 4, 0, 0, 62, 65, 5, 1,
		0, 0, 63, 64, 5, 14, 0, 0, 64, 66, 5, 1, 0, 0, 65, 63, 1, 0, 0, 0, 65,
		66, 1, 0, 0, 0, 66, 69, 1, 0, 0, 0, 67, 68, 5, 15, 0, 0, 68, 70, 5, 1,
		0, 0, 69, 67, 1, 0, 0, 0, 69, 70, 1, 0, 0, 0, 70, 79, 1, 0, 0, 0, 71, 73,
		5, 1, 0, 0, 72, 74, 5, 19, 0, 0, 73, 72, 1, 0, 0, 0, 74, 75, 1, 0, 0, 0,
		75, 73, 1, 0, 0, 0, 75, 76, 1, 0, 0, 0, 76, 77, 1, 0, 0, 0, 77, 79, 5,
		1, 0, 0, 78, 60, 1, 0, 0, 0, 78, 61, 1, 0, 0, 0, 78, 62, 1, 0, 0, 0, 78,
		71, 1, 0, 0, 0, 79, 9, 1, 0, 0, 0, 80, 101, 3, 14, 7, 0, 81, 101, 3, 20,
		10, 0, 82, 98, 3, 12, 6, 0, 83, 85, 5, 19, 0, 0, 84, 83, 1, 0, 0, 0, 85,
		88, 1, 0, 0, 0, 86, 84, 1, 0, 0, 0, 86, 87, 1, 0, 0, 0, 87, 89, 1, 0, 0,
		0, 88, 86, 1, 0, 0, 0, 89, 93, 3, 24, 12, 0, 90, 92, 5, 19, 0, 0, 91, 90,
		1, 0, 0, 0, 92, 95, 1, 0, 0, 0, 93, 91, 1, 0, 0, 0, 93, 94, 1, 0, 0, 0,
		94, 96, 1, 0, 0, 0, 95, 93, 1, 0, 0, 0, 96, 97, 3, 12, 6, 0, 97, 99, 1,
		0, 0, 0, 98, 86, 1, 0, 0, 0, 98, 99, 1, 0, 0, 0, 99, 101, 1, 0, 0, 0, 100,
		80, 1, 0, 0, 0, 100, 81, 1, 0, 0, 0, 100, 82, 1, 0, 0, 0, 101, 11, 1, 0,
		0, 0, 102, 103, 7, 0, 0, 0, 103, 13, 1, 0, 0, 0, 104, 107, 3, 16, 8, 0,
		105, 107, 3, 18, 9, 0, 106, 104, 1, 0, 0, 0, 106, 105, 1, 0, 0, 0, 107,
		15, 1, 0, 0, 0, 108, 112, 5, 16, 0, 0, 109, 111, 5, 19, 0, 0, 110, 109,
		1, 0, 0, 0, 111, 114, 1, 0, 0, 0, 112, 110, 1, 0, 0, 0, 112, 113, 1, 0,
		0, 0, 113, 115, 1, 0, 0, 0, 114, 112, 1, 0, 0, 0, 115, 119, 5, 12, 0, 0,
		116, 118, 5, 19, 0, 0, 117, 116, 1, 0, 0, 0, 118, 121, 1, 0, 0, 0, 119,
		117, 1, 0, 0, 0, 119, 120, 1, 0, 0, 0, 120, 122, 1, 0, 0, 0, 121, 119,
		1, 0, 0, 0, 122, 124, 5, 2, 0, 0, 123, 125, 5, 19, 0, 0, 124, 123, 1, 0,
		0, 0, 125, 126, 1, 0, 0, 0, 126, 124, 1, 0, 0, 0, 126, 127, 1, 0, 0, 0,
		127, 128, 1, 0, 0, 0, 128, 130, 5, 2, 0, 0, 129, 131, 5, 19, 0, 0, 130,
		129, 1, 0, 0, 0, 131, 132, 1, 0, 0, 0, 132, 130, 1, 0, 0, 0, 132, 133,
		1, 0, 0, 0, 133, 134, 1, 0, 0, 0, 134, 138, 5, 2, 0, 0, 135, 137, 5, 19,
		0, 0, 136, 135, 1, 0, 0, 0, 137, 140, 1, 0, 0, 0, 138, 136, 1, 0, 0, 0,
		138, 139, 1, 0, 0, 0, 139, 141, 1, 0, 0, 0, 140, 138, 1, 0, 0, 0, 141,
		142, 5, 13, 0, 0, 142, 17, 1, 0, 0, 0, 143, 147, 5, 12, 0, 0, 144, 146,
		5, 19, 0, 0, 145, 144, 1, 0, 0, 0, 146, 149, 1, 0, 0, 0, 147, 145, 1, 0,
		0, 0, 147, 148, 1, 0, 0, 0, 148, 150, 1, 0, 0, 0, 149, 147, 1, 0, 0, 0,
		150, 152, 5, 2, 0, 0, 151, 153, 5, 19, 0, 0, 152, 151, 1, 0, 0, 0, 153,
		154, 1, 0, 0, 0, 154, 152, 1, 0, 0, 0, 154, 155, 1, 0, 0, 0, 155, 156,
		1, 0, 0, 0, 156, 158, 5, 2, 0, 0, 157, 159, 5, 19, 0, 0, 158, 157, 1, 0,
		0, 0, 159, 160, 1, 0, 0, 0, 160, 158, 1, 0, 0, 0, 160, 161, 1, 0, 0, 0,
		161, 162, 1, 0, 0, 0, 162, 169, 5, 2, 0, 0, 163, 165, 5, 19, 0, 0, 164,
		163, 1, 0, 0, 0, 165, 166, 1, 0, 0, 0, 166, 164, 1, 0, 0, 0, 166, 167,
		1, 0, 0, 0, 167, 168, 1, 0, 0, 0, 168, 170, 5, 2, 0, 0, 169, 164, 1, 0,
		0, 0, 169, 170, 1, 0, 0, 0, 170, 174, 1, 0, 0, 0, 171, 173, 5, 19, 0, 0,
		172, 171, 1, 0, 0, 0, 173, 176, 1, 0, 0, 0, 174, 172, 1, 0, 0, 0, 174,
		175, 1, 0, 0, 0, 175, 177, 1, 0, 0, 0, 176, 174, 1, 0, 0, 0, 177, 178,
		5, 13, 0, 0, 178, 19, 1, 0, 0, 0, 179, 183, 5, 12, 0, 0, 180, 182, 3, 22,
		11, 0, 181, 180, 1, 0, 0, 0, 182, 185, 1, 0, 0, 0, 183, 181, 1, 0, 0, 0,
		183, 184, 1, 0, 0, 0, 184, 186, 1, 0, 0, 0, 185, 183, 1, 0, 0, 0, 186,
		187, 5, 13, 0, 0, 187, 21, 1, 0, 0, 0, 188, 193, 3, 4, 2, 0, 189, 193,
		3, 10, 5, 0, 190, 193, 5, 18, 0, 0, 191, 193, 5, 19, 0, 0, 192, 188, 1,
		0, 0, 0, 192, 189, 1, 0, 0, 0, 192, 190, 1, 0, 0, 0, 192, 191, 1, 0, 0,
		0, 193, 23, 1, 0, 0, 0, 194, 195, 7, 1, 0, 0, 195, 25, 1, 0, 0, 0, 27,
		27, 32, 40, 48, 55, 65, 69, 75, 78, 86, 93, 98, 100, 106, 112, 119, 126,
		132, 138, 147, 154, 160, 166, 169, 174, 183, 192,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// ParadoxParserInit initializes any static state used to implement ParadoxParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewParadoxParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func ParadoxParserInit() {
	staticData := &ParadoxParserStaticData
	staticData.once.Do(paradoxParserInit)
}

// NewParadoxParser produces a new parser instance for the optional input antlr.TokenStream.
func NewParadoxParser(input antlr.TokenStream) *ParadoxParser {
	ParadoxParserInit()
	this := new(ParadoxParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &ParadoxParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "Paradox.g4"

	return this
}

// ParadoxParser tokens.
const (
	ParadoxParserEOF      = antlr.TokenEOF
	ParadoxParserID       = 1
	ParadoxParserNUMBER   = 2
	ParadoxParserSTRING   = 3
	ParadoxParserVARIABLE = 4
	ParadoxParserEQ       = 5
	ParadoxParserLT       = 6
	ParadoxParserGT       = 7
	ParadoxParserLTE      = 8
	ParadoxParserGTE      = 9
	ParadoxParserNEQ      = 10
	ParadoxParserQEQ      = 11
	ParadoxParserLBRACE   = 12
	ParadoxParserRBRACE   = 13
	ParadoxParserCOLON    = 14
	ParadoxParserDOT      = 15
	ParadoxParserHSV      = 16
	ParadoxParserBOM      = 17
	ParadoxParserCOMMENT  = 18
	ParadoxParserWS       = 19
)

// ParadoxParser rules.
const (
	ParadoxParserRULE_file              = 0
	ParadoxParserRULE_line              = 1
	ParadoxParserRULE_statement         = 2
	ParadoxParserRULE_assignment        = 3
	ParadoxParserRULE_key               = 4
	ParadoxParserRULE_value             = 5
	ParadoxParserRULE_valueAtom         = 6
	ParadoxParserRULE_color_block       = 7
	ParadoxParserRULE_hsv_block         = 8
	ParadoxParserRULE_rgb_or_rgba_block = 9
	ParadoxParserRULE_block             = 10
	ParadoxParserRULE_blockContent      = 11
	ParadoxParserRULE_operator          = 12
)

// IFileContext is an interface to support dynamic dispatch.
type IFileContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EOF() antlr.TerminalNode
	BOM() antlr.TerminalNode
	AllLine() []ILineContext
	Line(i int) ILineContext

	// IsFileContext differentiates from other interfaces.
	IsFileContext()
}

type FileContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFileContext() *FileContext {
	var p = new(FileContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_file
	return p
}

func InitEmptyFileContext(p *FileContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_file
}

func (*FileContext) IsFileContext() {}

func NewFileContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FileContext {
	var p = new(FileContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_file

	return p
}

func (s *FileContext) GetParser() antlr.Parser { return s.parser }

func (s *FileContext) EOF() antlr.TerminalNode {
	return s.GetToken(ParadoxParserEOF, 0)
}

func (s *FileContext) BOM() antlr.TerminalNode {
	return s.GetToken(ParadoxParserBOM, 0)
}

func (s *FileContext) AllLine() []ILineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ILineContext); ok {
			len++
		}
	}

	tst := make([]ILineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ILineContext); ok {
			tst[i] = t.(ILineContext)
			i++
		}
	}

	return tst
}

func (s *FileContext) Line(i int) ILineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILineContext)
}

func (s *FileContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FileContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FileContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterFile(s)
	}
}

func (s *FileContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitFile(s)
	}
}

func (p *ParadoxParser) File() (localctx IFileContext) {
	localctx = NewFileContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, ParadoxParserRULE_file)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(27)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ParadoxParserBOM {
		{
			p.SetState(26)
			p.Match(ParadoxParserBOM)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(32)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&786450) != 0 {
		{
			p.SetState(29)
			p.Line()
		}

		p.SetState(34)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(35)
		p.Match(ParadoxParserEOF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILineContext is an interface to support dynamic dispatch.
type ILineContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Statement() IStatementContext
	COMMENT() antlr.TerminalNode
	WS() antlr.TerminalNode

	// IsLineContext differentiates from other interfaces.
	IsLineContext()
}

type LineContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLineContext() *LineContext {
	var p = new(LineContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_line
	return p
}

func InitEmptyLineContext(p *LineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_line
}

func (*LineContext) IsLineContext() {}

func NewLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LineContext {
	var p = new(LineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_line

	return p
}

func (s *LineContext) GetParser() antlr.Parser { return s.parser }

func (s *LineContext) Statement() IStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStatementContext)
}

func (s *LineContext) COMMENT() antlr.TerminalNode {
	return s.GetToken(ParadoxParserCOMMENT, 0)
}

func (s *LineContext) WS() antlr.TerminalNode {
	return s.GetToken(ParadoxParserWS, 0)
}

func (s *LineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterLine(s)
	}
}

func (s *LineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitLine(s)
	}
}

func (p *ParadoxParser) Line() (localctx ILineContext) {
	localctx = NewLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, ParadoxParserRULE_line)
	p.SetState(40)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ParadoxParserID, ParadoxParserVARIABLE:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(37)
			p.Statement()
		}

	case ParadoxParserCOMMENT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(38)
			p.Match(ParadoxParserCOMMENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ParadoxParserWS:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(39)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStatementContext is an interface to support dynamic dispatch.
type IStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Assignment() IAssignmentContext

	// IsStatementContext differentiates from other interfaces.
	IsStatementContext()
}

type StatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStatementContext() *StatementContext {
	var p = new(StatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_statement
	return p
}

func InitEmptyStatementContext(p *StatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_statement
}

func (*StatementContext) IsStatementContext() {}

func NewStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StatementContext {
	var p = new(StatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_statement

	return p
}

func (s *StatementContext) GetParser() antlr.Parser { return s.parser }

func (s *StatementContext) Assignment() IAssignmentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAssignmentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAssignmentContext)
}

func (s *StatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterStatement(s)
	}
}

func (s *StatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitStatement(s)
	}
}

func (p *ParadoxParser) Statement() (localctx IStatementContext) {
	localctx = NewStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, ParadoxParserRULE_statement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(42)
		p.Assignment()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAssignmentContext is an interface to support dynamic dispatch.
type IAssignmentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Key() IKeyContext
	Operator() IOperatorContext
	Value() IValueContext
	AllWS() []antlr.TerminalNode
	WS(i int) antlr.TerminalNode

	// IsAssignmentContext differentiates from other interfaces.
	IsAssignmentContext()
}

type AssignmentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAssignmentContext() *AssignmentContext {
	var p = new(AssignmentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_assignment
	return p
}

func InitEmptyAssignmentContext(p *AssignmentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_assignment
}

func (*AssignmentContext) IsAssignmentContext() {}

func NewAssignmentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AssignmentContext {
	var p = new(AssignmentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_assignment

	return p
}

func (s *AssignmentContext) GetParser() antlr.Parser { return s.parser }

func (s *AssignmentContext) Key() IKeyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IKeyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IKeyContext)
}

func (s *AssignmentContext) Operator() IOperatorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOperatorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOperatorContext)
}

func (s *AssignmentContext) Value() IValueContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IValueContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IValueContext)
}

func (s *AssignmentContext) AllWS() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserWS)
}

func (s *AssignmentContext) WS(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserWS, i)
}

func (s *AssignmentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AssignmentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AssignmentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterAssignment(s)
	}
}

func (s *AssignmentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitAssignment(s)
	}
}

func (p *ParadoxParser) Assignment() (localctx IAssignmentContext) {
	localctx = NewAssignmentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, ParadoxParserRULE_assignment)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(44)
		p.Key()
	}
	p.SetState(48)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(45)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(50)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(51)
		p.Operator()
	}
	p.SetState(55)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(52)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(57)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(58)
		p.Value()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IKeyContext is an interface to support dynamic dispatch.
type IKeyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllID() []antlr.TerminalNode
	ID(i int) antlr.TerminalNode
	VARIABLE() antlr.TerminalNode
	COLON() antlr.TerminalNode
	DOT() antlr.TerminalNode
	AllWS() []antlr.TerminalNode
	WS(i int) antlr.TerminalNode

	// IsKeyContext differentiates from other interfaces.
	IsKeyContext()
}

type KeyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyKeyContext() *KeyContext {
	var p = new(KeyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_key
	return p
}

func InitEmptyKeyContext(p *KeyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_key
}

func (*KeyContext) IsKeyContext() {}

func NewKeyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *KeyContext {
	var p = new(KeyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_key

	return p
}

func (s *KeyContext) GetParser() antlr.Parser { return s.parser }

func (s *KeyContext) AllID() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserID)
}

func (s *KeyContext) ID(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserID, i)
}

func (s *KeyContext) VARIABLE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserVARIABLE, 0)
}

func (s *KeyContext) COLON() antlr.TerminalNode {
	return s.GetToken(ParadoxParserCOLON, 0)
}

func (s *KeyContext) DOT() antlr.TerminalNode {
	return s.GetToken(ParadoxParserDOT, 0)
}

func (s *KeyContext) AllWS() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserWS)
}

func (s *KeyContext) WS(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserWS, i)
}

func (s *KeyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *KeyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *KeyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterKey(s)
	}
}

func (s *KeyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitKey(s)
	}
}

func (p *ParadoxParser) Key() (localctx IKeyContext) {
	localctx = NewKeyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, ParadoxParserRULE_key)
	var _la int

	p.SetState(78)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(60)
			p.Match(ParadoxParserID)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(61)
			p.Match(ParadoxParserVARIABLE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(62)
			p.Match(ParadoxParserID)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(65)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ParadoxParserCOLON {
			{
				p.SetState(63)
				p.Match(ParadoxParserCOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(64)
				p.Match(ParadoxParserID)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}
		p.SetState(69)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ParadoxParserDOT {
			{
				p.SetState(67)
				p.Match(ParadoxParserDOT)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(68)
				p.Match(ParadoxParserID)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(71)
			p.Match(ParadoxParserID)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(73)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for ok := true; ok; ok = _la == ParadoxParserWS {
			{
				p.SetState(72)
				p.Match(ParadoxParserWS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

			p.SetState(75)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(77)
			p.Match(ParadoxParserID)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IValueContext is an interface to support dynamic dispatch.
type IValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Color_block() IColor_blockContext
	Block() IBlockContext
	AllValueAtom() []IValueAtomContext
	ValueAtom(i int) IValueAtomContext
	Operator() IOperatorContext
	AllWS() []antlr.TerminalNode
	WS(i int) antlr.TerminalNode

	// IsValueContext differentiates from other interfaces.
	IsValueContext()
}

type ValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyValueContext() *ValueContext {
	var p = new(ValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_value
	return p
}

func InitEmptyValueContext(p *ValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_value
}

func (*ValueContext) IsValueContext() {}

func NewValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ValueContext {
	var p = new(ValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_value

	return p
}

func (s *ValueContext) GetParser() antlr.Parser { return s.parser }

func (s *ValueContext) Color_block() IColor_blockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IColor_blockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IColor_blockContext)
}

func (s *ValueContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ValueContext) AllValueAtom() []IValueAtomContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IValueAtomContext); ok {
			len++
		}
	}

	tst := make([]IValueAtomContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IValueAtomContext); ok {
			tst[i] = t.(IValueAtomContext)
			i++
		}
	}

	return tst
}

func (s *ValueContext) ValueAtom(i int) IValueAtomContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IValueAtomContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IValueAtomContext)
}

func (s *ValueContext) Operator() IOperatorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOperatorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOperatorContext)
}

func (s *ValueContext) AllWS() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserWS)
}

func (s *ValueContext) WS(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserWS, i)
}

func (s *ValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterValue(s)
	}
}

func (s *ValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitValue(s)
	}
}

func (p *ParadoxParser) Value() (localctx IValueContext) {
	localctx = NewValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, ParadoxParserRULE_value)
	var _la int

	p.SetState(100)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(80)
			p.Color_block()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(81)
			p.Block()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(82)
			p.ValueAtom()
		}
		p.SetState(98)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 11, p.GetParserRuleContext()) == 1 {
			p.SetState(86)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == ParadoxParserWS {
				{
					p.SetState(83)
					p.Match(ParadoxParserWS)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}

				p.SetState(88)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(89)
				p.Operator()
			}
			p.SetState(93)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == ParadoxParserWS {
				{
					p.SetState(90)
					p.Match(ParadoxParserWS)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}

				p.SetState(95)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(96)
				p.ValueAtom()
			}

		} else if p.HasError() { // JIM
			goto errorExit
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IValueAtomContext is an interface to support dynamic dispatch.
type IValueAtomContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NUMBER() antlr.TerminalNode
	STRING() antlr.TerminalNode
	ID() antlr.TerminalNode
	VARIABLE() antlr.TerminalNode

	// IsValueAtomContext differentiates from other interfaces.
	IsValueAtomContext()
}

type ValueAtomContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyValueAtomContext() *ValueAtomContext {
	var p = new(ValueAtomContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_valueAtom
	return p
}

func InitEmptyValueAtomContext(p *ValueAtomContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_valueAtom
}

func (*ValueAtomContext) IsValueAtomContext() {}

func NewValueAtomContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ValueAtomContext {
	var p = new(ValueAtomContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_valueAtom

	return p
}

func (s *ValueAtomContext) GetParser() antlr.Parser { return s.parser }

func (s *ValueAtomContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(ParadoxParserNUMBER, 0)
}

func (s *ValueAtomContext) STRING() antlr.TerminalNode {
	return s.GetToken(ParadoxParserSTRING, 0)
}

func (s *ValueAtomContext) ID() antlr.TerminalNode {
	return s.GetToken(ParadoxParserID, 0)
}

func (s *ValueAtomContext) VARIABLE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserVARIABLE, 0)
}

func (s *ValueAtomContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ValueAtomContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ValueAtomContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterValueAtom(s)
	}
}

func (s *ValueAtomContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitValueAtom(s)
	}
}

func (p *ParadoxParser) ValueAtom() (localctx IValueAtomContext) {
	localctx = NewValueAtomContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, ParadoxParserRULE_valueAtom)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(102)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&30) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IColor_blockContext is an interface to support dynamic dispatch.
type IColor_blockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Hsv_block() IHsv_blockContext
	Rgb_or_rgba_block() IRgb_or_rgba_blockContext

	// IsColor_blockContext differentiates from other interfaces.
	IsColor_blockContext()
}

type Color_blockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyColor_blockContext() *Color_blockContext {
	var p = new(Color_blockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_color_block
	return p
}

func InitEmptyColor_blockContext(p *Color_blockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_color_block
}

func (*Color_blockContext) IsColor_blockContext() {}

func NewColor_blockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Color_blockContext {
	var p = new(Color_blockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_color_block

	return p
}

func (s *Color_blockContext) GetParser() antlr.Parser { return s.parser }

func (s *Color_blockContext) Hsv_block() IHsv_blockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IHsv_blockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IHsv_blockContext)
}

func (s *Color_blockContext) Rgb_or_rgba_block() IRgb_or_rgba_blockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IRgb_or_rgba_blockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IRgb_or_rgba_blockContext)
}

func (s *Color_blockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Color_blockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Color_blockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterColor_block(s)
	}
}

func (s *Color_blockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitColor_block(s)
	}
}

func (p *ParadoxParser) Color_block() (localctx IColor_blockContext) {
	localctx = NewColor_blockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, ParadoxParserRULE_color_block)
	p.SetState(106)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ParadoxParserHSV:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(104)
			p.Hsv_block()
		}

	case ParadoxParserLBRACE:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(105)
			p.Rgb_or_rgba_block()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IHsv_blockContext is an interface to support dynamic dispatch.
type IHsv_blockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	HSV() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	AllNUMBER() []antlr.TerminalNode
	NUMBER(i int) antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllWS() []antlr.TerminalNode
	WS(i int) antlr.TerminalNode

	// IsHsv_blockContext differentiates from other interfaces.
	IsHsv_blockContext()
}

type Hsv_blockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyHsv_blockContext() *Hsv_blockContext {
	var p = new(Hsv_blockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_hsv_block
	return p
}

func InitEmptyHsv_blockContext(p *Hsv_blockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_hsv_block
}

func (*Hsv_blockContext) IsHsv_blockContext() {}

func NewHsv_blockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Hsv_blockContext {
	var p = new(Hsv_blockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_hsv_block

	return p
}

func (s *Hsv_blockContext) GetParser() antlr.Parser { return s.parser }

func (s *Hsv_blockContext) HSV() antlr.TerminalNode {
	return s.GetToken(ParadoxParserHSV, 0)
}

func (s *Hsv_blockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserLBRACE, 0)
}

func (s *Hsv_blockContext) AllNUMBER() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserNUMBER)
}

func (s *Hsv_blockContext) NUMBER(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserNUMBER, i)
}

func (s *Hsv_blockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserRBRACE, 0)
}

func (s *Hsv_blockContext) AllWS() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserWS)
}

func (s *Hsv_blockContext) WS(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserWS, i)
}

func (s *Hsv_blockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Hsv_blockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Hsv_blockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterHsv_block(s)
	}
}

func (s *Hsv_blockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitHsv_block(s)
	}
}

func (p *ParadoxParser) Hsv_block() (localctx IHsv_blockContext) {
	localctx = NewHsv_blockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, ParadoxParserRULE_hsv_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(108)
		p.Match(ParadoxParserHSV)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(112)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(109)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(114)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(115)
		p.Match(ParadoxParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(119)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(116)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(121)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(122)
		p.Match(ParadoxParserNUMBER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(124)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ParadoxParserWS {
		{
			p.SetState(123)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(126)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(128)
		p.Match(ParadoxParserNUMBER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(130)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ParadoxParserWS {
		{
			p.SetState(129)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(132)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(134)
		p.Match(ParadoxParserNUMBER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(138)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(135)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(140)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(141)
		p.Match(ParadoxParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IRgb_or_rgba_blockContext is an interface to support dynamic dispatch.
type IRgb_or_rgba_blockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	AllNUMBER() []antlr.TerminalNode
	NUMBER(i int) antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllWS() []antlr.TerminalNode
	WS(i int) antlr.TerminalNode

	// IsRgb_or_rgba_blockContext differentiates from other interfaces.
	IsRgb_or_rgba_blockContext()
}

type Rgb_or_rgba_blockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyRgb_or_rgba_blockContext() *Rgb_or_rgba_blockContext {
	var p = new(Rgb_or_rgba_blockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_rgb_or_rgba_block
	return p
}

func InitEmptyRgb_or_rgba_blockContext(p *Rgb_or_rgba_blockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_rgb_or_rgba_block
}

func (*Rgb_or_rgba_blockContext) IsRgb_or_rgba_blockContext() {}

func NewRgb_or_rgba_blockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Rgb_or_rgba_blockContext {
	var p = new(Rgb_or_rgba_blockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_rgb_or_rgba_block

	return p
}

func (s *Rgb_or_rgba_blockContext) GetParser() antlr.Parser { return s.parser }

func (s *Rgb_or_rgba_blockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserLBRACE, 0)
}

func (s *Rgb_or_rgba_blockContext) AllNUMBER() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserNUMBER)
}

func (s *Rgb_or_rgba_blockContext) NUMBER(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserNUMBER, i)
}

func (s *Rgb_or_rgba_blockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserRBRACE, 0)
}

func (s *Rgb_or_rgba_blockContext) AllWS() []antlr.TerminalNode {
	return s.GetTokens(ParadoxParserWS)
}

func (s *Rgb_or_rgba_blockContext) WS(i int) antlr.TerminalNode {
	return s.GetToken(ParadoxParserWS, i)
}

func (s *Rgb_or_rgba_blockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Rgb_or_rgba_blockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Rgb_or_rgba_blockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterRgb_or_rgba_block(s)
	}
}

func (s *Rgb_or_rgba_blockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitRgb_or_rgba_block(s)
	}
}

func (p *ParadoxParser) Rgb_or_rgba_block() (localctx IRgb_or_rgba_blockContext) {
	localctx = NewRgb_or_rgba_blockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, ParadoxParserRULE_rgb_or_rgba_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(143)
		p.Match(ParadoxParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(147)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(144)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(149)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(150)
		p.Match(ParadoxParserNUMBER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(152)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ParadoxParserWS {
		{
			p.SetState(151)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(154)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(156)
		p.Match(ParadoxParserNUMBER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(158)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ParadoxParserWS {
		{
			p.SetState(157)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(160)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(162)
		p.Match(ParadoxParserNUMBER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(169)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 23, p.GetParserRuleContext()) == 1 {
		p.SetState(164)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for ok := true; ok; ok = _la == ParadoxParserWS {
			{
				p.SetState(163)
				p.Match(ParadoxParserWS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

			p.SetState(166)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(168)
			p.Match(ParadoxParserNUMBER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}
	p.SetState(174)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(171)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(176)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(177)
		p.Match(ParadoxParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockContext is an interface to support dynamic dispatch.
type IBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllBlockContent() []IBlockContentContext
	BlockContent(i int) IBlockContentContext

	// IsBlockContext differentiates from other interfaces.
	IsBlockContext()
}

type BlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockContext() *BlockContext {
	var p = new(BlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_block
	return p
}

func InitEmptyBlockContext(p *BlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_block
}

func (*BlockContext) IsBlockContext() {}

func NewBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockContext {
	var p = new(BlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_block

	return p
}

func (s *BlockContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserLBRACE, 0)
}

func (s *BlockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserRBRACE, 0)
}

func (s *BlockContext) AllBlockContent() []IBlockContentContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IBlockContentContext); ok {
			len++
		}
	}

	tst := make([]IBlockContentContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IBlockContentContext); ok {
			tst[i] = t.(IBlockContentContext)
			i++
		}
	}

	return tst
}

func (s *BlockContext) BlockContent(i int) IBlockContentContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContentContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContentContext)
}

func (s *BlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterBlock(s)
	}
}

func (s *BlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitBlock(s)
	}
}

func (p *ParadoxParser) Block() (localctx IBlockContext) {
	localctx = NewBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, ParadoxParserRULE_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(179)
		p.Match(ParadoxParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(183)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&856094) != 0 {
		{
			p.SetState(180)
			p.BlockContent()
		}

		p.SetState(185)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(186)
		p.Match(ParadoxParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockContentContext is an interface to support dynamic dispatch.
type IBlockContentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Statement() IStatementContext
	Value() IValueContext
	COMMENT() antlr.TerminalNode
	WS() antlr.TerminalNode

	// IsBlockContentContext differentiates from other interfaces.
	IsBlockContentContext()
}

type BlockContentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockContentContext() *BlockContentContext {
	var p = new(BlockContentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_blockContent
	return p
}

func InitEmptyBlockContentContext(p *BlockContentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_blockContent
}

func (*BlockContentContext) IsBlockContentContext() {}

func NewBlockContentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockContentContext {
	var p = new(BlockContentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_blockContent

	return p
}

func (s *BlockContentContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockContentContext) Statement() IStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStatementContext)
}

func (s *BlockContentContext) Value() IValueContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IValueContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IValueContext)
}

func (s *BlockContentContext) COMMENT() antlr.TerminalNode {
	return s.GetToken(ParadoxParserCOMMENT, 0)
}

func (s *BlockContentContext) WS() antlr.TerminalNode {
	return s.GetToken(ParadoxParserWS, 0)
}

func (s *BlockContentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockContentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockContentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterBlockContent(s)
	}
}

func (s *BlockContentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitBlockContent(s)
	}
}

func (p *ParadoxParser) BlockContent() (localctx IBlockContentContext) {
	localctx = NewBlockContentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, ParadoxParserRULE_blockContent)
	p.SetState(192)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 26, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(188)
			p.Statement()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(189)
			p.Value()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(190)
			p.Match(ParadoxParserCOMMENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(191)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IOperatorContext is an interface to support dynamic dispatch.
type IOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EQ() antlr.TerminalNode
	LT() antlr.TerminalNode
	GT() antlr.TerminalNode
	LTE() antlr.TerminalNode
	GTE() antlr.TerminalNode
	NEQ() antlr.TerminalNode
	QEQ() antlr.TerminalNode

	// IsOperatorContext differentiates from other interfaces.
	IsOperatorContext()
}

type OperatorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOperatorContext() *OperatorContext {
	var p = new(OperatorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_operator
	return p
}

func InitEmptyOperatorContext(p *OperatorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ParadoxParserRULE_operator
}

func (*OperatorContext) IsOperatorContext() {}

func NewOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OperatorContext {
	var p = new(OperatorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ParadoxParserRULE_operator

	return p
}

func (s *OperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *OperatorContext) EQ() antlr.TerminalNode {
	return s.GetToken(ParadoxParserEQ, 0)
}

func (s *OperatorContext) LT() antlr.TerminalNode {
	return s.GetToken(ParadoxParserLT, 0)
}

func (s *OperatorContext) GT() antlr.TerminalNode {
	return s.GetToken(ParadoxParserGT, 0)
}

func (s *OperatorContext) LTE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserLTE, 0)
}

func (s *OperatorContext) GTE() antlr.TerminalNode {
	return s.GetToken(ParadoxParserGTE, 0)
}

func (s *OperatorContext) NEQ() antlr.TerminalNode {
	return s.GetToken(ParadoxParserNEQ, 0)
}

func (s *OperatorContext) QEQ() antlr.TerminalNode {
	return s.GetToken(ParadoxParserQEQ, 0)
}

func (s *OperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.EnterOperator(s)
	}
}

func (s *OperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ParadoxListener); ok {
		listenerT.ExitOperator(s)
	}
}

func (p *ParadoxParser) Operator() (localctx IOperatorContext) {
	localctx = NewOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, ParadoxParserRULE_operator)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(194)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&4064) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
