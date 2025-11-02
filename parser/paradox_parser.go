// Code generated from parser/Paradox.g4 by ANTLR 4.13.2. DO NOT EDIT.

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
		"", "", "", "", "'='", "'<'", "'>'", "'<='", "'>='", "'!='", "'?='",
		"'{'", "'}'", "':'", "'.'", "'\\uFEFF'",
	}
	staticData.SymbolicNames = []string{
		"", "ID", "NUMBER", "STRING", "EQ", "LT", "GT", "LTE", "GTE", "NEQ",
		"QEQ", "LBRACE", "RBRACE", "COLON", "DOT", "BOM", "COMMENT", "WS",
	}
	staticData.RuleNames = []string{
		"file", "line", "statement", "assignment", "key", "value", "valueAtom",
		"block", "blockContent", "operator",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 17, 114, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 1, 0, 3,
		0, 22, 8, 0, 1, 0, 5, 0, 25, 8, 0, 10, 0, 12, 0, 28, 9, 0, 1, 0, 1, 0,
		1, 1, 1, 1, 1, 1, 3, 1, 35, 8, 1, 1, 2, 1, 2, 1, 3, 1, 3, 5, 3, 41, 8,
		3, 10, 3, 12, 3, 44, 9, 3, 1, 3, 1, 3, 5, 3, 48, 8, 3, 10, 3, 12, 3, 51,
		9, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 4, 3, 4, 59, 8, 4, 1, 4, 1, 4, 3,
		4, 63, 8, 4, 1, 4, 1, 4, 4, 4, 67, 8, 4, 11, 4, 12, 4, 68, 1, 4, 3, 4,
		72, 8, 4, 1, 5, 1, 5, 1, 5, 5, 5, 77, 8, 5, 10, 5, 12, 5, 80, 9, 5, 1,
		5, 1, 5, 5, 5, 84, 8, 5, 10, 5, 12, 5, 87, 9, 5, 1, 5, 1, 5, 3, 5, 91,
		8, 5, 3, 5, 93, 8, 5, 1, 6, 1, 6, 1, 7, 1, 7, 5, 7, 99, 8, 7, 10, 7, 12,
		7, 102, 9, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1, 8, 1, 8, 3, 8, 110, 8, 8, 1, 9,
		1, 9, 1, 9, 0, 0, 10, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 0, 2, 1, 0, 1,
		3, 1, 0, 4, 10, 122, 0, 21, 1, 0, 0, 0, 2, 34, 1, 0, 0, 0, 4, 36, 1, 0,
		0, 0, 6, 38, 1, 0, 0, 0, 8, 71, 1, 0, 0, 0, 10, 92, 1, 0, 0, 0, 12, 94,
		1, 0, 0, 0, 14, 96, 1, 0, 0, 0, 16, 109, 1, 0, 0, 0, 18, 111, 1, 0, 0,
		0, 20, 22, 5, 15, 0, 0, 21, 20, 1, 0, 0, 0, 21, 22, 1, 0, 0, 0, 22, 26,
		1, 0, 0, 0, 23, 25, 3, 2, 1, 0, 24, 23, 1, 0, 0, 0, 25, 28, 1, 0, 0, 0,
		26, 24, 1, 0, 0, 0, 26, 27, 1, 0, 0, 0, 27, 29, 1, 0, 0, 0, 28, 26, 1,
		0, 0, 0, 29, 30, 5, 0, 0, 1, 30, 1, 1, 0, 0, 0, 31, 35, 3, 4, 2, 0, 32,
		35, 5, 16, 0, 0, 33, 35, 5, 17, 0, 0, 34, 31, 1, 0, 0, 0, 34, 32, 1, 0,
		0, 0, 34, 33, 1, 0, 0, 0, 35, 3, 1, 0, 0, 0, 36, 37, 3, 6, 3, 0, 37, 5,
		1, 0, 0, 0, 38, 42, 3, 8, 4, 0, 39, 41, 5, 17, 0, 0, 40, 39, 1, 0, 0, 0,
		41, 44, 1, 0, 0, 0, 42, 40, 1, 0, 0, 0, 42, 43, 1, 0, 0, 0, 43, 45, 1,
		0, 0, 0, 44, 42, 1, 0, 0, 0, 45, 49, 3, 18, 9, 0, 46, 48, 5, 17, 0, 0,
		47, 46, 1, 0, 0, 0, 48, 51, 1, 0, 0, 0, 49, 47, 1, 0, 0, 0, 49, 50, 1,
		0, 0, 0, 50, 52, 1, 0, 0, 0, 51, 49, 1, 0, 0, 0, 52, 53, 3, 10, 5, 0, 53,
		7, 1, 0, 0, 0, 54, 72, 5, 1, 0, 0, 55, 58, 5, 1, 0, 0, 56, 57, 5, 13, 0,
		0, 57, 59, 5, 1, 0, 0, 58, 56, 1, 0, 0, 0, 58, 59, 1, 0, 0, 0, 59, 62,
		1, 0, 0, 0, 60, 61, 5, 14, 0, 0, 61, 63, 5, 1, 0, 0, 62, 60, 1, 0, 0, 0,
		62, 63, 1, 0, 0, 0, 63, 72, 1, 0, 0, 0, 64, 66, 5, 1, 0, 0, 65, 67, 5,
		17, 0, 0, 66, 65, 1, 0, 0, 0, 67, 68, 1, 0, 0, 0, 68, 66, 1, 0, 0, 0, 68,
		69, 1, 0, 0, 0, 69, 70, 1, 0, 0, 0, 70, 72, 5, 1, 0, 0, 71, 54, 1, 0, 0,
		0, 71, 55, 1, 0, 0, 0, 71, 64, 1, 0, 0, 0, 72, 9, 1, 0, 0, 0, 73, 93, 3,
		14, 7, 0, 74, 90, 3, 12, 6, 0, 75, 77, 5, 17, 0, 0, 76, 75, 1, 0, 0, 0,
		77, 80, 1, 0, 0, 0, 78, 76, 1, 0, 0, 0, 78, 79, 1, 0, 0, 0, 79, 81, 1,
		0, 0, 0, 80, 78, 1, 0, 0, 0, 81, 85, 3, 18, 9, 0, 82, 84, 5, 17, 0, 0,
		83, 82, 1, 0, 0, 0, 84, 87, 1, 0, 0, 0, 85, 83, 1, 0, 0, 0, 85, 86, 1,
		0, 0, 0, 86, 88, 1, 0, 0, 0, 87, 85, 1, 0, 0, 0, 88, 89, 3, 12, 6, 0, 89,
		91, 1, 0, 0, 0, 90, 78, 1, 0, 0, 0, 90, 91, 1, 0, 0, 0, 91, 93, 1, 0, 0,
		0, 92, 73, 1, 0, 0, 0, 92, 74, 1, 0, 0, 0, 93, 11, 1, 0, 0, 0, 94, 95,
		7, 0, 0, 0, 95, 13, 1, 0, 0, 0, 96, 100, 5, 11, 0, 0, 97, 99, 3, 16, 8,
		0, 98, 97, 1, 0, 0, 0, 99, 102, 1, 0, 0, 0, 100, 98, 1, 0, 0, 0, 100, 101,
		1, 0, 0, 0, 101, 103, 1, 0, 0, 0, 102, 100, 1, 0, 0, 0, 103, 104, 5, 12,
		0, 0, 104, 15, 1, 0, 0, 0, 105, 110, 3, 4, 2, 0, 106, 110, 3, 10, 5, 0,
		107, 110, 5, 16, 0, 0, 108, 110, 5, 17, 0, 0, 109, 105, 1, 0, 0, 0, 109,
		106, 1, 0, 0, 0, 109, 107, 1, 0, 0, 0, 109, 108, 1, 0, 0, 0, 110, 17, 1,
		0, 0, 0, 111, 112, 7, 1, 0, 0, 112, 19, 1, 0, 0, 0, 15, 21, 26, 34, 42,
		49, 58, 62, 68, 71, 78, 85, 90, 92, 100, 109,
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
	ParadoxParserEOF     = antlr.TokenEOF
	ParadoxParserID      = 1
	ParadoxParserNUMBER  = 2
	ParadoxParserSTRING  = 3
	ParadoxParserEQ      = 4
	ParadoxParserLT      = 5
	ParadoxParserGT      = 6
	ParadoxParserLTE     = 7
	ParadoxParserGTE     = 8
	ParadoxParserNEQ     = 9
	ParadoxParserQEQ     = 10
	ParadoxParserLBRACE  = 11
	ParadoxParserRBRACE  = 12
	ParadoxParserCOLON   = 13
	ParadoxParserDOT     = 14
	ParadoxParserBOM     = 15
	ParadoxParserCOMMENT = 16
	ParadoxParserWS      = 17
)

// ParadoxParser rules.
const (
	ParadoxParserRULE_file         = 0
	ParadoxParserRULE_line         = 1
	ParadoxParserRULE_statement    = 2
	ParadoxParserRULE_assignment   = 3
	ParadoxParserRULE_key          = 4
	ParadoxParserRULE_value        = 5
	ParadoxParserRULE_valueAtom    = 6
	ParadoxParserRULE_block        = 7
	ParadoxParserRULE_blockContent = 8
	ParadoxParserRULE_operator     = 9
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
	p.SetState(21)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ParadoxParserBOM {
		{
			p.SetState(20)
			p.Match(ParadoxParserBOM)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(26)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&196610) != 0 {
		{
			p.SetState(23)
			p.Line()
		}

		p.SetState(28)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(29)
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
	p.SetState(34)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ParadoxParserID:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(31)
			p.Statement()
		}

	case ParadoxParserCOMMENT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(32)
			p.Match(ParadoxParserCOMMENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ParadoxParserWS:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(33)
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
		p.SetState(36)
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
		p.SetState(38)
		p.Key()
	}
	p.SetState(42)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(39)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(44)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(45)
		p.Operator()
	}
	p.SetState(49)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ParadoxParserWS {
		{
			p.SetState(46)
			p.Match(ParadoxParserWS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(51)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(52)
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

	p.SetState(71)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(54)
			p.Match(ParadoxParserID)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(55)
			p.Match(ParadoxParserID)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(58)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ParadoxParserCOLON {
			{
				p.SetState(56)
				p.Match(ParadoxParserCOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(57)
				p.Match(ParadoxParserID)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}
		p.SetState(62)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ParadoxParserDOT {
			{
				p.SetState(60)
				p.Match(ParadoxParserDOT)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(61)
				p.Match(ParadoxParserID)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(64)
			p.Match(ParadoxParserID)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(66)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for ok := true; ok; ok = _la == ParadoxParserWS {
			{
				p.SetState(65)
				p.Match(ParadoxParserWS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

			p.SetState(68)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(70)
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

	p.SetState(92)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ParadoxParserLBRACE:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(73)
			p.Block()
		}

	case ParadoxParserID, ParadoxParserNUMBER, ParadoxParserSTRING:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(74)
			p.ValueAtom()
		}
		p.SetState(90)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 11, p.GetParserRuleContext()) == 1 {
			p.SetState(78)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == ParadoxParserWS {
				{
					p.SetState(75)
					p.Match(ParadoxParserWS)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}

				p.SetState(80)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(81)
				p.Operator()
			}
			p.SetState(85)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == ParadoxParserWS {
				{
					p.SetState(82)
					p.Match(ParadoxParserWS)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}

				p.SetState(87)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(88)
				p.ValueAtom()
			}

		} else if p.HasError() { // JIM
			goto errorExit
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

// IValueAtomContext is an interface to support dynamic dispatch.
type IValueAtomContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NUMBER() antlr.TerminalNode
	STRING() antlr.TerminalNode
	ID() antlr.TerminalNode

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
		p.SetState(94)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&14) != 0) {
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
	p.EnterRule(localctx, 14, ParadoxParserRULE_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(96)
		p.Match(ParadoxParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(100)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&198670) != 0 {
		{
			p.SetState(97)
			p.BlockContent()
		}

		p.SetState(102)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(103)
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
	p.EnterRule(localctx, 16, ParadoxParserRULE_blockContent)
	p.SetState(109)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(105)
			p.Statement()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(106)
			p.Value()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(107)
			p.Match(ParadoxParserCOMMENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(108)
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
	p.EnterRule(localctx, 18, ParadoxParserRULE_operator)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(111)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&2032) != 0) {
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
